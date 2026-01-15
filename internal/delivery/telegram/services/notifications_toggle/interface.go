// internal/delivery/telegram/services/notifications_toggle/interface.go
package notifications_toggle

import "crypto-exchange-screener-bot/internal/core/domain/users"

type Service interface {
	// Exec выполняет переключение уведомлений
	Exec(params NotificationsToggleResultParams) (NotificationsToggleResult, error)
}

// NotificationsToggleResultParams параметры для Exec
type NotificationsToggleResultParams struct {
	Data interface{} // ИСПРАВЛЕНО: Data с большой буквы
}

// NotificationsToggleResult результат Exec
type NotificationsToggleResult struct {
	Processed bool   `json:"processed"`
	Message   string `json:"message,omitempty"`
	SentTo    int    `json:"sent_to,omitempty"`
}

// NewServiceWithDependencies фабрика с зависимостями
func NewServiceWithDependencies(
	userService *users.Service,
) Service {
	return NewService(userService)
}
