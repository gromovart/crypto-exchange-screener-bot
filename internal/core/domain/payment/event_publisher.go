// internal/core/domain/payment/event_publisher.go
package payment

import (
	"crypto-exchange-screener-bot/internal/types"
	"time"
)

// EventPublisher интерфейс для публикации событий платежей
type EventPublisher interface {
	Publish(event types.Event) error // ⬅️ Используем стандартный метод
}

// PaymentEventData данные события платежа
type PaymentEventData struct {
	PaymentID      string                 `json:"payment_id"`
	UserID         string                 `json:"user_id"`
	PlanID         string                 `json:"plan_id"`
	StarsAmount    int                    `json:"stars_amount"`
	PaymentType    string                 `json:"payment_type"`
	Timestamp      time.Time              `json:"timestamp"`
	InvoiceID      string                 `json:"invoice_id,omitempty"`
	SubscriptionID int                    `json:"subscription_id"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// PaymentEventTypes типы событий платежей
const ()

// CreatePaymentEventData создает данные события платежа
func CreatePaymentEventData(
	paymentID, userID, planID string,
	starsAmount int,
	paymentType string,
	invoiceID string,
	subscriptionID int,
) PaymentEventData {
	return PaymentEventData{
		PaymentID:      paymentID,
		UserID:         userID,
		PlanID:         planID,
		StarsAmount:    starsAmount,
		PaymentType:    paymentType,
		Timestamp:      time.Now(),
		InvoiceID:      invoiceID,
		Metadata:       make(map[string]interface{}),
		SubscriptionID: subscriptionID,
	}
}
