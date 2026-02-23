// internal/delivery/telegram/services/trading_session/interface.go
package trading_session

import "time"

// TradingSession хранит данные торговой сессии
type TradingSession struct {
	UserID    int       `json:"user_id"`
	ChatID    int64     `json:"chat_id"`
	StartedAt time.Time `json:"started_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Service интерфейс для управления торговыми сессиями
type Service interface {
	// Start запускает торговую сессию для пользователя
	Start(userID int, chatID int64, duration time.Duration) (*TradingSession, error)
	// Stop завершает торговую сессию пользователя
	Stop(userID int) error
	// GetActive возвращает активную сессию пользователя (nil, false если нет)
	GetActive(userID int) (*TradingSession, bool)
	// IsActive проверяет наличие активной сессии
	IsActive(userID int) bool
}
