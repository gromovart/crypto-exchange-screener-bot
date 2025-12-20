// internal/types/growth.go
package types

import (
	"time"
)

// GrowthSignal - сигнал непрерывного роста/падения
type GrowthSignal struct {
	Symbol        string    `json:"symbol"`
	PeriodMinutes int       `json:"period_minutes"`
	GrowthPercent float64   `json:"growth_percent"`
	FallPercent   float64   `json:"fall_percent"`
	IsContinuous  bool      `json:"is_continuous"`
	DataPoints    int       `json:"data_points"`
	StartPrice    float64   `json:"start_price"`
	EndPrice      float64   `json:"end_price"`
	Direction     string    `json:"direction"`
	Confidence    float64   `json:"confidence"`
	Timestamp     time.Time `json:"timestamp"`
	Volume24h     float64   `json:"volume_24h,omitempty"` // Добавьте это поле
}

type PriceDataPoint struct {
	Price     float64   `json:"price"`
	Timestamp time.Time `json:"timestamp"`
	Volume    float64   `json:"volume"`
}

type GrowthAnalysis struct {
	Symbol        string           `json:"symbol"`
	Period        int              `json:"period"`
	DataPoints    []PriceDataPoint `json:"data_points"`
	IsGrowing     bool             `json:"is_growing"`
	IsFalling     bool             `json:"is_falling"`
	GrowthPercent float64          `json:"growth_percent"`
	FallPercent   float64          `json:"fall_percent"`
	MinPrice      float64          `json:"min_price"`
	MaxPrice      float64          `json:"max_price"`
	Volatility    float64          `json:"volatility"`
}
