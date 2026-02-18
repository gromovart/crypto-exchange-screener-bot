// internal/delivery/telegram/app/bot/handlers/commands/notifications/handler.go
package notifications

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
)

// notificationsCommandHandler реализация обработчика команды /notifications
type notificationsCommandHandler struct {
	*base.BaseHandler
}

// NewHandler создает новый обработчик команды /notifications
func NewHandler() handlers.Handler {
	return &notificationsCommandHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "notifications_command_handler",
			Command: "notifications",
			Type:    handlers.TypeCommand,
		},
	}
}

// Execute выполняет обработку команды /notifications
func (h *notificationsCommandHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
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

// createNotificationsMessage создает сообщение для команды /notifications
func (h *notificationsCommandHandler) createNotificationsMessage(user *models.User) string {
	// Используем методы через h.BaseHandler или напрямую h
	notifyGrowthText := h.BaseHandler.GetToggleText("📈 Рост", user.NotifyGrowth)
	notifyFallText := h.BaseHandler.GetToggleText("📉 Падение", user.NotifyFall)

	return fmt.Sprintf(
		"%s\n\n"+
			"Текущие настройки:\n\n"+
			"🔊 Общие уведомления: %s\n"+
			"%s\n"+
			"%s\n"+
			"Выберите настройку для изменения:",
		constants.AuthButtonTexts.Notifications,
		h.BaseHandler.GetBoolDisplay(user.NotificationsEnabled),
		notifyGrowthText,
		notifyFallText,
	)
}

// createNotificationsKeyboard создает клавиатуру для команды /notifications
func (h *notificationsCommandHandler) createNotificationsKeyboard(user *models.User) interface{} {
	// ⭐ ИСПРАВЛЕНО: для ToggleAll не используем GetToggleText
	toggleAllText := constants.NotificationButtonTexts.ToggleAll

	// Для остальных кнопок используем GetToggleText как обычно
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
