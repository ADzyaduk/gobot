// Package bot contains text input handlers
package bot

import (
	"context"
	"fmt"

	tele "gopkg.in/telebot.v3"
)

// handleTextInput handles all text messages
func (b *Bot) handleTextInput(c tele.Context) error {
	state := b.getUserState(c.Sender().ID)

	// Handle "Главное меню" button
	if c.Text() == "🏠 Главное меню" || c.Text() == "/start" {
		ctx := context.Background()
		_, err := b.ensureUser(ctx, c.Sender())
		if err != nil {
			return c.Send("Произошла ошибка. Попробуйте позже.")
		}

		welcomeMsg := fmt.Sprintf(
			"🏠 <b>Главное меню</b>\n\n"+
				"Выберите действие:",
		)

		return c.Send(welcomeMsg, &tele.SendOptions{
			ParseMode:   tele.ModeHTML,
			ReplyMarkup: getMainMenuInlineKeyboard(b.isAdmin(c.Sender().ID)),
		})
	}

	// Check if admin is editing
	if b.isAdmin(c.Sender().ID) {
		// Service editing
		if state.EditMode != "" && state.EditServiceID != 0 {
			return b.handleAdminTextMessage(c)
		}

		// Service creation
		if state.EditMode != "" && state.TempServiceData != nil {
			// Check if it's discount creation or service creation
			if state.EditMode == "add_service_name" ||
				state.EditMode == "add_service_price" ||
				state.EditMode == "add_service_duration" ||
				state.EditMode == "add_service_description" {
				return b.handleAdminAddServiceMessage(c)
			}

			// Discount creation
			if state.EditMode == "add_discount_name" ||
				state.EditMode == "add_discount_percentage" ||
				state.EditMode == "add_discount_dates" {
				return b.handleAdminAddDiscountMessage(c)
			}
		}
	}

	// Default: no special handling
	return nil
}
