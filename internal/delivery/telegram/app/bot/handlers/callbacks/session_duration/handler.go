// internal/delivery/telegram/app/bot/handlers/callbacks/session_duration/handler.go
package session_duration

import (
	"fmt"
	"strings"
	"time"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	trading_session "crypto-exchange-screener-bot/internal/delivery/telegram/services/trading_session"
)

// sessionDurationHandler обрабатывает выбор длительности торговой сессии
type sessionDurationHandler struct {
	*base.BaseHandler
	service trading_session.Service
}

func newSessionDurationHandler(service trading_session.Service) handlers.Handler {
	return &sessionDurationHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "session_duration_handler",
			Command: constants.CallbackSessionDuration,
			Type:    handlers.TypeCallback,
		},
		service: service,
	}
}

// Execute стартует сессию с выбранной длительностью
func (h *sessionDurationHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	if params.User == nil {
		return handlers.HandlerResult{}, fmt.Errorf("пользователь не авторизован")
	}

	// Парсим длительность из callback data: "session_duration:2h"
	duration, label, err := parseDuration(params.Data)
	if err != nil {
		return handlers.HandlerResult{}, err
	}

	// Запускаем сессию
	session, err := h.service.Start(params.User.ID, params.ChatID, duration, "telegram")
	if err != nil {
		return handlers.HandlerResult{}, fmt.Errorf("не удалось запустить сессию: %w", err)
	}

	expiresAtStr := formatInUserTZ(session.ExpiresAt, params.User.Timezone)

	// Inline-кнопка завершения сессии
	remaining := time.Until(session.ExpiresAt)
	stopButtonText := fmt.Sprintf("%s (%s)",
		constants.SessionButtonTexts.Stop,
		formatRemaining(remaining),
	)
	stopKeyboard := map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			{{"text": stopButtonText, "callback_data": constants.CallbackSessionStop}},
		},
	}

	message := fmt.Sprintf(
		"🟢 *Сессия запущена!*\n\n"+
			"⏱ Длительность: *%s*\n"+
			"🕐 Завершится в: *%s*\n\n"+
			"✅ Уведомления включены.",
		label,
		expiresAtStr,
	)

	return handlers.HandlerResult{
		Message:  message,
		Keyboard: stopKeyboard,
		Metadata: map[string]interface{}{
			"session_started": true,
			"expires_at":      session.ExpiresAt,
		},
	}, nil
}

// formatRemaining форматирует оставшееся время в формате "Xч Yм" или "Yм"
func formatRemaining(d time.Duration) string {
	if d <= 0 {
		return "0м"
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dч %dм", h, m)
	}
	return fmt.Sprintf("%dм", m)
}

// formatInUserTZ форматирует время в часовом поясе пользователя
func formatInUserTZ(t time.Time, timezone string) string {
	if timezone == "" {
		timezone = "Europe/Moscow"
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return t.Format("15:04")
	}
	return t.In(loc).Format("15:04")
}

// parseDuration извлекает time.Duration и метку из callback data
func parseDuration(data string) (time.Duration, string, error) {
	// data = "session_duration:2h" или "session_duration:4h" ...
	suffix := strings.TrimPrefix(data, "session_duration:")
	switch suffix {
	case "2h":
		return 2 * time.Hour, "2 часа", nil
	case "4h":
		return 4 * time.Hour, "4 часа", nil
	case "8h":
		return 8 * time.Hour, "8 часов", nil
	case "day":
		return 24 * time.Hour, "весь день (24ч)", nil
	default:
		return 0, "", fmt.Errorf("неизвестная длительность: %s", suffix)
	}
}
