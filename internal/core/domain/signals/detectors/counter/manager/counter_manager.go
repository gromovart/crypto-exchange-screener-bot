// internal/core/domain/signals/detectors/counter/manager/counter_manager.go
package manager

import (
	"sync"
	"time"
)

// CounterManager - менеджер счетчиков сигналов
type CounterManager struct {
	counters map[string]*counterSignalCounter
	mu       sync.RWMutex
}

// CounterSettings - настройки счетчика (локальная копия)
type CounterSettings struct {
	BasePeriodMinutes int
	SelectedPeriod    string // Изменено с CounterPeriod на string
	TrackGrowth       bool
	TrackFall         bool
	ChartProvider     string
	NotifyOnSignal    bool
}

// SignalCounter - счетчик сигналов для символа
type SignalCounter struct {
	Symbol          string
	SelectedPeriod  string // Изменено
	BasePeriodCount int
	SignalCount     int
	GrowthCount     int
	FallCount       int
	PeriodStartTime time.Time
	PeriodEndTime   time.Time
	LastSignalTime  time.Time
	Settings        CounterSettings
}

type counterSignalCounter struct {
	SignalCounter
	mu sync.RWMutex
}

// GetCounter возвращает счетчик для символа (экспортированный метод)
func (m *CounterManager) GetCounter(symbol string) (*counterSignalCounter, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, exists := m.counters[symbol]
	return c, exists
}

// getCounter внутренний метод для получения счетчика (не экспортируется)
func (m *CounterManager) getCounter(symbol string) (*counterSignalCounter, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, exists := m.counters[symbol]
	return c, exists
}

// Методы для периодов (временное решение)
func getPeriodMinutes(period string) int {
	switch period {
	case "5m":
		return 5
	case "15m":
		return 15
	case "30m":
		return 30
	case "1h":
		return 60
	case "4h":
		return 240
	case "1d":
		return 1440
	default:
		return 15
	}
}

func getPeriodDuration(period string) time.Duration {
	return time.Duration(getPeriodMinutes(period)) * time.Minute
}

func NewCounterManager() *CounterManager {
	return &CounterManager{
		counters: make(map[string]*counterSignalCounter),
	}
}

func (m *CounterManager) GetOrCreateCounter(symbol string, period string, basePeriodMinutes int) *counterSignalCounter {
	if c, exists := m.getCounter(symbol); exists {
		return c
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if c, exists := m.counters[symbol]; exists {
		return c
	}

	c := &counterSignalCounter{
		SignalCounter: SignalCounter{
			Symbol:          symbol,
			SelectedPeriod:  period,
			PeriodStartTime: time.Now(),
			PeriodEndTime:   time.Now().Add(getPeriodDuration(period)),
			Settings: CounterSettings{
				BasePeriodMinutes: basePeriodMinutes,
				SelectedPeriod:    period,
				TrackGrowth:       true,
				TrackFall:         true,
				ChartProvider:     "coinglass",
				NotifyOnSignal:    true,
			},
		},
	}
	m.counters[symbol] = c
	return c
}

func (m *CounterManager) GetAllCounters() map[string]SignalCounter {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]SignalCounter)
	for symbol, cnt := range m.counters {
		cnt.RLock()
		result[symbol] = cnt.SignalCounter
		cnt.RUnlock()
	}
	return result
}

func (m *CounterManager) GetCounterStats(symbol string) (SignalCounter, bool) {
	c, exists := m.GetCounter(symbol) // Используем экспортированный метод
	if !exists {
		return SignalCounter{}, false
	}

	c.RLock()
	defer c.RUnlock()
	return c.SignalCounter, true
}

func (m *CounterManager) UpdateCounterSettings(symbol string, settings CounterSettings) error {
	c, exists := m.GetCounter(symbol) // Используем экспортированный метод
	if !exists {
		return m.createCounter(symbol, settings)
	}

	c.Lock()
	c.Settings = settings
	c.Unlock()
	return nil
}

func (m *CounterManager) ResetAllCounters(newPeriod string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, cnt := range m.counters {
		cnt.Lock()
		cnt.BasePeriodCount = 0
		cnt.SignalCount = 0
		cnt.GrowthCount = 0
		cnt.FallCount = 0
		cnt.PeriodStartTime = time.Now()
		cnt.PeriodEndTime = time.Now().Add(getPeriodDuration(newPeriod))
		cnt.SelectedPeriod = newPeriod
		cnt.Settings.SelectedPeriod = newPeriod
		cnt.Unlock()
	}
}

func (m *CounterManager) ResetCounter(symbol string) bool {
	c, exists := m.GetCounter(symbol) // Используем экспортированный метод
	if !exists {
		return false
	}

	c.Lock()
	c.BasePeriodCount = 0
	c.SignalCount = 0
	c.GrowthCount = 0
	c.FallCount = 0
	c.PeriodStartTime = time.Now()
	c.PeriodEndTime = time.Now().Add(getPeriodDuration(c.SelectedPeriod))
	c.Unlock()
	return true
}

func (m *CounterManager) DeleteCounter(symbol string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.counters[symbol]; exists {
		delete(m.counters, symbol)
		return true
	}
	return false
}

func (m *CounterManager) createCounter(symbol string, settings CounterSettings) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.counters[symbol]; exists {
		return nil
	}

	m.counters[symbol] = &counterSignalCounter{
		SignalCounter: SignalCounter{
			Symbol:          symbol,
			SelectedPeriod:  settings.SelectedPeriod,
			PeriodStartTime: time.Now(),
			PeriodEndTime:   time.Now().Add(getPeriodDuration(settings.SelectedPeriod)),
			Settings:        settings,
		},
	}
	return nil
}

func GetPeriodMinutes(period string) int {
	switch period {
	case "5m":
		return 5
	case "15m":
		return 15
	case "30m":
		return 30
	case "1h":
		return 60
	case "4h":
		return 240
	case "1d":
		return 1440
	default:
		return 15
	}
}

func (c *counterSignalCounter) Lock()    { c.mu.Lock() }
func (c *counterSignalCounter) Unlock()  { c.mu.Unlock() }
func (c *counterSignalCounter) RLock()   { c.mu.RLock() }
func (c *counterSignalCounter) RUnlock() { c.mu.RUnlock() }
