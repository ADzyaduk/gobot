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
	case "admin_discount_set_percentage":
		return b.handleAdminDiscountSetPercentage(ctx, c, data)
	case "admin_discount_set_start_date":
		return b.handleAdminDiscountSetStartDate(ctx, c, data)
	case "admin_discount_set_end_date":
		return b.handleAdminDiscountSetEndDate(ctx, c, data)
	case "admin_approve_booking":
		return b.handleAdminApproveBooking(ctx, c, data)
	case "admin_reject_booking":
		return b.handleAdminRejectBooking(ctx, c, data)
	case "catalog_service":
		return b.handleCatalogService(ctx, c, data)
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

	// Get service info
	var service database.Service
	if err := database.DB.First(&service, serviceID).Error; err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка загрузки услуги"})
	}

	// Debug: log detailed description
	fmt.Printf("DEBUG: Service %d - DetailedDescription length: %d\n", service.ID, len(service.DetailedDescription))
	if service.DetailedDescription != "" {
		previewLen := 50
		if len(service.DetailedDescription) < previewLen {
			previewLen = len(service.DetailedDescription)
		}
		fmt.Printf("DEBUG: DetailedDescription content: %s\n", service.DetailedDescription[:previewLen])
	}

	// Save service selection to user state
	state := b.getUserState(c.Sender().ID)
	state.ServiceID = uint(serviceID)

	// Show service details with detailed description
	serviceMsg := fmt.Sprintf(
		"✅ <b>Выбрана услуга: %s</b>\n\n"+
			"📝 <b>Описание:</b>\n%s\n\n",
		service.Name,
		service.Description,
	)

	// Add detailed description if available
	if service.DetailedDescription != "" {
		serviceMsg += fmt.Sprintf(
			"📖 <b>Подробное описание:</b>\n%s\n\n",
			service.DetailedDescription,
		)
	} else {
		// Show message if no detailed description
		serviceMsg += "ℹ️ <i>Подробное описание отсутствует</i>\n\n"
	}

	serviceMsg += fmt.Sprintf(
		"⏱ Длительность: <b>%d минут</b>\n"+
			"💰 Стоимость: <b>%d руб.</b>\n\n"+
			"📅 <b>Выберите дату:</b>",
		service.Duration,
		service.Price/100,
	)

	state.CurrentStep = "date"

	// Update message with service details and date selection
	return c.Edit(serviceMsg, &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: getDateKeyboard(),
	})
}

// handleDateSelection handles date selection
func (b *Bot) handleDateSelection(ctx context.Context, c tele.Context, dateStr string) error {
	// Parse date and normalize to local timezone
	parsedDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка выбора даты"})
	}

	// Normalize to local timezone (same as time.Now())
	now := time.Now()
	date := time.Date(
		parsedDate.Year(), parsedDate.Month(), parsedDate.Day(),
		0, 0, 0, 0, now.Location(),
	)

	// Save date selection to user state
	state := b.getUserState(c.Sender().ID)
	state.CurrentStep = "time"
	state.Date = date

	// Get service to know duration
	var service database.Service
	if err := database.DB.First(&service, state.ServiceID).Error; err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка загрузки услуги"})
	}

	// Update message with time selection
	msg := fmt.Sprintf("⏰ <b>Выберите время:</b>\n\n📅 Дата: %s", date.Format("02.01.2006"))
	return c.Edit(msg, &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: getTimeKeyboard(date, service.Duration),
	})
}

// handleTimeSelection handles time selection
func (b *Bot) handleTimeSelection(ctx context.Context, c tele.Context, timeStr string) error {
	state := b.getUserState(c.Sender().ID)

	// Validate time slot
	if err := b.validateTimeSlot(ctx, state.Date, timeStr, state.ServiceID); err != nil {
		return c.Respond(&tele.CallbackResponse{Text: err.Error()})
	}

	// Save time selection to user state
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

// validateTimeSlot validates if a time slot is available
func (b *Bot) validateTimeSlot(ctx context.Context, date time.Time, timeStr string, serviceID uint) error {
	// Parse time
	slotTime, err := time.Parse("15:04", timeStr)
	if err != nil {
		return fmt.Errorf("неверный формат времени")
	}

	// Check if time is in the past
	now := time.Now()
	nowLocation := now.Location()

	// Normalize dates to same location for comparison
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, nowLocation)
	selectedDayNormalized := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, nowLocation)

	// Combine date and time using now's location
	// Use normalized date to ensure correct timezone
	slotDateTime := time.Date(
		selectedDayNormalized.Year(), selectedDayNormalized.Month(), selectedDayNormalized.Day(),
		slotTime.Hour(), slotTime.Minute(), 0, 0, nowLocation,
	)

	// Check if selected date is today and time is in the past
	if selectedDayNormalized.Equal(today) {
		// Check if the slot time is before current time (with 1 minute buffer for safety)
		if slotDateTime.Before(now.Add(1 * time.Minute)) {
			return fmt.Errorf("❌ Нельзя записаться на прошедшее время")
		}
	}

	// Get service duration
	var service database.Service
	if err := database.DB.First(&service, serviceID).Error; err != nil {
		return fmt.Errorf("ошибка загрузки услуги")
	}

	// Check if slot is already booked
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	var bookings []database.Booking
	database.DB.WithContext(ctx).
		Preload("Service").
		Where("date >= ? AND date < ?", startOfDay, endOfDay).
		Where("status IN ?", []database.BookingStatus{
			database.BookingStatusPending,
			database.BookingStatusConfirmed,
		}).
		Find(&bookings)

	slotStart := slotDateTime
	slotEnd := slotDateTime.Add(time.Duration(service.Duration) * time.Minute)

	for _, booking := range bookings {
		bookedTime, err := time.Parse("15:04", booking.Time)
		if err != nil {
			continue
		}

		bookedDateTime := time.Date(
			booking.Date.Year(), booking.Date.Month(), booking.Date.Day(),
			bookedTime.Hour(), bookedTime.Minute(), 0, 0, booking.Date.Location(),
		)
		bookedEnd := bookedDateTime.Add(time.Duration(booking.Service.Duration) * time.Minute)

		// Check for overlap
		if slotStart.Before(bookedEnd) && slotEnd.After(bookedDateTime) {
			return fmt.Errorf("❌ Это время уже занято. Выберите другое время.")
		}
	}

	return nil
}

// handleBookingConfirmation handles booking confirmation
func (b *Bot) handleBookingConfirmation(ctx context.Context, c tele.Context) error {
	state := b.getUserState(c.Sender().ID)

	// Validate time slot again before creating booking (double check to prevent race conditions)
	if err := b.validateTimeSlot(ctx, state.Date, state.Time, state.ServiceID); err != nil {
		return c.Respond(&tele.CallbackResponse{Text: err.Error()})
	}

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

	// Notify admins about new booking with approve/reject buttons
	for _, adminID := range b.config.AdminUserIDs {
		adminMsg := fmt.Sprintf(
			"🔔 <b>Новая запись!</b>\n\n"+
				"👤 %s %s (@%s)\n"+
				"📋 %s\n"+
				"📆 %s в %s\n"+
				"💰 %d руб.\n\n"+
				"Подтвердите или отмените запись:",
			booking.User.FirstName,
			booking.User.LastName,
			booking.User.Username,
			booking.Service.Name,
			booking.Date.Format("02.01.2006"),
			booking.Time,
			booking.Service.Price/100,
		)
		b.notificationService.NotifyAdminWithActions(ctx, adminID, adminMsg, booking.ID)
	}

	// Clear user state
	b.clearUserState(c.Sender().ID)

	successMsg := fmt.Sprintf(
		"⏳ <b>Запись создана и ожидает подтверждения</b>\n\n"+
			"📋 Услуга: <b>%s</b>\n"+
			"📆 Дата: <b>%s</b>\n"+
			"⏰ Время: <b>%s</b>\n"+
			"💰 Стоимость: %d руб.\n\n"+
			"Администратор рассмотрит вашу заявку и подтвердит запись.\n"+
			"Вы получите уведомление о решении.\n\n"+
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
	return c.Edit("❌ Действие отменено.\nИспользуйте каталог услуг для новой записи.")
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

// handleAdminApproveBooking handles admin approval of a booking
func (b *Bot) handleAdminApproveBooking(ctx context.Context, c tele.Context, bookingIDStr string) error {
	if !b.isAdmin(c.Sender().ID) {
		return c.Respond(&tele.CallbackResponse{Text: "❌ Нет доступа"})
	}

	bookingID, err := strconv.ParseUint(bookingIDStr, 10, 32)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка обработки"})
	}

	// Get booking
	var booking database.Booking
	if err := database.DB.WithContext(ctx).
		Preload("Service").
		Preload("User").
		First(&booking, bookingID).Error; err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Запись не найдена"})
	}

	// Check if already processed
	if booking.Status != database.BookingStatusPending {
		return c.Respond(&tele.CallbackResponse{Text: "Запись уже обработана"})
	}

	// Update booking status
	booking.Status = database.BookingStatusConfirmed
	if err := database.DB.WithContext(ctx).Save(&booking).Error; err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка при обновлении записи"})
	}

	// Send confirmation to user
	if err := b.notificationService.SendBookingConfirmation(ctx, &booking); err != nil {
		fmt.Printf("Warning: failed to send confirmation to user: %v\n", err)
	}

	// Update admin message
	updatedMsg := fmt.Sprintf(
		"✅ <b>Запись подтверждена</b>\n\n"+
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

	c.Respond(&tele.CallbackResponse{Text: "✅ Запись подтверждена"})
	return c.Edit(updatedMsg, &tele.SendOptions{ParseMode: tele.ModeHTML})
}

// handleAdminRejectBooking handles admin rejection of a booking
func (b *Bot) handleAdminRejectBooking(ctx context.Context, c tele.Context, bookingIDStr string) error {
	if !b.isAdmin(c.Sender().ID) {
		return c.Respond(&tele.CallbackResponse{Text: "❌ Нет доступа"})
	}

	bookingID, err := strconv.ParseUint(bookingIDStr, 10, 32)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка обработки"})
	}

	// Get booking
	var booking database.Booking
	if err := database.DB.WithContext(ctx).
		Preload("Service").
		Preload("User").
		First(&booking, bookingID).Error; err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Запись не найдена"})
	}

	// Check if already processed
	if booking.Status != database.BookingStatusPending {
		return c.Respond(&tele.CallbackResponse{Text: "Запись уже обработана"})
	}

	// Update booking status
	booking.Status = database.BookingStatusCancelled
	if err := database.DB.WithContext(ctx).Save(&booking).Error; err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка при обновлении записи"})
	}

	// Send rejection notification to user
	userMsg := fmt.Sprintf(
		"❌ <b>Ваша запись была отменена администратором</b>\n\n"+
			"📋 Услуга: %s\n"+
			"📆 Дата: %s в %s\n\n"+
			"Вы можете создать новую запись через каталог услуг",
		booking.Service.Name,
		booking.Date.Format("02.01.2006"),
		booking.Time,
	)

	recipient := &tele.User{ID: booking.UserID}
	if _, err := b.tg.Send(recipient, userMsg, &tele.SendOptions{ParseMode: tele.ModeHTML}); err != nil {
		fmt.Printf("Warning: failed to send rejection notification to user: %v\n", err)
	}

	// Update admin message
	updatedMsg := fmt.Sprintf(
		"❌ <b>Запись отменена</b>\n\n"+
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

	c.Respond(&tele.CallbackResponse{Text: "❌ Запись отменена"})
	return c.Edit(updatedMsg, &tele.SendOptions{ParseMode: tele.ModeHTML})
}

// handleCatalogService shows service details from catalog
func (b *Bot) handleCatalogService(ctx context.Context, c tele.Context, serviceIDStr string) error {
	serviceID, err := strconv.ParseUint(serviceIDStr, 10, 32)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка выбора услуги"})
	}

	// Get service info
	var service database.Service
	if err := database.DB.First(&service, serviceID).Error; err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка загрузки услуги"})
	}

	// Debug: log detailed description
	fmt.Printf("DEBUG: Catalog Service %d - DetailedDescription length: %d\n", service.ID, len(service.DetailedDescription))
	if service.DetailedDescription != "" {
		previewLen := 50
		if len(service.DetailedDescription) < previewLen {
			previewLen = len(service.DetailedDescription)
		}
		fmt.Printf("DEBUG: DetailedDescription content: %s\n", service.DetailedDescription[:previewLen])
	}

	// Build service details message
	serviceMsg := fmt.Sprintf(
		"📋 <b>%s</b>\n\n"+
			"📝 <b>Описание:</b>\n%s\n\n",
		service.Name,
		service.Description,
	)

	// Add detailed description if available
	if service.DetailedDescription != "" {
		serviceMsg += fmt.Sprintf(
			"📖 <b>Подробное описание:</b>\n%s\n\n",
			service.DetailedDescription,
		)
	} else {
		// Show message if no detailed description
		serviceMsg += "ℹ️ <i>Подробное описание отсутствует</i>\n\n"
	}

	serviceMsg += fmt.Sprintf(
		"⏱ Длительность: <b>%d минут</b>\n"+
			"💰 Стоимость: <b>%d руб.</b>",
		service.Duration,
		service.Price/100,
	)

	return c.Edit(serviceMsg, &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: getServiceDetailsKeyboard(uint(serviceID), true),
	})
}
