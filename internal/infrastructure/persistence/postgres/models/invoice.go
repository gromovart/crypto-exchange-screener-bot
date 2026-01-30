package models

import (
	"time"
)

// InvoiceStatus статус инвойса
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
	ID     int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID int64  `gorm:"index;not null" json:"user_id"`            // ID пользователя
	PlanID string `gorm:"type:varchar(50);not null" json:"plan_id"` // ID плана подписки (basic, pro, enterprise)

	// Основная информация
	ExternalID  string `gorm:"index;size:255" json:"external_id"`       // ID инвойса во внешней системе
	Title       string `gorm:"type:varchar(255);not null" json:"title"` // Название инвойса
	Description string `gorm:"type:text" json:"description,omitempty"`  // Описание

	// Сумма и валюта
	AmountUSD   float64 `gorm:"type:decimal(10,2);not null" json:"amount_usd"` // Сумма в USD
	StarsAmount int     `gorm:"not null" json:"stars_amount"`                  // Количество Stars

	// Статус и провайдер
	Status   InvoiceStatus   `gorm:"type:varchar(20);not null;default:'created'" json:"status"` // Текущий статус
	Provider InvoiceProvider `gorm:"type:varchar(20);not null" json:"provider"`                 // Провайдер платежа

	// Ссылки и данные
	InvoiceURL string `gorm:"type:text;not null" json:"invoice_url"` // Ссылка на оплату
	Payload    string `gorm:"type:text" json:"payload,omitempty"`    // Данные для deep link (start parameter)
	Metadata   string `gorm:"type:jsonb" json:"metadata,omitempty"`  // Дополнительные данные (JSON)

	// Временные метки
	CreatedAt time.Time  `gorm:"autoCreateTime" json:"created_at"` // Дата создания
	UpdatedAt time.Time  `gorm:"autoUpdateTime" json:"updated_at"` // Дата обновления
	ExpiresAt time.Time  `gorm:"not null" json:"expires_at"`       // Срок действия
	PaidAt    *time.Time `json:"paid_at,omitempty"`                // Дата оплаты

	// Связи
	User    User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Payment Payment `gorm:"foreignKey:InvoiceID" json:"payment,omitempty"`
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
		"basic":      "📱 Basic",
		"pro":        "🚀 Pro",
		"enterprise": "🏢 Enterprise",
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
