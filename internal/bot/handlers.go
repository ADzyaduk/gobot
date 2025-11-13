// Package bot contains Telegram bot handlers
package bot

import (
	"context"
	"fmt"

	"gobot/internal/database"

	tele "gopkg.in/telebot.v3"
)

// handleStart handles the /start command
func (b *Bot) handleStart(c tele.Context) error {
	ctx := context.Background()

	// Ensure user exists in database
	_, err := b.ensureUser(ctx, c.Sender())
	if err != nil {
		return c.Send("Произошла ошибка при регистрации. Попробуйте позже.")
	}

	welcomeMsg := fmt.Sprintf(
		"👋 Привет, %s!\n\n"+
			"Добро пожаловать в систему записи на услуги массажа и депиляции.\n\n"+
			"Выберите действие:",
		c.Sender().FirstName,
	)

	return c.Send(welcomeMsg, &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: getMainMenuInlineKeyboard(b.isAdmin(c.Sender().ID)),
	})
}

// handleHelp handles the /help command
func (b *Bot) handleHelp(c tele.Context) error {
	helpMsg := "📋 <b>Справка:</b>\n\n" +
		"<b>📋 Каталог услуг</b>\n" +
		"Просмотр всех услуг с описаниями и запись\n\n" +
		"<b>📅 Мои записи</b>\n" +
		"Просмотр всех ваших записей\n\n" +
		"<b>🎉 Акции</b>\n" +
		"Просмотр текущих акций и скидок\n\n"

	if b.isAdmin(c.Sender().ID) {
		helpMsg += "<b>🔧 Админ-панель</b>\n" +
			"Управление записями, услугами и акциями\n\n"
	}

	helpMsg += "По всем вопросам обращайтесь к администратору."

	markup := &tele.ReplyMarkup{}
	btnMenu := markup.Data("🏠 Главное меню", "back_to_menu", "")
	markup.Inline(markup.Row(btnMenu))

	return c.Send(helpMsg, &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: markup,
	})
}

// handleBook handles the /book command - redirects to catalog
func (b *Bot) handleBook(c tele.Context) error {
	// Redirect to catalog
	return b.handleCatalog(c)
}

// handleMyBookings handles the /my_bookings command
func (b *Bot) handleMyBookings(c tele.Context) error {
	ctx := context.Background()

	bookings, err := b.bookingService.GetUserBookings(ctx, c.Sender().ID)
	if err != nil {
		return c.Send("Ошибка при загрузке записей. Попробуйте позже.")
	}

	if len(bookings) == 0 {
		return c.Send("У вас пока нет записей.\nИспользуйте каталог услуг для создания записи.")
	}

	msg := "📅 <b>Ваши записи:</b>\n\n"
	for i, booking := range bookings {
		statusEmoji := getStatusEmoji(booking.Status)
		msg += fmt.Sprintf(
			"%d. %s <b>%s</b>\n"+
				"   📍 %s\n"+
				"   📆 %s в %s\n"+
				"   💰 %d руб.\n"+
				"   %s %s\n\n",
			i+1,
			statusEmoji,
			booking.Service.Name,
			booking.Service.Description,
			booking.Date.Format("02.01.2006"),
			booking.Time,
			booking.Service.Price/100,
			statusEmoji,
			getStatusText(booking.Status),
		)
	}

	return c.Send(msg, &tele.SendOptions{
		ParseMode: tele.ModeHTML,
	})
}

// handleCancelStart handles the /cancel command
func (b *Bot) handleCancelStart(c tele.Context) error {
	ctx := context.Background()

	bookings, err := b.bookingService.GetUserBookings(ctx, c.Sender().ID)
	if err != nil {
		return c.Send("Ошибка при загрузке записей. Попробуйте позже.")
	}

	// Filter only active bookings
	activeBookings := make([]database.Booking, 0)
	for _, booking := range bookings {
		if booking.Status == database.BookingStatusPending || booking.Status == database.BookingStatusConfirmed {
			activeBookings = append(activeBookings, booking)
		}
	}

	if len(activeBookings) == 0 {
		return c.Send("У вас нет активных записей для отмены.")
	}

	return c.Send(
		"❌ Выберите запись для отмены:",
		getCancelBookingsKeyboard(activeBookings),
	)
}

// handleAdmin handles the /admin command
func (b *Bot) handleAdmin(c tele.Context) error {
	if !b.isAdmin(c.Sender().ID) {
		return c.Send("❌ У вас нет доступа к админ-панели.")
	}

	adminMsg := "🔧 <b>Админ-панель</b>\n\n" +
		"Доступные функции:\n" +
		"• Просмотр всех записей\n" +
		"• Управление услугами\n" +
		"• Управление временными слотами\n"

	return c.Send(adminMsg, &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: getAdminKeyboard(),
	})
}

// getStatusEmoji returns emoji for booking status
func getStatusEmoji(status database.BookingStatus) string {
	switch status {
	case database.BookingStatusPending:
		return "⏳"
	case database.BookingStatusConfirmed:
		return "✅"
	case database.BookingStatusCancelled:
		return "❌"
	case database.BookingStatusCompleted:
		return "✔️"
	default:
		return "❓"
	}
}

// getStatusText returns text for booking status
func getStatusText(status database.BookingStatus) string {
	switch status {
	case database.BookingStatusPending:
		return "Ожидает подтверждения"
	case database.BookingStatusConfirmed:
		return "Подтверждено"
	case database.BookingStatusCancelled:
		return "Отменено"
	case database.BookingStatusCompleted:
		return "Завершено"
	default:
		return "Неизвестно"
	}
}
