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
	bot *tele.Bot
}

// NewNotificationService creates a new notification service
func NewNotificationService(bot *tele.Bot) *NotificationService {
	return &NotificationService{
		bot: bot,
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
	// Get tomorrow's date
	tomorrow := time.Now().AddDate(0, 0, 1)
	startOfDay := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, tomorrow.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	// Find bookings for tomorrow
	var bookings []database.Booking
	err := database.DB.WithContext(ctx).
		Preload("Service").
		Preload("User").
		Where("date >= ? AND date < ?", startOfDay, endOfDay).
		Where("status IN ?", []database.BookingStatus{
			database.BookingStatusPending,
			database.BookingStatusConfirmed,
		}).
		Find(&bookings).Error

	if err != nil {
		log.Printf("Error fetching bookings for reminders: %v", err)
		return
	}

	log.Printf("Found %d bookings to remind", len(bookings))

	// Send reminders
	for _, booking := range bookings {
		if err := s.SendReminder(ctx, &booking); err != nil {
			log.Printf("Error sending reminder for booking %d: %v", booking.ID, err)
		} else {
			log.Printf("Reminder sent for booking %d to user %d", booking.ID, booking.UserID)
		}

		// Small delay to avoid rate limiting
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

