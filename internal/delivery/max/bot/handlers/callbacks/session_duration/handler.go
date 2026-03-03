// internal/delivery/max/bot/handlers/callbacks/session_duration/handler.go
package session_duration

import (
	"fmt"
	"strings"
	"time"

	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
	tradingSession "crypto-exchange-screener-bot/internal/delivery/telegram/services/trading_session"
)

// Handler — запускает торговую сессию с выбранной длительностью.
// Ожидаемый формат params.Data: "session_duration:2h|4h|8h|day"
type Handler struct {
	*base.BaseHandler
	sessionService tradingSession.Service
}

// New создаёт обработчик
func New(svc tradingSession.Service) handlers.Handler {
	return &Handler{
		BaseHandler:    base.New("session_duration", kb.CbSessionDuration, handlers.TypeCallback),
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

	// Парсим длительность из data: "session_duration:4h" → "4h"
	durationStr := ""
	if strings.Contains(params.Data, ":") {
		parts := strings.SplitN(params.Data, ":", 2)
		if len(parts) == 2 {
			durationStr = parts[1]
		}
	}

	duration, label := parseDuration(durationStr)
	if duration == 0 {
		return handlers.HandlerResult{
			Message:     "❌ Неверная длительность сессии.",
			Keyboard:    kb.Keyboard([][]map[string]string{kb.BackRow(kb.CbSessionStart)}),
			EditMessage: params.MessageID != "",
		}, nil
	}

	session, err := h.sessionService.Start(user.ID, params.ChatID, duration, "max")
	if err != nil {
		return handlers.HandlerResult{
			Message:     fmt.Sprintf("❌ Не удалось запустить сессию: %v", err),
			Keyboard:    kb.Keyboard([][]map[string]string{kb.BackRow(kb.CbMenuMain)}),
			EditMessage: params.MessageID != "",
		}, nil
	}

	msg := fmt.Sprintf(
		"🟢 Торговая сессия запущена!\n\n"+
			"Длительность: %s\n"+
			"Завершится: %s\n\n"+
			"Уведомления включены. Вы будете получать сигналы до конца сессии.",
		label,
		session.ExpiresAt.Format("15:04 02.01.2006"),
	)

	// Inline-кнопка для завершения (в сообщении)
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

// parseDuration преобразует строку длительности в time.Duration и читаемую метку
func parseDuration(s string) (time.Duration, string) {
	switch s {
	case "2h":
		return 2 * time.Hour, "2 часа"
	case "4h":
		return 4 * time.Hour, "4 часа"
	case "8h":
		return 8 * time.Hour, "8 часов"
	case "day":
		return 24 * time.Hour, "весь день (24ч)"
	default:
		return 0, ""
	}
}
