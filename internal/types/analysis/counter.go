// internal/types/analysis/counter.go
package analysis

import (
	"crypto_exchange_screener_bot/internal/types/common"
	"sync"
	"time"
)

// CounterSignalType - тип сигнала для счетчика
type CounterSignalType string

const (
	CounterTypeGrowth CounterSignalType = "growth"
	CounterTypeFall   CounterSignalType = "fall"
)

// CounterPeriod - период анализа для счетчика
type CounterPeriod string

const (
	Period5m  CounterPeriod = "5m"
	Period15m CounterPeriod = "15m"
	Period30m CounterPeriod = "30m"
	Period1h  CounterPeriod = "1h"
	Period4h  CounterPeriod = "4h"
	Period1d  CounterPeriod = "1d"
)

// CounterSettings - настройки счетчика
type CounterSettings struct {
	BasePeriodMinutes int           `json:"base_period_minutes"`
	SelectedPeriod    CounterPeriod `json:"selected_period"`
	TrackGrowth       bool          `json:"track_growth"`
	TrackFall         bool          `json:"track_fall"`
	ChartProvider     string        `json:"chart_provider"`
	NotifyOnSignal    bool          `json:"notify_on_signal"`
}

// SignalCounter - счетчик сигналов (без мьютекса)
type SignalCounter struct {
	Symbol          common.Symbol   `json:"symbol"`
	SelectedPeriod  CounterPeriod   `json:"selected_period"`
	BasePeriodCount int             `json:"base_period_count"`
	SignalCount     int             `json:"signal_count"`
	GrowthCount     int             `json:"growth_count"`
	FallCount       int             `json:"fall_count"`
	PeriodStartTime time.Time       `json:"period_start_time"`
	PeriodEndTime   time.Time       `json:"period_end_time"`
	LastSignalTime  time.Time       `json:"last_signal_time"`
	Settings        CounterSettings `json:"settings"`
}

// CounterNotification - уведомление счетчика
type CounterNotification struct {
	Symbol          common.Symbol     `json:"symbol"`
	SignalType      CounterSignalType `json:"signal_type"`
	CurrentCount    int               `json:"current_count"`
	TotalCount      int               `json:"total_count"`
	Period          CounterPeriod     `json:"period"`
	PeriodStartTime time.Time         `json:"period_start_time"`
	PeriodEndTime   time.Time         `json:"period_end_time"`
	Timestamp       time.Time         `json:"timestamp"`
	MaxSignals      int               `json:"max_signals"`
	Percentage      float64           `json:"percentage"`
	ChangePercent   float64           `json:"change_percent"`
}

// CounterConfig - конфигурация счетчика
type CounterConfig struct {
	Enabled               bool                  `json:"enabled"`
	BasePeriodMinutes     int                   `json:"base_period_minutes"`
	AnalysisPeriod        CounterPeriod         `json:"analysis_period"`
	MaxSignalsPerPeriod   map[CounterPeriod]int `json:"max_signals_per_period"`
	TrackGrowth           bool                  `json:"track_growth"`
	TrackFall             bool                  `json:"track_fall"`
	NotificationThreshold int                   `json:"notification_threshold"`
	ChartProvider         string                `json:"chart_provider"`
}

// InternalCounter - внутренний счетчик с мьютексом
type InternalCounter struct {
	SignalCounter
	mu sync.RWMutex
}

// Lock блокирует счетчик для записи
func (c *InternalCounter) Lock() {
	c.mu.Lock()
}

// Unlock разблокирует счетчик для записи
func (c *InternalCounter) Unlock() {
	c.mu.Unlock()
}

// RLock блокирует счетчик для чтения
func (c *InternalCounter) RLock() {
	c.mu.RLock()
}

// RUnlock разблокирует счетчик для чтения
func (c *InternalCounter) RUnlock() {
	c.mu.RUnlock()
}

// GetMinutes возвращает количество минут для периода
func (cp CounterPeriod) GetMinutes() int {
	switch cp {
	case Period5m:
		return 5
	case Period15m:
		return 15
	case Period30m:
		return 30
	case Period1h:
		return 60
	case Period4h:
		return 240
	case Period1d:
		return 1440
	default:
		return 15 // По умолчанию 15 минут
	}
}

// GetDuration возвращает длительность периода как time.Duration
func (cp CounterPeriod) GetDuration() time.Duration {
	return time.Duration(cp.GetMinutes()) * time.Minute
}

// GetMaxSignals возвращает максимальное количество сигналов для периода
func (cp CounterPeriod) GetMaxSignals(basePeriodMinutes int) int {
	if basePeriodMinutes <= 0 {
		basePeriodMinutes = 1 // По умолчанию 1 минута
	}

	// Выбранный период / базовый период = максимальное количество сигналов
	maxSignals := cp.GetMinutes() / basePeriodMinutes

	// Ограничиваем 5-15 сигналами согласно требованиям
	if maxSignals < 5 {
		return 5
	}
	if maxSignals > 15 {
		return 15
	}
	return maxSignals
}

// ToString возвращает строковое представление периода
func (cp CounterPeriod) ToString() string {
	switch cp {
	case Period5m:
		return "5 минут"
	case Period15m:
		return "15 минут"
	case Period30m:
		return "30 минут"
	case Period1h:
		return "1 час"
	case Period4h:
		return "4 часа"
	case Period1d:
		return "1 день"
	default:
		return "15 минут"
	}
}

// GetChartProvider возвращает провайдера графиков для типа сигнала
func (cst CounterSignalType) GetChartProvider() string {
	// В реальной реализации можно получить из настроек
	// Пока возвращаем пустую строку, будет использоваться дефолтный
	return ""
}
