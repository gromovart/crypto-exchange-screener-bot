// internal/analysis/analyzers/types.go
package analyzers

import (
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/types"
	"sync"
	"time"
)

// Analyzer - интерфейс анализатора
type Analyzer interface {
	Name() string
	Version() string
	Supports(symbol string) bool
	Analyze(data []types.PriceData, config AnalyzerConfig) ([]analysis.Signal, error)
	GetConfig() AnalyzerConfig
	GetStats() AnalyzerStats
}

// AnalyzerConfig - конфигурация анализатора
type AnalyzerConfig struct {
	Enabled        bool                   `json:"enabled"`
	Weight         float64                `json:"weight"`          // вес анализатора в общем результате
	MinConfidence  float64                `json:"min_confidence"`  // минимальная уверенность
	MinDataPoints  int                    `json:"min_data_points"` // минимальное количество точек
	CustomSettings map[string]interface{} `json:"custom_settings"`
}

// AnalyzerStats - статистика анализатора
type AnalyzerStats struct {
	TotalCalls   int           `json:"total_calls"`
	SuccessCount int           `json:"success_count"`
	ErrorCount   int           `json:"error_count"`
	TotalTime    time.Duration `json:"total_time"`
	AverageTime  time.Duration `json:"average_time"`
	LastCallTime time.Time     `json:"last_call_time"`
}

// Signal - структура сигнала анализа
type Signal struct {
	ID            string    `json:"id"`
	Symbol        string    `json:"symbol"`
	Type          string    `json:"type"`           // "growth", "fall", "breakout", "volume_spike"
	Direction     string    `json:"direction"`      // "up", "down"
	ChangePercent float64   `json:"change_percent"` // процент изменения
	Period        int       `json:"period"`         // период в минутах
	Confidence    float64   `json:"confidence"`     // уверенность 0-100
	DataPoints    int       `json:"data_points"`    // количество точек данных
	StartPrice    float64   `json:"start_price"`
	EndPrice      float64   `json:"end_price"`
	Volume        float64   `json:"volume"`
	Timestamp     time.Time `json:"timestamp"`
	Metadata      Metadata  `json:"metadata"`
}

// Metadata - метаданные сигнала
type Metadata struct {
	Strategy     string             `json:"strategy"`
	Tags         []string           `json:"tags"`
	Indicators   map[string]float64 `json:"indicators"`
	IsContinuous bool               `json:"is_continuous"`
	Patterns     []string           `json:"patterns"`
}

// AnalysisRequest - запрос на анализ
type AnalysisRequest struct {
	Symbol         string        `json:"symbol"`
	Period         time.Duration `json:"period"`
	Strategies     []string      `json:"strategies"`
	IncludeHistory bool          `json:"include_history"`
}

// AnalysisResult - результат анализа
type AnalysisResult struct {
	Symbol    string        `json:"symbol"`
	Signals   []Signal      `json:"signals"`
	Timestamp time.Time     `json:"timestamp"`
	Duration  time.Duration `json:"duration"`
}

// ============== НОВЫЕ ТИПЫ ДЛЯ СЧЕТЧИКА СИГНАЛОВ ==============

// CounterSignalType - тип сигнала для счетчика
type CounterSignalType string

const (
	CounterTypeGrowth CounterSignalType = "growth"
	CounterTypeFall   CounterSignalType = "fall"
)

// CounterPeriod - период анализа для счетчика
type CounterPeriod string

const (
	Period5Min   CounterPeriod = "5m"
	Period15Min  CounterPeriod = "15m"
	Period30Min  CounterPeriod = "30m"
	Period1Hour  CounterPeriod = "1h"
	Period4Hours CounterPeriod = "4h"
	Period1Day   CounterPeriod = "1d"
)

// CounterSettings - настройки счетчика
type CounterSettings struct {
	BasePeriodMinutes int           `json:"base_period_minutes"` // Базовый период в минутах (по умолчанию 1)
	SelectedPeriod    CounterPeriod `json:"selected_period"`     // Выбранный период анализа
	TrackGrowth       bool          `json:"track_growth"`        // Отслеживать рост
	TrackFall         bool          `json:"track_fall"`          // Отслеживать падение
	ChartProvider     string        `json:"chart_provider"`      // Провайдер графиков (coinglass/tradingview)
	NotifyOnSignal    bool          `json:"notify_on_signal"`    // Уведомлять при каждом сигнале
}

// SignalCounter - счетчик сигналов для символа
type SignalCounter struct {
	Symbol          string          `json:"symbol"`
	SelectedPeriod  CounterPeriod   `json:"selected_period"`
	BasePeriodCount int             `json:"base_period_count"` // Количество обработанных базовых периодов
	SignalCount     int             `json:"signal_count"`      // Общее количество сигналов в текущем периоде
	GrowthCount     int             `json:"growth_count"`      // Количество сигналов роста
	FallCount       int             `json:"fall_count"`        // Количество сигналов падения
	PeriodStartTime time.Time       `json:"period_start_time"` // Начало текущего периода
	PeriodEndTime   time.Time       `json:"period_end_time"`   // Конец текущего периода
	LastSignalTime  time.Time       `json:"last_signal_time"`  // Время последнего сигнала
	Settings        CounterSettings `json:"settings"`          // Настройки счетчика
}

// CounterNotification - уведомление счетчика
type CounterNotification struct {
	Symbol          string            `json:"symbol"`
	SignalType      CounterSignalType `json:"signal_type"`
	CurrentCount    int               `json:"current_count"`
	TotalCount      int               `json:"total_count"` // Общее количество сигналов в периоде
	Period          CounterPeriod     `json:"period"`
	PeriodStartTime time.Time         `json:"period_start_time"`
	PeriodEndTime   time.Time         `json:"period_end_time"`
	Timestamp       time.Time         `json:"timestamp"`
	MaxSignals      int               `json:"max_signals"` // Максимальное количество сигналов для периода
	Percentage      float64           `json:"percentage"`  // Процент заполнения (0-100)
	ChangePercent   float64           `json:"change_percent"`
}

// internalCounter - внутренняя структура счетчика с мьютексом
type internalCounter struct {
	SignalCounter
	mu sync.RWMutex
}
