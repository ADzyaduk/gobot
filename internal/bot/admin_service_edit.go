// Package bot contains service editing handlers
package bot

import (
	"context"
	"fmt"
	"strconv"

	tele "gopkg.in/telebot.v3"
)

// handleAdminEditServiceMenu shows editing menu for a service
func (b *Bot) handleAdminEditServiceMenu(ctx context.Context, c tele.Context, serviceIDStr string) error {
	serviceID, err := strconv.ParseUint(serviceIDStr, 10, 32)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка"})
	}

	service, err := b.adminService.GetServiceByID(ctx, uint(serviceID))
	if err != nil {
		return c.Edit("Услуга не найдена")
	}

	// Save to user state for editing
	state := b.getUserState(c.Sender().ID)
	state.EditServiceID = service.ID

	status := "Активна ✅"
	if !service.IsActive {
		status = "Неактивна ❌"
	}

	msg := fmt.Sprintf(
		"✏️ <b>Редактирование услуги</b>\n\n"+
			"<b>%s</b>\n\n"+
			"💰 Цена: <b>%d руб.</b>\n"+
			"⏱ Длительность: <b>%d мин</b>\n"+
			"📝 Описание: %s\n"+
			"Статус: %s\n\n"+
			"Выберите что хотите изменить:",
		service.Name,
		service.Price/100,
		service.Duration,
		service.Description,
		status,
	)

	return c.Edit(msg, &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: getServiceEditMenuKeyboard(service.ID),
	})
}

// getServiceEditMenuKeyboard returns keyboard for service editing menu
func getServiceEditMenuKeyboard(serviceID uint) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}

	btnName := markup.Data("📝 Изменить название", "admin_edit_field", fmt.Sprintf("name:%d", serviceID))
	btnPrice := markup.Data("💰 Изменить цену", "admin_edit_field", fmt.Sprintf("price:%d", serviceID))
	btnDuration := markup.Data("⏱ Изменить длительность", "admin_edit_field", fmt.Sprintf("duration:%d", serviceID))
	btnDesc := markup.Data("📝 Изменить описание", "admin_edit_field", fmt.Sprintf("description:%d", serviceID))
	btnToggle := markup.Data("🔄 Вкл/Выкл", "admin_toggle_service", fmt.Sprintf("%d", serviceID))
	btnDelete := markup.Data("🗑 Удалить", "admin_delete_service", fmt.Sprintf("%d", serviceID))
	btnBack := markup.Data("⬅️ Назад", "admin", "services")

	markup.Inline(
		markup.Row(btnName),
		markup.Row(btnPrice, btnDuration),
		markup.Row(btnDesc),
		markup.Row(btnToggle, btnDelete),
		markup.Row(btnBack),
	)

	return markup
}

// handleAdminEditField starts editing a specific field
func (b *Bot) handleAdminEditField(ctx context.Context, c tele.Context, data string) error {
	// Parse data: "field:serviceID"
	var field string
	var serviceID uint64
	_, err := fmt.Sscanf(data, "%[^:]:%d", &field, &serviceID)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка"})
	}

	service, err := b.adminService.GetServiceByID(ctx, uint(serviceID))
	if err != nil {
		return c.Edit("Услуга не найдена")
	}

	// Set edit mode in user state
	state := b.getUserState(c.Sender().ID)
	state.EditMode = field
	state.EditServiceID = uint(serviceID)

	var msg string
	switch field {
	case "name":
		msg = fmt.Sprintf(
			"📝 <b>Изменение названия</b>\n\n"+
				"Текущее: %s\n\n"+
				"Отправьте новое название:",
			service.Name,
		)
	case "price":
		msg = fmt.Sprintf(
			"💰 <b>Изменение цены</b>\n\n"+
				"Текущая: %d руб.\n\n"+
				"Отправьте новую цену (в рублях):",
			service.Price/100,
		)
	case "duration":
		msg = fmt.Sprintf(
			"⏱ <b>Изменение длительности</b>\n\n"+
				"Текущая: %d мин\n\n"+
				"Отправьте новую длительность (в минутах):",
			service.Duration,
		)
	case "description":
		msg = fmt.Sprintf(
			"📝 <b>Изменение описания</b>\n\n"+
				"Текущее: %s\n\n"+
				"Отправьте новое описание:",
			service.Description,
		)
	default:
		return c.Respond(&tele.CallbackResponse{Text: "Неизвестное поле"})
	}

	markup := &tele.ReplyMarkup{}
	btnCancel := markup.Data("❌ Отмена", "admin_cancel_edit", "")
	markup.Inline(markup.Row(btnCancel))

	return c.Edit(msg, &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: markup,
	})
}

// handleAdminCancelEdit cancels editing
func (b *Bot) handleAdminCancelEdit(ctx context.Context, c tele.Context) error {
	state := b.getUserState(c.Sender().ID)
	serviceID := state.EditServiceID
	state.EditMode = ""
	state.EditServiceID = 0

	return b.handleAdminEditServiceMenu(ctx, c, fmt.Sprintf("%d", serviceID))
}

// handleAdminTextMessage handles text messages during editing
func (b *Bot) handleAdminTextMessage(c tele.Context) error {
	if !b.isAdmin(c.Sender().ID) {
		return nil
	}

	state := b.getUserState(c.Sender().ID)
	if state.EditMode == "" {
		return nil // Not in edit mode
	}

	ctx := context.Background()
	serviceID := state.EditServiceID
	text := c.Text()

	var err error
	switch state.EditMode {
	case "name":
		err = b.adminService.UpdateServiceField(ctx, serviceID, "name", text)
	case "price":
		price, parseErr := strconv.Atoi(text)
		if parseErr != nil {
			return c.Send("❌ Неверный формат цены. Введите число (например: 2500)")
		}
		// Convert rubles to cents
		err = b.adminService.UpdateServiceField(ctx, serviceID, "price", price*100)
	case "duration":
		duration, parseErr := strconv.Atoi(text)
		if parseErr != nil {
			return c.Send("❌ Неверный формат длительности. Введите число (например: 60)")
		}
		err = b.adminService.UpdateServiceField(ctx, serviceID, "duration", duration)
	case "description":
		err = b.adminService.UpdateServiceField(ctx, serviceID, "description", text)
	default:
		return c.Send("❌ Ошибка редактирования")
	}

	if err != nil {
		return c.Send("❌ Ошибка сохранения: " + err.Error())
	}

	// Clear edit mode
	state.EditMode = ""

	service, _ := b.adminService.GetServiceByID(ctx, serviceID)
	msg := fmt.Sprintf(
		"✅ Успешно обновлено!\n\n"+
			"<b>%s</b>\n"+
			"💰 %d руб. | ⏱ %d мин\n"+
			"📝 %s",
		service.Name,
		service.Price/100,
		service.Duration,
		service.Description,
	)

	return c.Send(msg, &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: getServiceEditMenuKeyboard(serviceID),
	})
}

// handleAdminAddServiceStart starts the service creation dialog
func (b *Bot) handleAdminAddServiceStart(ctx context.Context, c tele.Context) error {
	state := b.getUserState(c.Sender().ID)
	state.EditMode = "add_service_name"
	state.TempServiceData = make(map[string]interface{})

	markup := &tele.ReplyMarkup{}
	btnCancel := markup.Data("❌ Отмена", "admin_cancel_add_service", "")
	markup.Inline(markup.Row(btnCancel))

	msg := "➕ <b>Добавление новой услуги</b>\n\n" +
		"Шаг 1/4: Введите название услуги:"

	return c.Edit(msg, &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: markup,
	})
}

// handleAdminAddServiceMessage handles messages during service creation
func (b *Bot) handleAdminAddServiceMessage(c tele.Context) error {
	if !b.isAdmin(c.Sender().ID) {
		return nil
	}

	state := b.getUserState(c.Sender().ID)
	if state.TempServiceData == nil {
		return nil
	}

	text := c.Text()
	ctx := context.Background()

	switch state.EditMode {
	case "add_service_name":
		state.TempServiceData["name"] = text
		state.EditMode = "add_service_price"
		return c.Send("✅ Название сохранено!\n\nШаг 2/4: Введите цену (в рублях):")

	case "add_service_price":
		price, err := strconv.Atoi(text)
		if err != nil {
			return c.Send("❌ Неверный формат. Введите число (например: 2500)")
		}
		state.TempServiceData["price"] = price * 100
		state.EditMode = "add_service_duration"
		return c.Send("✅ Цена сохранена!\n\nШаг 3/4: Введите длительность (в минутах):")

	case "add_service_duration":
		duration, err := strconv.Atoi(text)
		if err != nil {
			return c.Send("❌ Неверный формат. Введите число (например: 60)")
		}
		state.TempServiceData["duration"] = duration
		state.EditMode = "add_service_description"
		return c.Send("✅ Длительность сохранена!\n\nШаг 4/4: Введите описание услуги:")

	case "add_service_description":
		state.TempServiceData["description"] = text

		// Create service
		_, err := b.adminService.CreateService(
			ctx,
			state.TempServiceData["name"].(string),
			state.TempServiceData["description"].(string),
			state.TempServiceData["duration"].(int),
			state.TempServiceData["price"].(int),
		)

		// Clear state
		state.EditMode = ""
		state.TempServiceData = nil

		if err != nil {
			return c.Send("❌ Ошибка создания услуги: " + err.Error())
		}

		msg := fmt.Sprintf(
			"✅ <b>Услуга успешно создана!</b>\n\n"+
				"<b>%s</b>\n"+
				"💰 %d руб. | ⏱ %d мин\n"+
				"📝 %s",
			state.TempServiceData["name"].(string),
			state.TempServiceData["price"].(int)/100,
			state.TempServiceData["duration"].(int),
			text,
		)

		return c.Send(msg, &tele.SendOptions{ParseMode: tele.ModeHTML})
	}

	return nil
}

// handleAdminCancelAddService cancels service creation
func (b *Bot) handleAdminCancelAddService(ctx context.Context, c tele.Context) error {
	state := b.getUserState(c.Sender().ID)
	state.EditMode = ""
	state.TempServiceData = nil

	return c.Edit("❌ Создание услуги отменено")
}
