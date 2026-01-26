// internal/core/domain/signals/detectors/volume_analyzer/manager/state_manager.go
package manager

import (
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/common"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/volume_analyzer"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	"sync"
	"time"
)

// StateManager управляет состоянием VolumeAnalyzer
type StateManager struct {
	mu            sync.RWMutex
	stats         volume_analyzer.VolumeStats
	volumeMetrics map[string]*volume_analyzer.VolumeMetrics // по символам
	recentSignals map[string][]time.Time                    // история сигналов по типам
	config        common.AnalyzerConfig
}

// NewStateManager создает новый менеджер состояния
func NewStateManager(config common.AnalyzerConfig) *StateManager {
	return &StateManager{
		stats:         volume_analyzer.VolumeStats{},
		volumeMetrics: make(map[string]*volume_analyzer.VolumeMetrics),
		recentSignals: make(map[string][]time.Time),
		config:        config,
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
func (sm *StateManager) GetStats() volume_analyzer.VolumeStats {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.stats
}

// UpdateVolumeMetrics обновляет метрики объема для символа
func (sm *StateManager) UpdateVolumeMetrics(symbol string, metrics *volume_analyzer.VolumeMetrics) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.volumeMetrics[symbol] = metrics

	// Обновляем агрегированные метрики
	if metrics.IsSpike {
		sm.stats.SpikeDetections++
	}
	if metrics.IsConfirmation {
		sm.stats.ConfirmationDetections++
	}
	if metrics.IsDivergence {
		sm.stats.DivergenceDetections++
	}
}

// GetVolumeMetrics возвращает метрики объема для символа
func (sm *StateManager) GetVolumeMetrics(symbol string) (*volume_analyzer.VolumeMetrics, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	metrics, exists := sm.volumeMetrics[symbol]
	return metrics, exists
}

// RecordSignal записывает сигнал в историю
func (sm *StateManager) RecordSignal(signalType string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	key := signalType

	// Добавляем время сигнала
	sm.recentSignals[key] = append(sm.recentSignals[key], now)

	// Очищаем старые записи (старше 24 часов)
	cutoffTime := now.Add(-24 * time.Hour)
	validSignals := make([]time.Time, 0)

	for _, ts := range sm.recentSignals[key] {
		if ts.After(cutoffTime) {
			validSignals = append(validSignals, ts)
		}
	}

	sm.recentSignals[key] = validSignals
}

// GetSignalFrequency возвращает частоту сигналов за период
func (sm *StateManager) GetSignalFrequency(signalType string, period time.Duration) int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	now := time.Now()
	cutoffTime := now.Add(-period)
	count := 0

	if signals, exists := sm.recentSignals[signalType]; exists {
		for _, ts := range signals {
			if ts.After(cutoffTime) {
				count++
			}
		}
	}

	return count
}

// ShouldProcessSymbol проверяет, нужно ли обрабатывать символ
func (sm *StateManager) ShouldProcessSymbol(symbol string, data []redis_storage.PriceData) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Проверяем минимальное количество точек данных
	if len(data) < sm.config.MinDataPoints {
		return false
	}

	// Проверяем rate limiting по символу
	symbolKey := "symbol_" + symbol
	recentCalls := sm.GetSignalFrequency(symbolKey, time.Minute)
	if recentCalls >= 5 { // не более 5 вызовов в минуту на символ
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
	for symbol, metrics := range sm.volumeMetrics {
		if time.Unix(metrics.Timestamp, 0).Before(cutoffTime) {
			delete(sm.volumeMetrics, symbol)
		}
	}

	// Очищаем устаревшие сигналы
	for signalType, signals := range sm.recentSignals {
		validSignals := make([]time.Time, 0)
		for _, ts := range signals {
			if ts.After(cutoffTime) {
				validSignals = append(validSignals, ts)
			}
		}
		sm.recentSignals[signalType] = validSignals
	}
}
