// internal/core/domain/signals/detectors/counter/confirmation/confirmation.go
package confirmation

import (
	"sync"
	"time"
)

// PeriodCounter - счетчик подтверждений для символа и периода
type PeriodCounter struct {
	Symbol        string
	Period        string // "5m", "15m", "30m", "1h", "4h", "1d"
	Direction     string // "growth" или "fall" - текущее направление
	Confirmations int    // текущее количество подтверждений ПОДРЯД в одном направлении
	LastUpdate    time.Time
	LastReset     time.Time
}

// ConfirmationManager - менеджер подтверждений для CounterAnalyzer
type ConfirmationManager struct {
	counters map[string]*PeriodCounter // ключ: "symbol:period"
	mu       sync.RWMutex
}

// NewConfirmationManager создает новый менеджер подтверждений
func NewConfirmationManager() *ConfirmationManager {
	return &ConfirmationManager{
		counters: make(map[string]*PeriodCounter),
	}
}

// GetRequiredConfirmations возвращает визуальное необходимое количество подтверждений
// ВСЕГДА 6 для единой шкалы прогресса (100%)
func GetRequiredConfirmations(period string) int {
	return 6 // Визуальная цель всегда 6
}

// GetSignalThreshold возвращает порог отправки сигнала (каждые 3 подтверждения)
func GetSignalThreshold() int {
	return 3
}

// AddConfirmation добавляет подтверждение для символа и периода
// direction: "growth" (рост) или "fall" (падение)
// Возвращает true, если достигнут порог сигнала (3, 6, 9... подтверждений ПОДРЯД)
func (cm *ConfirmationManager) AddConfirmation(symbol, period, direction string) (bool, int) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	key := symbol + ":" + period
	now := time.Now()

	// Получаем или создаем счетчик
	counter, exists := cm.counters[key]
	if !exists {
		counter = &PeriodCounter{
			Symbol:     symbol,
			Period:     period,
			Direction:  direction,
			LastUpdate: now,
			LastReset:  now,
		}
		cm.counters[key] = counter
	}

	// Проверяем, не нужно ли сбросить счетчик (если прошло больше времени чем период)
	if shouldResetCounter(counter, now) {
		counter.Confirmations = 0
		counter.Direction = direction
		counter.LastReset = now
	}

	// Если направление изменилось → сбрасываем счетчик
	if counter.Direction != direction {
		counter.Confirmations = 0
		counter.Direction = direction
	}

	// Добавляем подтверждение
	counter.Confirmations++
	counter.LastUpdate = now

	// Проверяем, достигнут ли порог сигнала (каждые 3 подтверждения)
	if counter.Confirmations >= GetSignalThreshold() && counter.Confirmations%GetSignalThreshold() == 0 {
		return true, counter.Confirmations
	}

	return false, counter.Confirmations
}

// Reset сбрасывает счетчик для символа и периода
func (cm *ConfirmationManager) Reset(symbol, period string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	key := symbol + ":" + period
	if counter, exists := cm.counters[key]; exists {
		counter.Confirmations = 0
		counter.LastReset = time.Now()
	}
}

// GetProgress возвращает текущий прогресс для символа и периода
func (cm *ConfirmationManager) GetProgress(symbol, period string) (int, string) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	key := symbol + ":" + period
	if counter, exists := cm.counters[key]; exists {
		return counter.Confirmations, counter.Direction
	}

	return 0, ""
}

// shouldResetCounter проверяет, нужно ли сбросить счетчик
func shouldResetCounter(counter *PeriodCounter, now time.Time) bool {
	// Определяем длительность периода
	var periodDuration time.Duration
	switch counter.Period {
	case "5m":
		periodDuration = 5 * time.Minute
	case "15m":
		periodDuration = 15 * time.Minute
	case "30m":
		periodDuration = 30 * time.Minute
	case "1h":
		periodDuration = 1 * time.Hour
	case "4h":
		periodDuration = 4 * time.Hour
	case "1d":
		periodDuration = 24 * time.Hour
	default:
		periodDuration = 15 * time.Minute
	}

	// Если с последнего сброса прошло больше времени чем период
	return now.Sub(counter.LastReset) > periodDuration
}

// Cleanup удаляет старые счетчики
func (cm *ConfirmationManager) Cleanup(maxAge time.Duration) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	now := time.Now()
	for key, counter := range cm.counters {
		if now.Sub(counter.LastUpdate) > maxAge {
			delete(cm.counters, key)
		}
	}
}

// GetDirection возвращает текущее направление для символа и периода
func (cm *ConfirmationManager) GetDirection(symbol, period string) string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	key := symbol + ":" + period
	if counter, exists := cm.counters[key]; exists {
		return counter.Direction
	}
	return ""
}
