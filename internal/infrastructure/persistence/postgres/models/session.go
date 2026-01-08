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
	IP            string                 `db:"ip_address" json:"ip,omitempty"`
	UserAgent     string                 `db:"user_agent" json:"user_agent,omitempty"`
	Data          map[string]interface{} `db:"session_data" json:"data,omitempty"`
	ExpiresAt     time.Time              `db:"expires_at" json:"expires_at"`
	CreatedAt     time.Time              `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time              `db:"updated_at" json:"updated_at"`
	LastActivity  time.Time              `db:"last_activity" json:"last_activity,omitempty"`
	IsActive      bool                   `db:"is_active" json:"is_active"`
	RevokedAt     *time.Time             `db:"revoked_at" json:"revoked_at,omitempty"`
	RevokedReason string                 `db:"revoked_reason" json:"revoked_reason,omitempty"`
	User          *User                  `db:"-" json:"user,omitempty"`
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
