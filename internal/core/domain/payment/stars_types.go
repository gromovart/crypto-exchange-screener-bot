// internal/core/domain/payment/stars_types.go
package payment

import (
	"context"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
	"time"
)

// StarsCommissionRate комиссия Telegram Stars (5%)
const StarsCommissionRate = 0.05

// UserManager интерфейс менеджера пользователей
type UserManager interface {
	GetUserByID(userID int) (*models.User, error) // ⬅️ ИЗМЕНЕНО: int вместо string
	UpdateSubscriptionTier(userID int, tier string) error
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
	Success        bool
	PaymentID      string
	InvoiceID      string
	UserID         string
	PlanID         string
	SubscriptionID int // ⭐ ДОБАВЛЕНО: ID созданной или обновленной подписки
	Timestamp      time.Time
}

// InvoiceData данные из парсинга payload
type InvoiceData struct {
	UserID             string
	SubscriptionPlanID string
	InvoiceID          string
}


// ToMap конвертирует PaymentEventData в map
func (d PaymentEventData) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"payment_id":      d.PaymentID,
		"user_id":         d.UserID,
		"plan_id":         d.PlanID,
		"stars_amount":    d.StarsAmount,
		"payment_type":    d.PaymentType,
		"timestamp":       d.Timestamp,
		"invoice_id":      d.InvoiceID,
		"subscription_id": d.SubscriptionID,
		"metadata":        d.Metadata,
	}
}

// ⭐ НОВЫЙ ИНТЕРФЕЙС: SubscriptionService для работы с подписками
type SubscriptionService interface {
	GetActiveSubscription(ctx context.Context, userID int) (*models.UserSubscription, error)
	CreateSubscription(ctx context.Context, userID int, planCode string, paymentID *int64, isTrial bool) (*models.UserSubscription, error)
	UpgradeSubscription(ctx context.Context, userID int, newPlanCode string, paymentID *int64) (*models.UserSubscription, error)
}
