// internal/delivery/telegram/app/bot/handlers/session_start/interface.go
package session_start

import (
	trading_session "crypto-exchange-screener-bot/internal/delivery/telegram/services/trading_session"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
)

// NewHandler создает обработчик кнопки "🟢 Начать сессию"
func NewHandler(service trading_session.Service) handlers.Handler {
	return newSessionStartHandler(service)
}
