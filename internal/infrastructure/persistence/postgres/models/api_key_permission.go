// internal/infrastructure/persistence/postgres/models/api_key_permission.go
package models

import (
	"time"
)

// APIKeyPermission модель разрешения API ключа
type APIKeyPermission struct {
	ID         int       `db:"id"`
	APIKeyID   int       `db:"api_key_id"`
	Permission string    `db:"permission"`
	GrantedAt  time.Time `db:"granted_at"`
	GrantedBy  *int      `db:"granted_by"`

	// Связи (для eager loading)
	APIKey        *APIKey `db:"-"`
	GrantedByUser *User   `db:"-"`
}

// IsValidPermission проверяет, является ли разрешение валидным
func (p *APIKeyPermission) IsValidPermission() bool {
	switch PermissionType(p.Permission) {
	case PermissionReadOnly,
		PermissionTrade,
		PermissionWithdraw,
		PermissionMargin,
		PermissionFutures,
		PermissionSpot,
		PermissionWallet,
		PermissionSubAccount:
		return true
	default:
		return false
	}
}

// ToMap возвращает представление разрешения в виде map
func (p *APIKeyPermission) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"id":         p.ID,
		"api_key_id": p.APIKeyID,
		"permission": p.Permission,
		"granted_at": p.GrantedAt,
		"is_valid":   p.IsValidPermission(),
	}

	if p.GrantedBy != nil {
		result["granted_by"] = *p.GrantedBy
	}

	return result
}
