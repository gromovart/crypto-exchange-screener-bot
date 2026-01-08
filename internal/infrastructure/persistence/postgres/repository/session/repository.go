// internal/infrastructure/persistence/postgres/repository/session/repository.go
package session

import (
	"context"
	"crypto-exchange-screener-bot/internal/infrastructure/cache/redis"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// SessionRepository интерфейс для работы с сессиями
type SessionRepository interface {
	// Основные методы
	Create(session *models.Session) error
	FindByID(sessionID string) (*models.Session, error)
	FindByToken(token string) (*models.Session, error)
	FindByUserID(userID int) ([]*models.Session, error)
	UpdateActivity(sessionID string) error
	Revoke(sessionID, reason string) error
	RevokeByToken(token, reason string) error
	RevokeAllUserSessions(userID int, reason string) (int, error)
	Extend(sessionID string, newExpiry time.Time) error
	Delete(sessionID string) error

	// Дополнительные методы
	CreateWithDetails(session *SessionRecord) error
	FindByTokenWithDetails(token string) (*SessionRecord, error)
	CleanupExpired(ctx context.Context) (int, error)
	GetActiveSessionsCount() (int, error)
	GetUserActiveSessionsCount(userID int) (int, error)
	GetSessionStats(ctx context.Context) (*SessionStats, error)
	UpdateSessionData(sessionID string, data map[string]interface{}) error
	GetRecentlyActiveSessions(hours int) ([]*SessionRecord, error)
	GetSessionActivityLog(sessionID string, limit int) ([]map[string]interface{}, error)
	LogSessionActivity(sessionID, activityType string, details map[string]interface{}) error
	GenerateNewToken(sessionID string) (string, error)
	GetSessionsByIP(ipAddress string) ([]*SessionRecord, error)
	BulkRevokeInactive(ctx context.Context, inactiveFor time.Duration) (int, error)
	CreateSessionForUser(userID int, userAgent, ipAddress string) (*models.Session, error)
	CreateSessionForUserWithDetails(userID int, userAgent, ipAddress string, deviceInfo, data map[string]interface{}) (*SessionRecord, error)
	ConvertToUserSession(record *SessionRecord) *models.Session
}

// JSONMap для работы с JSON полями
type JSONMap map[string]interface{}

// SessionRecord структура для работы с базой данных
type SessionRecord struct {
	ID            string                 `db:"id"`
	UserID        int                    `db:"user_id"`
	Token         string                 `db:"token"`
	DeviceInfo    map[string]interface{} `db:"device_info"`
	IPAddress     string                 `db:"ip_address"`
	UserAgent     string                 `db:"user_agent"`
	Data          map[string]interface{} `db:"data"`
	ExpiresAt     time.Time              `db:"expires_at"`
	CreatedAt     time.Time              `db:"created_at"`
	UpdatedAt     time.Time              `db:"updated_at"`
	LastActivity  time.Time              `db:"last_activity"`
	IsActive      bool                   `db:"is_active"`
	RevokedAt     *time.Time             `db:"revoked_at"`
	RevokedReason string                 `db:"revoked_reason"`
}

// SessionStats статистика сессий
type SessionStats struct {
	TotalSessions      int           `json:"total_sessions"`
	ActiveSessions     int           `json:"active_sessions"`
	AvgSessionDuration time.Duration `json:"avg_session_duration"`
	MaxConcurrent      int           `json:"max_concurrent"`
	MostActiveHour     int           `json:"most_active_hour"`
}

// SessionRepositoryImpl реализация репозитория сессий
type SessionRepositoryImpl struct {
	db    *sqlx.DB
	cache *redis.Cache
}

// NewSessionRepository создает новый репозиторий сессий
func NewSessionRepository(db *sqlx.DB, cache *redis.Cache) *SessionRepositoryImpl {
	return &SessionRepositoryImpl{db: db, cache: cache}
}

// Create создает новую сессию
func (r *SessionRepositoryImpl) Create(session *models.Session) error {
	// Начинаем транзакцию
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
    INSERT INTO user_sessions (
        id, user_id, token, ip_address,
        user_agent, expires_at, created_at
    ) VALUES ($1, $2, $3, $4, $5, $6, $7)
    RETURNING created_at
    `

	err = tx.QueryRow(
		query,
		session.ID,
		session.UserID,
		session.Token,
		session.IP,
		session.UserAgent,
		session.ExpiresAt,
		time.Now(),
	).Scan(&session.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	// Фиксируем транзакцию
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Инвалидируем кэш
	r.invalidateSessionCache(session.ID, session.UserID)

	return nil
}

// CreateWithDetails создает сессию с дополнительными данными
func (r *SessionRepositoryImpl) CreateWithDetails(session *SessionRecord) error {
	// Начинаем транзакцию
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	deviceInfoJSON, err := json.Marshal(session.DeviceInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal device info: %w", err)
	}

	dataJSON, err := json.Marshal(session.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	query := `
    INSERT INTO user_sessions (
        id, user_id, token, device_info, ip_address,
        user_agent, expires_at, data, created_at
    ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    RETURNING created_at
    `

	err = tx.QueryRow(
		query,
		session.ID,
		session.UserID,
		session.Token,
		deviceInfoJSON,
		session.IPAddress,
		session.UserAgent,
		session.ExpiresAt,
		dataJSON,
		time.Now(),
	).Scan(&session.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create session with details: %w", err)
	}

	// Фиксируем транзакцию
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Инвалидируем кэш
	r.invalidateSessionCache(session.ID, session.UserID)

	return nil
}

// FindByToken находит сессию по токену
func (r *SessionRepositoryImpl) FindByToken(token string) (*models.Session, error) {
	// Попробуем получить из кэша
	cacheKey := fmt.Sprintf("session:token:%s", token)
	var cachedSession models.Session
	if err := r.cache.Get(context.Background(), cacheKey, &cachedSession); err == nil {
		// Проверяем не истекла ли сессия
		if cachedSession.ExpiresAt.After(time.Now()) {
			return &cachedSession, nil
		}
	}

	query := `
    SELECT
        s.id, s.user_id, s.token, s.ip_address,
        s.user_agent, s.expires_at, s.created_at,
        u.telegram_id, u.username, u.first_name, u.last_name, u.chat_id,
        u.role, u.is_active as user_active
    FROM user_sessions s
    JOIN users u ON s.user_id = u.id
    WHERE s.token = $1
      AND s.is_active = TRUE
      AND s.expires_at > NOW()
      AND s.revoked_at IS NULL
      AND u.is_active = TRUE
    `

	var session models.Session
	var user models.User
	var ipAddress string

	err := r.db.QueryRow(query, token).Scan(
		&session.ID,
		&session.UserID,
		&session.Token,
		&ipAddress,
		&session.UserAgent,
		&session.ExpiresAt,
		&session.CreatedAt,
		&user.TelegramID,
		&user.Username,
		&user.FirstName,
		&user.LastName,
		&user.ChatID,
		&user.Role,
		&user.IsActive,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Присваиваем IP
	session.IP = ipAddress
	session.User = &user

	// Сохраняем в кэш
	r.cache.Set(context.Background(), cacheKey, session, 5*time.Minute)

	return &session, nil
}

// FindByTokenWithDetails находит сессию с деталями
func (r *SessionRepositoryImpl) FindByTokenWithDetails(token string) (*SessionRecord, error) {
	// Попробуем получить из кэша
	cacheKey := fmt.Sprintf("session:details:token:%s", token)
	var cachedSession SessionRecord
	if err := r.cache.Get(context.Background(), cacheKey, &cachedSession); err == nil {
		return &cachedSession, nil
	}

	query := `
    SELECT
        id, user_id, token, device_info, ip_address,
        user_agent, expires_at, data, created_at, updated_at,
        last_activity, is_active, revoked_at, revoked_reason
    FROM user_sessions
    WHERE token = $1
    `

	var session SessionRecord
	var deviceInfoJSON, dataJSON []byte

	err := r.db.QueryRow(query, token).Scan(
		&session.ID,
		&session.UserID,
		&session.Token,
		&deviceInfoJSON,
		&session.IPAddress,
		&session.UserAgent,
		&session.ExpiresAt,
		&dataJSON,
		&session.CreatedAt,
		&session.UpdatedAt,
		&session.LastActivity,
		&session.IsActive,
		&session.RevokedAt,
		&session.RevokedReason,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Декодируем JSON данные
	if len(deviceInfoJSON) > 0 {
		if err := json.Unmarshal(deviceInfoJSON, &session.DeviceInfo); err != nil {
			return nil, fmt.Errorf("failed to unmarshal device info: %w", err)
		}
	}

	if len(dataJSON) > 0 {
		if err := json.Unmarshal(dataJSON, &session.Data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal session data: %w", err)
		}
	}

	// Сохраняем в кэш
	r.cache.Set(context.Background(), cacheKey, session, 5*time.Minute)

	return &session, nil
}

// FindByID находит сессию по ID
func (r *SessionRepositoryImpl) FindByID(sessionID string) (*SessionRecord, error) {
	// Попробуем получить из кэша
	cacheKey := fmt.Sprintf("session:id:%s", sessionID)
	var cachedSession SessionRecord
	if err := r.cache.Get(context.Background(), cacheKey, &cachedSession); err == nil {
		return &cachedSession, nil
	}

	query := `
    SELECT
        id, user_id, token, device_info, ip_address,
        user_agent, expires_at, data, created_at, updated_at,
        last_activity, is_active, revoked_at, revoked_reason
    FROM user_sessions
    WHERE id = $1
    `

	var session SessionRecord
	var deviceInfoJSON, dataJSON []byte

	err := r.db.QueryRow(query, sessionID).Scan(
		&session.ID,
		&session.UserID,
		&session.Token,
		&deviceInfoJSON,
		&session.IPAddress,
		&session.UserAgent,
		&session.ExpiresAt,
		&dataJSON,
		&session.CreatedAt,
		&session.UpdatedAt,
		&session.LastActivity,
		&session.IsActive,
		&session.RevokedAt,
		&session.RevokedReason,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Декодируем JSON данные
	if len(deviceInfoJSON) > 0 {
		if err := json.Unmarshal(deviceInfoJSON, &session.DeviceInfo); err != nil {
			return nil, fmt.Errorf("failed to unmarshal device info: %w", err)
		}
	}

	if len(dataJSON) > 0 {
		if err := json.Unmarshal(dataJSON, &session.Data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal session data: %w", err)
		}
	}

	// Сохраняем в кэш
	r.cache.Set(context.Background(), cacheKey, session, 10*time.Minute)

	return &session, nil
}

// FindByUserID находит все активные сессии пользователя
func (r *SessionRepositoryImpl) FindByUserID(userID int) ([]*models.Session, error) {
	// Попробуем получить из кэша
	cacheKey := fmt.Sprintf("sessions:user:%d", userID)
	var cachedSessions []*models.Session
	if err := r.cache.Get(context.Background(), cacheKey, &cachedSessions); err == nil {
		return cachedSessions, nil
	}

	query := `
    SELECT
        id, user_id, token, ip_address,
        user_agent, expires_at, created_at, last_activity
    FROM user_sessions
    WHERE user_id = $1
      AND is_active = TRUE
      AND expires_at > NOW()
      AND revoked_at IS NULL
    ORDER BY created_at DESC
    `

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*models.Session

	for rows.Next() {
		var session models.Session
		var lastActivity sql.NullTime

		err := rows.Scan(
			&session.ID,
			&session.UserID,
			&session.Token,
			&session.IP,
			&session.UserAgent,
			&session.ExpiresAt,
			&session.CreatedAt,
			&lastActivity,
		)

		if err != nil {
			return nil, err
		}

		if lastActivity.Valid {
			session.LastActivity = lastActivity.Time
		}

		sessions = append(sessions, &session)
	}

	// Сохраняем в кэш
	r.cache.Set(context.Background(), cacheKey, sessions, 2*time.Minute)

	return sessions, nil
}

// UpdateActivity обновляет время последней активности сессии
func (r *SessionRepositoryImpl) UpdateActivity(sessionID string) error {
	query := `
    UPDATE user_sessions
    SET last_activity = NOW(),
        updated_at = NOW()
    WHERE id = $1 AND is_active = TRUE
    `

	result, err := r.db.Exec(query, sessionID)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	// Инвалидируем кэш
	r.invalidateSessionCache(sessionID, 0)

	return nil
}

// Revoke отзывает сессию
func (r *SessionRepositoryImpl) Revoke(sessionID, reason string) error {
	// Начинаем транзакцию
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
    UPDATE user_sessions
    SET is_active = FALSE,
        revoked_at = NOW(),
        revoked_reason = $1,
        updated_at = NOW()
    WHERE id = $2 AND is_active = TRUE
    `

	result, err := tx.Exec(query, reason, sessionID)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	// Фиксируем транзакцию
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Инвалидируем кэш
	r.invalidateSessionCache(sessionID, 0)

	return nil
}

// RevokeAllUserSessions отзывает все сессии пользователя
func (r *SessionRepositoryImpl) RevokeAllUserSessions(userID int, reason string) (int, error) {
	// Начинаем транзакцию
	tx, err := r.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
    UPDATE user_sessions
    SET is_active = FALSE,
        revoked_at = NOW(),
        revoked_reason = $1,
        updated_at = NOW()
    WHERE user_id = $2
      AND is_active = TRUE
      AND revoked_at IS NULL
    `

	result, err := tx.Exec(query, reason, userID)
	if err != nil {
		return 0, err
	}

	rowsAffected, _ := result.RowsAffected()

	// Фиксируем транзакцию
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Инвалидируем кэш
	r.invalidateUserSessionsCache(userID)

	return int(rowsAffected), nil
}

// RevokeByToken отзывает сессию по токену
func (r *SessionRepositoryImpl) RevokeByToken(token, reason string) error {
	// Сначала найдем сессию по токену
	session, err := r.FindByTokenWithDetails(token)
	if err != nil {
		return err
	}
	if session == nil {
		return sql.ErrNoRows
	}

	// Отзываем сессию
	return r.Revoke(session.ID, reason)
}

// Extend продлевает срок действия сессии
func (r *SessionRepositoryImpl) Extend(sessionID string, newExpiry time.Time) error {
	query := `
    UPDATE user_sessions
    SET expires_at = $1,
        updated_at = NOW()
    WHERE id = $2 AND is_active = TRUE
    `

	result, err := r.db.Exec(query, newExpiry, sessionID)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	// Инвалидируем кэш
	r.invalidateSessionCache(sessionID, 0)

	return nil
}

// Delete удаляет сессию
func (r *SessionRepositoryImpl) Delete(sessionID string) error {
	// Сначала получим информацию о сессии для инвалидации кэша
	session, err := r.FindByID(sessionID)
	if err != nil {
		return err
	}
	if session == nil {
		return sql.ErrNoRows
	}

	// Удаляем сессию
	query := `DELETE FROM user_sessions WHERE id = $1`
	result, err := r.db.Exec(query, sessionID)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	// Инвалидируем кэш
	r.invalidateSessionCache(sessionID, session.UserID)

	return nil
}

// CleanupExpired очищает истекшие сессии
func (r *SessionRepositoryImpl) CleanupExpired(ctx context.Context) (int, error) {
	query := `
    DELETE FROM user_sessions
    WHERE expires_at < NOW() - INTERVAL '7 days'
      AND is_active = FALSE
    `

	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}

	rowsAffected, _ := result.RowsAffected()
	return int(rowsAffected), nil
}

// GetActiveSessionsCount возвращает количество активных сессий
func (r *SessionRepositoryImpl) GetActiveSessionsCount() (int, error) {
	// Попробуем получить из кэша
	cacheKey := "sessions:active:count"
	var cachedCount int
	if err := r.cache.Get(context.Background(), cacheKey, &cachedCount); err == nil {
		return cachedCount, nil
	}

	query := `
    SELECT COUNT(*)
    FROM user_sessions
    WHERE is_active = TRUE
      AND expires_at > NOW()
      AND revoked_at IS NULL
    `

	var count int
	err := r.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, err
	}

	// Сохраняем в кэш
	r.cache.Set(context.Background(), cacheKey, count, 1*time.Minute)

	return count, nil
}

// GetUserActiveSessionsCount возвращает количество активных сессий пользователя
func (r *SessionRepositoryImpl) GetUserActiveSessionsCount(userID int) (int, error) {
	// Попробуем получить из кэша
	cacheKey := fmt.Sprintf("sessions:active:user:%d:count", userID)
	var cachedCount int
	if err := r.cache.Get(context.Background(), cacheKey, &cachedCount); err == nil {
		return cachedCount, nil
	}

	query := `
    SELECT COUNT(*)
    FROM user_sessions
    WHERE user_id = $1
      AND is_active = TRUE
      AND expires_at > NOW()
      AND revoked_at IS NULL
    `

	var count int
	err := r.db.QueryRow(query, userID).Scan(&count)
	if err != nil {
		return 0, err
	}

	// Сохраняем в кэш
	r.cache.Set(context.Background(), cacheKey, count, 2*time.Minute)

	return count, nil
}

// GetSessionStats возвращает статистику сессий
func (r *SessionRepositoryImpl) GetSessionStats(ctx context.Context) (*SessionStats, error) {
	// Попробуем получить из кэша
	cacheKey := "sessions:stats"
	var cachedStats SessionStats
	if err := r.cache.Get(ctx, cacheKey, &cachedStats); err == nil {
		return &cachedStats, nil
	}

	stats := &SessionStats{}

	// Общее количество сессий
	err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM user_sessions").Scan(&stats.TotalSessions)
	if err != nil {
		return nil, err
	}

	// Активные сессии
	err = r.db.QueryRowContext(ctx, `
        SELECT COUNT(*)
        FROM user_sessions
        WHERE is_active = TRUE
          AND expires_at > NOW()
          AND revoked_at IS NULL
    `).Scan(&stats.ActiveSessions)
	if err != nil {
		return nil, err
	}

	// Средняя продолжительность сессии
	var avgSeconds float64
	err = r.db.QueryRowContext(ctx, `
        SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (last_activity - created_at))), 0)
        FROM user_sessions
        WHERE is_active = FALSE
          AND last_activity IS NOT NULL
    `).Scan(&avgSeconds)
	if err != nil {
		return nil, err
	}
	stats.AvgSessionDuration = time.Duration(avgSeconds * float64(time.Second))

	// Максимальное количество одновременных сессий за последние 24 часа
	err = r.db.QueryRowContext(ctx, `
        SELECT COALESCE(MAX(concurrent_count), 0)
        FROM (
            SELECT COUNT(*) as concurrent_count
            FROM user_sessions
            WHERE created_at >= NOW() - INTERVAL '24 hours'
            GROUP BY DATE_TRUNC('hour', created_at)
        ) as hourly_counts
    `).Scan(&stats.MaxConcurrent)
	if err != nil {
		return nil, err
	}

	// Самый активный час
	err = r.db.QueryRowContext(ctx, `
        SELECT EXTRACT(HOUR FROM created_at) as hour
        FROM user_sessions
        WHERE created_at >= NOW() - INTERVAL '7 days'
        GROUP BY EXTRACT(HOUR FROM created_at)
        ORDER BY COUNT(*) DESC
        LIMIT 1
    `).Scan(&stats.MostActiveHour)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Сохраняем в кэш
	r.cache.Set(ctx, cacheKey, stats, 5*time.Minute)

	return stats, nil
}

// UpdateSessionData обновляет дополнительные данные сессии
func (r *SessionRepositoryImpl) UpdateSessionData(sessionID string, data map[string]interface{}) error {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	query := `
    UPDATE user_sessions
    SET data = $1,
        updated_at = NOW()
    WHERE id = $2
    `

	result, err := r.db.Exec(query, dataJSON, sessionID)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	// Инвалидируем кэш
	r.invalidateSessionCache(sessionID, 0)

	return nil
}

// GetRecentlyActiveSessions возвращает недавно активные сессии
func (r *SessionRepositoryImpl) GetRecentlyActiveSessions(hours int) ([]*SessionRecord, error) {
	query := `
    SELECT
        id, user_id, token, device_info, ip_address,
        user_agent, expires_at, data, created_at, updated_at,
        last_activity, is_active, revoked_at, revoked_reason
    FROM user_sessions
    WHERE last_activity >= NOW() - INTERVAL '1 hour' * $1
      AND is_active = TRUE
    ORDER BY last_activity DESC
    `

	rows, err := r.db.Query(query, hours)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*SessionRecord

	for rows.Next() {
		var session SessionRecord
		var deviceInfoJSON, dataJSON []byte

		err := rows.Scan(
			&session.ID,
			&session.UserID,
			&session.Token,
			&deviceInfoJSON,
			&session.IPAddress,
			&session.UserAgent,
			&session.ExpiresAt,
			&dataJSON,
			&session.CreatedAt,
			&session.UpdatedAt,
			&session.LastActivity,
			&session.IsActive,
			&session.RevokedAt,
			&session.RevokedReason,
		)

		if err != nil {
			return nil, err
		}

		// Декодируем JSON данные
		if len(deviceInfoJSON) > 0 {
			if err := json.Unmarshal(deviceInfoJSON, &session.DeviceInfo); err != nil {
				return nil, fmt.Errorf("failed to unmarshal device info: %w", err)
			}
		}

		if len(dataJSON) > 0 {
			if err := json.Unmarshal(dataJSON, &session.Data); err != nil {
				return nil, fmt.Errorf("failed to unmarshal session data: %w", err)
			}
		}

		sessions = append(sessions, &session)
	}

	return sessions, nil
}

// GetSessionActivityLog возвращает лог активности сессии
func (r *SessionRepositoryImpl) GetSessionActivityLog(sessionID string, limit int) ([]map[string]interface{}, error) {
	query := `
    SELECT
        activity_type, details, ip_address, created_at
    FROM session_activities
    WHERE session_id = $1
    ORDER BY created_at DESC
    LIMIT $2
    `

	rows, err := r.db.Query(query, sessionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []map[string]interface{}

	for rows.Next() {
		var activityType, ipAddress string
		var createdAt time.Time
		var detailsJSON []byte

		err := rows.Scan(&activityType, &detailsJSON, &ipAddress, &createdAt)
		if err != nil {
			return nil, err
		}

		var detailsMap map[string]interface{}
		if len(detailsJSON) > 0 {
			if err := json.Unmarshal(detailsJSON, &detailsMap); err != nil {
				detailsMap = map[string]interface{}{"raw": string(detailsJSON)}
			}
		} else {
			detailsMap = make(map[string]interface{})
		}

		activities = append(activities, map[string]interface{}{
			"activity_type": activityType,
			"details":       detailsMap,
			"ip_address":    ipAddress,
			"created_at":    createdAt,
		})
	}

	return activities, nil
}

// LogSessionActivity записывает активность сессии
func (r *SessionRepositoryImpl) LogSessionActivity(sessionID, activityType string, details map[string]interface{}) error {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("failed to marshal activity details: %w", err)
	}

	query := `
    INSERT INTO session_activities (session_id, activity_type, details)
    VALUES ($1, $2, $3)
    `

	_, err = r.db.Exec(query, sessionID, activityType, detailsJSON)
	return err
}

// GenerateNewToken генерирует новый токен для сессии
func (r *SessionRepositoryImpl) GenerateNewToken(sessionID string) (string, error) {
	newToken := uuid.New().String()

	// Начинаем транзакцию
	tx, err := r.db.Begin()
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
    UPDATE user_sessions
    SET token = $1,
        updated_at = NOW()
    WHERE id = $2 AND is_active = TRUE
    `

	result, err := tx.Exec(query, newToken, sessionID)
	if err != nil {
		return "", err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return "", sql.ErrNoRows
	}

	// Фиксируем транзакцию
	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Инвалидируем кэш
	r.invalidateSessionCache(sessionID, 0)

	return newToken, nil
}

// GetSessionsByIP возвращает сессии по IP адресу
func (r *SessionRepositoryImpl) GetSessionsByIP(ipAddress string) ([]*SessionRecord, error) {
	query := `
    SELECT
        id, user_id, token, device_info, ip_address,
        user_agent, expires_at, data, created_at, updated_at,
        last_activity, is_active, revoked_at, revoked_reason
    FROM user_sessions
    WHERE ip_address = $1
      AND created_at >= NOW() - INTERVAL '30 days'
    ORDER BY created_at DESC
    `

	rows, err := r.db.Query(query, ipAddress)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*SessionRecord

	for rows.Next() {
		var session SessionRecord
		var deviceInfoJSON, dataJSON []byte

		err := rows.Scan(
			&session.ID,
			&session.UserID,
			&session.Token,
			&deviceInfoJSON,
			&session.IPAddress,
			&session.UserAgent,
			&session.ExpiresAt,
			&dataJSON,
			&session.CreatedAt,
			&session.UpdatedAt,
			&session.LastActivity,
			&session.IsActive,
			&session.RevokedAt,
			&session.RevokedReason,
		)

		if err != nil {
			return nil, err
		}

		// Декодируем JSON данные
		if len(deviceInfoJSON) > 0 {
			if err := json.Unmarshal(deviceInfoJSON, &session.DeviceInfo); err != nil {
				return nil, fmt.Errorf("failed to unmarshal device info: %w", err)
			}
		}

		if len(dataJSON) > 0 {
			if err := json.Unmarshal(dataJSON, &session.Data); err != nil {
				return nil, fmt.Errorf("failed to unmarshal session data: %w", err)
			}
		}

		sessions = append(sessions, &session)
	}

	return sessions, nil
}

// BulkRevokeInactive массово отзывает неактивные сессии
func (r *SessionRepositoryImpl) BulkRevokeInactive(ctx context.Context, inactiveFor time.Duration) (int, error) {
	query := `
    UPDATE user_sessions
    SET is_active = FALSE,
        revoked_at = NOW(),
        revoked_reason = 'inactive',
        updated_at = NOW()
    WHERE is_active = TRUE
      AND last_activity < NOW() - $1
      AND revoked_at IS NULL
    `

	result, err := r.db.ExecContext(ctx, query, inactiveFor)
	if err != nil {
		return 0, err
	}

	rowsAffected, _ := result.RowsAffected()

	// Инвалидируем общий кэш сессий
	r.invalidateSessionsCache()

	return int(rowsAffected), nil
}

// CreateSessionForUser создает новую сессию для пользователя
func (r *SessionRepositoryImpl) CreateSessionForUser(userID int, userAgent, ipAddress string) (*models.Session, error) {
	session := &models.Session{
		ID:        uuid.New().String(),
		UserID:    userID,
		Token:     uuid.New().String(),
		IP:        ipAddress,
		UserAgent: userAgent,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour), // 30 дней
		CreatedAt: time.Now(),
	}

	if err := r.Create(session); err != nil {
		return nil, err
	}

	return session, nil
}

// CreateSessionForUserWithDetails создает сессию с деталями
func (r *SessionRepositoryImpl) CreateSessionForUserWithDetails(userID int, userAgent, ipAddress string, deviceInfo, data map[string]interface{}) (*SessionRecord, error) {
	session := &SessionRecord{
		ID:         uuid.New().String(),
		UserID:     userID,
		Token:      uuid.New().String(),
		DeviceInfo: deviceInfo,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		Data:       data,
		ExpiresAt:  time.Now().Add(30 * 24 * time.Hour), // 30 дней
		IsActive:   true,
	}

	if err := r.CreateWithDetails(session); err != nil {
		return nil, err
	}

	return session, nil
}

// ConvertToUserSession конвертирует SessionRecord в users.Session
func (r *SessionRepositoryImpl) ConvertToUserSession(record *SessionRecord) *models.Session {
	return &models.Session{
		ID:        record.ID,
		UserID:    record.UserID,
		Token:     record.Token,
		IP:        record.IPAddress,
		UserAgent: record.UserAgent,
		ExpiresAt: record.ExpiresAt,
		CreatedAt: record.CreatedAt,
	}
}

// Вспомогательные методы для инвалидации кэша

// invalidateSessionCache инвалидирует кэш сессии
func (r *SessionRepositoryImpl) invalidateSessionCache(sessionID string, userID int) {
	ctx := context.Background()
	keys := []string{
		"sessions:active:count",
		"sessions:stats",
	}

	if sessionID != "" {
		keys = append(keys, fmt.Sprintf("session:id:%s", sessionID))
		keys = append(keys, fmt.Sprintf("session:details:id:%s", sessionID))
	}
	if userID > 0 {
		keys = append(keys, fmt.Sprintf("sessions:user:%d", userID))
		keys = append(keys, fmt.Sprintf("sessions:active:user:%d:count", userID))
	}

	_ = r.cache.DeleteMulti(ctx, keys...)
}

// invalidateUserSessionsCache инвалидирует кэш сессий пользователя
func (r *SessionRepositoryImpl) invalidateUserSessionsCache(userID int) {
	ctx := context.Background()
	keys := []string{
		fmt.Sprintf("sessions:user:%d", userID),
		fmt.Sprintf("sessions:active:user:%d:count", userID),
	}

	_ = r.cache.DeleteMulti(ctx, keys...)
}

// invalidateSessionsCache инвалидирует общий кэш сессий
func (r *SessionRepositoryImpl) invalidateSessionsCache() {
	ctx := context.Background()
	keys := []string{
		"sessions:active:count",
		"sessions:stats",
	}

	_ = r.cache.DeleteMulti(ctx, keys...)
}

// Вспомогательная функция для преобразования времени в NullTime
func getNullTime(t time.Time) sql.NullTime {
	if t.IsZero() {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{
		Time:  t,
		Valid: true,
	}
}
