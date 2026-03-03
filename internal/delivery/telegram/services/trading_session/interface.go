// internal/delivery/telegram/services/trading_session/interface.go
package trading_session

import (
	"time"
)

// TradingSession данные сессии для внешнего использования
type TradingSession struct {
	UserID    int
	ChatID    int64
	Platform  string // "telegram" | "max"
	StartedAt time.Time
	ExpiresAt time.Time
}

// Service интерфейс для управления торговыми сессиями
// platform — идентификатор мессенджера: "telegram" или "max".
// Сессии разных платформ независимы: можно одновременно иметь
// активную сессию в Telegram и активную сессию в MAX.
type Service interface {
	Start(userID int, chatID int64, duration time.Duration, platform string) (*TradingSession, error)
	Stop(userID int, platform string) error
	GetActive(userID int, platform string) (*TradingSession, bool)
	IsActive(userID int, platform string) bool
}
