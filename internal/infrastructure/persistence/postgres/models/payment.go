// internal/infrastructure/persistence/postgres/models/payment.go
package models

import (
	"fmt"
	"time"
)

// PaymentStatus статус платежа (совместимость с payment.PaymentStatus)
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusCompleted PaymentStatus = "completed"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusRefunded  PaymentStatus = "refunded"
	// Дополнительные статусы
	PaymentStatusProcessing PaymentStatus = "processing"
	PaymentStatusExpired    PaymentStatus = "expired"
	PaymentStatusCancelled  PaymentStatus = "cancelled"
)

// PaymentType тип платежа
type PaymentType string

const (
	PaymentTypeStars    PaymentType = "stars"     // Telegram Stars
	PaymentTypeCrypto   PaymentType = "crypto"    // Криптовалюта
	PaymentTypeBankCard PaymentType = "bank_card" // Банковская карта
)

// Currency валюта платежа
type Currency string

const (
	CurrencyUSD Currency = "USD" // Доллары США
	CurrencyEUR Currency = "EUR" // Евро
	CurrencyRUB Currency = "RUB" // Рубли
)

// Payment модель платежа в системе (совместимость с текущей реализацией)
type Payment struct {
	ID             int64  `db:"id" json:"id"`
	UserID         int64  `db:"user_id" json:"user_id"`                           // ID пользователя (числовой)
	SubscriptionID *int64 `db:"subscription_id" json:"subscription_id,omitempty"` // ID подписки
	InvoiceID      *int64 `db:"invoice_id" json:"invoice_id,omitempty"`           // ID инвойса

	// Основная информация (совместимость с stars_types.go)
	ExternalID  string   `db:"external_id" json:"external_id"`   // TelegramPaymentID в системе
	Amount      float64  `db:"amount" json:"amount"`             // Сумма в USD
	Currency    Currency `db:"currency" json:"currency"`         // Валюта
	StarsAmount int      `db:"stars_amount" json:"stars_amount"` // Количество Stars
	FiatAmount  int      `db:"fiat_amount" json:"fiat_amount"`   // Сумма в центах (для совместимости)

	// Тип и статус
	PaymentType PaymentType   `db:"payment_type" json:"payment_type"` // Тип платежа
	Status      PaymentStatus `db:"status" json:"status"`             // Текущий статус
	Provider    string        `db:"provider" json:"provider"`         // Платежный провайдер

	// Детали платежа
	Description string `db:"description" json:"description,omitempty"` // Описание платежа
	Payload     string `db:"payload" json:"payload,omitempty"`         // Payload из инвойса
	Metadata    string `db:"metadata" json:"metadata,omitempty"`       // Дополнительные данные (JSON)

	// Временные метки
	CreatedAt time.Time  `db:"created_at" json:"created_at"`           // Дата создания
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`           // Дата обновления
	PaidAt    *time.Time `db:"paid_at" json:"paid_at,omitempty"`       // Дата оплаты
	ExpiresAt *time.Time `db:"expires_at" json:"expires_at,omitempty"` // Срок действия платежа
}

// TableName задает имя таблицы в БД
func (Payment) TableName() string {
	return "payments"
}

// IsPending проверяет, ожидает ли платеж оплаты
func (p *Payment) IsPending() bool {
	return p.Status == PaymentStatusPending
}

// IsCompleted проверяет, завершен ли платеж успешно
func (p *Payment) IsCompleted() bool {
	return p.Status == PaymentStatusCompleted
}

// IsFailed проверяет, завершен ли платеж с ошибкой
func (p *Payment) IsFailed() bool {
	return p.Status == PaymentStatusFailed
}

// CanBeRefunded проверяет, можно ли сделать возврат
func (p *Payment) CanBeRefunded() bool {
	return p.Status == PaymentStatusCompleted &&
		p.PaidAt != nil &&
		time.Since(*p.PaidAt) < 30*24*time.Hour // 30 дней для возврата
}

// GetAmountWithCurrency возвращает сумму с валютой
func (p *Payment) GetAmountWithCurrency() string {
	return FormatCurrency(p.Amount, p.Currency)
}

// FormatCurrency форматирует сумму с валютой
func FormatCurrency(amount float64, currency Currency) string {
	switch currency {
	case CurrencyUSD:
		return fmt.Sprintf("$%.2f", amount)
	case CurrencyEUR:
		return fmt.Sprintf("€%.2f", amount)
	case CurrencyRUB:
		return fmt.Sprintf("₽%.0f", amount)
	default:
		return fmt.Sprintf("%.2f %s", amount, currency)
	}
}

// PaymentSummary краткая статистика по платежам
type PaymentSummary struct {
	TotalPayments   int     `json:"total_payments"`
	TotalAmount     float64 `json:"total_amount"`
	SuccessfulCount int     `json:"successful_count"`
	FailedCount     int     `json:"failed_count"`
	PendingCount    int     `json:"pending_count"`
}

// PaymentFilter фильтр для поиска платежей
type PaymentFilter struct {
	UserID      int64         `json:"user_id,omitempty"`
	Status      PaymentStatus `json:"status,omitempty"`
	PaymentType PaymentType   `json:"payment_type,omitempty"`
	StartDate   *time.Time    `json:"start_date,omitempty"`
	EndDate     *time.Time    `json:"end_date,omitempty"`
	Limit       int           `json:"limit,omitempty"`
	Offset      int           `json:"offset,omitempty"`
}

// NewPaymentFilter создает новый фильтр с настройками по умолчанию
func NewPaymentFilter() PaymentFilter {
	return PaymentFilter{
		Limit:  50,
		Offset: 0,
	}
}

// GetTelegramPaymentID возвращает ExternalID как TelegramPaymentID (для совместимости)
func (p *Payment) GetTelegramPaymentID() string {
	return p.ExternalID
}

// GetUserIDString возвращает UserID как строку (для совместимости)
func (p *Payment) GetUserIDString() string {
	return fmt.Sprintf("%d", p.UserID)
}

// SetTelegramPaymentID устанавливает TelegramPaymentID
func (p *Payment) SetTelegramPaymentID(paymentID string) {
	p.ExternalID = paymentID
}
