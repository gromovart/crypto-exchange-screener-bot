// internal/delivery/telegram/services/signal_settings/interface.go
package signal_settings

import "crypto-exchange-screener-bot/internal/core/domain/users"

// Service интерфейс сервиса настройки сигналов
type Service interface {
	// Exec выполняет операции с настройками сигналов
	Exec(params SignalSettingsParams) (SignalSettingsResult, error)
}

// SignalSettingsParams параметры для Exec
type SignalSettingsParams struct {
	Action string      `json:"action"` // Действие: toggle_growth, toggle_fall, set_growth_threshold, set_fall_threshold
	UserID int         `json:"user_id"`
	ChatID int64       `json:"chat_id,omitempty"`
	Value  interface{} `json:"value,omitempty"`
	Data   interface{} `json:"data,omitempty"` // Дополнительные данные
}

// SignalSettingsResult результат Exec
type SignalSettingsResult struct {
	Success      bool                   `json:"success"`
	Message      string                 `json:"message,omitempty"`
	UpdatedField string                 `json:"updated_field,omitempty"`
	NewValue     interface{}            `json:"new_value,omitempty"`
	UserID       int                    `json:"user_id"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// NewServiceWithDependencies фабрика с зависимостями
func NewServiceWithDependencies(
	userService *users.Service,
) Service {
	return NewService(userService)
}
