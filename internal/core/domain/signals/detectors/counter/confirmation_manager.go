// internal/core/domain/signals/detectors/counter/confirmation_manager.go
package counter

import (
	"sync"
	"time"
)

// PeriodCounter - счетчик подтверждений для символа и периода
type PeriodCounter struct {
	Symbol        string
	Period        string // "5m", "15m", "30m", "1h", "4h", "1d"
	Confirmations int    // текущее количество подтверждений
	Required      int    // сколько нужно подтверждений
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

// GetRequiredConfirmations возвращает необходимое количество подтверждений для периода
func GetRequiredConfirmations(period string) int {
	switch period {
	case "5m":
		return 3 // 3 подтверждения за 5 минут
	case "15m":
		return 3 // 3 подтверждения за 15 минут
	case "30m":
		return 4 // 4 подтверждения за 30 минут
	case "1h":
		return 6 // 6 подтверждений за 1 час
	case "4h":
		return 8 // 8 подтверждений за 4 часа
	case "1d":
		return 12 // 12 подтверждений за 1 день
	default:
		return 3 // по умолчанию
	}
}

// AddConfirmation добавляет подтверждение для символа и периода
// Возвращает true, если достигнуто необходимое количество подтверждений
func (cm *ConfirmationManager) AddConfirmation(symbol, period string) (bool, int) {
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
			Required:   GetRequiredConfirmations(period),
			LastUpdate: now,
			LastReset:  now,
		}
		cm.counters[key] = counter
	}

	// Проверяем, не нужно ли сбросить счетчик
	// (если прошло больше времени чем период)
	if shouldResetCounter(counter, now) {
		counter.Confirmations = 0
		counter.LastReset = now
	}

	// Добавляем подтверждение
	counter.Confirmations++
	counter.LastUpdate = now

	// Проверяем, достигнуто ли необходимое количество
	if counter.Confirmations >= counter.Required {
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
func (cm *ConfirmationManager) GetProgress(symbol, period string) (int, int) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	key := symbol + ":" + period
	if counter, exists := cm.counters[key]; exists {
		return counter.Confirmations, counter.Required
	}

	return 0, GetRequiredConfirmations(period)
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
