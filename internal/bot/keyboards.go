// Package bot contains keyboard layouts for Telegram bot
package bot

import (
	"fmt"
	"time"

	"gobot/internal/database"

	tele "gopkg.in/telebot.v3"
)

// getMainMenuInlineKeyboard returns the main menu inline keyboard
func getMainMenuInlineKeyboard(isAdmin bool) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}

	btnBook := markup.Data("📝 Записаться", "main_menu", "book")
	btnMyBookings := markup.Data("📅 Мои записи", "main_menu", "my_bookings")
	btnHelp := markup.Data("❓ Помощь", "main_menu", "help")

	if isAdmin {
		btnAdmin := markup.Data("🔧 Админ-панель", "main_menu", "admin")
		btnDiscounts := markup.Data("🎉 Акции", "admin_discounts", "main")
		markup.Inline(
			markup.Row(btnBook),
			markup.Row(btnMyBookings),
			markup.Row(btnAdmin, btnDiscounts),
			markup.Row(btnHelp),
		)
	} else {
		markup.Inline(
			markup.Row(btnBook),
			markup.Row(btnMyBookings),
			markup.Row(btnHelp),
		)
	}

	return markup
}

// getServicesKeyboard returns keyboard with available services
func getServicesKeyboard(services []database.Service) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	rows := make([]tele.Row, 0)

	for _, service := range services {
		btn := markup.Data(
			fmt.Sprintf("%s (%d руб.)", service.Name, service.Price/100),
			"service",
			fmt.Sprintf("%d", service.ID),
		)
		rows = append(rows, markup.Row(btn))
	}

	// Add cancel button
	btnCancel := markup.Data("❌ Отмена", "cancel", "booking")
	rows = append(rows, markup.Row(btnCancel))

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
		btn := markup.Data(
			date.Format("02.01.2006 (Mon)"),
			"date",
			date.Format("2006-01-02"),
		)
		rows = append(rows, markup.Row(btn))
	}

	// Add back and cancel buttons
	btnBack := markup.Data("⬅️ Назад", "back", "services")
	btnCancel := markup.Data("❌ Отмена", "cancel", "booking")
	rows = append(rows, markup.Row(btnBack, btnCancel))

	markup.Inline(rows...)
	return markup
}

// getTimeKeyboard returns keyboard with available time slots
func getTimeKeyboard() *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	rows := make([]tele.Row, 0)

	// Default time slots (можно расширить на основе TimeSlots из БД)
	timeSlots := []string{
		"09:00", "10:00", "11:00", "12:00",
		"13:00", "14:00", "15:00", "16:00",
		"17:00", "18:00", "19:00", "20:00",
	}

	// Create rows with 3 buttons each
	for i := 0; i < len(timeSlots); i += 3 {
		row := tele.Row{}
		for j := 0; j < 3 && i+j < len(timeSlots); j++ {
			timeSlot := timeSlots[i+j]
			btn := markup.Data(timeSlot, "time", timeSlot)
			row = append(row, btn)
		}
		rows = append(rows, row)
	}

	// Add back and cancel buttons
	btnBack := markup.Data("⬅️ Назад", "back", "date")
	btnCancel := markup.Data("❌ Отмена", "cancel", "booking")
	rows = append(rows, markup.Row(btnBack, btnCancel))

	markup.Inline(rows...)
	return markup
}

// getConfirmKeyboard returns keyboard for booking confirmation
func getConfirmKeyboard() *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}

	btnConfirm := markup.Data("✅ Подтвердить", "confirm", "booking")
	btnCancel := markup.Data("❌ Отмена", "cancel", "booking")

	markup.Inline(
		markup.Row(btnConfirm),
		markup.Row(btnCancel),
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

	// Add cancel button
	btnBack := markup.Data("⬅️ Назад", "back", "main")
	rows = append(rows, markup.Row(btnBack))

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
