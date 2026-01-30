// internal/infrastructure/persistence/postgres/models/payment.go
package models

import (
	"fmt"
	"time"
)

// PaymentStatus статус платежа
type PaymentStatus string

const (
	PaymentStatusPending    PaymentStatus = "pending"    // Ожидает оплаты
	PaymentStatusProcessing PaymentStatus = "processing" // В обработке
	PaymentStatusCompleted  PaymentStatus = "completed"  // Успешно завершен
	PaymentStatusFailed     PaymentStatus = "failed"     // Ошибка оплаты
	PaymentStatusRefunded   PaymentStatus = "refunded"   // Возврат средств
	PaymentStatusExpired    PaymentStatus = "expired"    // Истек срок оплаты
	PaymentStatusCancelled  PaymentStatus = "cancelled"  // Отменен пользователем
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

// Payment модель платежа в системе
type Payment struct {
	ID             int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID         int64  `gorm:"index;not null" json:"user_id"`          // ID пользователя
	SubscriptionID *int64 `gorm:"index" json:"subscription_id,omitempty"` // ID подписки (если есть)
	InvoiceID      *int64 `gorm:"index" json:"invoice_id,omitempty"`      // ID инвойса (если есть)

	// Основная информация
	ExternalID  string   `gorm:"index;size:255" json:"external_id"`                      // ID платежа во внешней системе
	Amount      float64  `gorm:"type:decimal(10,2);not null" json:"amount"`              // Сумма платежа
	Currency    Currency `gorm:"type:varchar(3);not null;default:'USD'" json:"currency"` // Валюта
	StarsAmount int      `gorm:"not null" json:"stars_amount"`                           // Количество Stars

	// Тип и статус
	PaymentType PaymentType   `gorm:"type:varchar(20);not null" json:"payment_type"`             // Тип платежа
	Status      PaymentStatus `gorm:"type:varchar(20);not null;default:'pending'" json:"status"` // Текущий статус
	Provider    string        `gorm:"type:varchar(50);not null" json:"provider"`                 // Платежный провайдер (telegram, stripe, etc)

	// Детали платежа
	Description string `gorm:"type:text" json:"description,omitempty"` // Описание платежа
	Metadata    string `gorm:"type:jsonb" json:"metadata,omitempty"`   // Дополнительные данные (JSON)

	// Временные метки
	CreatedAt time.Time  `gorm:"autoCreateTime" json:"created_at"` // Дата создания
	UpdatedAt time.Time  `gorm:"autoUpdateTime" json:"updated_at"` // Дата обновления
	PaidAt    *time.Time `json:"paid_at,omitempty"`                // Дата оплаты
	ExpiresAt *time.Time `json:"expires_at,omitempty"`             // Срок действия платежа

	// Связи
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
	// Subscription и Invoice будут добавлены позже, когда создадим эти модели
	// Subscription Subscription  `gorm:"foreignKey:SubscriptionID" json:"subscription,omitempty"`
	// Invoice      Invoice       `gorm:"foreignKey:InvoiceID" json:"invoice,omitempty"`
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
