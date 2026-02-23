// internal/delivery/telegram/app/bot/handlers/callbacks/session_duration/interface.go
package session_duration

import (
	trading_session "crypto-exchange-screener-bot/internal/delivery/telegram/services/trading_session"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
)

// NewHandler создает обработчик выбора длительности сессии
func NewHandler(service trading_session.Service) handlers.Handler {
	return newSessionDurationHandler(service)
}
