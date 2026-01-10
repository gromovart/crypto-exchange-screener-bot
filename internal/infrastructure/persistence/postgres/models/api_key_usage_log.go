// internal/infrastructure/persistence/postgres/models/api_key_usage_log.go
package models

import (
	"time"
)

// APIKeyUsageLog модель лога использования API ключа
type APIKeyUsageLog struct {
	ID             int                    `db:"id"`
	APIKeyID       int                    `db:"api_key_id"`
	Action         string                 `db:"action"`
	Endpoint       *string                `db:"endpoint"`
	RequestBody    map[string]interface{} `db:"request_body"`
	ResponseStatus *int                   `db:"response_status"`
	ResponseBody   map[string]interface{} `db:"response_body"`
	IPAddress      *string                `db:"ip_address"`
	UserAgent      *string                `db:"user_agent"`
	LatencyMS      *int                   `db:"latency_ms"`
	ErrorMessage   *string                `db:"error_message"`
	CreatedAt      time.Time              `db:"created_at"`

	// Связи (для eager loading)
	APIKey *APIKey `db:"-"`
}

// IsSuccess проверяет, был ли запрос успешным
func (l *APIKeyUsageLog) IsSuccess() bool {
	if l.ResponseStatus == nil {
		return l.ErrorMessage == nil
	}
	return *l.ResponseStatus >= 200 && *l.ResponseStatus < 400
}

// IsError проверяет, была ли ошибка
func (l *APIKeyUsageLog) IsError() bool {
	if l.ErrorMessage != nil {
		return true
	}
	if l.ResponseStatus != nil {
		return *l.ResponseStatus >= 400
	}
	return false
}

// GetLatency возвращает латенцию в миллисекундах
func (l *APIKeyUsageLog) GetLatency() int {
	if l.LatencyMS == nil {
		return 0
	}
	return *l.LatencyMS
}

// ToMap возвращает представление лога в виде map
func (l *APIKeyUsageLog) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"id":         l.ID,
		"api_key_id": l.APIKeyID,
		"action":     l.Action,
		"created_at": l.CreatedAt,
		"is_success": l.IsSuccess(),
		"is_error":   l.IsError(),
		"latency_ms": l.GetLatency(),
	}

	if l.Endpoint != nil {
		result["endpoint"] = *l.Endpoint
	}
	if l.IPAddress != nil {
		result["ip_address"] = *l.IPAddress
	}
	if l.UserAgent != nil {
		result["user_agent"] = *l.UserAgent
	}
	if l.ResponseStatus != nil {
		result["response_status"] = *l.ResponseStatus
	}
	if l.ErrorMessage != nil {
		result["error_message"] = *l.ErrorMessage
	}

	return result
}
