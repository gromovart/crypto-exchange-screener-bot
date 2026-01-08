// internal/infrastructure/persistence/postgres/models/session.go
package models

import (
	"encoding/json"
	"time"
)

// Session - сессия пользователя
type Session struct {
	ID            string                 `db:"id" json:"id"`
	UserID        int                    `db:"user_id" json:"user_id"`
	Token         string                 `db:"token" json:"token"`
	DeviceInfo    map[string]interface{} `db:"device_info" json:"device_info,omitempty"`
	IPAddress     *string                `db:"ip_address" json:"ip_address,omitempty"`
	UserAgent     *string                `db:"user_agent" json:"user_agent,omitempty"`
	Data          map[string]interface{} `db:"data" json:"data,omitempty"`
	ExpiresAt     time.Time              `db:"expires_at" json:"expires_at"`
	CreatedAt     time.Time              `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time              `db:"updated_at" json:"updated_at"`
	LastActivity  time.Time              `db:"last_activity" json:"last_activity,omitempty"`
	IsActive      bool                   `db:"is_active" json:"is_active"`
	RevokedAt     *time.Time             `db:"revoked_at" json:"revoked_at,omitempty"`
	RevokedReason *string                `db:"revoked_reason" json:"revoked_reason,omitempty"`
	User          *User                  `db:"-" json:"user,omitempty"`
}

// SessionActivity - активность сессии
type SessionActivity struct {
	ID           int                    `db:"id" json:"id"`
	SessionID    string                 `db:"session_id" json:"session_id"`
	ActivityType string                 `db:"activity_type" json:"activity_type"`
	Details      map[string]interface{} `db:"details" json:"details,omitempty"`
	IPAddress    *string                `db:"ip_address" json:"ip_address,omitempty"`
	CreatedAt    time.Time              `db:"created_at" json:"created_at"`
}

// SessionStats - статистика сессий
type SessionStats struct {
	TotalSessions      int64              `json:"total_sessions"`
	ActiveSessions     int64              `json:"active_sessions"`
	UniqueUsers        int64              `json:"unique_users"`
	ExpiredSessions    int64              `json:"expired_sessions"`
	SessionsLast24h    int64              `json:"sessions_last_24h"`
	AvgSessionDuration time.Duration      `json:"avg_session_duration"`
	MostActiveUsers    []SessionUserStats `json:"most_active_users,omitempty"`
}

// SessionUserStats - статистика пользователя по сессиям
type SessionUserStats struct {
	UserID       int   `json:"user_id"`
	SessionCount int64 `json:"session_count"`
}

// SessionFilter фильтр для поиска сессий
type SessionFilter struct {
	UserID    *int       `json:"user_id,omitempty"`
	IsActive  *bool      `json:"is_active,omitempty"`
	Token     *string    `json:"token,omitempty"`
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`
	Limit     int        `json:"limit,omitempty"`
	Offset    int        `json:"offset,omitempty"`
	OrderBy   string     `json:"order_by,omitempty"`
	OrderDir  string     `json:"order_dir,omitempty"`
}

// DefaultSessionFilter возвращает фильтр по умолчанию
func DefaultSessionFilter() SessionFilter {
	return SessionFilter{
		Limit:    50,
		OrderBy:  "last_activity",
		OrderDir: "DESC",
	}
}

// MarshalJSON кастомная сериализация для Session
func (s *Session) MarshalJSON() ([]byte, error) {
	type Alias Session
	return json.Marshal(&struct {
		*Alias
		ExpiresAt    string `json:"expires_at"`
		CreatedAt    string `json:"created_at"`
		UpdatedAt    string `json:"updated_at"`
		LastActivity string `json:"last_activity,omitempty"`
		RevokedAt    string `json:"revoked_at,omitempty"`
	}{
		Alias:        (*Alias)(s),
		ExpiresAt:    s.ExpiresAt.Format(time.RFC3339),
		CreatedAt:    s.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    s.UpdatedAt.Format(time.RFC3339),
		LastActivity: formatTime(s.LastActivity),
		RevokedAt:    formatTimePtr(s.RevokedAt),
	})
}

// formatTime вспомогательная функция для форматирования времени
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

// formatTimePtr вспомогательная функция для форматирования указателя на время
func formatTimePtr(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}
