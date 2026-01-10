// internal/infrastructure/persistence/postgres/models/api_key.go
package models

import (
	"time"
)

// APIKey модель API ключа пользователя
type APIKey struct {
	ID                 int                    `db:"id"`
	UserID             int                    `db:"user_id"`
	Exchange           ExchangeType           `db:"exchange"`
	APIKeyEncrypted    string                 `db:"api_key_encrypted"`
	APISecretEncrypted string                 `db:"api_secret_encrypted"`
	Label              *string                `db:"label"`
	Permissions        map[string]interface{} `db:"permissions"`
	IsActive           bool                   `db:"is_active"`
	LastUsedAt         *time.Time             `db:"last_used_at"`
	ExpiresAt          *time.Time             `db:"expires_at"`
	CreatedAt          time.Time              `db:"created_at"`
	UpdatedAt          time.Time              `db:"updated_at"`

	// Связи (для eager loading)
	User           *User              `db:"-"`
	UsageLogs      []APIKeyUsageLog   `db:"-"`
	PermissionList []APIKeyPermission `db:"-"`
}

// APIKeyWithSecrets структура с расшифрованными ключами (для обработки в памяти)
type APIKeyWithSecrets struct {
	APIKey
	APIKeyPlain    string `json:"api_key" db:"-"`
	APISecretPlain string `json:"api_secret" db:"-"`
}

// IsExpired проверяет, истек ли срок действия ключа
func (k *APIKey) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false
	}
	return k.ExpiresAt.Before(time.Now())
}

// CanUse проверяет, можно ли использовать ключ
func (k *APIKey) CanUse() bool {
	if !k.IsActive {
		return false
	}
	if k.IsExpired() {
		return false
	}
	return true
}

// HasPermission проверяет наличие разрешения
func (k *APIKey) HasPermission(permission PermissionType) bool {
	// Проверяем в JSON поле permissions
	if k.Permissions != nil {
		if allowList, ok := k.Permissions["allow"].([]interface{}); ok {
			for _, p := range allowList {
				if pStr, ok := p.(string); ok && pStr == string(permission) {
					return true
				}
			}
		}
		// Проверяем deny list
		if denyList, ok := k.Permissions["deny"].([]interface{}); ok {
			for _, p := range denyList {
				if pStr, ok := p.(string); ok && pStr == string(permission) {
					return false
				}
			}
		}
	}

	// Проверяем в связанных разрешениях
	for _, perm := range k.PermissionList {
		if perm.Permission == string(permission) {
			return true
		}
	}

	return false
}

// UpdateLastUsed обновляет время последнего использования
func (k *APIKey) UpdateLastUsed() {
	now := time.Now()
	k.LastUsedAt = &now
	k.UpdatedAt = now
}

// Deactivate деактивирует ключ
func (k *APIKey) Deactivate() {
	k.IsActive = false
	k.UpdatedAt = time.Now()
}

// ToSafeMap возвращает безопасное представление ключа (без зашифрованных данных)
func (k *APIKey) ToSafeMap() map[string]interface{} {
	return map[string]interface{}{
		"id":           k.ID,
		"user_id":      k.UserID,
		"exchange":     k.Exchange,
		"label":        k.Label,
		"is_active":    k.IsActive,
		"last_used_at": k.LastUsedAt,
		"expires_at":   k.ExpiresAt,
		"created_at":   k.CreatedAt,
		"updated_at":   k.UpdatedAt,
		"can_use":      k.CanUse(),
		"is_expired":   k.IsExpired(),
	}
}
