// Package bot contains callback handlers for inline keyboards
package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gobot/internal/database"

	tele "gopkg.in/telebot.v3"
)

// handleCallback handles all callback queries from inline keyboards
func (b *Bot) handleCallback(c tele.Context) error {
	callback := c.Callback()
	if callback == nil {
		return nil
	}

	// DEBUG: Log callback data
	fmt.Printf("📲 Callback received: %s from user %d\n", callback.Data, c.Sender().ID)

	// Parse callback data - telebot uses "|" as separator
	// Clean the callback data from whitespace
	cleanCallbackData := strings.TrimSpace(callback.Data)

	parts := strings.Split(cleanCallbackData, "|")
	if len(parts) < 1 {
		fmt.Printf("❌ Error: empty callback parts\n")
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка обработки действия"})
	}

	action := strings.TrimSpace(parts[0])
	data := ""
	if len(parts) > 1 {
		data = strings.TrimSpace(parts[1])
	}

	fmt.Printf("🔍 Action: '%s', Data: '%s'\n", action, data)

	ctx := context.Background()

	// Answer callback first to remove loading state
	c.Respond()

	switch action {
	case "main_menu":
		return b.handleMainMenuAction(ctx, c, data)
	case "service":
		return b.handleServiceSelection(ctx, c, data)
	case "date":
		return b.handleDateSelection(ctx, c, data)
	case "time":
		return b.handleTimeSelection(ctx, c, data)
	case "confirm":
		return b.handleBookingConfirmation(ctx, c)
	case "cancel":
		return b.handleCancel(ctx, c, data)
	case "cancel_booking":
		return b.handleBookingCancellation(ctx, c, data)
	case "back":
		return b.handleBack(ctx, c, data)
	case "back_to_menu":
		return b.handleBackToMainMenu(ctx, c)
	case "admin":
		return b.handleAdminAction(ctx, c, data)
	case "admin_edit_service":
		return b.handleAdminEditService(ctx, c, data)
	case "admin_toggle_service":
		return b.handleAdminToggleService(ctx, c, data)
	case "admin_delete_service":
		return b.handleAdminDeleteService(ctx, c, data)
	case "admin_add_service":
		return b.handleAdminAddServiceStart(ctx, c)
	case "admin_edit_service_menu":
		return b.handleAdminEditServiceMenu(ctx, c, data)
	case "admin_edit_field":
		return b.handleAdminEditField(ctx, c, data)
	case "admin_cancel_edit":
		return b.handleAdminCancelEdit(ctx, c)
	case "admin_cancel_add_service":
		return b.handleAdminCancelAddService(ctx, c)
	case "admin_discounts":
		return b.handleAdminDiscounts(ctx, c)
	case "admin_add_discount":
		return b.handleAdminAddDiscountStart(ctx, c)
	case "admin_discount_select_service":
		return b.handleAdminDiscountSelectService(ctx, c, data)
	case "admin_edit_discount":
		return b.handleAdminEditDiscount(ctx, c, data)
	case "admin_toggle_discount":
		return b.handleAdminToggleDiscount(ctx, c, data)
	case "admin_delete_discount":
		return b.handleAdminDeleteDiscount(ctx, c, data)
	case "admin_cancel_add_discount":
		return b.handleAdminCancelAddDiscount(ctx, c)
	default:
		return c.Respond(&tele.CallbackResponse{Text: "Неизвестное действие"})
	}
}

// handleServiceSelection handles service selection
func (b *Bot) handleServiceSelection(ctx context.Context, c tele.Context, serviceIDStr string) error {
	serviceID, err := strconv.ParseUint(serviceIDStr, 10, 32)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка выбора услуги"})
	}

	// Save service selection to user state
	state := b.getUserState(c.Sender().ID)
	state.CurrentStep = "date"
	state.ServiceID = uint(serviceID)

	// Update message with date selection
	return c.Edit("📅 Выберите дату:", getDateKeyboard())
}

// handleDateSelection handles date selection
func (b *Bot) handleDateSelection(ctx context.Context, c tele.Context, dateStr string) error {
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка выбора даты"})
	}

	// Save date selection to user state
	state := b.getUserState(c.Sender().ID)
	state.CurrentStep = "time"
	state.Date = date

	// Update message with time selection
	return c.Edit("⏰ Выберите время:", getTimeKeyboard())
}

// handleTimeSelection handles time selection
func (b *Bot) handleTimeSelection(ctx context.Context, c tele.Context, timeStr string) error {
	// Save time selection to user state
	state := b.getUserState(c.Sender().ID)
	state.CurrentStep = "confirm"
	state.Time = timeStr

	// Get service info
	var service database.Service
	if err := database.DB.First(&service, state.ServiceID).Error; err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка загрузки услуги"})
	}

	// Show confirmation
	confirmMsg := fmt.Sprintf(
		"✅ <b>Подтверждение записи</b>\n\n"+
			"📋 Услуга: <b>%s</b>\n"+
			"📝 Описание: %s\n"+
			"⏱ Длительность: %d минут\n"+
			"💰 Стоимость: %d руб.\n\n"+
			"📆 Дата: <b>%s</b>\n"+
			"⏰ Время: <b>%s</b>\n\n"+
			"Подтвердите запись:",
		service.Name,
		service.Description,
		service.Duration,
		service.Price/100,
		state.Date.Format("02.01.2006"),
		state.Time,
	)

	return c.Edit(confirmMsg, &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: getConfirmKeyboard(),
	})
}

// handleBookingConfirmation handles booking confirmation
func (b *Bot) handleBookingConfirmation(ctx context.Context, c tele.Context) error {
	state := b.getUserState(c.Sender().ID)

	// Create booking
	booking, err := b.bookingService.CreateBooking(
		ctx,
		c.Sender().ID,
		state.ServiceID,
		state.Date,
		state.Time,
	)
	if err != nil {
		return c.Edit("❌ Ошибка при создании записи. Попробуйте позже.")
	}

	// Send confirmation notification
	if err := b.notificationService.SendBookingConfirmation(ctx, booking); err != nil {
		// Log error but don't fail the booking
		fmt.Printf("Warning: failed to send confirmation: %v\n", err)
	}

	// Notify admins about new booking
	for _, adminID := range b.config.AdminUserIDs {
		adminMsg := fmt.Sprintf(
			"🔔 <b>Новая запись!</b>\n\n"+
				"👤 %s %s (@%s)\n"+
				"📋 %s\n"+
				"📆 %s в %s\n"+
				"💰 %d руб.",
			booking.User.FirstName,
			booking.User.LastName,
			booking.User.Username,
			booking.Service.Name,
			booking.Date.Format("02.01.2006"),
			booking.Time,
			booking.Service.Price/100,
		)
		b.notificationService.NotifyAdmin(ctx, adminID, adminMsg)
	}

	// Clear user state
	b.clearUserState(c.Sender().ID)

	successMsg := fmt.Sprintf(
		"✅ <b>Запись успешно создана!</b>\n\n"+
			"📋 Услуга: <b>%s</b>\n"+
			"📆 Дата: <b>%s</b>\n"+
			"⏰ Время: <b>%s</b>\n"+
			"💰 Стоимость: %d руб.\n\n"+
			"Мы ждем вас! 🌟\n"+
			"За день до визита вы получите напоминание.\n\n"+
			"Для просмотра записей используйте /my_bookings",
		booking.Service.Name,
		booking.Date.Format("02.01.2006"),
		booking.Time,
		booking.Service.Price/100,
	)

	return c.Edit(successMsg, &tele.SendOptions{ParseMode: tele.ModeHTML})
}

// handleCancel handles cancellation
func (b *Bot) handleCancel(ctx context.Context, c tele.Context, cancelType string) error {
	b.clearUserState(c.Sender().ID)
	return c.Edit("❌ Действие отменено.\nИспользуйте /book для новой записи.")
}

// handleBookingCancellation handles booking cancellation
func (b *Bot) handleBookingCancellation(ctx context.Context, c tele.Context, bookingIDStr string) error {
	bookingID, err := strconv.ParseUint(bookingIDStr, 10, 32)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка отмены записи"})
	}

	// Get booking info before cancellation
	var booking database.Booking
	if err := database.DB.WithContext(ctx).
		Preload("Service").
		Preload("User").
		First(&booking, bookingID).Error; err != nil {
		return c.Edit("❌ Запись не найдена")
	}

	// Cancel booking
	if err := b.bookingService.CancelBooking(ctx, uint(bookingID), c.Sender().ID); err != nil {
		return c.Edit("❌ Ошибка при отмене записи: " + err.Error())
	}

	// Send cancellation notification
	booking.Status = database.BookingStatusCancelled
	if err := b.notificationService.SendBookingCancellation(ctx, &booking); err != nil {
		fmt.Printf("Warning: failed to send cancellation notification: %v\n", err)
	}

	// Notify admins about cancellation
	for _, adminID := range b.config.AdminUserIDs {
		adminMsg := fmt.Sprintf(
			"❌ <b>Отмена записи</b>\n\n"+
				"👤 %s %s (@%s)\n"+
				"📋 %s\n"+
				"📆 %s в %s",
			booking.User.FirstName,
			booking.User.LastName,
			booking.User.Username,
			booking.Service.Name,
			booking.Date.Format("02.01.2006"),
			booking.Time,
		)
		b.notificationService.NotifyAdmin(ctx, adminID, adminMsg)
	}

	return c.Edit(
		"✅ Запись успешно отменена!\n\n" +
			"Для создания новой записи используйте /book",
	)
}

// handleBack handles back button
func (b *Bot) handleBack(ctx context.Context, c tele.Context, backTo string) error {
	state := b.getUserState(c.Sender().ID)

	switch backTo {
	case "services":
		services, err := b.bookingService.GetAvailableServices(ctx)
		if err != nil {
			return c.Edit("Ошибка при загрузке услуг")
		}
		state.CurrentStep = "service"
		return c.Edit("📋 Выберите услугу:", getServicesKeyboard(services))

	case "date":
		state.CurrentStep = "date"
		return c.Edit("📅 Выберите дату:", getDateKeyboard())

	case "main":
		b.clearUserState(c.Sender().ID)
		return c.Edit("Возврат в главное меню")

	default:
		return c.Respond(&tele.CallbackResponse{Text: "Неизвестное действие"})
	}
}

// handleAdminAction handles admin panel actions
func (b *Bot) handleAdminAction(ctx context.Context, c tele.Context, actionType string) error {
	if !b.isAdmin(c.Sender().ID) {
		return c.Respond(&tele.CallbackResponse{Text: "❌ Нет доступа"})
	}

	switch actionType {
	case "bookings":
		return b.handleAdminBookingsDetailed(ctx, c)
	case "services":
		return b.handleAdminServicesManagement(ctx, c)
	case "discounts":
		return b.handleAdminDiscounts(ctx, c)
	case "slots":
		return c.Edit("⏰ Управление временными слотами\n\nФункция в разработке...")
	case "stats":
		return b.handleAdminStatsDetailed(ctx, c)
	case "main":
		return b.handleAdmin(c)
	default:
		return c.Respond(&tele.CallbackResponse{Text: "Неизвестное действие"})
	}
}

// handleAdminBookings shows all bookings to admin
func (b *Bot) handleAdminBookings(ctx context.Context, c tele.Context) error {
	var bookings []database.Booking
	err := database.DB.WithContext(ctx).
		Preload("Service").
		Preload("User").
		Order("date DESC, time DESC").
		Limit(20).
		Find(&bookings).Error

	if err != nil {
		return c.Edit("Ошибка при загрузке записей")
	}

	if len(bookings) == 0 {
		return c.Edit("Записей пока нет")
	}

	msg := "📋 <b>Последние записи:</b>\n\n"
	for i, booking := range bookings {
		statusEmoji := getStatusEmoji(booking.Status)
		msg += fmt.Sprintf(
			"%d. %s <b>%s</b>\n"+
				"   👤 %s (@%s)\n"+
				"   📆 %s в %s\n"+
				"   %s %s\n\n",
			i+1,
			statusEmoji,
			booking.Service.Name,
			booking.User.FirstName,
			booking.User.Username,
			booking.Date.Format("02.01.2006"),
			booking.Time,
			statusEmoji,
			getStatusText(booking.Status),
		)
	}

	return c.Edit(msg, &tele.SendOptions{ParseMode: tele.ModeHTML})
}

// handleAdminStats shows statistics to admin
func (b *Bot) handleAdminStats(ctx context.Context, c tele.Context) error {
	var totalBookings int64
	var activeBookings int64
	var totalUsers int64

	database.DB.Model(&database.Booking{}).Count(&totalBookings)
	database.DB.Model(&database.Booking{}).Where("status IN ?", []string{"pending", "confirmed"}).Count(&activeBookings)
	database.DB.Model(&database.User{}).Count(&totalUsers)

	msg := fmt.Sprintf(
		"📊 <b>Статистика:</b>\n\n"+
			"👥 Пользователей: %d\n"+
			"📋 Всего записей: %d\n"+
			"✅ Активных записей: %d\n",
		totalUsers,
		totalBookings,
		activeBookings,
	)

	return c.Edit(msg, &tele.SendOptions{ParseMode: tele.ModeHTML})
}
