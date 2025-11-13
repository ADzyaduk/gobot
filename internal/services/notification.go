// Package services contains notification logic
package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"gobot/internal/database"

	tele "gopkg.in/telebot.v3"
)

// NotificationService handles notifications and reminders
type NotificationService struct {
	bot       *tele.Bot
	adminIDs  []int64
	channelID string
}

// NewNotificationService creates a new notification service
func NewNotificationService(bot *tele.Bot, adminIDs []int64, channelID string) *NotificationService {
	return &NotificationService{
		bot:       bot,
		adminIDs:  adminIDs,
		channelID: channelID,
	}
}

// SendBookingConfirmation sends confirmation message to user
func (s *NotificationService) SendBookingConfirmation(ctx context.Context, booking *database.Booking) error {
	msg := fmt.Sprintf(
		"✅ <b>Ваша запись подтверждена!</b>\n\n"+
			"📋 Услуга: <b>%s</b>\n"+
			"📆 Дата: <b>%s</b>\n"+
			"⏰ Время: <b>%s</b>\n"+
			"💰 Стоимость: %d руб.\n\n"+
			"Мы ждем вас! 🌟\n"+
			"За день до визита мы отправим напоминание.",
		booking.Service.Name,
		booking.Date.Format("02.01.2006"),
		booking.Time,
		booking.Service.Price/100,
	)

	recipient := &tele.User{ID: booking.UserID}
	_, err := s.bot.Send(recipient, msg, &tele.SendOptions{ParseMode: tele.ModeHTML})
	if err != nil {
		return fmt.Errorf("failed to send confirmation: %w", err)
	}

	return nil
}

// SendBookingCancellation sends cancellation message to user
func (s *NotificationService) SendBookingCancellation(ctx context.Context, booking *database.Booking) error {
	msg := fmt.Sprintf(
		"❌ <b>Запись отменена</b>\n\n"+
			"📋 Услуга: %s\n"+
			"📆 Дата: %s в %s\n\n"+
			"Вы можете создать новую запись с помощью команды /book",
		booking.Service.Name,
		booking.Date.Format("02.01.2006"),
		booking.Time,
	)

	recipient := &tele.User{ID: booking.UserID}
	_, err := s.bot.Send(recipient, msg, &tele.SendOptions{ParseMode: tele.ModeHTML})
	if err != nil {
		return fmt.Errorf("failed to send cancellation: %w", err)
	}

	return nil
}

// SendReminder sends reminder to user about upcoming booking
func (s *NotificationService) SendReminder(ctx context.Context, booking *database.Booking) error {
	msg := fmt.Sprintf(
		"🔔 <b>Напоминание о записи</b>\n\n"+
			"Завтра в <b>%s</b> у вас запись:\n"+
			"📋 %s\n"+
			"⏱ Длительность: %d мин\n"+
			"💰 Стоимость: %d руб.\n\n"+
			"Будем рады вас видеть! 🌟",
		booking.Time,
		booking.Service.Name,
		booking.Service.Duration,
		booking.Service.Price/100,
	)

	recipient := &tele.User{ID: booking.UserID}
	_, err := s.bot.Send(recipient, msg, &tele.SendOptions{ParseMode: tele.ModeHTML})
	if err != nil {
		return fmt.Errorf("failed to send reminder: %w", err)
	}

	return nil
}

// StartReminderWorker starts a background worker to send reminders
func (s *NotificationService) StartReminderWorker(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour) // Check every hour
	defer ticker.Stop()

	log.Println("Reminder worker started")

	for {
		select {
		case <-ctx.Done():
			log.Println("Reminder worker stopped")
			return
		case <-ticker.C:
			s.checkAndSendReminders(ctx)
		}
	}
}

// checkAndSendReminders checks for bookings that need reminders
func (s *NotificationService) checkAndSendReminders(ctx context.Context) {
	now := time.Now()

	// 1. Check for daily admin reminders (send at 8:00 AM)
	if now.Hour() == 8 && now.Minute() < 5 {
		s.sendDailyAdminReminder(ctx)
	}

	// 2. Check for 1-day reminders (tomorrow's bookings)
	s.checkDayBeforeReminders(ctx)

	// 3. Check for 1-hour reminders (today's bookings)
	s.checkHourBeforeReminders(ctx)
}

// sendDailyAdminReminder sends daily reminder to admins about today's bookings
func (s *NotificationService) sendDailyAdminReminder(ctx context.Context) {
	today := time.Now()
	startOfDay := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	var bookings []database.Booking
	err := database.DB.WithContext(ctx).
		Preload("Service").
		Preload("User").
		Where("date >= ? AND date < ?", startOfDay, endOfDay).
		Where("status IN ?", []database.BookingStatus{
			database.BookingStatusPending,
			database.BookingStatusConfirmed,
		}).
		Where("admin_daily_reminder_sent = ?", false).
		Find(&bookings).Error

	if err != nil {
		log.Printf("Error fetching today's bookings: %v", err)
		return
	}

	if len(bookings) == 0 {
		return
	}

	// Build message for admins
	msg := fmt.Sprintf("📅 <b>Записи на сегодня (%s)</b>\n\n", today.Format("02.01.2006"))

	for i, booking := range bookings {
		msg += fmt.Sprintf(
			"%d. <b>%s</b>\n"+
				"   👤 %s %s\n"+
				"   📋 %s\n"+
				"   💰 %d руб.\n\n",
			i+1,
			booking.Time,
			booking.User.FirstName,
			booking.User.LastName,
			booking.Service.Name,
			booking.Service.Price/100,
		)
	}

	msg += fmt.Sprintf("Всего записей: <b>%d</b>", len(bookings))

	// Send to all admins
	for _, adminID := range s.adminIDs {
		recipient := &tele.User{ID: adminID}
		if _, err := s.bot.Send(recipient, msg, &tele.SendOptions{ParseMode: tele.ModeHTML}); err != nil {
			log.Printf("Error sending daily reminder to admin %d: %v", adminID, err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Mark as sent
	for _, booking := range bookings {
		booking.AdminDailyReminderSent = true
		database.DB.WithContext(ctx).Save(&booking)
	}

	log.Printf("Daily admin reminder sent for %d bookings", len(bookings))
}

// checkDayBeforeReminders checks for bookings tomorrow and sends reminders
func (s *NotificationService) checkDayBeforeReminders(ctx context.Context) {
	tomorrow := time.Now().AddDate(0, 0, 1)
	startOfDay := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, tomorrow.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	var bookings []database.Booking
	err := database.DB.WithContext(ctx).
		Preload("Service").
		Preload("User").
		Where("date >= ? AND date < ?", startOfDay, endOfDay).
		Where("status IN ?", []database.BookingStatus{
			database.BookingStatusPending,
			database.BookingStatusConfirmed,
		}).
		Where("reminder_sent = ?", false).
		Find(&bookings).Error

	if err != nil {
		log.Printf("Error fetching bookings for reminders: %v", err)
		return
	}

	for _, booking := range bookings {
		if err := s.SendReminder(ctx, &booking); err != nil {
			log.Printf("Error sending reminder for booking %d: %v", booking.ID, err)
		} else {
			booking.ReminderSent = true
			database.DB.WithContext(ctx).Save(&booking)
			log.Printf("Reminder sent for booking %d to user %d", booking.ID, booking.UserID)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// checkHourBeforeReminders checks for bookings in 1 hour and sends reminders
func (s *NotificationService) checkHourBeforeReminders(ctx context.Context) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrow := today.Add(24 * time.Hour)

	var bookings []database.Booking
	err := database.DB.WithContext(ctx).
		Preload("Service").
		Preload("User").
		Where("date >= ? AND date < ?", today, tomorrow).
		Where("status IN ?", []database.BookingStatus{
			database.BookingStatusPending,
			database.BookingStatusConfirmed,
		}).
		Where("hour_reminder_sent = ?", false).
		Find(&bookings).Error

	if err != nil {
		log.Printf("Error fetching bookings for hour reminders: %v", err)
		return
	}

	for _, booking := range bookings {
		// Parse booking time
		bookingTime, err := time.Parse("15:04", booking.Time)
		if err != nil {
			continue
		}

		bookingDateTime := time.Date(
			booking.Date.Year(), booking.Date.Month(), booking.Date.Day(),
			bookingTime.Hour(), bookingTime.Minute(), 0, 0, now.Location(),
		)

		// Check if booking is within 1 hour window
		diff := bookingDateTime.Sub(now)
		if diff >= 55*time.Minute && diff <= 65*time.Minute {
			// Send reminder to user
			if err := s.SendHourReminder(ctx, &booking); err != nil {
				log.Printf("Error sending hour reminder for booking %d: %v", booking.ID, err)
			} else {
				booking.HourReminderSent = true
				database.DB.WithContext(ctx).Save(&booking)
				log.Printf("Hour reminder sent for booking %d to user %d", booking.ID, booking.UserID)
			}

			// Send reminder to admins
			s.sendHourReminderToAdmins(ctx, &booking)

			time.Sleep(100 * time.Millisecond)
		}
	}
}

// SendHourReminder sends reminder to user 1 hour before booking
func (s *NotificationService) SendHourReminder(ctx context.Context, booking *database.Booking) error {
	msg := fmt.Sprintf(
		"⏰ <b>Напоминание: запись через час!</b>\n\n"+
			"📋 Услуга: <b>%s</b>\n"+
			"⏰ Время: <b>%s</b>\n"+
			"💰 Стоимость: %d руб.\n\n"+
			"До встречи! 🌟",
		booking.Service.Name,
		booking.Time,
		booking.Service.Price/100,
	)

	recipient := &tele.User{ID: booking.UserID}
	_, err := s.bot.Send(recipient, msg, &tele.SendOptions{ParseMode: tele.ModeHTML})
	if err != nil {
		return fmt.Errorf("failed to send hour reminder: %w", err)
	}

	return nil
}

// sendHourReminderToAdmins sends reminder to admins 1 hour before booking
func (s *NotificationService) sendHourReminderToAdmins(ctx context.Context, booking *database.Booking) {
	msg := fmt.Sprintf(
		"⏰ <b>Напоминание: запись через час!</b>\n\n"+
			"👤 Клиент: %s %s\n"+
			"📋 Услуга: <b>%s</b>\n"+
			"⏰ Время: <b>%s</b>\n"+
			"💰 Стоимость: %d руб.",
		booking.User.FirstName,
		booking.User.LastName,
		booking.Service.Name,
		booking.Time,
		booking.Service.Price/100,
	)

	for _, adminID := range s.adminIDs {
		recipient := &tele.User{ID: adminID}
		if _, err := s.bot.Send(recipient, msg, &tele.SendOptions{ParseMode: tele.ModeHTML}); err != nil {
			log.Printf("Error sending hour reminder to admin %d: %v", adminID, err)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// NotifyAdmin sends notification to admin
func (s *NotificationService) NotifyAdmin(ctx context.Context, adminID int64, message string) error {
	recipient := &tele.User{ID: adminID}
	_, err := s.bot.Send(recipient, message, &tele.SendOptions{ParseMode: tele.ModeHTML})
	if err != nil {
		return fmt.Errorf("failed to notify admin: %w", err)
	}
	return nil
}

// NotifyAdminWithActions sends notification to admin with approve/reject buttons
func (s *NotificationService) NotifyAdminWithActions(ctx context.Context, adminID int64, message string, bookingID uint) error {
	recipient := &tele.User{ID: adminID}

	// Create keyboard with approve/reject buttons
	markup := &tele.ReplyMarkup{}
	btnApprove := markup.Data("✅ Подтвердить", "admin_approve_booking", fmt.Sprintf("%d", bookingID))
	btnReject := markup.Data("❌ Отменить", "admin_reject_booking", fmt.Sprintf("%d", bookingID))
	markup.Inline(
		markup.Row(btnApprove, btnReject),
	)

	_, err := s.bot.Send(recipient, message, &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: markup,
	})
	if err != nil {
		return fmt.Errorf("failed to notify admin with actions: %w", err)
	}
	return nil
}

// SendPromotionToChannel sends promotion message to configured channel
// This function can be called when creating a discount to automatically post to channel
// Usage: notificationService.SendPromotionToChannel(ctx, discount)
// Note: Requires CHANNEL_ID in .env file (format: @channelname or -1001234567890)
func (s *NotificationService) SendPromotionToChannel(ctx context.Context, discount *database.Discount) error {
	if s.channelID == "" {
		log.Println("Channel ID not configured, skipping promotion")
		return nil
	}

	originalPrice := discount.Service.Price / 100
	discountAmount := (discount.Service.Price * discount.Percentage) / 10000
	newPrice := originalPrice - discountAmount

	msg := fmt.Sprintf(
		"🎉 <b>%s</b>\n\n"+
			"📋 Услуга: <b>%s</b>\n"+
			"💰 Скидка: <b>%d%%</b>\n"+
			"💵 Цена: <s>%d руб.</s> <b>%d руб.</b>\n"+
			"📅 Действует: %s - %s\n\n"+
			"Записывайтесь через бота! 👇",
		discount.Name,
		discount.Service.Name,
		discount.Percentage,
		originalPrice,
		newPrice,
		discount.StartDate.Format("02.01.2006"),
		discount.EndDate.Format("02.01.2006"),
	)

	// Try to parse channel ID (can be @channelname or -1001234567890)
	var recipient tele.Recipient
	if len(s.channelID) > 0 && s.channelID[0] == '@' {
		recipient = &tele.Chat{Username: s.channelID[1:]}
	} else {
		// For numeric channel IDs, use ChatID directly
		// Note: This requires the bot to be added to the channel as admin
		// Format: -1001234567890 (negative number for channels)
		chatID, err := parseChannelID(s.channelID)
		if err != nil {
			return fmt.Errorf("invalid channel ID format: %w", err)
		}
		recipient = &tele.Chat{ID: chatID}
	}

	_, err := s.bot.Send(recipient, msg, &tele.SendOptions{ParseMode: tele.ModeHTML})
	if err != nil {
		return fmt.Errorf("failed to send promotion to channel: %w", err)
	}

	log.Printf("Promotion sent to channel: %s", s.channelID)
	return nil
}

// parseChannelID parses channel ID string to int64
func parseChannelID(channelID string) (int64, error) {
	// Remove @ if present
	if len(channelID) > 0 && channelID[0] == '@' {
		channelID = channelID[1:]
	}

	// Try to parse as int64
	var chatID int64
	_, err := fmt.Sscanf(channelID, "%d", &chatID)
	if err != nil {
		return 0, fmt.Errorf("cannot parse channel ID: %w", err)
	}

	return chatID, nil
}
