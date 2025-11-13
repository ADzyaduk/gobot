// Package bot contains main menu handlers
package bot

import (
	"context"
	"fmt"

	tele "gopkg.in/telebot.v3"
)

// handleMainMenuAction handles main menu button clicks
func (b *Bot) handleMainMenuAction(ctx context.Context, c tele.Context, action string) error {
	fmt.Printf("🎯 Main menu action: %s\n", action)

	switch action {
	case "book":
		fmt.Println("🗑️ Deleting old menu message...")
		err := c.Delete()
		if err != nil {
			fmt.Printf("⚠️ Failed to delete message: %v\n", err)
		}
		fmt.Println("➡️ Calling handleBook...")
		return b.handleBook(c)
	case "my_bookings":
		fmt.Println("➡️ Calling handleMyBookings...")
		c.Delete()
		return b.handleMyBookings(c)
	case "help":
		fmt.Println("➡️ Calling handleHelp...")
		c.Delete()
		return b.handleHelp(c)
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
