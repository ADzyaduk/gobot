// Package bot contains main menu handlers
package bot

import (
	"context"
	"fmt"
	"time"

	"gobot/internal/database"

	tele "gopkg.in/telebot.v3"
)

// handleMainMenuAction handles main menu button clicks
func (b *Bot) handleMainMenuAction(ctx context.Context, c tele.Context, action string) error {
	fmt.Printf("🎯 Main menu action: %s\n", action)

	switch action {
	case "my_bookings":
		fmt.Println("➡️ Calling handleMyBookings...")
		c.Delete()
		return b.handleMyBookings(c)
	case "help":
		fmt.Println("➡️ Calling handleHelp...")
		c.Delete()
		return b.handleHelp(c)
	case "discounts":
		fmt.Println("➡️ Calling handleDiscounts...")
		c.Delete()
		return b.handleDiscounts(c)
	case "catalog":
		fmt.Println("➡️ Calling handleCatalog...")
		c.Delete()
		return b.handleCatalog(c)
	case "admin":
		if b.isAdmin(c.Sender().ID) {
			fmt.Println("➡️ Calling handleAdmin...")
			c.Delete()
			return b.handleAdmin(c)
		}
		fmt.Println("❌ User is not admin")
		c.Send("❌ Нет доступа")
		return nil
	default:
		fmt.Printf("❌ Unknown action: %s\n", action)
		c.Send("❌ Неизвестное действие")
		return nil
	}
}

// handleBackToMainMenu returns user to main menu
func (b *Bot) handleBackToMainMenu(ctx context.Context, c tele.Context) error {
	welcomeMsg := "🏠 Главное меню\n\nВыберите действие:"

	return c.Edit(welcomeMsg, &tele.SendOptions{
		ReplyMarkup: getMainMenuInlineKeyboard(b.isAdmin(c.Sender().ID)),
	})
}

// handleDiscounts shows active discounts to user
func (b *Bot) handleDiscounts(c tele.Context) error {
	ctx := context.Background()

	// Get active discounts
	var discounts []database.Discount
	err := database.DB.WithContext(ctx).
		Preload("Service").
		Where("is_active = ? AND start_date <= ? AND end_date >= ?", true, time.Now(), time.Now()).
		Order("end_date ASC").
		Find(&discounts).Error

	if err != nil {
		return c.Send("❌ Ошибка при загрузке акций. Попробуйте позже.")
	}

	if len(discounts) == 0 {
		msg := "🎉 <b>Акции</b>\n\n" +
			"К сожалению, сейчас нет активных акций.\n\n" +
			"Следите за обновлениями! Мы регулярно проводим специальные предложения."

		markup := &tele.ReplyMarkup{}
		btnMenu := markup.Data("🏠 Главное меню", "back_to_menu", "")
		markup.Inline(markup.Row(btnMenu))

		return c.Send(msg, &tele.SendOptions{
			ParseMode:   tele.ModeHTML,
			ReplyMarkup: markup,
		})
	}

	msg := "🎉 <b>Актуальные акции:</b>\n\n"
	for i, discount := range discounts {
		originalPrice := discount.Service.Price / 100
		discountAmount := (discount.Service.Price * discount.Percentage) / 10000
		newPrice := originalPrice - discountAmount

		msg += fmt.Sprintf(
			"%d. <b>%s</b>\n"+
				"   📋 Услуга: %s\n"+
				"   💰 Скидка: %d%%\n"+
				"   💵 Цена: <s>%d руб.</s> <b>%d руб.</b>\n"+
				"   📅 Действует до: %s\n\n",
			i+1,
			discount.Name,
			discount.Service.Name,
			discount.Percentage,
			originalPrice,
			newPrice,
			discount.EndDate.Format("02.01.2006"),
		)
	}

	msg += "💡 <i>Чтобы записаться на услугу со скидкой, выберите услугу при бронировании.</i>"

	markup := &tele.ReplyMarkup{}
	btnMenu := markup.Data("🏠 Главное меню", "back_to_menu", "")
	markup.Inline(markup.Row(btnMenu))

	return c.Send(msg, &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: markup,
	})
}

// handleCatalog shows services catalog
func (b *Bot) handleCatalog(c tele.Context) error {
	ctx := context.Background()

	// Get all active services
	services, err := b.bookingService.GetAvailableServices(ctx)
	if err != nil {
		return c.Send("❌ Ошибка при загрузке услуг. Попробуйте позже.")
	}

	if len(services) == 0 {
		return c.Send("К сожалению, сейчас нет доступных услуг.")
	}

	msg := "📋 <b>КАТАЛОГ УСЛУГ</b>\n\n" +
		"👇 <i>Нажмите на услугу, чтобы увидеть полное описание и записаться</i>\n\n"

	for i, service := range services {
		msg += fmt.Sprintf(
			"<b>%d. %s</b>\n"+
				"💰 %d руб. | ⏱ %d мин\n"+
				"📝 %s\n\n",
			i+1,
			service.Name,
			service.Price/100,
			service.Duration,
			service.Description,
		)
	}

	return c.Send(msg, &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: getServicesCatalogKeyboard(services),
	})
}
