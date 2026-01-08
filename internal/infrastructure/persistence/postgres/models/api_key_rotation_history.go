package models

import (
	"time"
)

// APIKeyRotationHistory история ротации API ключей
type APIKeyRotationHistory struct {
	ID             int       `db:"id" json:"id"`
	UserID         int       `db:"user_id" json:"user_id"`
	Exchange       string    `db:"exchange" json:"exchange"`
	OldKeyID       *int      `db:"old_key_id" json:"old_key_id,omitempty"` // NULLable
	NewKeyID       int       `db:"new_key_id" json:"new_key_id"`           // NOT NULL
	RotatedBy      *int      `db:"rotated_by" json:"rotated_by,omitempty"` // NULLable
	RotationReason string    `db:"rotation_reason" json:"rotation_reason"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`

	// Связные данные (не хранятся в БД напрямую)
	OldKey        *APIKey `db:"-" json:"old_key,omitempty"`
	NewKey        *APIKey `db:"-" json:"new_key,omitempty"`
	RotatedByUser *User   `db:"-" json:"rotated_by_user,omitempty"`
	User          *User   `db:"-" json:"user,omitempty"`
}

// APIKeyRotationRequest запрос на ротацию API ключа
type APIKeyRotationRequest struct {
	UserID         int    `json:"user_id" validate:"required"`
	Exchange       string `json:"exchange" validate:"required"`
	NewAPIKey      string `json:"new_api_key" validate:"required"`
	NewAPISecret   string `json:"new_api_secret" validate:"required"`
	RotationReason string `json:"rotation_reason,omitempty"`
}

// APIKeyRotationStats статистика ротаций
type APIKeyRotationStats struct {
	TotalRotations              int            `json:"total_rotations"`
	LastRotation                time.Time      `json:"last_rotation"`
	RotationsByMonth            map[string]int `json:"rotations_by_month"` // ключ: "YYYY-MM"
	RotationsByExchange         map[string]int `json:"rotations_by_exchange"`
	AverageRotationIntervalDays float64        `json:"average_rotation_interval_days"`
}
