// internal/infrastructure/persistence/postgres/repository/session/repository.go
package session

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"crypto-exchange-screener-bot/internal/infrastructure/cache/redis"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"

	"github.com/jmoiron/sqlx"
)

// SessionRepository интерфейс для работы с сессиями
type SessionRepository interface {
	// Базовые CRUD операции
	Create(session *models.Session) error
	FindByID(id string) (*models.Session, error)
	FindByToken(token string) (*models.Session, error)
	Update(session *models.Session) error
	Delete(id string) error
	Revoke(id, reason string) error
	RevokeAllUserSessions(userID int, reason string) error
	CleanupExpiredSessions() (int64, error)

	// Поиск и фильтрация
	FindByUserID(userID int, limit, offset int) ([]*models.Session, error)
	FindByFilter(filter models.SessionFilter) ([]*models.Session, int64, error)
	GetActiveSessions(userID int) ([]*models.Session, error)
	GetAll(limit, offset int) ([]*models.Session, error)

	// Управление активностью
	UpdateLastActivity(id string) error
	LogActivity(activity *models.SessionActivity) error
	GetSessionActivities(sessionID string, limit, offset int) ([]*models.SessionActivity, error)

	// Статистика
	GetUserSessionStats(userID int) (map[string]interface{}, error)
	GetSystemSessionStats() (*models.SessionStats, error)
	GetSessionCount(userID int) (int, error)
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
	query := `
	INSERT INTO user_sessions (
		id, user_id, token, device_info, ip_address, user_agent,
		data, expires_at, is_active, revoked_at, revoked_reason
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	RETURNING created_at, updated_at, last_activity
	`

	deviceInfoJSON, err := json.Marshal(session.DeviceInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal device info: %w", err)
	}

	dataJSON, err := json.Marshal(session.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	return r.db.QueryRow(
		query,
		session.ID,
		session.UserID,
		session.Token,
		deviceInfoJSON,
		session.IPAddress, // Исправлено: IPAddress вместо IP
		session.UserAgent,
		dataJSON,
		session.ExpiresAt,
		session.IsActive,
		session.RevokedAt,
		session.RevokedReason,
	).Scan(&session.CreatedAt, &session.UpdatedAt, &session.LastActivity)
}

// FindByID находит сессию по ID
func (r *SessionRepositoryImpl) FindByID(id string) (*models.Session, error) {
	cacheKey := fmt.Sprintf("session:%s", id)

	// Попытка получить из кэша
	var session models.Session
	if err := r.cache.Get(context.Background(), cacheKey, &session); err == nil {
		return &session, nil
	}

	query := `
	SELECT
		s.id, s.user_id, s.token, s.device_info, s.ip_address, s.user_agent,
		s.data, s.expires_at, s.created_at, s.updated_at, s.last_activity,
		s.is_active, s.revoked_at, s.revoked_reason,
		u.telegram_id, u.username, u.first_name, u.role
	FROM user_sessions s
	LEFT JOIN users u ON s.user_id = u.id
	WHERE s.id = $1
	`

	var deviceInfoJSON, dataJSON []byte
	var telegramID sql.NullInt64
	var username, firstName, role sql.NullString

	err := r.db.QueryRow(query, id).Scan(
		&session.ID,
		&session.UserID,
		&session.Token,
		&deviceInfoJSON,
		&session.IPAddress, // Исправлено: IPAddress вместо IP
		&session.UserAgent,
		&dataJSON,
		&session.ExpiresAt,
		&session.CreatedAt,
		&session.UpdatedAt,
		&session.LastActivity,
		&session.IsActive,
		&session.RevokedAt,
		&session.RevokedReason,
		&telegramID,
		&username,
		&firstName,
		&role,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Декодируем JSON
	if err := json.Unmarshal(deviceInfoJSON, &session.DeviceInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal device info: %w", err)
	}

	if err := json.Unmarshal(dataJSON, &session.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session data: %w", err)
	}

	// Заполняем информацию о пользователе если есть
	if telegramID.Valid || username.Valid || firstName.Valid {
		session.User = &models.User{
			ID:         session.UserID,
			TelegramID: telegramID.Int64,
			Username:   username.String,
			FirstName:  firstName.String,
			Role:       role.String,
		}
	}

	// Сохраняем в кэш
	if data, err := json.Marshal(session); err == nil {
		_ = r.cache.Set(context.Background(), cacheKey, string(data), 10*time.Minute)
	}

	return &session, nil
}

// FindByToken находит сессию по токену
func (r *SessionRepositoryImpl) FindByToken(token string) (*models.Session, error) {
	cacheKey := fmt.Sprintf("session:token:%s", token)

	// Попытка получить из кэша
	var session models.Session
	if err := r.cache.Get(context.Background(), cacheKey, &session); err == nil {
		return &session, nil
	}

	query := `
	SELECT
		s.id, s.user_id, s.token, s.device_info, s.ip_address, s.user_agent,
		s.data, s.expires_at, s.created_at, s.updated_at, s.last_activity,
		s.is_active, s.revoked_at, s.revoked_reason,
		u.telegram_id, u.username, u.first_name, u.role
	FROM user_sessions s
	LEFT JOIN users u ON s.user_id = u.id
	WHERE s.token = $1
	`

	var deviceInfoJSON, dataJSON []byte
	var telegramID sql.NullInt64
	var username, firstName, role sql.NullString

	err := r.db.QueryRow(query, token).Scan(
		&session.ID,
		&session.UserID,
		&session.Token,
		&deviceInfoJSON,
		&session.IPAddress, // Исправлено: IPAddress вместо IP
		&session.UserAgent,
		&dataJSON,
		&session.ExpiresAt,
		&session.CreatedAt,
		&session.UpdatedAt,
		&session.LastActivity,
		&session.IsActive,
		&session.RevokedAt,
		&session.RevokedReason,
		&telegramID,
		&username,
		&firstName,
		&role,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Декодируем JSON
	if err := json.Unmarshal(deviceInfoJSON, &session.DeviceInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal device info: %w", err)
	}

	if err := json.Unmarshal(dataJSON, &session.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session data: %w", err)
	}

	// Заполняем информацию о пользователе если есть
	if telegramID.Valid || username.Valid || firstName.Valid {
		session.User = &models.User{
			ID:         session.UserID,
			TelegramID: telegramID.Int64,
			Username:   username.String,
			FirstName:  firstName.String,
			Role:       role.String,
		}
	}

	// Сохраняем в кэш
	if data, err := json.Marshal(session); err == nil {
		_ = r.cache.Set(context.Background(), cacheKey, string(data), 10*time.Minute)
	}

	return &session, nil
}

// Update обновляет сессию
func (r *SessionRepositoryImpl) Update(session *models.Session) error {
	query := `
	UPDATE user_sessions SET
		token = $1,
		device_info = $2,
		ip_address = $3,
		user_agent = $4,
		data = $5,
		expires_at = $6,
		is_active = $7,
		revoked_at = $8,
		revoked_reason = $9,
		last_activity = $10
	WHERE id = $11
	RETURNING updated_at
	`

	deviceInfoJSON, err := json.Marshal(session.DeviceInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal device info: %w", err)
	}

	dataJSON, err := json.Marshal(session.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	err = r.db.QueryRow(
		query,
		session.Token,
		deviceInfoJSON,
		session.IPAddress, // Исправлено: IPAddress вместо IP
		session.UserAgent,
		dataJSON,
		session.ExpiresAt,
		session.IsActive,
		session.RevokedAt,
		session.RevokedReason,
		session.LastActivity,
		session.ID,
	).Scan(&session.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	// Инвалидируем кэш
	r.invalidateSessionCache(session.ID, session.Token)
	return nil
}

// Delete удаляет сессию
func (r *SessionRepositoryImpl) Delete(id string) error {
	// Сначала получаем сессию для инвалидации кэша
	session, err := r.FindByID(id)
	if err != nil {
		return err
	}
	if session == nil {
		return sql.ErrNoRows
	}

	query := `DELETE FROM user_sessions WHERE id = $1`
	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	// Инвалидируем кэш
	r.invalidateSessionCache(id, session.Token)
	return nil
}

// Revoke отзывает сессию
func (r *SessionRepositoryImpl) Revoke(id, reason string) error {
	now := time.Now()
	reasonPtr := &reason

	query := `
	UPDATE user_sessions SET
		is_active = false,
		revoked_at = $1,
		revoked_reason = $2,
		updated_at = NOW()
	WHERE id = $3
	RETURNING token
	`

	var token string
	err := r.db.QueryRow(query, now, reasonPtr, id).Scan(&token)
	if err != nil {
		return fmt.Errorf("failed to revoke session: %w", err)
	}

	// Инвалидируем кэш
	r.invalidateSessionCache(id, token)
	return nil
}

// RevokeAllUserSessions отзывает все сессии пользователя
func (r *SessionRepositoryImpl) RevokeAllUserSessions(userID int, reason string) error {
	now := time.Now()
	reasonPtr := &reason

	query := `
	UPDATE user_sessions SET
		is_active = false,
		revoked_at = $1,
		revoked_reason = $2,
		updated_at = NOW()
	WHERE user_id = $3 AND is_active = true
	RETURNING id, token
	`

	rows, err := r.db.Query(query, now, reasonPtr, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke user sessions: %w", err)
	}
	defer rows.Close()

	var sessionIDs []string
	var tokens []string

	for rows.Next() {
		var id, token string
		if err := rows.Scan(&id, &token); err != nil {
			return err
		}
		sessionIDs = append(sessionIDs, id)
		tokens = append(tokens, token)
	}

	// Инвалидируем кэш для всех отозванных сессий
	for i, id := range sessionIDs {
		r.invalidateSessionCache(id, tokens[i])
	}

	return nil
}

// CleanupExpiredSessions очищает истекшие сессии
func (r *SessionRepositoryImpl) CleanupExpiredSessions() (int64, error) {
	query := `
	DELETE FROM user_sessions
	WHERE expires_at < NOW() OR (is_active = false AND revoked_at < NOW() - INTERVAL '30 days')
	RETURNING id, token
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}
	defer rows.Close()

	var count int64
	var sessionIDs []string
	var tokens []string

	for rows.Next() {
		var id, token string
		if err := rows.Scan(&id, &token); err != nil {
			return 0, err
		}
		sessionIDs = append(sessionIDs, id)
		tokens = append(tokens, token)
		count++
	}

	// Инвалидируем кэш
	for i, id := range sessionIDs {
		r.invalidateSessionCache(id, tokens[i])
	}

	return count, nil
}

// FindByUserID находит сессии пользователя
func (r *SessionRepositoryImpl) FindByUserID(userID int, limit, offset int) ([]*models.Session, error) {
	cacheKey := fmt.Sprintf("sessions:user:%d:%d:%d", userID, limit, offset)

	// Попытка получить из кэша
	var sessions []*models.Session
	if err := r.cache.Get(context.Background(), cacheKey, &sessions); err == nil {
		return sessions, nil
	}

	query := `
	SELECT
		s.id, s.user_id, s.token, s.device_info, s.ip_address, s.user_agent,
		s.data, s.expires_at, s.created_at, s.updated_at, s.last_activity,
		s.is_active, s.revoked_at, s.revoked_reason,
		u.telegram_id, u.username, u.first_name, u.role
	FROM user_sessions s
	LEFT JOIN users u ON s.user_id = u.id
	WHERE s.user_id = $1
	ORDER BY s.last_activity DESC
	LIMIT $2 OFFSET $3
	`

	sessions, err := r.querySessions(query, userID, limit, offset)
	if err != nil {
		return nil, err
	}

	// Сохраняем в кэш
	if data, err := json.Marshal(sessions); err == nil {
		_ = r.cache.Set(context.Background(), cacheKey, string(data), 5*time.Minute)
	}

	return sessions, nil
}

// FindByFilter находит сессии по фильтру
func (r *SessionRepositoryImpl) FindByFilter(filter models.SessionFilter) ([]*models.Session, int64, error) {
	query, args, err := r.buildFilterQuery(filter)
	if err != nil {
		return nil, 0, err
	}

	sessions, err := r.querySessions(query, args...)
	if err != nil {
		return nil, 0, err
	}

	countQuery, countArgs := r.buildCountQuery(filter)
	var total int64
	err = r.db.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	return sessions, total, nil
}

// GetActiveSessions возвращает активные сессии пользователя
func (r *SessionRepositoryImpl) GetActiveSessions(userID int) ([]*models.Session, error) {
	cacheKey := fmt.Sprintf("sessions:active:user:%d", userID)

	// Попытка получить из кэша
	var sessions []*models.Session
	if err := r.cache.Get(context.Background(), cacheKey, &sessions); err == nil {
		return sessions, nil
	}

	query := `
	SELECT
		s.id, s.user_id, s.token, s.device_info, s.ip_address, s.user_agent,
		s.data, s.expires_at, s.created_at, s.updated_at, s.last_activity,
		s.is_active, s.revoked_at, s.revoked_reason,
		u.telegram_id, u.username, u.first_name, u.role
	FROM user_sessions s
	LEFT JOIN users u ON s.user_id = u.id
	WHERE s.user_id = $1 AND s.is_active = true AND s.expires_at > NOW()
	ORDER BY s.last_activity DESC
	`

	sessions, err := r.querySessions(query, userID)
	if err != nil {
		return nil, err
	}

	// Сохраняем в кэш
	if data, err := json.Marshal(sessions); err == nil {
		_ = r.cache.Set(context.Background(), cacheKey, string(data), 2*time.Minute)
	}

	return sessions, nil
}

// GetAll возвращает все сессии с пагинацией
func (r *SessionRepositoryImpl) GetAll(limit, offset int) ([]*models.Session, error) {
	query := `
	SELECT
		s.id, s.user_id, s.token, s.device_info, s.ip_address, s.user_agent,
		s.data, s.expires_at, s.created_at, s.updated_at, s.last_activity,
		s.is_active, s.revoked_at, s.revoked_reason,
		u.telegram_id, u.username, u.first_name, u.role
	FROM user_sessions s
	LEFT JOIN users u ON s.user_id = u.id
	ORDER BY s.last_activity DESC
	LIMIT $1 OFFSET $2
	`

	return r.querySessions(query, limit, offset)
}

// UpdateLastActivity обновляет время последней активности
func (r *SessionRepositoryImpl) UpdateLastActivity(id string) error {
	query := `
	UPDATE user_sessions SET
		last_activity = NOW(),
		updated_at = NOW()
	WHERE id = $1
	RETURNING token
	`

	var token string
	err := r.db.QueryRow(query, id).Scan(&token)
	if err != nil {
		return fmt.Errorf("failed to update last activity: %w", err)
	}

	// Инвалидируем кэш
	r.invalidateSessionCache(id, token)
	return nil
}

// LogActivity логирует активность сессии
func (r *SessionRepositoryImpl) LogActivity(activity *models.SessionActivity) error {
	query := `
	INSERT INTO session_activities (
		session_id, activity_type, details, ip_address
	) VALUES ($1, $2, $3, $4)
	RETURNING id, created_at
	`

	detailsJSON, err := json.Marshal(activity.Details)
	if err != nil {
		return fmt.Errorf("failed to marshal details: %w", err)
	}

	return r.db.QueryRow(
		query,
		activity.SessionID,
		activity.ActivityType,
		detailsJSON,
		activity.IPAddress,
	).Scan(&activity.ID, &activity.CreatedAt)
}

// GetSessionActivities возвращает активность сессии
func (r *SessionRepositoryImpl) GetSessionActivities(sessionID string, limit, offset int) ([]*models.SessionActivity, error) {
	query := `
	SELECT id, session_id, activity_type, details, ip_address, created_at
	FROM session_activities
	WHERE session_id = $1
	ORDER BY created_at DESC
	LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, sessionID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []*models.SessionActivity

	for rows.Next() {
		var activity models.SessionActivity
		var detailsJSON []byte

		err := rows.Scan(
			&activity.ID,
			&activity.SessionID,
			&activity.ActivityType,
			&detailsJSON,
			&activity.IPAddress,
			&activity.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		// Декодируем JSON
		if err := json.Unmarshal(detailsJSON, &activity.Details); err != nil {
			return nil, fmt.Errorf("failed to unmarshal details: %w", err)
		}

		activities = append(activities, &activity)
	}

	return activities, nil
}

// GetUserSessionStats возвращает статистику сессий пользователя
func (r *SessionRepositoryImpl) GetUserSessionStats(userID int) (map[string]interface{}, error) {
	cacheKey := fmt.Sprintf("session:stats:user:%d", userID)
	var stats map[string]interface{}

	if err := r.cache.Get(context.Background(), cacheKey, &stats); err == nil {
		return stats, nil
	}

	query := `
	SELECT
		COUNT(*) as total_sessions,
		COUNT(CASE WHEN is_active = true AND expires_at > NOW() THEN 1 END) as active_sessions,
		COUNT(CASE WHEN is_active = false THEN 1 END) as revoked_sessions,
		COUNT(CASE WHEN expires_at < NOW() THEN 1 END) as expired_sessions,
		MIN(created_at) as first_session,
		MAX(last_activity) as last_activity,
		COUNT(DISTINCT ip_address) as unique_ips
	FROM user_sessions
	WHERE user_id = $1
	`

	var (
		totalSessions   int
		activeSessions  int
		revokedSessions int
		expiredSessions int
		firstSession    sql.NullTime
		lastActivity    sql.NullTime
		uniqueIPs       int
	)

	err := r.db.QueryRow(query, userID).Scan(
		&totalSessions,
		&activeSessions,
		&revokedSessions,
		&expiredSessions,
		&firstSession,
		&lastActivity,
		&uniqueIPs,
	)
	if err != nil {
		return nil, err
	}

	stats = make(map[string]interface{})
	stats["total_sessions"] = totalSessions
	stats["active_sessions"] = activeSessions
	stats["revoked_sessions"] = revokedSessions
	stats["expired_sessions"] = expiredSessions
	stats["unique_ips"] = uniqueIPs

	if firstSession.Valid {
		stats["first_session"] = firstSession.Time
	}
	if lastActivity.Valid {
		stats["last_activity"] = lastActivity.Time
	}

	// Сохраняем в кэш
	if data, err := json.Marshal(stats); err == nil {
		_ = r.cache.Set(context.Background(), cacheKey, string(data), 5*time.Minute)
	}

	return stats, nil
}

// GetSystemSessionStats возвращает системную статистику сессий
func (r *SessionRepositoryImpl) GetSystemSessionStats() (*models.SessionStats, error) {
	cacheKey := "session:stats:system"

	// Попытка получить из кэша
	var cachedStats models.SessionStats
	if err := r.cache.Get(context.Background(), cacheKey, &cachedStats); err == nil {
		return &cachedStats, nil
	}

	stats := &models.SessionStats{}

	// Основная статистика
	query := `
	SELECT
		COUNT(*) as total_sessions,
		COUNT(CASE WHEN is_active = true AND expires_at > NOW() THEN 1 END) as active_sessions,
		COUNT(DISTINCT user_id) as unique_users,
		COUNT(CASE WHEN expires_at < NOW() THEN 1 END) as expired_sessions,
		AVG(EXTRACT(EPOCH FROM (last_activity - created_at))) as avg_session_duration
	FROM user_sessions
	`

	var (
		totalSessions      int64
		activeSessions     int64
		uniqueUsers        int64
		expiredSessions    int64
		avgDurationSeconds sql.NullFloat64
	)

	err := r.db.QueryRow(query).Scan(
		&totalSessions,
		&activeSessions,
		&uniqueUsers,
		&expiredSessions,
		&avgDurationSeconds,
	)
	if err != nil {
		return nil, err
	}

	stats.TotalSessions = totalSessions
	stats.ActiveSessions = activeSessions
	stats.UniqueUsers = uniqueUsers
	stats.ExpiredSessions = expiredSessions

	if avgDurationSeconds.Valid {
		stats.AvgSessionDuration = time.Duration(avgDurationSeconds.Float64) * time.Second
	}

	// Сессии за последние 24 часа
	last24hQuery := `SELECT COUNT(*) FROM user_sessions WHERE created_at >= NOW() - INTERVAL '24 hours'`
	err = r.db.QueryRow(last24hQuery).Scan(&stats.SessionsLast24h)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Наиболее активные пользователи
	mostActiveQuery := `
	SELECT user_id, COUNT(*) as session_count
	FROM user_sessions
	GROUP BY user_id
	ORDER BY session_count DESC
	LIMIT 5
	`

	rows, err := r.db.Query(mostActiveQuery)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var userID int
			var count int64
			rows.Scan(&userID, &count)
			stats.MostActiveUsers = append(stats.MostActiveUsers, models.SessionUserStats{
				UserID:       userID,
				SessionCount: count,
			})
		}
	}

	// Сохраняем в кэш
	if data, err := json.Marshal(stats); err == nil {
		_ = r.cache.Set(context.Background(), cacheKey, string(data), 5*time.Minute)
	}

	return stats, nil
}

// GetSessionCount возвращает количество сессий пользователя
func (r *SessionRepositoryImpl) GetSessionCount(userID int) (int, error) {
	cacheKey := fmt.Sprintf("session:count:user:%d", userID)

	// Попытка получить из кэша
	var count int
	if err := r.cache.Get(context.Background(), cacheKey, &count); err == nil {
		return count, nil
	}

	query := `SELECT COUNT(*) FROM user_sessions WHERE user_id = $1`
	err := r.db.QueryRow(query, userID).Scan(&count)
	if err != nil {
		return 0, err
	}

	// Сохраняем в кэш
	_ = r.cache.Set(context.Background(), cacheKey, count, 5*time.Minute)

	return count, nil
}

// Вспомогательные методы

func (r *SessionRepositoryImpl) querySessions(query string, args ...interface{}) ([]*models.Session, error) {
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*models.Session

	for rows.Next() {
		var session models.Session
		var deviceInfoJSON, dataJSON []byte
		var telegramID sql.NullInt64
		var username, firstName, role sql.NullString

		err := rows.Scan(
			&session.ID,
			&session.UserID,
			&session.Token,
			&deviceInfoJSON,
			&session.IPAddress, // Исправлено: IPAddress вместо IP
			&session.UserAgent,
			&dataJSON,
			&session.ExpiresAt,
			&session.CreatedAt,
			&session.UpdatedAt,
			&session.LastActivity,
			&session.IsActive,
			&session.RevokedAt,
			&session.RevokedReason,
			&telegramID,
			&username,
			&firstName,
			&role,
		)

		if err != nil {
			return nil, err
		}

		// Декодируем JSON
		if err := json.Unmarshal(deviceInfoJSON, &session.DeviceInfo); err != nil {
			return nil, fmt.Errorf("failed to unmarshal device info: %w", err)
		}

		if err := json.Unmarshal(dataJSON, &session.Data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal session data: %w", err)
		}

		// Заполняем информацию о пользователе если есть
		if telegramID.Valid || username.Valid || firstName.Valid {
			session.User = &models.User{
				ID:         session.UserID,
				TelegramID: telegramID.Int64,
				Username:   username.String,
				FirstName:  firstName.String,
				Role:       role.String,
			}
		}

		sessions = append(sessions, &session)
	}

	return sessions, nil
}

func (r *SessionRepositoryImpl) buildFilterQuery(filter models.SessionFilter) (string, []interface{}, error) {
	baseQuery := `
	SELECT
		s.id, s.user_id, s.token, s.device_info, s.ip_address, s.user_agent,
		s.data, s.expires_at, s.created_at, s.updated_at, s.last_activity,
		s.is_active, s.revoked_at, s.revoked_reason,
		u.telegram_id, u.username, u.first_name, u.role
	FROM user_sessions s
	LEFT JOIN users u ON s.user_id = u.id
	`

	whereClauses := []string{"1=1"}
	args := []interface{}{}
	argIndex := 1

	if filter.UserID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("s.user_id = $%d", argIndex))
		args = append(args, *filter.UserID)
		argIndex++
	}

	if filter.IsActive != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("s.is_active = $%d", argIndex))
		args = append(args, *filter.IsActive)
		argIndex++
	}

	if filter.Token != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("s.token = $%d", argIndex))
		args = append(args, *filter.Token)
		argIndex++
	}

	if filter.StartDate != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("s.created_at >= $%d", argIndex))
		args = append(args, *filter.StartDate)
		argIndex++
	}

	if filter.EndDate != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("s.created_at <= $%d", argIndex))
		args = append(args, *filter.EndDate)
		argIndex++
	}

	// Собираем запрос
	query := baseQuery + " WHERE " + joinStrings(whereClauses, " AND ")

	// Сортировка
	orderBy := "s.last_activity"
	orderDir := "DESC"

	if filter.OrderBy != "" {
		validOrderFields := map[string]bool{
			"created_at":    true,
			"last_activity": true,
			"expires_at":    true,
			"user_id":       true,
		}
		if validOrderFields[filter.OrderBy] {
			orderBy = "s." + filter.OrderBy
		}
	}

	if filter.OrderDir != "" && (filter.OrderDir == "ASC" || filter.OrderDir == "DESC") {
		orderDir = filter.OrderDir
	}

	query += fmt.Sprintf(" ORDER BY %s %s", orderBy, orderDir)

	// Лимит и оффсет
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++

		if filter.Offset > 0 {
			query += fmt.Sprintf(" OFFSET $%d", argIndex)
			args = append(args, filter.Offset)
		}
	}

	return query, args, nil
}

func (r *SessionRepositoryImpl) buildCountQuery(filter models.SessionFilter) (string, []interface{}) {
	baseQuery := "SELECT COUNT(*) FROM user_sessions s WHERE 1=1"

	whereClauses := []string{}
	args := []interface{}{}
	argIndex := 1

	if filter.UserID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("user_id = $%d", argIndex))
		args = append(args, *filter.UserID)
		argIndex++
	}

	if filter.IsActive != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("is_active = $%d", argIndex))
		args = append(args, *filter.IsActive)
		argIndex++
	}

	if filter.Token != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("token = $%d", argIndex))
		args = append(args, *filter.Token)
		argIndex++
	}

	if filter.StartDate != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("created_at >= $%d", argIndex))
		args = append(args, *filter.StartDate)
		argIndex++
	}

	if filter.EndDate != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("created_at <= $%d", argIndex))
		args = append(args, *filter.EndDate)
		argIndex++
	}

	query := baseQuery
	if len(whereClauses) > 0 {
		query += " AND " + joinStrings(whereClauses, " AND ")
	}

	return query, args
}

func (r *SessionRepositoryImpl) invalidateSessionCache(sessionID, token string) {
	ctx := context.Background()
	keys := []string{
		fmt.Sprintf("session:%s", sessionID),
		fmt.Sprintf("session:token:%s", token),
		"sessions:user:*",
		"sessions:active:*",
		"session:stats:*",
		"session:count:*",
	}

	_ = r.cache.DeleteMulti(ctx, keys...)
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}
	result := strs[0]
	for _, s := range strs[1:] {
		result += sep + s
	}
	return result
}
