// internal/infrastructure/persistence/postgres/models/api_key_rotation_history.go
package models

import (
	"time"
)

// APIKeyRotationHistory модель истории ротации API ключей
type APIKeyRotationHistory struct {
	ID             int       `db:"id"`
	UserID         int       `db:"user_id"`
	Exchange       string    `db:"exchange"`
	OldKeyID       *int      `db:"old_key_id"`
	NewKeyID       int       `db:"new_key_id"`
	RotatedBy      *int      `db:"rotated_by"`
	RotationReason *string   `db:"rotation_reason"`
	CreatedAt      time.Time `db:"created_at"`

	// Связи (для eager loading)
	User          *User   `db:"-"`
	OldKey        *APIKey `db:"-"`
	NewKey        *APIKey `db:"-"`
	RotatedByUser *User   `db:"-"`
}

// GetRotationReason возвращает причину ротации или дефолтную
func (h *APIKeyRotationHistory) GetRotationReason() string {
	if h.RotationReason != nil {
		return *h.RotationReason
	}
	return "security_rotation"
}

// ToMap возвращает представление истории ротации в виде map
func (h *APIKeyRotationHistory) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"id":              h.ID,
		"user_id":         h.UserID,
		"exchange":        h.Exchange,
		"new_key_id":      h.NewKeyID,
		"created_at":      h.CreatedAt,
		"rotation_reason": h.GetRotationReason(),
	}

	if h.OldKeyID != nil {
		result["old_key_id"] = *h.OldKeyID
	}
	if h.RotatedBy != nil {
		result["rotated_by"] = *h.RotatedBy
	}

	return result
}
