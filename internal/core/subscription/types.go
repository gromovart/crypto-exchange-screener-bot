// internal/subscription/types.go
package subscription

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
	ID               int                    `json:"id"`
	Name             string                 `json:"name"`
	Code             string                 `json:"code"`
	Description      string                 `json:"description"`
	PriceMonthly     float64                `json:"price_monthly"`
	PriceYearly      float64                `json:"price_yearly"`
	MaxSymbols       int                    `json:"max_symbols"`         // -1 = неограниченно
	MaxSignalsPerDay int                    `json:"max_signals_per_day"` // -1 = неограниченно
	MaxAPIRequests   int                    `json:"max_api_requests"`    // -1 = неограниченно
	Features         map[string]interface{} `json:"features"`
	IsActive         bool                   `json:"is_active"`
	CreatedAt        time.Time              `json:"created_at"`
}

// Подписка пользователя
type UserSubscription struct {
	ID                   int                    `json:"id"`
	UserID               int                    `json:"user_id"`
	PlanID               int                    `json:"plan_id"`
	PlanName             string                 `json:"plan_name"`
	PlanCode             string                 `json:"plan_code"`
	StripeSubscriptionID string                 `json:"stripe_subscription_id"`
	Status               string                 `json:"status"`
	CurrentPeriodStart   time.Time              `json:"current_period_start"`
	CurrentPeriodEnd     time.Time              `json:"current_period_end"`
	CancelAtPeriodEnd    bool                   `json:"cancel_at_period_end"`
	Metadata             map[string]interface{} `json:"metadata"`
	CreatedAt            time.Time              `json:"created_at"`

	// Информация о пользователе (для нотификаций)
	TelegramID    int64  `json:"telegram_id,omitempty"`
	ChatID        string `json:"chat_id,omitempty"`
	UserFirstName string `json:"user_first_name,omitempty"`
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

// NotificationSettings настройки уведомлений
type NotificationSettings struct {
	Enabled             bool     `json:"enabled"`
	BeforeExpiration    []int    `json:"before_expiration"` // дни до истечения
	OnPaymentSuccess    bool     `json:"on_payment_success"`
	OnPaymentFailure    bool     `json:"on_payment_failure"`
	OnPlanChange        bool     `json:"on_plan_change"`
	OnTrialEnding       bool     `json:"on_trial_ending"`
	NotificationMethods []string `json:"notification_methods"` // telegram, email
}
