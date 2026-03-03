package trading_session_repo

import "crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"

// TradingSessionRepository интерфейс доступа к данным торговых сессий
type TradingSessionRepository interface {
	// Save деактивирует предыдущую сессию пользователя на той же платформе и сохраняет новую
	Save(session *models.TradingSession) error
	// Deactivate помечает все активные сессии пользователя как завершённые (все платформы)
	Deactivate(userID int) error
	// DeactivateByPlatform помечает активную сессию пользователя на указанной платформе как завершённую
	DeactivateByPlatform(userID int, platform string) error
	// FindAllActive возвращает все активные сессии, срок которых ещё не истёк
	FindAllActive() ([]*models.TradingSession, error)
}
