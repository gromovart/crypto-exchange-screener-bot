// internal/core/domain/signals/detectors/fall_analyzer/manager/state_manager.go
package manager

import (
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	"math"
	"sync"
	"time"
)

// FallState - состояние анализа падений для символа
type FallState struct {
	Symbol       string
	CurrentPrice float64
	PriceChange  float64         // изменение цены в %
	LastFallTime time.Time       // время последнего падения
	FallCount    int             // счетчик падений
	IsFalling    bool            // флаг текущего падения
	FallSince    time.Time       // время с начала падения
	History      []FallDataPoint // история для анализа непрерывности
}

// FallDataPoint - точка данных для анализа падений
type FallDataPoint struct {
	Timestamp   time.Time
	Price       float64
	Volume      float64
	PriceChange float64 // изменение цены с предыдущей точки
	IsFall      bool    // является ли падением
}

// FallConfigForState - конфигурация для управления состояниями падений
type FallConfigForState struct {
	MinFall float64
}

// StateManager - менеджер состояний падений для символов
type StateManager struct {
	states map[string]*FallState // symbol -> state
	mu     sync.RWMutex
}

// NewStateManager создает новый менеджер состояний
func NewStateManager() *StateManager {
	return &StateManager{
		states: make(map[string]*FallState),
	}
}

// GetState возвращает состояние для символа
func (sm *StateManager) GetState(symbol string) *FallState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if state, exists := sm.states[symbol]; exists {
		return state
	}
	return nil
}

// UpdateState обновляет состояние для символа на основе новых данных
func (sm *StateManager) UpdateState(symbol string, data []redis_storage.PriceData, config FallConfigForState) *FallState {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	state, exists := sm.states[symbol]
	if !exists {
		state = &FallState{
			Symbol:  symbol,
			History: make([]FallDataPoint, 0),
		}
		sm.states[symbol] = state
	}

	// Обновляем текущие значения
	if len(data) > 0 {
		latest := data[len(data)-1]
		previous := state.CurrentPrice

		state.CurrentPrice = latest.Price

		// Рассчитываем изменение цены
		if previous > 0 {
			state.PriceChange = ((state.CurrentPrice - previous) / previous) * 100
		}

		// Определяем, является ли текущее движение падением
		isFalling := state.PriceChange < 0 && math.Abs(state.PriceChange) >= config.MinFall

		if isFalling && !state.IsFalling {
			// Начинается новое падение
			state.IsFalling = true
			state.FallSince = time.Now()
			state.FallCount++
			state.LastFallTime = time.Now()
		} else if !isFalling && state.IsFalling {
			// Падение закончилось
			state.IsFalling = false
			state.FallSince = time.Time{}
		} else if isFalling {
			// Падение продолжается
			state.LastFallTime = time.Now()
		}

		// Добавляем точку в историю
		dataPoint := FallDataPoint{
			Timestamp: latest.Timestamp,
			Price:     latest.Price,
			Volume:    latest.Volume24h,
			IsFall:    isFalling,
		}

		// Рассчитываем изменения с предыдущей точки
		if len(state.History) > 0 {
			prev := state.History[len(state.History)-1]
			dataPoint.PriceChange = ((dataPoint.Price - prev.Price) / prev.Price) * 100
		}

		state.History = append(state.History, dataPoint)

		// Ограничиваем размер истории (сохраняем последние 50 точек)
		if len(state.History) > 50 {
			state.History = state.History[len(state.History)-50:]
		}
	}

	return state
}

// GetFallHistory возвращает историю падений для символа
func (sm *StateManager) GetFallHistory(symbol string, maxPoints int) []FallDataPoint {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if state, exists := sm.states[symbol]; exists {
		if maxPoints <= 0 || maxPoints >= len(state.History) {
			return append([]FallDataPoint{}, state.History...)
		}
		return append([]FallDataPoint{}, state.History[len(state.History)-maxPoints:]...)
	}
	return nil
}

// CalculateFallDuration рассчитывает длительность текущего падения
func (sm *StateManager) CalculateFallDuration(symbol string) time.Duration {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if state, exists := sm.states[symbol]; exists && state.IsFalling && !state.FallSince.IsZero() {
		return time.Since(state.FallSince)
	}
	return 0
}

// GetFallCount возвращает количество падений для символа
func (sm *StateManager) GetFallCount(symbol string, period time.Duration) int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if state, exists := sm.states[symbol]; exists {
		if period <= 0 {
			return state.FallCount
		}

		// Подсчитываем падения за указанный период
		count := 0
		cutoff := time.Now().Add(-period)

		for _, point := range state.History {
			if point.IsFall && point.Timestamp.After(cutoff) {
				count++
			}
		}

		return count
	}
	return 0
}

// Cleanup очищает старые состояния
func (sm *StateManager) Cleanup(maxAge time.Duration) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	for symbol, state := range sm.states {
		if len(state.History) > 0 {
			lastUpdate := state.History[len(state.History)-1].Timestamp
			if lastUpdate.Before(cutoff) {
				delete(sm.states, symbol)
			}
		}
	}
}

// GetAllSymbols возвращает все символы с состояниями
func (sm *StateManager) GetAllSymbols() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	symbols := make([]string, 0, len(sm.states))
	for symbol := range sm.states {
		symbols = append(symbols, symbol)
	}
	return symbols
}

// Reset сбрасывает состояние для символа
func (sm *StateManager) Reset(symbol string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.states, symbol)
}

// ResetAll сбрасывает все состояния
func (sm *StateManager) ResetAll() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.states = make(map[string]*FallState)
}

// GetStats возвращает статистику по состояниям
func (sm *StateManager) GetStats() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_symbols"] = len(sm.states)

	fallingCount := 0
	totalFalls := 0

	for _, state := range sm.states {
		if state.IsFalling {
			fallingCount++
		}
		totalFalls += state.FallCount
	}

	stats["falling_symbols"] = fallingCount
	stats["total_falls"] = totalFalls
	stats["falling_percentage"] = 0.0

	if len(sm.states) > 0 {
		stats["falling_percentage"] = float64(fallingCount) / float64(len(sm.states)) * 100
		stats["avg_falls_per_symbol"] = float64(totalFalls) / float64(len(sm.states))
	}

	return stats
}
