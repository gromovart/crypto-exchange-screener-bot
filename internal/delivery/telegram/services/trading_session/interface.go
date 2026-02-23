// internal/delivery/telegram/services/trading_session/interface.go
package trading_session

import (
	"time"
)

// TradingSession данные сессии для внешнего использования
type TradingSession struct {
	UserID    int
	ChatID    int64
	StartedAt time.Time
	ExpiresAt time.Time
}

// Service интерфейс для управления торговыми сессиями
type Service interface {
	Start(userID int, chatID int64, duration time.Duration) (*TradingSession, error)
	Stop(userID int) error
	GetActive(userID int) (*TradingSession, bool)
	IsActive(userID int) bool
}
