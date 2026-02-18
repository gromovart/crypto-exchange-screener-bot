// internal/delivery/telegram/services/payment/interface.go
package payment

import (
	"crypto-exchange-screener-bot/internal/core/domain/payment"
	"crypto-exchange-screener-bot/internal/core/domain/subscription"
	"crypto-exchange-screener-bot/internal/core/domain/users"
	"time"
)

// Service интерфейс сервиса обработки платежей
type Service interface {
	// Exec выполняет операции с платежами
	Exec(params PaymentParams) (PaymentResult, error)
}

// PaymentParams параметры для Exec
type PaymentParams struct {
	Action           string                 `json:"action"`              // Действие: pre_checkout, successful_payment, activate_subscription
	UserID           int                    `json:"user_id"`             // ID пользователя
	ChatID           int64                  `json:"chat_id,omitempty"`   // ID чата
	Data             map[string]interface{} `json:"data,omitempty"`      // Данные платежа
	TelegramUpdateID string                 `json:"update_id,omitempty"` // ID обновления Telegram
}

// PaymentResult результат Exec
type PaymentResult struct {
	Success        bool                   `json:"success"`
	Message        string                 `json:"message,omitempty"`         // Сообщение для пользователя
	PaymentID      string                 `json:"payment_id,omitempty"`      // ID платежа
	SubscriptionID string                 `json:"subscription_id,omitempty"` // ID подписки
	InvoiceURL     string                 `json:"invoice_url,omitempty"`     // Ссылка на инвойс
	StarsAmount    int                    `json:"stars_amount,omitempty"`    // Сумма в Stars
	ActivatedUntil time.Time              `json:"activated_until,omitempty"` // Подписка активна до
	Metadata       map[string]interface{} `json:"metadata,omitempty"`        // Дополнительные метаданные
}

// Dependencies зависимости для сервиса платежей
type Dependencies struct {
	PaymentService      *payment.PaymentService // ⭐ Изменено
	SubscriptionService *subscription.Service
	UserService         *users.Service
}

// NewServiceWithDependencies фабрика с зависимостями (обновленная без factory.go)
func NewServiceWithDependencies(deps Dependencies) Service {
	return NewService(deps)
}
