// persistence/postgres/repository/session_repository.go
package session

// import (
// 	"context"
// 	"crypto-exchange-screener-bot/persistence/postgres/repository/users"
// 	"database/sql"
// 	"encoding/json"
// 	"fmt"
// 	"time"

// 	"github.com/google/uuid"
// 	"github.com/jmoiron/sqlx"
// )

// // SessionRepository управляет сессиями пользователей
// type SessionRepository struct {
// 	db *sqlx.DB
// }

// // SessionRecord структура для работы с базой данных
// type SessionRecord struct {
// 	ID            string                 `db:"id"`
// 	UserID        int                    `db:"user_id"`
// 	Token         string                 `db:"token"`
// 	DeviceInfo    map[string]interface{} `db:"device_info"`
// 	IPAddress     string                 `db:"ip_address"`
// 	UserAgent     string                 `db:"user_agent"`
// 	Data          map[string]interface{} `db:"data"`
// 	ExpiresAt     time.Time              `db:"expires_at"`
// 	CreatedAt     time.Time              `db:"created_at"`
// 	UpdatedAt     time.Time              `db:"updated_at"`
// 	LastActivity  time.Time              `db:"last_activity"`
// 	IsActive      bool                   `db:"is_active"`
// 	RevokedAt     *time.Time             `db:"revoked_at"`
// 	RevokedReason string                 `db:"revoked_reason"`
// }

// // SessionStats статистика сессий
// type SessionStats struct {
// 	TotalSessions      int           `json:"total_sessions"`
// 	ActiveSessions     int           `json:"active_sessions"`
// 	AvgSessionDuration time.Duration `json:"avg_session_duration"`
// 	MaxConcurrent      int           `json:"max_concurrent"`
// 	MostActiveHour     int           `json:"most_active_hour"`
// }

// // NewSessionRepository создает новый репозиторий сессий
// func NewSessionRepository(db *sqlx.DB) *SessionRepository {
// 	return &SessionRepository{db: db}
// }

// // Create создает новую сессию
// func (r *SessionRepository) Create(session *users.Session) error {
// 	query := `
//     INSERT INTO user_sessions (
//         id, user_id, token, ip_address,
//         user_agent, expires_at, created_at
//     ) VALUES ($1, $2, $3, $4, $5, $6, $7)
//     RETURNING created_at
//     `

// 	return r.db.QueryRow(
// 		query,
// 		session.ID,
// 		session.UserID,
// 		session.Token,
// 		session.IP, // Исправлено: IPAddress -> IP
// 		session.UserAgent,
// 		session.ExpiresAt,
// 		time.Now(),
// 	).Scan(&session.CreatedAt)
// }

// // CreateWithDetails создает сессию с дополнительными данными
// func (r *SessionRepository) CreateWithDetails(session *SessionRecord) error {
// 	query := `
//     INSERT INTO user_sessions (
//         id, user_id, token, device_info, ip_address,
//         user_agent, expires_at, data, created_at
//     ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
//     RETURNING created_at
//     `

// 	deviceInfoJSON, err := json.Marshal(session.DeviceInfo)
// 	if err != nil {
// 		return fmt.Errorf("failed to marshal device info: %w", err)
// 	}

// 	dataJSON, err := json.Marshal(session.Data)
// 	if err != nil {
// 		return fmt.Errorf("failed to marshal session data: %w", err)
// 	}

// 	return r.db.QueryRow(
// 		query,
// 		session.ID,
// 		session.UserID,
// 		session.Token,
// 		deviceInfoJSON,
// 		session.IPAddress,
// 		session.UserAgent,
// 		session.ExpiresAt,
// 		dataJSON,
// 		time.Now(),
// 	).Scan(&session.CreatedAt)
// }

// // FindByToken находит сессию по токену
// func (r *SessionRepository) FindByToken(token string) (*users.Session, error) {
// 	query := `
//     SELECT
//         s.id, s.user_id, s.token, s.ip_address,
//         s.user_agent, s.expires_at, s.created_at,
//         u.telegram_id, u.username, u.first_name, u.last_name, u.chat_id,
//         u.role, u.is_active as user_active
//     FROM user_sessions s
//     JOIN users u ON s.user_id = u.id
//     WHERE s.token = $1
//       AND s.is_active = TRUE
//       AND s.expires_at > NOW()
//       AND s.revoked_at IS NULL
//       AND u.is_active = TRUE
//     `

// 	var session users.Session
// 	var user users.User
// 	var ipAddress string // Используем временную переменную

// 	err := r.db.QueryRow(query, token).Scan(
// 		&session.ID,
// 		&session.UserID,
// 		&session.Token,
// 		&ipAddress, // Сканируем в ipAddress
// 		&session.UserAgent,
// 		&session.ExpiresAt,
// 		&session.CreatedAt,
// 		&user.TelegramID,
// 		&user.Username,
// 		&user.FirstName,
// 		&user.LastName,
// 		&user.ChatID,
// 		&user.Role,
// 		&user.IsActive,
// 	)

// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			return nil, nil
// 		}
// 		return nil, err
// 	}

// 	// Присваиваем IP
// 	session.IP = ipAddress
// 	session.User = &user
// 	return &session, nil
// }

// // FindByTokenWithDetails находит сессию с деталями
// func (r *SessionRepository) FindByTokenWithDetails(token string) (*SessionRecord, error) {
// 	query := `
//     SELECT
//         id, user_id, token, device_info, ip_address,
//         user_agent, expires_at, data, created_at, updated_at,
//         last_activity, is_active, revoked_at, revoked_reason
//     FROM user_sessions
//     WHERE token = $1
//     `

// 	var session SessionRecord
// 	var deviceInfoJSON, dataJSON []byte

// 	err := r.db.QueryRow(query, token).Scan(
// 		&session.ID,
// 		&session.UserID,
// 		&session.Token,
// 		&deviceInfoJSON,
// 		&session.IPAddress,
// 		&session.UserAgent,
// 		&session.ExpiresAt,
// 		&dataJSON,
// 		&session.CreatedAt,
// 		&session.UpdatedAt,
// 		&session.LastActivity,
// 		&session.IsActive,
// 		&session.RevokedAt,
// 		&session.RevokedReason,
// 	)

// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			return nil, nil
// 		}
// 		return nil, err
// 	}

// 	// Декодируем JSON данные
// 	if len(deviceInfoJSON) > 0 {
// 		if err := json.Unmarshal(deviceInfoJSON, &session.DeviceInfo); err != nil {
// 			return nil, fmt.Errorf("failed to unmarshal device info: %w", err)
// 		}
// 	}

// 	if len(dataJSON) > 0 {
// 		if err := json.Unmarshal(dataJSON, &session.Data); err != nil {
// 			return nil, fmt.Errorf("failed to unmarshal session data: %w", err)
// 		}
// 	}

// 	return &session, nil
// }

// // FindByID находит сессию по ID
// func (r *SessionRepository) FindByID(sessionID string) (*SessionRecord, error) {
// 	query := `
//     SELECT
//         id, user_id, token, device_info, ip_address,
//         user_agent, expires_at, data, created_at, updated_at,
//         last_activity, is_active, revoked_at, revoked_reason
//     FROM user_sessions
//     WHERE id = $1
//     `

// 	var session SessionRecord
// 	var deviceInfoJSON, dataJSON []byte

// 	err := r.db.QueryRow(query, sessionID).Scan(
// 		&session.ID,
// 		&session.UserID,
// 		&session.Token,
// 		&deviceInfoJSON,
// 		&session.IPAddress,
// 		&session.UserAgent,
// 		&session.ExpiresAt,
// 		&dataJSON,
// 		&session.CreatedAt,
// 		&session.UpdatedAt,
// 		&session.LastActivity,
// 		&session.IsActive,
// 		&session.RevokedAt,
// 		&session.RevokedReason,
// 	)

// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			return nil, nil
// 		}
// 		return nil, err
// 	}

// 	// Декодируем JSON данные
// 	if len(deviceInfoJSON) > 0 {
// 		if err := json.Unmarshal(deviceInfoJSON, &session.DeviceInfo); err != nil {
// 			return nil, fmt.Errorf("failed to unmarshal device info: %w", err)
// 		}
// 	}

// 	if len(dataJSON) > 0 {
// 		if err := json.Unmarshal(dataJSON, &session.Data); err != nil {
// 			return nil, fmt.Errorf("failed to unmarshal session data: %w", err)
// 		}
// 	}

// 	return &session, nil
// }

// // FindByUserID находит все активные сессии пользователя
// func (r *SessionRepository) FindByUserID(userID int) ([]*SessionRecord, error) {
// 	query := `
//     SELECT
//         id, user_id, token, device_info, ip_address,
//         user_agent, expires_at, data, created_at, updated_at,
//         last_activity, is_active, revoked_at, revoked_reason
//     FROM user_sessions
//     WHERE user_id = $1
//       AND is_active = TRUE
//       AND expires_at > NOW()
//       AND revoked_at IS NULL
//     ORDER BY created_at DESC
//     `

// 	rows, err := r.db.Query(query, userID)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	var sessions []*SessionRecord

// 	for rows.Next() {
// 		var session SessionRecord
// 		var deviceInfoJSON, dataJSON []byte

// 		err := rows.Scan(
// 			&session.ID,
// 			&session.UserID,
// 			&session.Token,
// 			&deviceInfoJSON,
// 			&session.IPAddress,
// 			&session.UserAgent,
// 			&session.ExpiresAt,
// 			&dataJSON,
// 			&session.CreatedAt,
// 			&session.UpdatedAt,
// 			&session.LastActivity,
// 			&session.IsActive,
// 			&session.RevokedAt,
// 			&session.RevokedReason,
// 		)

// 		if err != nil {
// 			return nil, err
// 		}

// 		// Декодируем JSON данные
// 		if len(deviceInfoJSON) > 0 {
// 			if err := json.Unmarshal(deviceInfoJSON, &session.DeviceInfo); err != nil {
// 				return nil, fmt.Errorf("failed to unmarshal device info: %w", err)
// 			}
// 		}

// 		if len(dataJSON) > 0 {
// 			if err := json.Unmarshal(dataJSON, &session.Data); err != nil {
// 				return nil, fmt.Errorf("failed to unmarshal session data: %w", err)
// 			}
// 		}

// 		sessions = append(sessions, &session)
// 	}

// 	return sessions, nil
// }

// // UpdateActivity обновляет время последней активности сессии
// func (r *SessionRepository) UpdateActivity(sessionID string) error {
// 	query := `
//     UPDATE user_sessions
//     SET last_activity = NOW(),
//         updated_at = NOW()
//     WHERE id = $1 AND is_active = TRUE
//     `

// 	result, err := r.db.Exec(query, sessionID)
// 	if err != nil {
// 		return err
// 	}

// 	rowsAffected, _ := result.RowsAffected()
// 	if rowsAffected == 0 {
// 		return sql.ErrNoRows
// 	}

// 	return nil
// }

// // Revoke отзывает сессию
// func (r *SessionRepository) Revoke(sessionID, reason string) error {
// 	query := `
//     UPDATE user_sessions
//     SET is_active = FALSE,
//         revoked_at = NOW(),
//         revoked_reason = $1,
//         updated_at = NOW()
//     WHERE id = $2 AND is_active = TRUE
//     `

// 	result, err := r.db.Exec(query, reason, sessionID)
// 	if err != nil {
// 		return err
// 	}

// 	rowsAffected, _ := result.RowsAffected()
// 	if rowsAffected == 0 {
// 		return sql.ErrNoRows
// 	}

// 	return nil
// }

// // RevokeAllUserSessions отзывает все сессии пользователя
// func (r *SessionRepository) RevokeAllUserSessions(userID int, reason string) (int, error) {
// 	query := `
//     UPDATE user_sessions
//     SET is_active = FALSE,
//         revoked_at = NOW(),
//         revoked_reason = $1,
//         updated_at = NOW()
//     WHERE user_id = $2
//       AND is_active = TRUE
//       AND revoked_at IS NULL
//     `

// 	result, err := r.db.Exec(query, reason, userID)
// 	if err != nil {
// 		return 0, err
// 	}

// 	rowsAffected, _ := result.RowsAffected()
// 	return int(rowsAffected), nil
// }

// // RevokeByToken отзывает сессию по токену
// func (r *SessionRepository) RevokeByToken(token, reason string) error {
// 	query := `
//     UPDATE user_sessions
//     SET is_active = FALSE,
//         revoked_at = NOW(),
//         revoked_reason = $1,
//         updated_at = NOW()
//     WHERE token = $2 AND is_active = TRUE
//     `

// 	result, err := r.db.Exec(query, reason, token)
// 	if err != nil {
// 		return err
// 	}

// 	rowsAffected, _ := result.RowsAffected()
// 	if rowsAffected == 0 {
// 		return sql.ErrNoRows
// 	}

// 	return nil
// }

// // Extend продлевает срок действия сессии
// func (r *SessionRepository) Extend(sessionID string, newExpiry time.Time) error {
// 	query := `
//     UPDATE user_sessions
//     SET expires_at = $1,
//         updated_at = NOW()
//     WHERE id = $2 AND is_active = TRUE
//     `

// 	result, err := r.db.Exec(query, newExpiry, sessionID)
// 	if err != nil {
// 		return err
// 	}

// 	rowsAffected, _ := result.RowsAffected()
// 	if rowsAffected == 0 {
// 		return sql.ErrNoRows
// 	}

// 	return nil
// }

// // CleanupExpired очищает истекшие сессии
// func (r *SessionRepository) CleanupExpired(ctx context.Context) (int, error) {
// 	query := `
//     DELETE FROM user_sessions
//     WHERE expires_at < NOW() - INTERVAL '7 days'
//       AND is_active = FALSE
//     `

// 	result, err := r.db.ExecContext(ctx, query)
// 	if err != nil {
// 		return 0, err
// 	}

// 	rowsAffected, _ := result.RowsAffected()
// 	return int(rowsAffected), nil
// }

// // GetActiveSessionsCount возвращает количество активных сессий
// func (r *SessionRepository) GetActiveSessionsCount() (int, error) {
// 	query := `
//     SELECT COUNT(*)
//     FROM user_sessions
//     WHERE is_active = TRUE
//       AND expires_at > NOW()
//       AND revoked_at IS NULL
//     `

// 	var count int
// 	err := r.db.QueryRow(query).Scan(&count)
// 	if err != nil {
// 		return 0, err
// 	}

// 	return count, nil
// }

// // GetUserActiveSessionsCount возвращает количество активных сессий пользователя
// func (r *SessionRepository) GetUserActiveSessionsCount(userID int) (int, error) {
// 	query := `
//     SELECT COUNT(*)
//     FROM user_sessions
//     WHERE user_id = $1
//       AND is_active = TRUE
//       AND expires_at > NOW()
//       AND revoked_at IS NULL
//     `

// 	var count int
// 	err := r.db.QueryRow(query, userID).Scan(&count)
// 	if err != nil {
// 		return 0, err
// 	}

// 	return count, nil
// }

// // GetSessionStats возвращает статистику сессий
// func (r *SessionRepository) GetSessionStats(ctx context.Context) (*SessionStats, error) {
// 	stats := &SessionStats{}

// 	// Общее количество сессий
// 	err := r.db.QueryRowContext(ctx,
// 		"SELECT COUNT(*) FROM user_sessions").Scan(&stats.TotalSessions)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Активные сессии
// 	err = r.db.QueryRowContext(ctx, `
//         SELECT COUNT(*)
//         FROM user_sessions
//         WHERE is_active = TRUE
//           AND expires_at > NOW()
//           AND revoked_at IS NULL
//     `).Scan(&stats.ActiveSessions)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Средняя продолжительность сессии
// 	var avgSeconds float64
// 	err = r.db.QueryRowContext(ctx, `
//         SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (last_activity - created_at))), 0)
//         FROM user_sessions
//         WHERE is_active = FALSE
//           AND last_activity IS NOT NULL
//     `).Scan(&avgSeconds)
// 	if err != nil {
// 		return nil, err
// 	}
// 	stats.AvgSessionDuration = time.Duration(avgSeconds * float64(time.Second))

// 	// Максимальное количество одновременных сессий за последние 24 часа
// 	err = r.db.QueryRowContext(ctx, `
//         SELECT COALESCE(MAX(concurrent_count), 0)
//         FROM (
//             SELECT COUNT(*) as concurrent_count
//             FROM user_sessions
//             WHERE created_at >= NOW() - INTERVAL '24 hours'
//             GROUP BY DATE_TRUNC('hour', created_at)
//         ) as hourly_counts
//     `).Scan(&stats.MaxConcurrent)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Самый активный час
// 	err = r.db.QueryRowContext(ctx, `
//         SELECT EXTRACT(HOUR FROM created_at) as hour
//         FROM user_sessions
//         WHERE created_at >= NOW() - INTERVAL '7 days'
//         GROUP BY EXTRACT(HOUR FROM created_at)
//         ORDER BY COUNT(*) DESC
//         LIMIT 1
//     `).Scan(&stats.MostActiveHour)
// 	if err != nil && err != sql.ErrNoRows {
// 		return nil, err
// 	}

// 	return stats, nil
// }

// // UpdateSessionData обновляет дополнительные данные сессии
// func (r *SessionRepository) UpdateSessionData(sessionID string, data map[string]interface{}) error {
// 	dataJSON, err := json.Marshal(data)
// 	if err != nil {
// 		return fmt.Errorf("failed to marshal session data: %w", err)
// 	}

// 	query := `
//     UPDATE user_sessions
//     SET data = $1,
//         updated_at = NOW()
//     WHERE id = $2
//     `

// 	result, err := r.db.Exec(query, dataJSON, sessionID)
// 	if err != nil {
// 		return err
// 	}

// 	rowsAffected, _ := result.RowsAffected()
// 	if rowsAffected == 0 {
// 		return sql.ErrNoRows
// 	}

// 	return nil
// }

// // GetRecentlyActiveSessions возвращает недавно активные сессии
// func (r *SessionRepository) GetRecentlyActiveSessions(hours int) ([]*SessionRecord, error) {
// 	query := `
//     SELECT
//         id, user_id, token, device_info, ip_address,
//         user_agent, expires_at, data, created_at, updated_at,
//         last_activity, is_active, revoked_at, revoked_reason
//     FROM user_sessions
//     WHERE last_activity >= NOW() - INTERVAL '1 hour' * $1
//       AND is_active = TRUE
//     ORDER BY last_activity DESC
//     `

// 	rows, err := r.db.Query(query, hours)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	var sessions []*SessionRecord

// 	for rows.Next() {
// 		var session SessionRecord
// 		var deviceInfoJSON, dataJSON []byte

// 		err := rows.Scan(
// 			&session.ID,
// 			&session.UserID,
// 			&session.Token,
// 			&deviceInfoJSON,
// 			&session.IPAddress,
// 			&session.UserAgent,
// 			&session.ExpiresAt,
// 			&dataJSON,
// 			&session.CreatedAt,
// 			&session.UpdatedAt,
// 			&session.LastActivity,
// 			&session.IsActive,
// 			&session.RevokedAt,
// 			&session.RevokedReason,
// 		)

// 		if err != nil {
// 			return nil, err
// 		}

// 		// Декодируем JSON данные
// 		if len(deviceInfoJSON) > 0 {
// 			if err := json.Unmarshal(deviceInfoJSON, &session.DeviceInfo); err != nil {
// 				return nil, fmt.Errorf("failed to unmarshal device info: %w", err)
// 			}
// 		}

// 		if len(dataJSON) > 0 {
// 			if err := json.Unmarshal(dataJSON, &session.Data); err != nil {
// 				return nil, fmt.Errorf("failed to unmarshal session data: %w", err)
// 			}
// 		}

// 		sessions = append(sessions, &session)
// 	}

// 	return sessions, nil
// }

// // GetSessionActivityLog возвращает лог активности сессии
// func (r *SessionRepository) GetSessionActivityLog(sessionID string, limit int) ([]map[string]interface{}, error) {
// 	query := `
//     SELECT
//         activity_type, details, ip_address, created_at
//     FROM session_activities
//     WHERE session_id = $1
//     ORDER BY created_at DESC
//     LIMIT $2
//     `

// 	rows, err := r.db.Query(query, sessionID, limit)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	var activities []map[string]interface{}

// 	for rows.Next() {
// 		var activityType, ipAddress string
// 		var createdAt time.Time
// 		var detailsJSON []byte

// 		err := rows.Scan(&activityType, &detailsJSON, &ipAddress, &createdAt)
// 		if err != nil {
// 			return nil, err
// 		}

// 		var detailsMap map[string]interface{}
// 		if len(detailsJSON) > 0 {
// 			if err := json.Unmarshal(detailsJSON, &detailsMap); err != nil {
// 				detailsMap = map[string]interface{}{"raw": string(detailsJSON)}
// 			}
// 		} else {
// 			detailsMap = make(map[string]interface{})
// 		}

// 		activities = append(activities, map[string]interface{}{
// 			"activity_type": activityType,
// 			"details":       detailsMap,
// 			"ip_address":    ipAddress,
// 			"created_at":    createdAt,
// 		})
// 	}

// 	return activities, nil
// }

// // LogSessionActivity записывает активность сессии
// func (r *SessionRepository) LogSessionActivity(sessionID, activityType string, details map[string]interface{}) error {
// 	detailsJSON, err := json.Marshal(details)
// 	if err != nil {
// 		return fmt.Errorf("failed to marshal activity details: %w", err)
// 	}

// 	query := `
//     INSERT INTO session_activities (session_id, activity_type, details)
//     VALUES ($1, $2, $3)
//     `

// 	_, err = r.db.Exec(query, sessionID, activityType, detailsJSON)
// 	return err
// }

// // GenerateNewToken генерирует новый токен для сессии
// func (r *SessionRepository) GenerateNewToken(sessionID string) (string, error) {
// 	newToken := uuid.New().String()

// 	query := `
//     UPDATE user_sessions
//     SET token = $1,
//         updated_at = NOW()
//     WHERE id = $2 AND is_active = TRUE
//     `

// 	result, err := r.db.Exec(query, newToken, sessionID)
// 	if err != nil {
// 		return "", err
// 	}

// 	rowsAffected, _ := result.RowsAffected()
// 	if rowsAffected == 0 {
// 		return "", sql.ErrNoRows
// 	}

// 	return newToken, nil
// }

// // GetSessionsByIP возвращает сессии по IP адресу
// func (r *SessionRepository) GetSessionsByIP(ipAddress string) ([]*SessionRecord, error) {
// 	query := `
//     SELECT
//         id, user_id, token, device_info, ip_address,
//         user_agent, expires_at, data, created_at, updated_at,
//         last_activity, is_active, revoked_at, revoked_reason
//     FROM user_sessions
//     WHERE ip_address = $1
//       AND created_at >= NOW() - INTERVAL '30 days'
//     ORDER BY created_at DESC
//     `

// 	rows, err := r.db.Query(query, ipAddress)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	var sessions []*SessionRecord

// 	for rows.Next() {
// 		var session SessionRecord
// 		var deviceInfoJSON, dataJSON []byte

// 		err := rows.Scan(
// 			&session.ID,
// 			&session.UserID,
// 			&session.Token,
// 			&deviceInfoJSON,
// 			&session.IPAddress,
// 			&session.UserAgent,
// 			&session.ExpiresAt,
// 			&dataJSON,
// 			&session.CreatedAt,
// 			&session.UpdatedAt,
// 			&session.LastActivity,
// 			&session.IsActive,
// 			&session.RevokedAt,
// 			&session.RevokedReason,
// 		)

// 		if err != nil {
// 			return nil, err
// 		}

// 		// Декодируем JSON данные
// 		if len(deviceInfoJSON) > 0 {
// 			if err := json.Unmarshal(deviceInfoJSON, &session.DeviceInfo); err != nil {
// 				return nil, fmt.Errorf("failed to unmarshal device info: %w", err)
// 			}
// 		}

// 		if len(dataJSON) > 0 {
// 			if err := json.Unmarshal(dataJSON, &session.Data); err != nil {
// 				return nil, fmt.Errorf("failed to unmarshal session data: %w", err)
// 			}
// 		}

// 		sessions = append(sessions, &session)
// 	}

// 	return sessions, nil
// }

// // BulkRevokeInactive массово отзывает неактивные сессии
// func (r *SessionRepository) BulkRevokeInactive(ctx context.Context, inactiveFor time.Duration) (int, error) {
// 	query := `
//     UPDATE user_sessions
//     SET is_active = FALSE,
//         revoked_at = NOW(),
//         revoked_reason = 'inactive',
//         updated_at = NOW()
//     WHERE is_active = TRUE
//       AND last_activity < NOW() - $1
//       AND revoked_at IS NULL
//     `

// 	result, err := r.db.ExecContext(ctx, query, inactiveFor)
// 	if err != nil {
// 		return 0, err
// 	}

// 	rowsAffected, _ := result.RowsAffected()
// 	return int(rowsAffected), nil
// }

// // CreateSessionForUser создает новую сессию для пользователя
// func (r *SessionRepository) CreateSessionForUser(userID int, userAgent, ipAddress string) (*users.Session, error) {
// 	session := &users.Session{
// 		ID:        uuid.New().String(),
// 		UserID:    userID,
// 		Token:     uuid.New().String(),
// 		IP:        ipAddress, // Исправлено: IPAddress -> IP
// 		UserAgent: userAgent,
// 		ExpiresAt: time.Now().Add(30 * 24 * time.Hour), // 30 дней
// 		CreatedAt: time.Now(),
// 	}

// 	if err := r.Create(session); err != nil {
// 		return nil, err
// 	}

// 	return session, nil
// }

// // CreateSessionForUserWithDetails создает сессию с деталями
// func (r *SessionRepository) CreateSessionForUserWithDetails(userID int, userAgent, ipAddress string, deviceInfo, data map[string]interface{}) (*SessionRecord, error) {
// 	session := &SessionRecord{
// 		ID:         uuid.New().String(),
// 		UserID:     userID,
// 		Token:      uuid.New().String(),
// 		DeviceInfo: deviceInfo,
// 		IPAddress:  ipAddress,
// 		UserAgent:  userAgent,
// 		Data:       data,
// 		ExpiresAt:  time.Now().Add(30 * 24 * time.Hour), // 30 дней
// 		IsActive:   true,
// 	}

// 	if err := r.CreateWithDetails(session); err != nil {
// 		return nil, err
// 	}

// 	return session, nil
// }

// // ConvertToUserSession конвертирует SessionRecord в users.Session
// func (r *SessionRepository) ConvertToUserSession(record *SessionRecord) *users.Session {
// 	return &users.Session{
// 		ID:        record.ID,
// 		UserID:    record.UserID,
// 		Token:     record.Token,
// 		IP:        record.IPAddress,
// 		UserAgent: record.UserAgent,
// 		ExpiresAt: record.ExpiresAt,
// 		CreatedAt: record.CreatedAt,
// 	}
// }
