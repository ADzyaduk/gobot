// Package bot contains admin-specific handlers
package bot

import (
	"context"
	"fmt"
	"strconv"

	"gobot/internal/database"

	tele "gopkg.in/telebot.v3"
)

// handleAdminServicesManagement shows services management interface
func (b *Bot) handleAdminServicesManagement(ctx context.Context, c tele.Context) error {
	if !b.isAdmin(c.Sender().ID) {
		return c.Respond(&tele.CallbackResponse{Text: "❌ Нет доступа"})
	}

	services, err := b.adminService.GetAllServices(ctx)
	if err != nil {
		return c.Edit("Ошибка при загрузке услуг")
	}

	if len(services) == 0 {
		return c.Edit("Услуг пока нет", getAddServiceKeyboard())
	}

	msg := "🛠 <b>Управление услугами</b>\n\n"
	for _, service := range services {
		status := "✅"
		if !service.IsActive {
			status = "❌"
		}
		msg += fmt.Sprintf(
			"%s <b>%s</b>\n"+
				"   💰 %d руб. | ⏱ %d мин\n"+
				"   📝 %s\n\n",
			status,
			service.Name,
			service.Price/100,
			service.Duration,
			service.Description,
		)
	}

	return c.Edit(msg, &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: getServicesManagementKeyboard(services),
	})
}

// getServicesManagementKeyboard returns keyboard for services management
func getServicesManagementKeyboard(services []database.Service) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	rows := make([]tele.Row, 0)

	for _, service := range services {
		statusBtn := "✅"
		if !service.IsActive {
			statusBtn = "❌"
		}

		btnEdit := markup.Data(
			fmt.Sprintf("%s %s", statusBtn, service.Name),
			"admin_edit_service_menu",
			fmt.Sprintf("%d", service.ID),
		)
		rows = append(rows, markup.Row(btnEdit))
	}

	// Add service button
	btnAdd := markup.Data("➕ Добавить услугу", "admin_add_service", "new")
	btnBack := markup.Data("⬅️ Назад", "admin", "main")

	rows = append(rows, markup.Row(btnAdd))
	rows = append(rows, markup.Row(btnBack))

	markup.Inline(rows...)
	return markup
}

// getAddServiceKeyboard returns keyboard to add service
func getAddServiceKeyboard() *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}

	btnAdd := markup.Data("➕ Добавить услугу", "admin_add_service", "new")
	btnBack := markup.Data("⬅️ Назад", "admin", "main")

	markup.Inline(
		markup.Row(btnAdd),
		markup.Row(btnBack),
	)

	return markup
}

// getServiceEditKeyboard returns keyboard for editing specific service
func getServiceEditKeyboard(serviceID uint) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}

	btnToggle := markup.Data("🔄 Вкл/Выкл", "admin_toggle_service", fmt.Sprintf("%d", serviceID))
	btnDelete := markup.Data("🗑 Удалить", "admin_delete_service", fmt.Sprintf("%d", serviceID))
	btnBack := markup.Data("⬅️ Назад", "admin", "services")

	markup.Inline(
		markup.Row(btnToggle),
		markup.Row(btnDelete),
		markup.Row(btnBack),
	)

	return markup
}

// handleAdminEditService shows service edit options
func (b *Bot) handleAdminEditService(ctx context.Context, c tele.Context, serviceIDStr string) error {
	serviceID, err := strconv.ParseUint(serviceIDStr, 10, 32)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка"})
	}

	service, err := b.adminService.GetServiceByID(ctx, uint(serviceID))
	if err != nil {
		return c.Edit("Услуга не найдена")
	}

	status := "Активна ✅"
	if !service.IsActive {
		status = "Неактивна ❌"
	}

	msg := fmt.Sprintf(
		"📋 <b>%s</b>\n\n"+
			"💰 Цена: %d руб.\n"+
			"⏱ Длительность: %d мин\n"+
			"📝 Описание: %s\n"+
			"Статус: %s",
		service.Name,
		service.Price/100,
		service.Duration,
		service.Description,
		status,
	)

	return c.Edit(msg, &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: getServiceEditKeyboard(service.ID),
	})
}

// handleAdminToggleService toggles service active status
func (b *Bot) handleAdminToggleService(ctx context.Context, c tele.Context, serviceIDStr string) error {
	serviceID, err := strconv.ParseUint(serviceIDStr, 10, 32)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка"})
	}

	if err := b.adminService.ToggleServiceStatus(ctx, uint(serviceID)); err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка изменения статуса"})
	}

	return b.handleAdminEditService(ctx, c, serviceIDStr)
}

// handleAdminDeleteService deletes a service
func (b *Bot) handleAdminDeleteService(ctx context.Context, c tele.Context, serviceIDStr string) error {
	serviceID, err := strconv.ParseUint(serviceIDStr, 10, 32)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Ошибка"})
	}

	if err := b.adminService.DeleteService(ctx, uint(serviceID)); err != nil {
		return c.Edit("❌ Ошибка удаления услуги")
	}

	c.Respond(&tele.CallbackResponse{Text: "✅ Услуга удалена"})
	return b.handleAdminServicesManagement(ctx, c)
}

// handleAdminBookingsDetailed shows detailed bookings list
func (b *Bot) handleAdminBookingsDetailed(ctx context.Context, c tele.Context) error {
	if !b.isAdmin(c.Sender().ID) {
		return c.Respond(&tele.CallbackResponse{Text: "❌ Нет доступа"})
	}

	bookings, err := b.adminService.GetAllBookings(ctx, 50, 0)
	if err != nil {
		return c.Edit("Ошибка при загрузке записей")
	}

	if len(bookings) == 0 {
		return c.Edit("📋 Записей пока нет")
	}

	msg := "📋 <b>Все записи:</b>\n\n"
	for i, booking := range bookings {
		if i >= 15 { // Limit for message size
			msg += fmt.Sprintf("\n... и еще %d записей", len(bookings)-15)
			break
		}

		statusEmoji := getStatusEmoji(booking.Status)
		msg += fmt.Sprintf(
			"%d. %s <b>%s</b>\n"+
				"   👤 %s %s (@%s)\n"+
				"   📆 %s в %s\n"+
				"   💰 %d руб.\n\n",
			i+1,
			statusEmoji,
			booking.Service.Name,
			booking.User.FirstName,
			booking.User.LastName,
			booking.User.Username,
			booking.Date.Format("02.01.2006"),
			booking.Time,
			booking.Service.Price/100,
		)
	}

	markup := &tele.ReplyMarkup{}
	btnBack := markup.Data("⬅️ Назад", "admin", "main")
	markup.Inline(markup.Row(btnBack))

	return c.Edit(msg, &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: markup,
	})
}

// handleAdminStatsDetailed shows detailed statistics
func (b *Bot) handleAdminStatsDetailed(ctx context.Context, c tele.Context) error {
	if !b.isAdmin(c.Sender().ID) {
		return c.Respond(&tele.CallbackResponse{Text: "❌ Нет доступа"})
	}

	stats, err := b.adminService.GetStats(ctx)
	if err != nil {
		return c.Edit("Ошибка загрузки статистики")
	}

	msg := fmt.Sprintf(
		"📊 <b>Статистика системы</b>\n\n"+
			"👥 Всего пользователей: <b>%d</b>\n"+
			"📋 Всего записей: <b>%d</b>\n"+
			"✅ Активных записей: <b>%d</b>\n"+
			"✔️ Завершенных записей: <b>%d</b>\n"+
			"🛠 Активных услуг: <b>%d</b>\n",
		stats["total_users"],
		stats["total_bookings"],
		stats["active_bookings"],
		stats["completed_bookings"],
		stats["active_services"],
	)

	markup := &tele.ReplyMarkup{}
	btnBack := markup.Data("⬅️ Назад", "admin", "main")
	markup.Inline(markup.Row(btnBack))

	return c.Edit(msg, &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: markup,
	})
}
