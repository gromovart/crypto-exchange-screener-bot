// internal/infrastructure/persistence/postgres/models/plan.go
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

// План подписки
type Plan struct {
	ID                int                    `db:"id" json:"id"`
	Name              string                 `db:"name" json:"name"`
	Code              string                 `db:"code" json:"code"`
	Description       string                 `db:"description" json:"description"`
	PriceMonthly      float64                `db:"price_monthly" json:"price_monthly"`
	PriceYearly       float64                `db:"price_yearly" json:"price_yearly"`
	StarsPriceMonthly int                    `db:"stars_price_monthly" json:"stars_price_monthly"` // Цена в Stars (месяц)
	StarsPriceYearly  int                    `db:"stars_price_yearly" json:"stars_price_yearly"`   // Цена в Stars (год)
	MaxSymbols        int                    `db:"max_symbols" json:"max_symbols"`                 // -1 = неограниченно
	MaxSignalsPerDay  int                    `db:"max_signals_per_day" json:"max_signals_per_day"` // -1 = неограниченно
	Features          map[string]interface{} `db:"features" json:"features"`
	IsActive          bool                   `db:"is_active" json:"is_active"`
	CreatedAt         time.Time              `db:"created_at" json:"created_at"`
}

// Лимиты по тарифам
type PlanLimits struct {
	MaxSymbols       int
	MaxSignalsPerDay int
	MaxAPIRequests   int
	Features         map[string]interface{}
}

// GetStarsPrice возвращает цену в Stars в зависимости от периода
func (p *Plan) GetStarsPrice(isYearly bool) int {
	if isYearly && p.StarsPriceYearly > 0 {
		return p.StarsPriceYearly
	}
	return p.StarsPriceMonthly
}

// GetUSDPrice возвращает цену в USD в зависимости от периода
func (p *Plan) GetUSDPrice(isYearly bool) float64 {
	if isYearly && p.PriceYearly > 0 {
		return p.PriceYearly
	}
	return p.PriceMonthly
}

// GetMaxAPIRequests возвращает количество API запросов для плана
func (p *Plan) GetMaxAPIRequests() int {
	// Извлекаем из features или используем значения по умолчанию
	if p.Features != nil {
		if apiRequests, ok := p.Features["max_api_requests"].(float64); ok {
			return int(apiRequests)
		}
	}

	// Значения по умолчанию по планам
	switch p.Code {
	case PlanFree:
		return 100
	case PlanBasic:
		return 1000
	case PlanPro:
		return 5000
	case PlanEnterprise:
		return -1 // неограниченно
	default:
		return 100
	}
}
