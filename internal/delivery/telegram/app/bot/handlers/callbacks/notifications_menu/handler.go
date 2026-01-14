// internal/delivery/telegram/app/bot/handlers/callbacks/notifications_menu/handler.go
package notifications_menu

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
)

// notificationsMenuHandler реализация обработчика меню уведомлений
type notificationsMenuHandler struct {
	*base.BaseHandler
}

// NewHandler создает новый обработчик меню уведомлений
func NewHandler() handlers.Handler {
	return &notificationsMenuHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "notifications_menu_handler",
			Command: constants.CallbackNotificationsMenu,
			Type:    handlers.TypeCallback,
		},
	}
}

// Execute выполняет обработку callback меню уведомлений
func (h *notificationsMenuHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	if params.User == nil {
		return handlers.HandlerResult{}, fmt.Errorf("пользователь не авторизован")
	}

	message := h.createNotificationsMessage(params.User)
	keyboard := h.createNotificationsKeyboard(params.User)

	return handlers.HandlerResult{
		Message:  message,
		Keyboard: keyboard,
		Metadata: map[string]interface{}{
			"user_id": params.User.ID,
		},
	}, nil
}

// createNotificationsMessage создает сообщение для меню уведомлений
func (h *notificationsMenuHandler) createNotificationsMessage(user *models.User) string {
	// Используем методы через h.BaseHandler или напрямую h
	notifyGrowthText := h.BaseHandler.GetToggleText("📈 Рост", user.NotifyGrowth)
	notifyFallText := h.BaseHandler.GetToggleText("📉 Падение", user.NotifyFall)

	return fmt.Sprintf(
		"%s\n\n"+
			"Текущие настройки:\n\n"+
			"🔊 Общие уведомления: %s\n"+
			"%s\n"+
			"%s\n"+
			"⏰ Тихие часы: %02d:00 - %02d:00\n\n"+
			"Выберите настройку для изменения:",
		constants.MenuButtonTexts.Notifications,
		h.BaseHandler.GetBoolDisplay(user.NotificationsEnabled),
		notifyGrowthText,
		notifyFallText,
		user.QuietHoursStart,
		user.QuietHoursEnd,
	)
}

// createNotificationsKeyboard создает клавиатуру для меню уведомлений
func (h *notificationsMenuHandler) createNotificationsKeyboard(user *models.User) interface{} {
	// Используем методы через h.BaseHandler
	toggleAllText := h.BaseHandler.GetToggleText(constants.NotificationButtonTexts.ToggleAll, user.NotificationsEnabled)
	growthText := h.BaseHandler.GetToggleText(constants.NotificationButtonTexts.GrowthOnly, user.NotifyGrowth)
	fallText := h.BaseHandler.GetToggleText(constants.NotificationButtonTexts.FallOnly, user.NotifyFall)

	return map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			{
				{"text": toggleAllText, "callback_data": constants.CallbackNotifyToggleAll},
			},
			{
				{"text": growthText, "callback_data": constants.CallbackNotifyGrowthOnly},
				{"text": fallText, "callback_data": constants.CallbackNotifyFallOnly},
			},
			{
				{"text": constants.NotificationButtonTexts.Both, "callback_data": constants.CallbackNotifyBoth},
			},
			{
				{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackMenuMain},
			},
		},
	}
}
