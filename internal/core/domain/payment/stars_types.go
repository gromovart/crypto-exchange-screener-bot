// internal/core/domain/payment/stars_types.go
package payment

import (
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
	"time"
)

// StarsCommissionRate комиссия Telegram Stars (5%)
const StarsCommissionRate = 0.05

// UserManager интерфейс менеджера пользователей
type UserManager interface {
	GetUserByID(userID int) (*models.User, error) // ⬅️ ИЗМЕНЕНО: int вместо string
}

// SubscriptionPlan интерфейс плана подписки
type SubscriptionPlan interface {
	GetID() string
	GetName() string
	GetPriceCents() int
}

// ActivationResult результат активации подписки
type ActivationResult struct {
	ActiveUntil time.Time
}

// CreateInvoiceRequest запрос на создание инвойса
type CreateInvoiceRequest struct {
	UserID           string
	SubscriptionPlan SubscriptionPlan
}

// ProcessPaymentRequest запрос на обработку платежа
type ProcessPaymentRequest struct {
	Payload           string
	TelegramPaymentID string
	StarsAmount       int
}

// StarsInvoice инвойс для оплаты Stars
type StarsInvoice struct {
	ID                 string
	UserID             string
	SubscriptionPlanID string
	StarsAmount        int
	FiatAmount         int
	Currency           string
	Payload            string
	InvoiceURL         string
	Status             PaymentStatus
	CreatedAt          time.Time
	ExpiresAt          time.Time
}

// PaymentStatus статус платежа
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusCompleted PaymentStatus = "completed"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusRefunded  PaymentStatus = "refunded"
)

// StarsPaymentResult результат обработки платежа
type StarsPaymentResult struct {
	Success   bool
	PaymentID string
	InvoiceID string
	UserID    string
	PlanID    string
	Timestamp time.Time
}

// InvoiceData данные из парсинга payload
type InvoiceData struct {
	UserID             string
	SubscriptionPlanID string
	InvoiceID          string
}
