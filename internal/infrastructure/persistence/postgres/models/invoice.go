// internal/infrastructure/persistence/postgres/models/invoice.go
package models

import (
	"encoding/json"
	"fmt"
	"time"
)

// InvoiceStatus статус инвойса (совместимость с текущей реализацией)
type InvoiceStatus string

const (
	InvoiceStatusCreated   InvoiceStatus = "created"   // Создан
	InvoiceStatusPending   InvoiceStatus = "pending"   // Ожидает оплаты
	InvoiceStatusPaid      InvoiceStatus = "paid"      // Оплачен
	InvoiceStatusExpired   InvoiceStatus = "expired"   // Истек срок
	InvoiceStatusCancelled InvoiceStatus = "cancelled" // Отменен
	InvoiceStatusFailed    InvoiceStatus = "failed"    // Ошибка оплаты
)

// InvoiceProvider провайдер инвойса
type InvoiceProvider string

const (
	InvoiceProviderTelegram InvoiceProvider = "telegram" // Telegram Stars
	InvoiceProviderStripe   InvoiceProvider = "stripe"   // Stripe
	InvoiceProviderManual   InvoiceProvider = "manual"   // Ручной платеж
)

// Invoice модель инвойса (счета на оплату)
type Invoice struct {
	ID     int64  `db:"id" json:"id"`
	UserID int64  `db:"user_id" json:"user_id"` // ID пользователя
	PlanID string `db:"plan_id" json:"plan_id"` // ID плана подписки

	// Основная информация
	ExternalID  string `db:"external_id" json:"external_id"`           // Внешний ID инвойса
	Title       string `db:"title" json:"title"`                       // Название инвойса
	Description string `db:"description" json:"description,omitempty"` // Описание

	// Сумма и валюта
	AmountUSD   float64 `db:"amount_usd" json:"amount_usd"`     // Сумма в USD
	StarsAmount int     `db:"stars_amount" json:"stars_amount"` // Количество Stars
	FiatAmount  int     `db:"fiat_amount" json:"fiat_amount"`   // Сумма в центах
	Currency    string  `db:"currency" json:"currency"`         // Валюта (USD, EUR, RUB)

	// Статус и провайдер
	Status   InvoiceStatus   `db:"status" json:"status"`     // Текущий статус
	Provider InvoiceProvider `db:"provider" json:"provider"` // Провайдер платежа

	// Ссылки и данные
	InvoiceURL string          `db:"invoice_url" json:"invoice_url"`     // Ссылка на оплату
	Payload    string          `db:"payload" json:"payload,omitempty"`   // Данные для deep link
	Metadata   json.RawMessage `db:"metadata" json:"metadata,omitempty"` // ⭐ Изменено с string на json.RawMessage

	// Временные метки
	CreatedAt time.Time  `db:"created_at" json:"created_at"`     // Дата создания
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`     // Дата обновления
	ExpiresAt time.Time  `db:"expires_at" json:"expires_at"`     // Срок действия
	PaidAt    *time.Time `db:"paid_at" json:"paid_at,omitempty"` // Дата оплаты
}

// TableName задает имя таблицы в БД
func (Invoice) TableName() string {
	return "invoices"
}

// IsActive проверяет, активен ли инвойс (не оплачен и не истек)
func (i *Invoice) IsActive() bool {
	now := time.Now()
	return (i.Status == InvoiceStatusCreated || i.Status == InvoiceStatusPending) &&
		now.Before(i.ExpiresAt)
}

// IsPaid проверяет, оплачен ли инвойс
func (i *Invoice) IsPaid() bool {
	return i.Status == InvoiceStatusPaid
}

// IsExpired проверяет, истек ли срок инвойса
func (i *Invoice) IsExpired() bool {
	now := time.Now()
	return i.Status != InvoiceStatusPaid && now.After(i.ExpiresAt)
}

// MarkAsPaid помечает инвойс как оплаченный
func (i *Invoice) MarkAsPaid() {
	now := time.Now()
	i.Status = InvoiceStatusPaid
	i.PaidAt = &now
}

// MarkAsExpired помечает инвойс как просроченный
func (i *Invoice) MarkAsExpired() {
	i.Status = InvoiceStatusExpired
}

// MarkAsCancelled помечает инвойс как отмененный
func (i *Invoice) MarkAsCancelled() {
	i.Status = InvoiceStatusCancelled
}

// GetPlanName возвращает читаемое название плана
func (i *Invoice) GetPlanName() string {
	plans := map[string]string{
		"basic":      "📱 Доступ на 1 месяц",
		"pro":        "🚀 Доступ на 3 месяца",
		"enterprise": "🏢 Доступ на 12 месяцев",
	}

	if name, exists := plans[i.PlanID]; exists {
		return name
	}
	return "Неизвестный план"
}

// GetStatusDisplay возвращает читаемый статус
func (i *Invoice) GetStatusDisplay() string {
	statuses := map[InvoiceStatus]string{
		InvoiceStatusCreated:   "🆕 Создан",
		InvoiceStatusPending:   "⏳ Ожидает оплаты",
		InvoiceStatusPaid:      "✅ Оплачен",
		InvoiceStatusExpired:   "⌛ Истек срок",
		InvoiceStatusCancelled: "❌ Отменен",
		InvoiceStatusFailed:    "⚠️ Ошибка",
	}

	if display, exists := statuses[i.Status]; exists {
		return display
	}
	return string(i.Status)
}

// GetProviderDisplay возвращает читаемое название провайдера
func (i *Invoice) GetProviderDisplay() string {
	providers := map[InvoiceProvider]string{
		InvoiceProviderTelegram: "💎 Telegram Stars",
		InvoiceProviderStripe:   "💳 Stripe",
		InvoiceProviderManual:   "👤 Ручной платеж",
	}

	if display, exists := providers[i.Provider]; exists {
		return display
	}
	return string(i.Provider)
}

// InvoiceFilter фильтр для поиска инвойсов
type InvoiceFilter struct {
	UserID    int64           `json:"user_id,omitempty"`
	Status    InvoiceStatus   `json:"status,omitempty"`
	Provider  InvoiceProvider `json:"provider,omitempty"`
	PlanID    string          `json:"plan_id,omitempty"`
	StartDate *time.Time      `json:"start_date,omitempty"`
	EndDate   *time.Time      `json:"end_date,omitempty"`
	Limit     int             `json:"limit,omitempty"`
	Offset    int             `json:"offset,omitempty"`
}

// NewInvoiceFilter создает новый фильтр с настройками по умолчанию
func NewInvoiceFilter() InvoiceFilter {
	return InvoiceFilter{
		Limit:  50,
		Offset: 0,
	}
}

// InvoiceSummary краткая статистика по инвойсам
type InvoiceSummary struct {
	TotalInvoices  int     `json:"total_invoices"`
	TotalAmountUSD float64 `json:"total_amount_usd"`
	PaidCount      int     `json:"paid_count"`
	PendingCount   int     `json:"pending_count"`
	ExpiredCount   int     `json:"expired_count"`
}

// GetIDString возвращает ID как строку (для совместимости)
func (i *Invoice) GetIDString() string {
	return fmt.Sprintf("%d", i.ID)
}

// GetUserIDString возвращает UserID как строку (для совместимости)
func (i *Invoice) GetUserIDString() string {
	return fmt.Sprintf("%d", i.UserID)
}

// GetSubscriptionPlanID возвращает PlanID (для совместимости со StarsInvoice)
func (i *Invoice) GetSubscriptionPlanID() string {
	return i.PlanID
}

// GetFiatAmountCents возвращает сумму в центах (для совместимости)
func (i *Invoice) GetFiatAmountCents() int {
	return i.FiatAmount
}

// SetFiatAmountFromUSD устанавливает FiatAmount из суммы в USD
func (i *Invoice) SetFiatAmountFromUSD(amountUSD float64) {
	i.FiatAmount = int(amountUSD * 100)
}
