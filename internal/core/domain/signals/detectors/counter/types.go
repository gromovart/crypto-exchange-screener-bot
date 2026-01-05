// internal/core/domain/signals/detectors/counter/types.go
package counter

import (
	"sync"
	"time"
)

// ==================== ТИПЫ СЧЕТЧИКА ====================

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

// SignalCounter - счетчик сигналов для символа (публичная структура)
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

// internalCounter - внутренняя структура счетчика с мьютексом
type internalCounter struct {
	SignalCounter
	mu sync.RWMutex
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

// ==================== МЕТОДЫ ДЛЯ CounterPeriod ====================

// GetMinutes возвращает количество минут для периода
func (cp CounterPeriod) GetMinutes() int {
	switch cp {
	case Period5Min:
		return 5
	case Period15Min:
		return 15
	case Period30Min:
		return 30
	case Period1Hour:
		return 60
	case Period4Hours:
		return 240
	case Period1Day:
		return 1440
	default:
		return 15 // По умолчанию 15 минут
	}
}

// GetDuration возвращает длительность периода как time.Duration
func (cp CounterPeriod) GetDuration() time.Duration {
	return time.Duration(cp.GetMinutes()) * time.Minute
}

// ToString возвращает строковое представление периода
func (cp CounterPeriod) ToString() string {
	switch cp {
	case Period5Min:
		return "5 минут"
	case Period15Min:
		return "15 минут"
	case Period30Min:
		return "30 минут"
	case Period1Hour:
		return "1 час"
	case Period4Hours:
		return "4 часа"
	case Period1Day:
		return "1 день"
	default:
		return "15 минут"
	}
}

// ==================== МЕТОДЫ ДЛЯ internalCounter ====================

// Lock блокирует счетчик для записи
func (c *internalCounter) Lock() {
	c.mu.Lock()
}

// Unlock разблокирует счетчик для записи
func (c *internalCounter) Unlock() {
	c.mu.Unlock()
}

// RLock блокирует счетчик для чтения
func (c *internalCounter) RLock() {
	c.mu.RLock()
}

// RUnlock разблокирует счетчика для чтения
func (c *internalCounter) RUnlock() {
	c.mu.RUnlock()
}
