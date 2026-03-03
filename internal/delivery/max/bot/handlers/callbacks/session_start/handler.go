// internal/delivery/max/bot/handlers/callbacks/session_start/handler.go
package session_start

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
	tradingSession "crypto-exchange-screener-bot/internal/delivery/telegram/services/trading_session"
)

// Handler — показывает меню выбора длительности торговой сессии.
// Если сессия уже активна — показывает её статус.
type Handler struct {
	*base.BaseHandler
	sessionService tradingSession.Service
}

// New создаёт обработчик
func New(svc tradingSession.Service) handlers.Handler {
	return &Handler{
		BaseHandler:    base.New("session_start", kb.CbSessionStart, handlers.TypeCallback),
		sessionService: svc,
	}
}

// Execute выполняет обработку
func (h *Handler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	user := params.User
	if user == nil {
		return handlers.HandlerResult{Message: "❌ Пользователь не найден"}, nil
	}

	// Если сессия уже активна — показываем статус и кнопку завершения
	if h.sessionService != nil {
		if session, ok := h.sessionService.GetActive(user.ID, "max"); ok {
			msg := fmt.Sprintf(
				"🟢 Торговая сессия активна\n\n"+
					"Начата: %s\n"+
					"Завершится: %s\n\n"+
					"Уведомления включены.",
				session.StartedAt.Format("15:04"),
				session.ExpiresAt.Format("15:04 02.01"),
			)
			rows := [][]map[string]string{
				{kb.B(kb.Btn.SessionStop, kb.CbSessionStop)},
				kb.BackRow(kb.CbMenuMain),
			}
			return handlers.HandlerResult{
				Message:     msg,
				Keyboard:    kb.Keyboard(rows),
				EditMessage: params.MessageID != "",
			}, nil
		}
	}

	// Сессии нет — предлагаем выбрать длительность
	msg := "🟢 Начать торговую сессию\n\n" +
		"Выберите длительность сессии.\n" +
		"На это время включатся уведомления о сигналах."

	rows := [][]map[string]string{
		{
			kb.B(kb.Btn.Duration2h, kb.CbSessionDuration+":2h"),
			kb.B(kb.Btn.Duration4h, kb.CbSessionDuration+":4h"),
		},
		{
			kb.B(kb.Btn.Duration8h, kb.CbSessionDuration+":8h"),
			kb.B(kb.Btn.DurationDay, kb.CbSessionDuration+":day"),
		},
		kb.BackRow(kb.CbMenuMain),
	}

	return handlers.HandlerResult{
		Message:     msg,
		Keyboard:    kb.Keyboard(rows),
		EditMessage: params.MessageID != "",
	}, nil
}
