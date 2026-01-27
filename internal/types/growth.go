// internal/types/growth.go
package types

import (
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"time"
)

// GrowthSignal - сигнал непрерывного роста/падения
type GrowthSignal struct {
	Symbol        string             `json:"symbol"`
	PeriodMinutes int                `json:"period_minutes"`
	GrowthPercent float64            `json:"growth_percent"`
	FallPercent   float64            `json:"fall_percent"`
	IsContinuous  bool               `json:"is_continuous"`
	DataPoints    int                `json:"data_points"`
	StartPrice    float64            `json:"start_price"`
	EndPrice      float64            `json:"end_price"`
	Direction     string             `json:"direction"` // "growth" или "fall"
	Confidence    float64            `json:"confidence"`
	Timestamp     time.Time          `json:"timestamp"`
	Volume24h     float64            `json:"volume_24h,omitempty"`
	Type          string             `json:"type,omitempty"`     // Тип сигнала: "counter_growth", "counter_fall", "growth", "fall"
	Metadata      *analysis.Metadata `json:"metadata,omitempty"` // Метаданные анализа
}
