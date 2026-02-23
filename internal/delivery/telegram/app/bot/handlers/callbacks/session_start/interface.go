// internal/delivery/telegram/app/bot/handlers/callbacks/session_start/interface.go
package session_start

import (
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	trading_session "crypto-exchange-screener-bot/internal/delivery/telegram/services/trading_session"
)

// NewHandler создает обработчик кнопки "🟢 Начать сессию"
func NewHandler(service trading_session.Service) handlers.Handler {
	return newSessionStartHandler(service)
}
