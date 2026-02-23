// internal/delivery/telegram/app/bot/handlers/session_start/handler.go
package session_start

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	trading_session "crypto-exchange-screener-bot/internal/delivery/telegram/services/trading_session"
)

// sessionStartHandler обрабатывает нажатие кнопки "🟢 Начать сессию"
type sessionStartHandler struct {
	*base.BaseHandler
	service trading_session.Service
}

func newSessionStartHandler(service trading_session.Service) handlers.Handler {
	return &sessionStartHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "session_start_handler",
			Command: constants.SessionButtonTexts.Start,
			Type:    handlers.TypeMessage,
		},
		service: service,
	}
}

// Execute показывает выбор длительности сессии
func (h *sessionStartHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	if params.User == nil {
		return handlers.HandlerResult{}, fmt.Errorf("пользователь не авторизован")
	}

	// Если сессия уже активна — сообщаем об этом
	if h.service.IsActive(params.User.ID) {
		session, _ := h.service.GetActive(params.User.ID)
		remaining := session.ExpiresAt.Sub(session.StartedAt)
		_ = remaining

		keyboard := map[string]interface{}{
			"inline_keyboard": [][]map[string]string{
				{
					{"text": constants.SessionButtonTexts.Duration2h, "callback_data": "session_duration:2h"},
					{"text": constants.SessionButtonTexts.Duration4h, "callback_data": "session_duration:4h"},
				},
				{
					{"text": constants.SessionButtonTexts.Duration8h, "callback_data": "session_duration:8h"},
					{"text": constants.SessionButtonTexts.DurationDay, "callback_data": "session_duration:day"},
				},
			},
		}

		message := fmt.Sprintf(
			"⚡ *Сессия уже активна*\n\n"+
				"Истекает: *%s*\n\n"+
				"Вы можете выбрать новую длительность (сессия перезапустится):",
			session.ExpiresAt.Format("15:04"),
		)

		return handlers.HandlerResult{
			Message:  message,
			Keyboard: keyboard,
		}, nil
	}

	// Показываем выбор длительности
	keyboard := map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			{
				{"text": constants.SessionButtonTexts.Duration2h, "callback_data": "session_duration:2h"},
				{"text": constants.SessionButtonTexts.Duration4h, "callback_data": "session_duration:4h"},
			},
			{
				{"text": constants.SessionButtonTexts.Duration8h, "callback_data": "session_duration:8h"},
				{"text": constants.SessionButtonTexts.DurationDay, "callback_data": "session_duration:day"},
			},
		},
	}

	return handlers.HandlerResult{
		Message:  "⏱ *Выберите длительность торговой сессии:*\n\nПосле старта уведомления включатся автоматически.",
		Keyboard: keyboard,
	}, nil
}
