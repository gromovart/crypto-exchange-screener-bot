package trading_session_repo

import "crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"

// TradingSessionRepository интерфейс доступа к данным торговых сессий
type TradingSessionRepository interface {
	// Save деактивирует предыдущую сессию пользователя и сохраняет новую
	Save(session *models.TradingSession) error
	// Deactivate помечает активную сессию пользователя как завершённую
	Deactivate(userID int) error
	// FindAllActive возвращает все активные сессии, срок которых ещё не истёк
	FindAllActive() ([]*models.TradingSession, error)
}
