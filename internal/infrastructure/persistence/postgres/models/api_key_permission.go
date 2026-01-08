package models

import (
	"time"
)

// APIKeyPermission разрешение API ключа
type APIKeyPermission struct {
	ID         int       `db:"id" json:"id"`
	APIKeyID   int       `db:"api_key_id" json:"api_key_id"`
	Permission string    `db:"permission" json:"permission"`
	GrantedAt  time.Time `db:"granted_at" json:"granted_at"`
	GrantedBy  *int      `db:"granted_by" json:"granted_by,omitempty"` // NULLable
}

// APIKeyPermissionRequest запрос на добавление разрешения
type APIKeyPermissionRequest struct {
	APIKeyID   int    `json:"api_key_id" validate:"required"`
	Permission string `json:"permission" validate:"required"`
}

// APIKeyPermissionsList список разрешений
type APIKeyPermissionsList struct {
	APIKeyID     int                  `json:"api_key_id"`
	Permissions  []string             `json:"permissions"`
	GrantedByMap map[string]int       `json:"granted_by_map,omitempty"`
	GrantedAtMap map[string]time.Time `json:"granted_at_map,omitempty"`
}
