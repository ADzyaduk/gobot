// Package bot contains discount management handlers
package bot

import (
	"context"
	"fmt"
	"log"
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
	btnMenu := markup.Data("🏠 Главное меню", "back_to_menu", "")

	rows = append(rows, markup.Row(btnAdd))
	rows = append(rows, markup.Row(btnBack, btnMenu))

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
		"Шаг 1/5: Выберите услугу для акции:"

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
	btnMenu := markup.Data("🏠 Главное меню", "back_to_menu", "")
	rows = append(rows, markup.Row(btnCancel))
	rows = append(rows, markup.Row(btnMenu))

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
	btnMenu := markup.Data("🏠 Главное меню", "back_to_menu", "")
	markup.Inline(
		markup.Row(btnCancel),
		markup.Row(btnMenu),
	)

	msg := fmt.Sprintf(
		"✅ Услуга: <b>%s</b>\n\n"+
			"Шаг 2/5: Введите название акции\n"+
			"💡 Например: \"Новогодняя распродажа\", \"Скидка 20%%\", \"Летняя акция\"",
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
				"Шаг 3/5: Выберите процент скидки\n"+
				"💡 Или введите свой процент (от 1 до 99):",
			&tele.SendOptions{
				ParseMode:   tele.ModeHTML,
				ReplyMarkup: getDiscountPercentageKeyboard(),
			},
		)

	case "add_discount_percentage":
		percentage, err := strconv.Atoi(text)
		if err != nil || percentage < 1 || percentage > 99 {
			return c.Send("❌ Неверный формат. Введите число от 1 до 99 или выберите из кнопок")
		}
		state.TempServiceData["percentage"] = percentage
		state.EditMode = "add_discount_dates"

		markup := &tele.ReplyMarkup{}
		btnCancel := markup.Data("❌ Отмена", "admin_cancel_add_discount", "")
		btnMenu := markup.Data("🏠 Главное меню", "back_to_menu", "")
		markup.Inline(
			markup.Row(btnCancel),
			markup.Row(btnMenu),
		)

		return c.Send(
			fmt.Sprintf("✅ Скидка: <b>%d%%</b>\n\n", percentage)+
				"Шаг 4/5: Выберите дату начала акции:",
			&tele.SendOptions{
				ParseMode:   tele.ModeHTML,
				ReplyMarkup: getDiscountStartDateKeyboard(),
			},
		)

	case "add_discount_start_date":
		// Parse start date
		startDate, err := time.Parse("02.01.2006", text)
		if err != nil {
			return c.Send("❌ Неверный формат. Используйте: ДД.ММ.ГГГГ или выберите из кнопок")
		}
		state.TempServiceData["start_date"] = startDate
		state.EditMode = "add_discount_end_date"

		return c.Send(
			fmt.Sprintf("✅ Дата начала: <b>%s</b>\n\n", startDate.Format("02.01.2006"))+
				"Шаг 5/5: Выберите дату окончания акции:",
			&tele.SendOptions{
				ParseMode:   tele.ModeHTML,
				ReplyMarkup: getDiscountEndDateKeyboard(startDate),
			},
		)

	case "add_discount_end_date":
		// Parse end date
		endDate, err := time.Parse("02.01.2006", text)
		if err != nil {
			return c.Send("❌ Неверный формат. Используйте: ДД.ММ.ГГГГ или выберите из кнопок")
		}

		startDate := state.TempServiceData["start_date"].(time.Time)
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

		serviceName := state.TempServiceData["service_name"].(string)

		// Clear state
		state.EditMode = ""
		state.TempServiceData = nil

		if err != nil {
			return c.Send("❌ Ошибка создания акции: " + err.Error())
		}

		markup := &tele.ReplyMarkup{}
		btnBack := markup.Data("⬅️ К акциям", "admin_discounts", "main")
		btnMenu := markup.Data("🏠 Главное меню", "back_to_menu", "")
		markup.Inline(
			markup.Row(btnBack),
			markup.Row(btnMenu),
		)

		msg := fmt.Sprintf(
			"✅ <b>Акция создана!</b>\n\n"+
				"🎉 <b>%s</b>\n"+
				"📋 Услуга: %s\n"+
				"💰 Скидка: <b>%d%%</b>\n"+
				"📅 Период: %s - %s\n\n"+
				"✨ Клиенты увидят сниженную цену автоматически!",
			discount.Name,
			serviceName,
			discount.Percentage,
			discount.StartDate.Format("02.01.2006"),
			discount.EndDate.Format("02.01.2006"),
		)

		return c.Send(msg, &tele.SendOptions{
			ParseMode:   tele.ModeHTML,
			ReplyMarkup: markup,
		})
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
	btnMenu := markup.Data("🏠 Главное меню", "back_to_menu", "")

	markup.Inline(
		markup.Row(btnToggle),
		markup.Row(btnDelete),
		markup.Row(btnBack, btnMenu),
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

	markup := &tele.ReplyMarkup{}
	btnBack := markup.Data("⬅️ К акциям", "admin_discounts", "main")
	btnMenu := markup.Data("🏠 Главное меню", "back_to_menu", "")
	markup.Inline(
		markup.Row(btnBack),
		markup.Row(btnMenu),
	)

	return c.Edit("❌ Создание акции отменено", &tele.SendOptions{ReplyMarkup: markup})
}

// handleAdminDiscountSetPercentage handles percentage selection from keyboard
func (b *Bot) handleAdminDiscountSetPercentage(ctx context.Context, c tele.Context, percentageStr string) error {
	percentage, err := strconv.Atoi(percentageStr)
	if err != nil || percentage < 1 || percentage > 99 {
		return c.Respond(&tele.CallbackResponse{Text: "❌ Неверный процент"})
	}

	state := b.getUserState(c.Sender().ID)
	state.TempServiceData["percentage"] = percentage
	state.EditMode = "add_discount_dates"

	c.Respond(&tele.CallbackResponse{Text: fmt.Sprintf("✅ Скидка: %d%%", percentage)})

	return c.Edit(
		fmt.Sprintf("✅ Скидка: <b>%d%%</b>\n\n", percentage)+
			"Шаг 4/5: Выберите дату начала акции:",
		&tele.SendOptions{
			ParseMode:   tele.ModeHTML,
			ReplyMarkup: getDiscountStartDateKeyboard(),
		},
	)
}

// handleAdminDiscountSetStartDate handles start date selection from keyboard
func (b *Bot) handleAdminDiscountSetStartDate(ctx context.Context, c tele.Context, dateStr string) error {
	startDate, err := time.Parse("02.01.2006", dateStr)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "❌ Неверная дата"})
	}

	state := b.getUserState(c.Sender().ID)
	state.TempServiceData["start_date"] = startDate
	state.EditMode = "add_discount_end_date"

	c.Respond(&tele.CallbackResponse{Text: fmt.Sprintf("✅ Дата начала: %s", startDate.Format("02.01.2006"))})

	return c.Edit(
		fmt.Sprintf("✅ Дата начала: <b>%s</b>\n\n", startDate.Format("02.01.2006"))+
			"Шаг 5/5: Выберите дату окончания акции:",
		&tele.SendOptions{
			ParseMode:   tele.ModeHTML,
			ReplyMarkup: getDiscountEndDateKeyboard(startDate),
		},
	)
}

// handleAdminDiscountSetEndDate handles end date selection and creates discount
func (b *Bot) handleAdminDiscountSetEndDate(ctx context.Context, c tele.Context, dateStr string) error {
	endDate, err := time.Parse("02.01.2006", dateStr)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "❌ Неверная дата"})
	}

	state := b.getUserState(c.Sender().ID)
	startDate := state.TempServiceData["start_date"].(time.Time)

	// Set time to end of day for end date
	endDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 0, endDate.Location())

	if endDate.Before(startDate) {
		return c.Respond(&tele.CallbackResponse{Text: "❌ Дата окончания не может быть раньше даты начала"})
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

	serviceName := state.TempServiceData["service_name"].(string)

	// Clear state
	state.EditMode = ""
	state.TempServiceData = nil

	if err != nil {
		c.Respond(&tele.CallbackResponse{Text: "❌ Ошибка создания акции"})
		return c.Edit("❌ Ошибка создания акции: " + err.Error())
	}

	c.Respond(&tele.CallbackResponse{Text: "✅ Акция создана!"})

	// Send promotion to channel if configured
	if err := b.notificationService.SendPromotionToChannel(ctx, discount); err != nil {
		log.Printf("Error sending promotion to channel: %v", err)
		// Don't fail the whole operation if channel send fails
	}

	markup := &tele.ReplyMarkup{}
	btnBack := markup.Data("⬅️ К акциям", "admin_discounts", "main")
	btnMenu := markup.Data("🏠 Главное меню", "back_to_menu", "")
	markup.Inline(
		markup.Row(btnBack),
		markup.Row(btnMenu),
	)

	msg := fmt.Sprintf(
		"✅ <b>Акция создана!</b>\n\n"+
			"🎉 <b>%s</b>\n"+
			"📋 Услуга: %s\n"+
			"💰 Скидка: <b>%d%%</b>\n"+
			"📅 Период: %s - %s\n\n"+
			"✨ Клиенты увидят сниженную цену автоматически!",
		discount.Name,
		serviceName,
		discount.Percentage,
		discount.StartDate.Format("02.01.2006"),
		discount.EndDate.Format("02.01.2006"),
	)

	return c.Edit(msg, &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: markup,
	})
}
