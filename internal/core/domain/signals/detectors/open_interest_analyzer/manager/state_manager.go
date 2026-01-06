// internal/core/domain/signals/detectors/open_interest_analyzer/manager/state_manager.go
package manager

import (
	"sync"
	"time"

	"crypto-exchange-screener-bot/internal/types"
)

// OIState - состояние OI для символа
type OIState struct {
	Symbol       string
	CurrentOI    float64
	AvgOI        float64
	OIRatio      float64 // отношение текущего OI к среднему
	PriceChange  float64 // изменение цены в %
	OIChange     float64 // изменение OI в %
	LastUpdated  time.Time
	History      []OIDataPoint // история OI для анализа дивергенций
	ExtremeFlag  bool          // флаг экстремального значения
	ExtremeSince time.Time     // время с начала экстремального состояния
}

// OIDataPoint - точка данных OI
type OIDataPoint struct {
	Timestamp   time.Time
	Price       float64
	OI          float64
	Volume      float64
	PriceChange float64 // изменение цены с предыдущей точки
	OIChange    float64 // изменение OI с предыдущей точки
}

// OIConfigForState - конфигурация для управления состояниями
type OIConfigForState struct {
	ExtremeOIThreshold float64
}

// StateManager - менеджер состояний OI для символов
type StateManager struct {
	states map[string]*OIState // symbol -> state
	mu     sync.RWMutex
}

// NewStateManager создает новый менеджер состояний
func NewStateManager() *StateManager {
	return &StateManager{
		states: make(map[string]*OIState),
	}
}

// GetState возвращает состояние для символа
func (sm *StateManager) GetState(symbol string) *OIState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if state, exists := sm.states[symbol]; exists {
		return state
	}
	return nil
}

// UpdateState обновляет состояние для символа на основе новых данных
func (sm *StateManager) UpdateState(symbol string, data []types.PriceData, config OIConfigForState) *OIState {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	state, exists := sm.states[symbol]
	if !exists {
		state = &OIState{
			Symbol:  symbol,
			History: make([]OIDataPoint, 0),
		}
		sm.states[symbol] = state
	}

	// Обновляем текущие значения
	if len(data) > 0 {
		latest := data[len(data)-1]
		previous := state.CurrentOI

		state.CurrentOI = latest.OpenInterest
		state.LastUpdated = time.Now()

		// Рассчитываем изменение OI
		if previous > 0 {
			state.OIChange = ((state.CurrentOI - previous) / previous) * 100
		}

		// Рассчитываем изменение цены
		if len(data) >= 2 {
			firstPrice := data[0].Price
			lastPrice := latest.Price
			state.PriceChange = ((lastPrice - firstPrice) / firstPrice) * 100
		}

		// Добавляем точку в историю
		dataPoint := OIDataPoint{
			Timestamp: latest.Timestamp,
			Price:     latest.Price,
			OI:        latest.OpenInterest,
			Volume:    latest.Volume24h,
		}

		// Рассчитываем изменения с предыдущей точки
		if len(state.History) > 0 {
			prev := state.History[len(state.History)-1]
			dataPoint.PriceChange = ((dataPoint.Price - prev.Price) / prev.Price) * 100
			if prev.OI > 0 {
				dataPoint.OIChange = ((dataPoint.OI - prev.OI) / prev.OI) * 100
			}
		}

		state.History = append(state.History, dataPoint)

		// Ограничиваем размер истории (сохраняем последние 100 точек)
		if len(state.History) > 100 {
			state.History = state.History[len(state.History)-100:]
		}

		// Рассчитываем среднее OI
		state.AvgOI = sm.calculateAvgOI(state.History)
		if state.AvgOI > 0 {
			state.OIRatio = state.CurrentOI / state.AvgOI
		}

		// Проверяем на экстремальное значение
		state.ExtremeFlag = state.OIRatio > config.ExtremeOIThreshold
		if state.ExtremeFlag && state.ExtremeSince.IsZero() {
			state.ExtremeSince = time.Now()
		} else if !state.ExtremeFlag {
			state.ExtremeSince = time.Time{}
		}
	}

	return state
}

// GetOIHistory возвращает историю OI для символа
func (sm *StateManager) GetOIHistory(symbol string, maxPoints int) []OIDataPoint {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if state, exists := sm.states[symbol]; exists {
		if maxPoints <= 0 || maxPoints >= len(state.History) {
			return append([]OIDataPoint{}, state.History...)
		}
		return append([]OIDataPoint{}, state.History[len(state.History)-maxPoints:]...)
	}
	return nil
}

// CalculateExtremeDuration рассчитывает длительность экстремального состояния
func (sm *StateManager) CalculateExtremeDuration(symbol string) time.Duration {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if state, exists := sm.states[symbol]; exists && state.ExtremeFlag && !state.ExtremeSince.IsZero() {
		return time.Since(state.ExtremeSince)
	}
	return 0
}

// Cleanup очищает старые состояния
func (sm *StateManager) Cleanup(maxAge time.Duration) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	for symbol, state := range sm.states {
		if state.LastUpdated.Before(cutoff) {
			delete(sm.states, symbol)
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

	sm.states = make(map[string]*OIState)
}

// calculateAvgOI рассчитывает среднее значение OI из истории
func (sm *StateManager) calculateAvgOI(history []OIDataPoint) float64 {
	if len(history) == 0 {
		return 0
	}

	var sum float64
	var count int
	for _, point := range history {
		if point.OI > 0 {
			sum += point.OI
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

// GetStats возвращает статистику по состояниям
func (sm *StateManager) GetStats() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_symbols"] = len(sm.states)

	extremeCount := 0
	for _, state := range sm.states {
		if state.ExtremeFlag {
			extremeCount++
		}
	}
	stats["extreme_symbols"] = extremeCount
	stats["extreme_percentage"] = 0.0
	if len(sm.states) > 0 {
		stats["extreme_percentage"] = float64(extremeCount) / float64(len(sm.states)) * 100
	}

	return stats
}
