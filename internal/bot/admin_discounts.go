// Package bot contains discount management handlers
package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gobot/internal/database"
	"gobot/internal/services"

	tele "gopkg.in/telebot.v3"
)

// handleAdminDiscounts shows discounts management interface
func (b *Bot) handleAdminDiscounts(ctx context.Context, c tele.Context) error {
	if !b.isAdmin(c.Sender().ID) {
		return c.Respond(&tele.CallbackResponse{Text: "❌ Нет доступа"})
	}

	discountService := services.NewDiscountService()
	discounts, err := discountService.GetAllDiscounts(ctx)
	if err != nil {
		return c.Edit("Ошибка при загрузке акций")
	}

	msg := "🎉 <b>Управление акциями</b>\n\n"

	if len(discounts) == 0 {
		msg += "Нет активных акций\n\n"
	} else {
		for _, discount := range discounts {
			status := "✅"
			if !discount.IsActive {
				status = "❌"
			}

			active := ""
			now := time.Now()
			if now.After(discount.StartDate) && now.Before(discount.EndDate) && discount.IsActive {
				active = " 🔥 <b>АКТИВНА</b>"
			}

			msg += fmt.Sprintf(
				"%s <b>%s</b>%s\n"+
					"   Услуга: %s\n"+
					"   Скидка: <b>%d%%</b>\n"+
					"   Период: %s - %s\n\n",
				status,
				discount.Name,
				active,
				discount.Service.Name,
				discount.Percentage,
				discount.StartDate.Format("02.01.2006"),
				discount.EndDate.Format("02.01.2006"),
			)
		}
	}

	return c.Edit(msg, &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: getDiscountsManagementKeyboard(discounts),
	})
}

// getDiscountsManagementKeyboard returns keyboard for discount management
func getDiscountsManagementKeyboard(discounts []database.Discount) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	rows := make([]tele.Row, 0)

	for _, discount := range discounts {
		statusBtn := "✅"
		if !discount.IsActive {
			statusBtn = "❌"
		}

		btn := markup.Data(
			fmt.Sprintf("%s %s (%d%%)", statusBtn, discount.Name, discount.Percentage),
			"admin_edit_discount",
			fmt.Sprintf("%d", discount.ID),
		)
		rows = append(rows, markup.Row(btn))
	}

	// Add discount button
	btnAdd := markup.Data("➕ Создать акцию", "admin_add_discount", "new")
	btnBack := markup.Data("⬅️ Назад", "admin", "main")

	rows = append(rows, markup.Row(btnAdd))
	rows = append(rows, markup.Row(btnBack))

	markup.Inline(rows...)
	return markup
}

// handleAdminAddDiscountStart starts discount creation
func (b *Bot) handleAdminAddDiscountStart(ctx context.Context, c tele.Context) error {
	services, err := b.adminService.GetAllServices(ctx)
	if err != nil {
		return c.Edit("Ошибка загрузки услуг")
	}

	if len(services) == 0 {
		return c.Edit("Сначала создайте услуги")
	}

	state := b.getUserState(c.Sender().ID)
	state.EditMode = "add_discount_service"
	state.TempServiceData = make(map[string]interface{})

	msg := "➕ <b>Создание акции</b>\n\n" +
		"Шаг 1/4: Выберите услугу для акции:"

	return c.Edit(msg, &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: getServicesForDiscountKeyboard(services),
	})
}

// getServicesForDiscountKeyboard returns services selection keyboard
func getServicesForDiscountKeyboard(services []database.Service) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	rows := make([]tele.Row, 0)

	for _, service := range services {
		if !service.IsActive {
			continue
		}
		btn := markup.Data(
			fmt.Sprintf("%s (%d руб.)", service.Name, service.Price/100),
			"admin_discount_select_service",
			fmt.Sprintf("%d", service.ID),
		)
		rows = append(rows, markup.Row(btn))
	}

	btnCancel := markup.Data("❌ Отмена", "admin_cancel_add_discount", "")
	rows = append(rows, markup.Row(btnCancel))

	markup.Inline(rows...)
	return markup
}

// handleAdminDiscountSelectService handles service selection for discount
func (b *Bot) handleAdminDiscountSelectService(ctx context.Context, c tele.Context, serviceIDStr string) error {
	serviceID, err := strconv.ParseUint(serviceIDStr, 10, 32)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка"})
	}

	service, err := b.adminService.GetServiceByID(ctx, uint(serviceID))
	if err != nil {
		return c.Edit("Услуга не найдена")
	}

	state := b.getUserState(c.Sender().ID)
	state.TempServiceData["service_id"] = uint(serviceID)
	state.TempServiceData["service_name"] = service.Name
	state.EditMode = "add_discount_name"

	markup := &tele.ReplyMarkup{}
	btnCancel := markup.Data("❌ Отмена", "admin_cancel_add_discount", "")
	markup.Inline(markup.Row(btnCancel))

	msg := fmt.Sprintf(
		"✅ Услуга: <b>%s</b>\n\n"+
			"Шаг 2/4: Введите название акции\n"+
			"(например: \"Новогодняя распродажа\"):",
		service.Name,
	)

	return c.Edit(msg, &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: markup,
	})
}

// handleAdminAddDiscountMessage handles text input for discount creation
func (b *Bot) handleAdminAddDiscountMessage(c tele.Context) error {
	if !b.isAdmin(c.Sender().ID) {
		return nil
	}

	state := b.getUserState(c.Sender().ID)
	if state.TempServiceData == nil || !strings.HasPrefix(state.EditMode, "add_discount_") {
		return nil
	}

	text := c.Text()
	ctx := context.Background()

	switch state.EditMode {
	case "add_discount_name":
		state.TempServiceData["name"] = text
		state.EditMode = "add_discount_percentage"

		markup := &tele.ReplyMarkup{}
		btnCancel := markup.Data("❌ Отмена", "admin_cancel_add_discount", "")
		markup.Inline(markup.Row(btnCancel))

		return c.Send(
			"✅ Название сохранено!\n\n"+
				"Шаг 3/4: Введите процент скидки\n"+
				"(например: 20 для скидки 20%):",
			&tele.SendOptions{ReplyMarkup: markup},
		)

	case "add_discount_percentage":
		percentage, err := strconv.Atoi(text)
		if err != nil || percentage < 1 || percentage > 99 {
			return c.Send("❌ Неверный формат. Введите число от 1 до 99")
		}
		state.TempServiceData["percentage"] = percentage
		state.EditMode = "add_discount_dates"

		markup := &tele.ReplyMarkup{}
		btnCancel := markup.Data("❌ Отмена", "admin_cancel_add_discount", "")
		markup.Inline(markup.Row(btnCancel))

		return c.Send(
			fmt.Sprintf("✅ Скидка: <b>%d%%</b>\n\n", percentage)+
				"Шаг 4/4: Введите период акции\n"+
				"Формат: ДД.ММ.ГГГГ-ДД.ММ.ГГГГ\n"+
				"(например: 01.12.2024-31.12.2024):",
			&tele.SendOptions{
				ParseMode:   tele.ModeHTML,
				ReplyMarkup: markup,
			},
		)

	case "add_discount_dates":
		// Parse dates
		parts := strings.Split(text, "-")
		if len(parts) != 2 {
			return c.Send("❌ Неверный формат. Используйте: ДД.ММ.ГГГГ-ДД.ММ.ГГГГ")
		}

		startDate, err := time.Parse("02.01.2006", strings.TrimSpace(parts[0]))
		if err != nil {
			return c.Send("❌ Неверная дата начала. Используйте формат: ДД.ММ.ГГГГ")
		}

		endDate, err := time.Parse("02.01.2006", strings.TrimSpace(parts[1]))
		if err != nil {
			return c.Send("❌ Неверная дата окончания. Используйте формат: ДД.ММ.ГГГГ")
		}

		// Set time to end of day for end date
		endDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 0, endDate.Location())

		if endDate.Before(startDate) {
			return c.Send("❌ Дата окончания не может быть раньше даты начала")
		}

		// Create discount
		discountService := services.NewDiscountService()
		discount, err := discountService.CreateDiscount(
			ctx,
			state.TempServiceData["service_id"].(uint),
			state.TempServiceData["name"].(string),
			state.TempServiceData["percentage"].(int),
			startDate,
			endDate,
		)

		// Clear state
		state.EditMode = ""
		state.TempServiceData = nil

		if err != nil {
			return c.Send("❌ Ошибка создания акции: " + err.Error())
		}

		msg := fmt.Sprintf(
			"✅ <b>Акция создана!</b>\n\n"+
				"🎉 <b>%s</b>\n"+
				"Услуга: %s\n"+
				"Скидка: <b>%d%%</b>\n"+
				"Период: %s - %s\n\n"+
				"Клиенты увидят сниженную цену автоматически!",
			discount.Name,
			state.TempServiceData["service_name"].(string),
			discount.Percentage,
			discount.StartDate.Format("02.01.2006"),
			discount.EndDate.Format("02.01.2006"),
		)

		return c.Send(msg, &tele.SendOptions{ParseMode: tele.ModeHTML})
	}

	return nil
}

// handleAdminEditDiscount shows discount editing menu
func (b *Bot) handleAdminEditDiscount(ctx context.Context, c tele.Context, discountIDStr string) error {
	discountID, err := strconv.ParseUint(discountIDStr, 10, 32)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка"})
	}

	discountService := services.NewDiscountService()
	discount, err := discountService.GetDiscountByID(ctx, uint(discountID))
	if err != nil {
		return c.Edit("Акция не найдена")
	}

	status := "Активна ✅"
	if !discount.IsActive {
		status = "Неактивна ❌"
	}

	msg := fmt.Sprintf(
		"🎉 <b>%s</b>\n\n"+
			"Услуга: %s\n"+
			"Скидка: <b>%d%%</b>\n"+
			"Период: %s - %s\n"+
			"Статус: %s",
		discount.Name,
		discount.Service.Name,
		discount.Percentage,
		discount.StartDate.Format("02.01.2006"),
		discount.EndDate.Format("02.01.2006"),
		status,
	)

	markup := &tele.ReplyMarkup{}
	btnToggle := markup.Data("🔄 Вкл/Выкл", "admin_toggle_discount", fmt.Sprintf("%d", discount.ID))
	btnDelete := markup.Data("🗑 Удалить", "admin_delete_discount", fmt.Sprintf("%d", discount.ID))
	btnBack := markup.Data("⬅️ Назад", "admin_discounts", "main")

	markup.Inline(
		markup.Row(btnToggle),
		markup.Row(btnDelete),
		markup.Row(btnBack),
	)

	return c.Edit(msg, &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: markup,
	})
}

// handleAdminToggleDiscount toggles discount status
func (b *Bot) handleAdminToggleDiscount(ctx context.Context, c tele.Context, discountIDStr string) error {
	discountID, err := strconv.ParseUint(discountIDStr, 10, 32)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка"})
	}

	discountService := services.NewDiscountService()
	if err := discountService.ToggleDiscountStatus(ctx, uint(discountID)); err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка изменения статуса"})
	}

	return b.handleAdminEditDiscount(ctx, c, discountIDStr)
}

// handleAdminDeleteDiscount deletes a discount
func (b *Bot) handleAdminDeleteDiscount(ctx context.Context, c tele.Context, discountIDStr string) error {
	discountID, err := strconv.ParseUint(discountIDStr, 10, 32)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка"})
	}

	discountService := services.NewDiscountService()
	if err := discountService.DeleteDiscount(ctx, uint(discountID)); err != nil {
		return c.Edit("❌ Ошибка удаления акции")
	}

	c.Respond(&tele.CallbackResponse{Text: "✅ Акция удалена"})
	return b.handleAdminDiscounts(ctx, c)
}

// handleAdminCancelAddDiscount cancels discount creation
func (b *Bot) handleAdminCancelAddDiscount(ctx context.Context, c tele.Context) error {
	state := b.getUserState(c.Sender().ID)
	state.EditMode = ""
	state.TempServiceData = nil

	return c.Edit("❌ Создание акции отменено")
}

