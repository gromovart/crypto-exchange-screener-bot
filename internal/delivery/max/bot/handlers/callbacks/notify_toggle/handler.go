// internal/delivery/max/bot/handlers/callbacks/notify_toggle/handler.go
package notify_toggle

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
	notifySvc "crypto-exchange-screener-bot/internal/delivery/telegram/services/notifications_toggle"
)

// Handler — обработчик переключения уведомлений
type Handler struct {
	*base.BaseHandler
	service notifySvc.Service
}

// New создаёт обработчик с зависимостями
func New(svc notifySvc.Service) handlers.Handler {
	return &Handler{
		BaseHandler: base.New("notify_toggle", kb.CbNotifyToggleAll, handlers.TypeCallback),
		service:     svc,
	}
}

// Execute выполняет обработку
func (h *Handler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	user := params.User
	if user == nil {
		return handlers.HandlerResult{Message: "❌ Пользователь не найден"}, nil
	}

	newState := !user.NotificationsEnabled

	_, err := h.service.Exec(notifySvc.NotificationsToggleResultParams{
		Data: map[string]interface{}{
			"user_id":   user.ID,
			"new_state": newState,
			"chat_id":   params.ChatID,
		},
	})
	if err != nil {
		return handlers.HandlerResult{
			Message:     fmt.Sprintf("❌ Ошибка: %v", err),
			Keyboard:    kb.Keyboard([][]map[string]string{{kb.B(kb.Btn.Back, kb.CbNotificationsMenu)}}),
			EditMessage: params.MessageID != "",
		}, nil
	}

	statusMsg := "Уведомления выключены ❌"
	if newState {
		statusMsg = "Уведомления включены ✅"
	}

	msg := fmt.Sprintf(
		"🔔 *Настройки уведомлений*\n\n%s\n\n"+
			"Настройки для конкретных типов сигналов можно изменить в меню уведомлений.",
		statusMsg,
	)

	return handlers.HandlerResult{
		Message:     msg,
		Keyboard:    kb.Keyboard([][]map[string]string{{kb.B(kb.Btn.Back, kb.CbNotificationsMenu)}}),
		EditMessage: params.MessageID != "",
	}, nil
}
