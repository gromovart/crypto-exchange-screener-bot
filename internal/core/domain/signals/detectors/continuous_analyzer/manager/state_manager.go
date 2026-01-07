// internal/core/domain/signals/detectors/continuous_analyzer/manager/state_manager.go
package manager

import (
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/common"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/continuous_analyzer"
	"crypto-exchange-screener-bot/internal/types"
	"sync"
	"time"
)

// StateManager управляет состоянием ContinuousAnalyzer
type StateManager struct {
	mu              sync.RWMutex
	stats           common.AnalyzerStats
	sequenceMetrics map[string]*continuous_analyzer.SequenceMetrics // по символам
	recentSequences map[string][]time.Time                          // история последовательностей
	config          common.AnalyzerConfig
}

// NewStateManager создает новый менеджер состояния
func NewStateManager(config common.AnalyzerConfig) *StateManager {
	return &StateManager{
		stats:           common.AnalyzerStats{},
		sequenceMetrics: make(map[string]*continuous_analyzer.SequenceMetrics),
		recentSequences: make(map[string][]time.Time),
		config:          config,
	}
}

// UpdateStats обновляет статистику анализатора
func (sm *StateManager) UpdateStats(duration time.Duration, success bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.stats.TotalCalls++
	sm.stats.TotalTime += duration
	sm.stats.LastCallTime = time.Now()

	if success {
		sm.stats.SuccessCount++
	} else {
		sm.stats.ErrorCount++
	}

	if sm.stats.TotalCalls > 0 {
		sm.stats.AverageTime = time.Duration(
			int64(sm.stats.TotalTime) / int64(sm.stats.TotalCalls),
		)
	}
}

// GetStats возвращает статистику
func (sm *StateManager) GetStats() common.AnalyzerStats {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.stats
}

// UpdateSequenceMetrics обновляет метрики последовательности для символа
func (sm *StateManager) UpdateSequenceMetrics(symbol string, metrics *continuous_analyzer.SequenceMetrics) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.sequenceMetrics[symbol] = metrics
	sm.recentSequences[symbol] = append(sm.recentSequences[symbol], time.Now())
}

// GetSequenceMetrics возвращает метрики последовательности для символа
func (sm *StateManager) GetSequenceMetrics(symbol string) (*continuous_analyzer.SequenceMetrics, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	metrics, exists := sm.sequenceMetrics[symbol]
	return metrics, exists
}

// GetRecentSequenceCount возвращает количество последовательностей за период
func (sm *StateManager) GetRecentSequenceCount(symbol string, period time.Duration) int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	now := time.Now()
	cutoffTime := now.Add(-period)
	count := 0

	if sequences, exists := sm.recentSequences[symbol]; exists {
		for _, ts := range sequences {
			if ts.After(cutoffTime) {
				count++
			}
		}
	}

	return count
}

// ShouldProcessSymbol проверяет, нужно ли обрабатывать символ
func (sm *StateManager) ShouldProcessSymbol(symbol string, data []types.PriceData) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Проверяем минимальное количество точек данных
	if len(data) < sm.config.MinDataPoints {
		return false
	}

	// Проверяем rate limiting по символу
	recentSequences := sm.GetRecentSequenceCount(symbol, time.Minute)
	if recentSequences >= 3 { // не более 3 последовательностей в минуту на символ
		return false
	}

	return true
}

// UpdateConfig обновляет конфигурацию
func (sm *StateManager) UpdateConfig(config common.AnalyzerConfig) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.config = config
}

// GetConfig возвращает конфигурацию
func (sm *StateManager) GetConfig() common.AnalyzerConfig {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.config
}

// Cleanup очищает устаревшие данные
func (sm *StateManager) Cleanup() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	cutoffTime := now.Add(-1 * time.Hour)

	// Очищаем устаревшие метрики
	for symbol, metrics := range sm.sequenceMetrics {
		if time.Unix(metrics.Timestamp, 0).Before(cutoffTime) {
			delete(sm.sequenceMetrics, symbol)
		}
	}

	// Очищаем устаревшие последовательности
	for symbol, sequences := range sm.recentSequences {
		validSequences := make([]time.Time, 0)
		for _, ts := range sequences {
			if ts.After(cutoffTime) {
				validSequences = append(validSequences, ts)
			}
		}
		sm.recentSequences[symbol] = validSequences
	}
}
