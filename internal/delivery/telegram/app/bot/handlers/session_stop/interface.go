// internal/delivery/telegram/app/bot/handlers/session_stop/interface.go
package session_stop

import (
	trading_session "crypto-exchange-screener-bot/internal/delivery/telegram/services/trading_session"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
)

// NewHandler создает обработчик кнопки "🔴 Завершить сессию"
func NewHandler(service trading_session.Service) handlers.Handler {
	return newSessionStopHandler(service)
}
