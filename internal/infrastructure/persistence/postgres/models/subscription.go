// internal/infrastructure/persistence/postgres/models/subscription.go
package models

import (
	"time"
)

// Типы подписок
const (
	PlanFree       = "free"
	PlanBasic      = "basic"
	PlanPro        = "pro"
	PlanEnterprise = "enterprise"
)

// Состояния подписки
const (
	StatusPending    = "pending"
	StatusActive     = "active"
	StatusTrialing   = "trialing"
	StatusPastDue    = "past_due"
	StatusCanceled   = "canceled"
	StatusExpired    = "expired"
	StatusIncomplete = "incomplete"
)

// План подписки
type Plan struct {
	ID               int                    `db:"id" json:"id"`
	Name             string                 `db:"name" json:"name"`
	Code             string                 `db:"code" json:"code"`
	Description      string                 `db:"description" json:"description"`
	PriceMonthly     float64                `db:"price_monthly" json:"price_monthly"`
	PriceYearly      float64                `db:"price_yearly" json:"price_yearly"`
	MaxSymbols       int                    `db:"max_symbols" json:"max_symbols"`                 // -1 = неограниченно
	MaxSignalsPerDay int                    `db:"max_signals_per_day" json:"max_signals_per_day"` // -1 = неограниченно
	Features         map[string]interface{} `db:"features" json:"features"`
	IsActive         bool                   `db:"is_active" json:"is_active"`
	CreatedAt        time.Time              `db:"created_at" json:"created_at"`
}

// Подписка пользователя
type UserSubscription struct {
	ID                   int                    `db:"id" json:"id"`
	UserID               int                    `db:"user_id" json:"user_id"`
	PlanID               int                    `db:"plan_id" json:"plan_id"`
	StripeSubscriptionID *string                `db:"stripe_subscription_id" json:"stripe_subscription_id,omitempty"` // NULLable
	Status               string                 `db:"status" json:"status"`
	CurrentPeriodStart   *time.Time             `db:"current_period_start" json:"current_period_start,omitempty"` // NULLable
	CurrentPeriodEnd     *time.Time             `db:"current_period_end" json:"current_period_end,omitempty"`     // NULLable
	CancelAtPeriodEnd    bool                   `db:"cancel_at_period_end" json:"cancel_at_period_end"`
	Metadata             map[string]interface{} `db:"metadata" json:"metadata"`
	CreatedAt            time.Time              `db:"created_at" json:"created_at"`
	UpdatedAt            time.Time              `db:"updated_at" json:"updated_at"`

	// Дополнительные поля (join или вычисляемые)
	PlanName      string `db:"-" json:"plan_name,omitempty"`
	PlanCode      string `db:"-" json:"plan_code,omitempty"`
	TelegramID    int64  `db:"-" json:"telegram_id,omitempty"`
	ChatID        string `db:"-" json:"chat_id,omitempty"`
	UserFirstName string `db:"-" json:"user_first_name,omitempty"`
}

// RevenueReport отчет по доходам
type RevenueReport struct {
	PeriodStart      time.Time          `json:"period_start"`
	PeriodEnd        time.Time          `json:"period_end"`
	TotalRevenue     float64            `json:"total_revenue"`
	NewSubscriptions int                `json:"new_subscriptions"`
	ARPU             float64            `json:"arpu"` // Average Revenue Per User
	MostPopularPlan  string             `json:"most_popular_plan"`
	MonthlyBreakdown []MonthlyBreakdown `json:"monthly_breakdown"`
}

// MonthlyBreakdown месячная разбивка
type MonthlyBreakdown struct {
	Month       time.Time `json:"month"`
	Revenue     float64   `json:"revenue"`
	Subscribers int       `json:"subscribers"`
}

// Лимиты по тарифам
type PlanLimits struct {
	MaxSymbols       int
	MaxSignalsPerDay int
	MaxAPIRequests   int
	Features         map[string]interface{}
}

// События подписки
type SubscriptionEvent struct {
	Type           string                 `json:"type"`
	UserID         int                    `json:"user_id"`
	SubscriptionID int                    `json:"subscription_id"`
	PlanCode       string                 `json:"plan_code"`
	OldPlanCode    string                 `json:"old_plan_code,omitempty"`
	Status         string                 `json:"status"`
	Timestamp      time.Time              `json:"timestamp"`
	Metadata       map[string]interface{} `json:"metadata"`
}
