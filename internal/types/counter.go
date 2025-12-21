// internal/types/counter.go
package types

import (
	"time"
)

// SignalCounter - структура счетчика сигналов (БЕЗ встроенного мьютекса!)
type SignalCounter struct {
	Symbol          string        `json:"symbol"`
	GrowthCount     int           `json:"growth_count"`
	FallCount       int           `json:"fall_count"`
	Period          CounterPeriod `json:"period"`
	PeriodStartTime time.Time     `json:"period_start_time"`
	LastGrowthTime  time.Time     `json:"last_growth_time"`
	LastFallTime    time.Time     `json:"last_fall_time"`
}

// CounterSignalType - тип сигнала для счетчика
type CounterSignalType string

const (
	CounterTypeGrowth CounterSignalType = "growth"
	CounterTypeFall   CounterSignalType = "fall"
)

// CounterPeriod - период для подсчета сигналов
type CounterPeriod string

// CounterConfig - конфигурация счетчика
type CounterConfig struct {
	Enabled               bool                  `json:"enabled"`
	BasePeriodMinutes     int                   `json:"base_period_minutes"` // Базовый период (1 минута)
	AnalysisPeriod        CounterPeriod         `json:"analysis_period"`
	MaxSignalsPerPeriod   map[CounterPeriod]int `json:"max_signals_per_period"`
	TrackGrowth           bool                  `json:"track_growth"`
	TrackFall             bool                  `json:"track_fall"`
	NotificationThreshold int                   `json:"notification_threshold"` // Порог для уведомлений
	ChartProvider         string                `json:"chart_provider"`
}

// CounterNotification - уведомление от счетчика
type CounterNotification struct {
	Symbol          string            `json:"symbol"`
	SignalType      CounterSignalType `json:"signal_type"`
	CurrentCount    int               `json:"current_count"`
	Period          CounterPeriod     `json:"period"`
	PeriodStartTime time.Time         `json:"period_start_time"`
	Timestamp       time.Time         `json:"timestamp"`
	MaxSignals      int               `json:"max_signals"`
	Percentage      float64           `json:"percentage"`
}
