// internal/infrastructure/persistence/postgres/models/api_key.go
package models

import (
	"time"
)

// APIKey модель для хранения API ключа
type APIKey struct {
	ID          int                    `db:"id" json:"id"`
	UserID      int                    `db:"user_id" json:"user_id"`
	Exchange    string                 `db:"exchange" json:"exchange"`
	Label       string                 `db:"label" json:"label"`
	Permissions map[string]interface{} `db:"permissions" json:"permissions"`
	IsActive    bool                   `db:"is_active" json:"is_active"`
	LastUsedAt  time.Time              `db:"last_used_at" json:"last_used_at"`
	ExpiresAt   time.Time              `db:"expires_at" json:"expires_at"`
	CreatedAt   time.Time              `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time              `db:"updated_at" json:"updated_at"`
}

// APIKeyWithSecrets структура с расшифрованными ключами
type APIKeyWithSecrets struct {
	APIKey
	APIKeyPlain    string `json:"api_key" db:"-"`
	APISecretPlain string `json:"api_secret" db:"-"`
}

// APIKeyUsageLog лог использования API ключа
type APIKeyUsageLog struct {
	ID             int                    `db:"id" json:"id"`
	APIKeyID       int                    `db:"api_key_id" json:"api_key_id"`
	Action         string                 `db:"action" json:"action"`
	Endpoint       string                 `db:"endpoint" json:"endpoint"`
	RequestBody    map[string]interface{} `db:"request_body" json:"request_body"`
	ResponseStatus int                    `db:"response_status" json:"response_status"`
	ResponseBody   map[string]interface{} `db:"response_body" json:"response_body"`
	IPAddress      string                 `db:"ip_address" json:"ip_address"`
	UserAgent      string                 `db:"user_agent" json:"user_agent"`
	LatencyMs      int                    `db:"latency_ms" json:"latency_ms"`
	ErrorMessage   string                 `db:"error_message" json:"error_message"`
	CreatedAt      time.Time              `db:"created_at" json:"created_at"`
}
