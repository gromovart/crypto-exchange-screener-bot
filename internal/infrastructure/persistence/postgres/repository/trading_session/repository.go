// /internal/infrastructure/persistence/postgres/repository/trading_session/repository.go
package trading_session_repo

import (
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type tradingSessionRepoImpl struct {
	db *sqlx.DB
}

// NewTradingSessionRepository создаёт реализацию TradingSessionRepository
func NewTradingSessionRepository(db *sqlx.DB) TradingSessionRepository {
	return &tradingSessionRepoImpl{db: db}
}

// Save деактивирует предыдущую активную сессию на той же платформе и вставляет новую
func (r *tradingSessionRepoImpl) Save(session *models.TradingSession) error {
	if session.Platform == "" {
		session.Platform = "telegram"
	}
	if err := r.DeactivateByPlatform(session.UserID, session.Platform); err != nil {
		logger.Warn("⚠️ TradingSessionRepo.Save: не удалось деактивировать старую сессию user=%d platform=%s: %v",
			session.UserID, session.Platform, err)
	}

	query := `
		INSERT INTO trading_sessions (user_id, chat_id, platform, started_at, expires_at, is_active)
		VALUES (:user_id, :chat_id, :platform, :started_at, :expires_at, TRUE)
	`
	_, err := r.db.NamedExec(query, session)
	if err != nil {
		return fmt.Errorf("TradingSessionRepo.Save: %w", err)
	}

	logger.Info("💾 Торговая сессия сохранена в БД: user=%d platform=%s, expires=%s",
		session.UserID, session.Platform, session.ExpiresAt.Format("15:04:05"))
	return nil
}

// Deactivate помечает все активные сессии пользователя как завершённые (все платформы)
func (r *tradingSessionRepoImpl) Deactivate(userID int) error {
	query := `
		UPDATE trading_sessions
		SET is_active = FALSE, updated_at = NOW()
		WHERE user_id = $1 AND is_active = TRUE
	`
	_, err := r.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("TradingSessionRepo.Deactivate: %w", err)
	}
	return nil
}

// DeactivateByPlatform помечает активную сессию пользователя на указанной платформе как завершённую
func (r *tradingSessionRepoImpl) DeactivateByPlatform(userID int, platform string) error {
	query := `
		UPDATE trading_sessions
		SET is_active = FALSE, updated_at = NOW()
		WHERE user_id = $1 AND platform = $2 AND is_active = TRUE
	`
	_, err := r.db.Exec(query, userID, platform)
	if err != nil {
		return fmt.Errorf("TradingSessionRepo.DeactivateByPlatform: %w", err)
	}
	return nil
}

// FindAllActive возвращает все активные сессии, срок которых ещё не истёк
func (r *tradingSessionRepoImpl) FindAllActive() ([]*models.TradingSession, error) {
	query := `
		SELECT id, user_id, chat_id, platform, started_at, expires_at, is_active, created_at, updated_at
		FROM trading_sessions
		WHERE is_active = TRUE AND expires_at > NOW()
	`
	var sessions []*models.TradingSession
	if err := r.db.Select(&sessions, query); err != nil {
		return nil, fmt.Errorf("TradingSessionRepo.FindAllActive: %w", err)
	}
	return sessions, nil
}
