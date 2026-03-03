// internal/delivery/max/bot/handlers/callbacks/session_stop/handler.go
package session_stop

import (
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
	tradingSession "crypto-exchange-screener-bot/internal/delivery/telegram/services/trading_session"
)

// Handler — завершает торговую сессию
type Handler struct {
	*base.BaseHandler
	sessionService tradingSession.Service
}

// New создаёт обработчик
func New(svc tradingSession.Service) handlers.Handler {
	return &Handler{
		BaseHandler:    base.New("session_stop", kb.CbSessionStop, handlers.TypeCallback),
		sessionService: svc,
	}
}

// Execute выполняет обработку
func (h *Handler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	user := params.User
	if user == nil {
		return handlers.HandlerResult{Message: "❌ Пользователь не найден"}, nil
	}

	if h.sessionService == nil {
		return handlers.HandlerResult{Message: "❌ Сервис сессий недоступен"}, nil
	}

	if err := h.sessionService.Stop(user.ID, "max"); err != nil {
		return handlers.HandlerResult{
			Message:     "❌ Не удалось завершить сессию. Попробуйте позже.",
			Keyboard:    kb.Keyboard([][]map[string]string{kb.BackRow(kb.CbMenuMain)}),
			EditMessage: params.MessageID != "",
		}, nil
	}

	// Inline-кнопка для начала новой сессии (в сообщении)
	rows := [][]map[string]string{
		{kb.B(kb.Btn.SessionStart, kb.CbSessionStart)},
		kb.BackRow(kb.CbMenuMain),
	}

	return handlers.HandlerResult{
		Message:     "🔴 Торговая сессия завершена.\n\nУведомления отключены.",
		Keyboard:    kb.Keyboard(rows),
		EditMessage: params.MessageID != "",
	}, nil
}
