// internal/types/analysis/common.go
package analysis

import (
	"crypto_exchange_screener_bot/internal/types/common"
	"time"
)

// SignalType - тип сигнала
type SignalType string

const (
	SignalTypeGrowth     SignalType = "growth"
	SignalTypeFall       SignalType = "fall"
	SignalTypeCounter    SignalType = "counter"
	SignalTypeVolume     SignalType = "volume"
	SignalTypeContinuous SignalType = "continuous"
	SignalTypeBreakout   SignalType = "breakout"
)

// TrendDirection - направление тренда
type TrendDirection string

const (
	TrendBullish  TrendDirection = "bullish"
	TrendBearish  TrendDirection = "bearish"
	TrendSideways TrendDirection = "sideways"
)

// SignalMetadata - метаданные сигнала
type SignalMetadata struct {
	Strategy       string             `json:"strategy"`
	Tags           []string           `json:"tags"`
	IsContinuous   bool               `json:"is_continuous"`
	ContinuousFrom int                `json:"continuous_from"`
	ContinuousTo   int                `json:"continuous_to"`
	Indicators     map[string]float64 `json:"indicators"`

	// Дополнительные поля для совместимости
	IsCounter      bool    `json:"is_counter,omitempty"`
	CounterType    string  `json:"counter_type,omitempty"`
	CurrentCount   int     `json:"current_count,omitempty"`
	MaxSignals     int     `json:"max_signals,omitempty"`
	PeriodProgress float64 `json:"period_progress,omitempty"`
}

// Signal - базовый сигнал
type Signal struct {
	ID            string           `json:"id"`
	Type          SignalType       `json:"type"`
	Symbol        common.Symbol    `json:"symbol"`
	Exchange      common.Exchange  `json:"exchange"`
	Timeframe     common.Timeframe `json:"timeframe"`
	Direction     TrendDirection   `json:"direction"`
	Confidence    float64          `json:"confidence"` // 0-1
	Strength      float64          `json:"strength"`   // 0-1
	ChangePercent float64          `json:"change_percent"`
	Period        int              `json:"period"` // в минутах
	StartPrice    float64          `json:"start_price"`
	EndPrice      float64          `json:"end_price"`
	Volume        float64          `json:"volume"`
	DataPoints    int              `json:"data_points"`
	Timestamp     time.Time        `json:"timestamp"`
	Metadata      SignalMetadata   `json:"metadata,omitempty"`
}

// TrendSignal сигнал тренда
type TrendSignal struct {
	Symbol        common.Symbol `json:"symbol"`
	Direction     string        `json:"direction"` // "growth" или "fall"
	ChangePercent float64       `json:"change_percent"`
	PeriodMinutes int           `json:"period_minutes"`
	Confidence    float64       `json:"confidence"`
	Timestamp     time.Time     `json:"timestamp"`
	DataPoints    int           `json:"data_points"`
}
