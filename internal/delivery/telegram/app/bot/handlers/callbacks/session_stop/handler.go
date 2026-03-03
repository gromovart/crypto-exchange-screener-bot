// internal/delivery/telegram/app/bot/handlers/callbacks/session_stop/handler.go
package session_stop

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	trading_session "crypto-exchange-screener-bot/internal/delivery/telegram/services/trading_session"
)

// sessionStopHandler обрабатывает нажатие кнопки "🔴 Завершить сессию"
type sessionStopHandler struct {
	*base.BaseHandler
	service trading_session.Service
}

func newSessionStopHandler(service trading_session.Service) handlers.Handler {
	return &sessionStopHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "session_stop_handler",
			Command: constants.CallbackSessionStop,
			Type:    handlers.TypeCallback,
		},
		service: service,
	}
}

// Execute завершает торговую сессию
func (h *sessionStopHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	if params.User == nil {
		return handlers.HandlerResult{}, fmt.Errorf("пользователь не авторизован")
	}

	startKeyboard := map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			{{"text": constants.SessionButtonTexts.Start, "callback_data": constants.CallbackSessionStart}},
		},
	}

	if !h.service.IsActive(params.User.ID, "telegram") {
		return handlers.HandlerResult{
			Message:  "ℹ️ Активной торговой сессии нет.",
			Keyboard: startKeyboard,
		}, nil
	}

	if err := h.service.Stop(params.User.ID, "telegram"); err != nil {
		return handlers.HandlerResult{}, fmt.Errorf("ошибка завершения сессии: %w", err)
	}

	return handlers.HandlerResult{
		Message:  "🔴 *Сессия завершена*\n\nУведомления отключены.",
		Keyboard: startKeyboard,
	}, nil
}
