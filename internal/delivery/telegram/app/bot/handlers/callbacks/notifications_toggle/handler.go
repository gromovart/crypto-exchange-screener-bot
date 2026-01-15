// internal/delivery/telegram/app/bot/handlers/callbacks/notifications_toggle/handler.go
package notifications_toggle

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	notificationsToggleSvc "crypto-exchange-screener-bot/internal/delivery/telegram/services/notifications_toggle"
)

// notifyToggleHandler реализация обработчика toggle уведомлений
type notifyToggleHandler struct {
	*base.BaseHandler
	service notificationsToggleSvc.Service
}

// NewHandler создает новый обработчик toggle уведомлений
func NewHandler(service notificationsToggleSvc.Service) handlers.Handler {
	return &notifyToggleHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "notify_toggle_handler",
			Command: constants.CallbackNotifyToggleAll,
			Type:    handlers.TypeCallback,
		},
		service: service,
	}
}

// Execute выполняет обработку toggle уведомлений
func (h *notifyToggleHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	if params.User == nil {
		return handlers.HandlerResult{}, fmt.Errorf("пользователь не авторизован")
	}

	// Подготавливаем данные для сервиса
	serviceParams := notificationsToggleSvc.NotificationsToggleResultParams{
		Data: map[string]interface{}{
			"user_id":   params.User.ID,
			"new_state": !params.User.NotificationsEnabled,
			"chat_id":   params.ChatID,
		},
	}

	// Вызываем сервис
	result, err := h.service.Exec(serviceParams)
	if err != nil {
		return handlers.HandlerResult{}, fmt.Errorf("ошибка в сервисе уведомлений: %w", err)
	}

	// Создаем простое сообщение с результатом
	message := fmt.Sprintf(
		"🔔 *Настройки уведомлений*\n\n%s\n\n"+
			"Настройки для конкретных типов сигналов можно изменить в меню уведомлений.",
		result.Message,
	)

	// Простая клавиатура с кнопкой "Назад"
	keyboard := map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			{
				{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackNotificationsMenu},
			},
		},
	}

	return handlers.HandlerResult{
		Message:  message,
		Keyboard: keyboard,
		Metadata: map[string]interface{}{
			"user_id":                 params.User.ID,
			"notifications_new_state": !params.User.NotificationsEnabled,
		},
	}, nil
}
