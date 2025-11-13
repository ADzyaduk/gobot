// Package bot contains keyboard layouts for Telegram bot
package bot

import (
	"fmt"
	"time"

	"gobot/internal/database"

	tele "gopkg.in/telebot.v3"
)

// getRussianWeekday returns Russian name of weekday
func getRussianWeekday(t time.Time) string {
	weekdays := map[time.Weekday]string{
		time.Sunday:    "Вс",
		time.Monday:    "Пн",
		time.Tuesday:   "Вт",
		time.Wednesday: "Ср",
		time.Thursday:  "Чт",
		time.Friday:    "Пт",
		time.Saturday:  "Сб",
	}
	return weekdays[t.Weekday()]
}

// getMainMenuInlineKeyboard returns the main menu inline keyboard
func getMainMenuInlineKeyboard(isAdmin bool) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}

	btnCatalog := markup.Data("📋 Каталог услуг", "main_menu", "catalog")
	btnMyBookings := markup.Data("📅 Мои записи", "main_menu", "my_bookings")
	btnDiscounts := markup.Data("🎉 Акции", "main_menu", "discounts")
	btnHelp := markup.Data("❓ Помощь", "main_menu", "help")

	if isAdmin {
		btnAdmin := markup.Data("🔧 Админ-панель", "main_menu", "admin")
		btnAdminDiscounts := markup.Data("🎉 Управление акциями", "admin_discounts", "main")
		markup.Inline(
			markup.Row(btnCatalog),
			markup.Row(btnMyBookings),
			markup.Row(btnDiscounts),
			markup.Row(btnAdmin, btnAdminDiscounts),
			markup.Row(btnHelp),
		)
	} else {
		markup.Inline(
			markup.Row(btnCatalog),
			markup.Row(btnMyBookings),
			markup.Row(btnDiscounts),
			markup.Row(btnHelp),
		)
	}

	return markup
}

// getServicesKeyboard returns keyboard with available services for booking
func getServicesKeyboard(services []database.Service) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	rows := make([]tele.Row, 0)

	for _, service := range services {
		btn := markup.Data(
			fmt.Sprintf("%s - %d руб.", service.Name, service.Price/100),
			"service",
			fmt.Sprintf("%d", service.ID),
		)
		rows = append(rows, markup.Row(btn))
	}

	// Add cancel button
	btnCancel := markup.Data("❌ Отмена", "cancel", "booking")
	rows = append(rows, markup.Row(btnCancel))

	// Add main menu button
	btnMenu := markup.Data("🏠 Главное меню", "back_to_menu", "")
	rows = append(rows, markup.Row(btnMenu))

	markup.Inline(rows...)
	return markup
}

// getServicesCatalogKeyboard returns keyboard for services catalog
func getServicesCatalogKeyboard(services []database.Service) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	rows := make([]tele.Row, 0)

	for _, service := range services {
		btn := markup.Data(
			fmt.Sprintf("📋 %s", service.Name),
			"catalog_service",
			fmt.Sprintf("%d", service.ID),
		)
		rows = append(rows, markup.Row(btn))
	}

	// Add main menu button
	btnMenu := markup.Data("🏠 Главное меню", "back_to_menu", "")
	rows = append(rows, markup.Row(btnMenu))

	markup.Inline(rows...)
	return markup
}

// getServiceDetailsKeyboard returns keyboard for service details view
func getServiceDetailsKeyboard(serviceID uint, showBookButton bool) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	rows := make([]tele.Row, 0)

	if showBookButton {
		btnBook := markup.Data("📝 Записаться на эту услугу", "service", fmt.Sprintf("%d", serviceID))
		rows = append(rows, markup.Row(btnBook))
	}

	btnBack := markup.Data("⬅️ Назад к каталогу", "main_menu", "catalog")
	rows = append(rows, markup.Row(btnBack))

	// Add main menu button
	btnMenu := markup.Data("🏠 Главное меню", "back_to_menu", "")
	rows = append(rows, markup.Row(btnMenu))

	markup.Inline(rows...)
	return markup
}

// getDateKeyboard returns keyboard with available dates
func getDateKeyboard() *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	rows := make([]tele.Row, 0)

	// Generate next 7 days
	for i := 0; i < 7; i++ {
		date := getNextAvailableDate(i)
		weekday := getRussianWeekday(date)
		btn := markup.Data(
			fmt.Sprintf("%s (%s)", date.Format("02.01.2006"), weekday),
			"date",
			date.Format("2006-01-02"),
		)
		rows = append(rows, markup.Row(btn))
	}

	// Add back and cancel buttons
	btnBack := markup.Data("⬅️ Назад", "back", "services")
	btnCancel := markup.Data("❌ Отмена", "cancel", "booking")
	rows = append(rows, markup.Row(btnBack, btnCancel))

	// Add main menu button
	btnMenu := markup.Data("🏠 Главное меню", "back_to_menu", "")
	rows = append(rows, markup.Row(btnMenu))

	markup.Inline(rows...)
	return markup
}

// getTimeKeyboard returns keyboard with available time slots for a specific date
// Filters out past times and already booked slots
func getTimeKeyboard(selectedDate time.Time, serviceDuration int) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	rows := make([]tele.Row, 0)

	// Default time slots (можно расширить на основе TimeSlots из БД)
	// Исключены: 17:00, 18:00, 20:00 - не работаем в это время
	allTimeSlots := []string{
		"09:00", "10:00", "11:00", "12:00",
		"13:00", "14:00", "15:00", "16:00",
		"19:00",
	}

	now := time.Now()
	nowLocation := now.Location()

	// Normalize selected date to same location and timezone
	selectedDateNormalized := time.Date(
		selectedDate.Year(), selectedDate.Month(), selectedDate.Day(),
		0, 0, 0, 0, nowLocation,
	)

	// Get today's date normalized
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, nowLocation)

	// Check if selected date is today
	isToday := selectedDateNormalized.Equal(today)

	// Filter available time slots
	availableSlots := make([]string, 0)
	for _, timeSlot := range allTimeSlots {
		// Parse time
		slotTime, err := time.Parse("15:04", timeSlot)
		if err != nil {
			continue
		}

		// Combine date and time using now's location
		// Use normalized date to ensure correct timezone
		slotDateTime := time.Date(
			selectedDateNormalized.Year(), selectedDateNormalized.Month(), selectedDateNormalized.Day(),
			slotTime.Hour(), slotTime.Minute(), 0, 0, nowLocation,
		)

		// Check if time is in the past (only for today)
		if isToday {
			// Check if the slot time is before current time (with 1 minute buffer for safety)
			if slotDateTime.Before(now.Add(1 * time.Minute)) {
				continue // Skip past times
			}
		}

		// Check if slot is already booked
		// Use normalized date for consistency
		if isTimeSlotBooked(timeSlot, selectedDateNormalized, serviceDuration) {
			continue // Skip booked slots
		}

		availableSlots = append(availableSlots, timeSlot)
	}

	if len(availableSlots) == 0 {
		// No available slots
		markup.Inline(markup.Row(markup.Data("❌ Нет доступного времени", "no_time", "")))
		return markup
	}

	// Create rows with 3 buttons each
	for i := 0; i < len(availableSlots); i += 3 {
		row := tele.Row{}
		for j := 0; j < 3 && i+j < len(availableSlots); j++ {
			timeSlot := availableSlots[i+j]
			btn := markup.Data(timeSlot, "time", timeSlot)
			row = append(row, btn)
		}
		rows = append(rows, row)
	}

	// Add back and cancel buttons
	btnBack := markup.Data("⬅️ Назад", "back", "date")
	btnCancel := markup.Data("❌ Отмена", "cancel", "booking")
	rows = append(rows, markup.Row(btnBack, btnCancel))

	// Add main menu button
	btnMenu := markup.Data("🏠 Главное меню", "back_to_menu", "")
	rows = append(rows, markup.Row(btnMenu))

	markup.Inline(rows...)
	return markup
}

// getBookedTimesForDate returns list of booked times with their durations for a specific date
func getBookedTimesForDate(date time.Time) []struct {
	Time     string
	Duration int
} {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	var bookings []database.Booking
	database.DB.
		Preload("Service").
		Where("date >= ? AND date < ?", startOfDay, endOfDay).
		Where("status IN ?", []database.BookingStatus{
			database.BookingStatusPending,
			database.BookingStatusConfirmed,
		}).
		Find(&bookings)

	bookedSlots := make([]struct {
		Time     string
		Duration int
	}, 0, len(bookings))
	for _, booking := range bookings {
		bookedSlots = append(bookedSlots, struct {
			Time     string
			Duration int
		}{
			Time:     booking.Time,
			Duration: booking.Service.Duration,
		})
	}

	return bookedSlots
}

// isTimeSlotBooked checks if a time slot conflicts with existing bookings
// Takes into account service duration to check for overlaps
func isTimeSlotBooked(timeSlot string, date time.Time, serviceDuration int) bool {
	slotTime, err := time.Parse("15:04", timeSlot)
	if err != nil {
		return true // If we can't parse, consider it booked to be safe
	}

	slotDateTime := time.Date(
		date.Year(), date.Month(), date.Day(),
		slotTime.Hour(), slotTime.Minute(), 0, 0, date.Location(),
	)
	slotStart := slotDateTime
	slotEnd := slotDateTime.Add(time.Duration(serviceDuration) * time.Minute)

	bookedSlots := getBookedTimesForDate(date)
	for _, booked := range bookedSlots {
		bookedTime, err := time.Parse("15:04", booked.Time)
		if err != nil {
			continue
		}

		bookedDateTime := time.Date(
			date.Year(), date.Month(), date.Day(),
			bookedTime.Hour(), bookedTime.Minute(), 0, 0, date.Location(),
		)
		bookedEnd := bookedDateTime.Add(time.Duration(booked.Duration) * time.Minute)

		// Check for overlap
		if slotStart.Before(bookedEnd) && slotEnd.After(bookedDateTime) {
			return true // Time slot is booked
		}
	}

	return false
}

// getConfirmKeyboard returns keyboard for booking confirmation
func getConfirmKeyboard() *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}

	btnConfirm := markup.Data("✅ Подтвердить", "confirm", "booking")
	btnCancel := markup.Data("❌ Отмена", "cancel", "booking")
	btnMenu := markup.Data("🏠 Главное меню", "back_to_menu", "")

	markup.Inline(
		markup.Row(btnConfirm),
		markup.Row(btnCancel),
		markup.Row(btnMenu),
	)

	return markup
}

// getCancelBookingsKeyboard returns keyboard with user's bookings for cancellation
func getCancelBookingsKeyboard(bookings []database.Booking) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	rows := make([]tele.Row, 0)

	for _, booking := range bookings {
		btn := markup.Data(
			fmt.Sprintf("%s - %s %s", booking.Service.Name, booking.Date.Format("02.01"), booking.Time),
			"cancel_booking",
			fmt.Sprintf("%d", booking.ID),
		)
		rows = append(rows, markup.Row(btn))
	}

	// Add main menu button
	btnMenu := markup.Data("🏠 Главное меню", "back_to_menu", "")
	rows = append(rows, markup.Row(btnMenu))

	markup.Inline(rows...)
	return markup
}

// getAdminKeyboard returns admin panel keyboard
func getAdminKeyboard() *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}

	btnBookings := markup.Data("📋 Все записи", "admin", "bookings")
	btnServices := markup.Data("🛠 Услуги", "admin", "services")
	btnDiscounts := markup.Data("🎉 Акции", "admin", "discounts")
	btnSlots := markup.Data("⏰ Временные слоты", "admin", "slots")
	btnStats := markup.Data("📊 Статистика", "admin", "stats")

	markup.Inline(
		markup.Row(btnBookings),
		markup.Row(btnServices, btnDiscounts),
		markup.Row(btnSlots),
		markup.Row(btnStats),
	)

	return markup
}

// getNextAvailableDate returns the next available date starting from today + offset
func getNextAvailableDate(offset int) time.Time {
	return time.Now().AddDate(0, 0, offset)
}

// getDiscountPercentageKeyboard returns keyboard with percentage options
func getDiscountPercentageKeyboard() *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	rows := make([]tele.Row, 0)

	// Popular percentages
	percentages := []int{5, 10, 15, 20, 25, 30, 50}
	for i := 0; i < len(percentages); i += 3 {
		row := tele.Row{}
		for j := 0; j < 3 && i+j < len(percentages); j++ {
			pct := percentages[i+j]
			btn := markup.Data(fmt.Sprintf("%d%%", pct), "admin_discount_set_percentage", fmt.Sprintf("%d", pct))
			row = append(row, btn)
		}
		rows = append(rows, row)
	}

	btnCancel := markup.Data("❌ Отмена", "admin_cancel_add_discount", "")
	btnMenu := markup.Data("🏠 Главное меню", "back_to_menu", "")
	rows = append(rows, markup.Row(btnCancel))
	rows = append(rows, markup.Row(btnMenu))

	markup.Inline(rows...)
	return markup
}

// getDiscountStartDateKeyboard returns keyboard with start date options
func getDiscountStartDateKeyboard() *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	rows := make([]tele.Row, 0)

	now := time.Now()
	dates := []struct {
		label string
		date  time.Time
	}{
		{"Сегодня", now},
		{"Завтра", now.AddDate(0, 0, 1)},
		{"Через 3 дня", now.AddDate(0, 0, 3)},
		{"Через неделю", now.AddDate(0, 0, 7)},
		{"Через месяц", now.AddDate(0, 1, 0)},
	}

	for _, d := range dates {
		btn := markup.Data(
			fmt.Sprintf("%s (%s)", d.label, d.date.Format("02.01.2006")),
			"admin_discount_set_start_date",
			d.date.Format("02.01.2006"),
		)
		rows = append(rows, markup.Row(btn))
	}

	btnCancel := markup.Data("❌ Отмена", "admin_cancel_add_discount", "")
	btnMenu := markup.Data("🏠 Главное меню", "back_to_menu", "")
	rows = append(rows, markup.Row(btnCancel))
	rows = append(rows, markup.Row(btnMenu))

	markup.Inline(rows...)
	return markup
}

// getDiscountEndDateKeyboard returns keyboard with end date options based on start date
func getDiscountEndDateKeyboard(startDate time.Time) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	rows := make([]tele.Row, 0)

	// Calculate end dates relative to start date
	dates := []struct {
		label string
		date  time.Time
	}{
		{"1 день", startDate.AddDate(0, 0, 1)},
		{"3 дня", startDate.AddDate(0, 0, 3)},
		{"Неделя", startDate.AddDate(0, 0, 7)},
		{"2 недели", startDate.AddDate(0, 0, 14)},
		{"Месяц", startDate.AddDate(0, 1, 0)},
		{"3 месяца", startDate.AddDate(0, 3, 0)},
	}

	for _, d := range dates {
		btn := markup.Data(
			fmt.Sprintf("%s (%s)", d.label, d.date.Format("02.01.2006")),
			"admin_discount_set_end_date",
			d.date.Format("02.01.2006"),
		)
		rows = append(rows, markup.Row(btn))
	}

	btnCancel := markup.Data("❌ Отмена", "admin_cancel_add_discount", "")
	btnMenu := markup.Data("🏠 Главное меню", "back_to_menu", "")
	rows = append(rows, markup.Row(btnCancel))
	rows = append(rows, markup.Row(btnMenu))

	markup.Inline(rows...)
	return markup
}
