// internal/core/domain/signals/detectors/growth_analyzer/manager/state_manager.go
package manager

import (
	calculator "crypto-exchange-screener-bot/internal/core/domain/signals/detectors/growth_analyzer/calculator"
	"crypto-exchange-screener-bot/internal/types"
	"sync"
	"time"
)

// StateManager - менеджер состояния анализатора роста
type StateManager struct {
	config     calculator.CalculatorConfig
	stats      GrowthStats
	mu         sync.RWMutex
	startTime  time.Time
	symbolData map[string]*symbolGrowthData
}

// GrowthStats - статистика анализатора роста
type GrowthStats struct {
	TotalCalls            int           `json:"total_calls"`
	SuccessCount          int           `json:"success_count"`
	ErrorCount            int           `json:"error_count"`
	TotalTime             time.Duration `json:"total_time"`
	AverageTime           time.Duration `json:"average_time"`
	LastCallTime          time.Time     `json:"last_call_time"`
	TotalGrowthSignals    int           `json:"total_growth_signals"`
	AverageGrowthPercent  float64       `json:"average_growth_percent"`
	MaxGrowthPercent      float64       `json:"max_growth_percent"`
	ContinuousGrowthCount int           `json:"continuous_growth_count"`
}

// symbolGrowthData - данные по росту для символа
type symbolGrowthData struct {
	LastSignalTime    time.Time
	SignalCount       int
	GrowthSignals     []GrowthAnalysisResult
	LastGrowthPercent float64
	LastConfidence    float64
}

// GrowthAnalysisResult - результат анализа роста
type GrowthAnalysisResult struct {
	Symbol        string            `json:"symbol"`
	GrowthPercent float64           `json:"growth_percent"`
	SignalType    string            `json:"signal_type"`
	Confidence    float64           `json:"confidence"`
	IsContinuous  bool              `json:"is_continuous"`
	TrendStrength float64           `json:"trend_strength"`
	Volatility    float64           `json:"volatility"`
	DataPoints    int               `json:"data_points"`
	StartPrice    float64           `json:"start_price"`
	EndPrice      float64           `json:"end_price"`
	Timestamp     time.Time         `json:"timestamp"`
	RawData       []types.PriceData `json:"-"` // Используем types.PriceData
}

// AnalyzerConfigWrapper - обертка для конфигурации (чтобы избежать импорта detectors)
type AnalyzerConfigWrapper struct {
	Enabled        bool
	Weight         float64
	MinConfidence  float64
	MinDataPoints  int
	CustomSettings map[string]interface{}
}

// AnalyzerStatsWrapper - обертка для статистики
type AnalyzerStatsWrapper struct {
	TotalCalls   int
	SuccessCount int
	ErrorCount   int
	TotalTime    time.Duration
	AverageTime  time.Duration
	LastCallTime time.Time
}

// NewStateManager - создает новый менеджер состояния
func NewStateManager(config AnalyzerConfigWrapper) *StateManager {
	calcConfig := convertToCalculatorConfig(config)

	return &StateManager{
		config:     calcConfig,
		stats:      GrowthStats{},
		startTime:  time.Now(),
		symbolData: make(map[string]*symbolGrowthData),
	}
}

// convertToCalculatorConfig - преобразует AnalyzerConfig в CalculatorConfig
func convertToCalculatorConfig(config AnalyzerConfigWrapper) calculator.CalculatorConfig {
	custom := config.CustomSettings
	return calculator.CalculatorConfig{
		MinGrowthPercent:      getFloatSetting(custom, "min_growth_percent", 2.0),
		ContinuityThreshold:   getFloatSetting(custom, "continuity_threshold", 0.7),
		AccelerationThreshold: getFloatSetting(custom, "acceleration_threshold", 0.5),
		VolumeWeight:          getFloatSetting(custom, "volume_weight", 0.2),
		TrendStrengthWeight:   getFloatSetting(custom, "trend_strength_weight", 0.4),
		VolatilityWeight:      getFloatSetting(custom, "volatility_weight", 0.2),
	}
}

// getFloatSetting - вспомогательная функция
func getFloatSetting(settings map[string]interface{}, key string, defaultValue float64) float64 {
	if value, ok := settings[key]; ok {
		if floatValue, ok := value.(float64); ok {
			return floatValue
		}
	}
	return defaultValue
}

// UpdateStats - обновляет статистику
func (sm *StateManager) UpdateStats(duration time.Duration, success bool, growthPercent float64, isContinuous bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.stats.TotalCalls++
	sm.stats.TotalTime += duration
	sm.stats.LastCallTime = time.Now()

	if success {
		sm.stats.SuccessCount++
		sm.stats.TotalGrowthSignals++

		sm.stats.AverageGrowthPercent = (sm.stats.AverageGrowthPercent*float64(sm.stats.TotalGrowthSignals-1) + growthPercent) / float64(sm.stats.TotalGrowthSignals)

		if growthPercent > sm.stats.MaxGrowthPercent {
			sm.stats.MaxGrowthPercent = growthPercent
		}

		if isContinuous {
			sm.stats.ContinuousGrowthCount++
		}
	} else {
		sm.stats.ErrorCount++
	}

	if sm.stats.TotalCalls > 0 {
		sm.stats.AverageTime = time.Duration(
			int64(sm.stats.TotalTime) / int64(sm.stats.TotalCalls),
		)
	}
}

// GetStats - возвращает статистику
func (sm *StateManager) GetStats() GrowthStats {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.stats
}

// GetAnalyzerStats - возвращает статистику в формате AnalyzerStats
func (sm *StateManager) GetAnalyzerStats() AnalyzerStatsWrapper {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return AnalyzerStatsWrapper{
		TotalCalls:   sm.stats.TotalCalls,
		SuccessCount: sm.stats.SuccessCount,
		ErrorCount:   sm.stats.ErrorCount,
		TotalTime:    sm.stats.TotalTime,
		AverageTime:  sm.stats.AverageTime,
		LastCallTime: sm.stats.LastCallTime,
	}
}

// GetConfig - возвращает конфигурацию
func (sm *StateManager) GetConfig() calculator.CalculatorConfig {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.config
}

// UpdateSymbolData - обновляет данные по символу
func (sm *StateManager) UpdateSymbolData(symbol string, result GrowthAnalysisResult) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	data, exists := sm.symbolData[symbol]
	if !exists {
		data = &symbolGrowthData{
			GrowthSignals: make([]GrowthAnalysisResult, 0),
		}
		sm.symbolData[symbol] = data
	}

	if len(data.GrowthSignals) >= 100 {
		data.GrowthSignals = data.GrowthSignals[1:]
	}
	data.GrowthSignals = append(data.GrowthSignals, result)

	data.LastSignalTime = result.Timestamp
	data.SignalCount++
	data.LastGrowthPercent = result.GrowthPercent
	data.LastConfidence = result.Confidence
}

// GetSymbolData - возвращает данные по символу
func (sm *StateManager) GetSymbolData(symbol string) (*symbolGrowthData, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	data, exists := sm.symbolData[symbol]
	return data, exists
}

// GetRecentSignals - возвращает последние сигналы для символа
func (sm *StateManager) GetRecentSignals(symbol string, limit int) []GrowthAnalysisResult {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	data, exists := sm.symbolData[symbol]
	if !exists || len(data.GrowthSignals) == 0 {
		return []GrowthAnalysisResult{}
	}

	startIdx := len(data.GrowthSignals) - limit
	if startIdx < 0 {
		startIdx = 0
	}

	result := make([]GrowthAnalysisResult, len(data.GrowthSignals)-startIdx)
	copy(result, data.GrowthSignals[startIdx:])

	return result
}

// ShouldProcessSymbol - проверяет, нужно ли обрабатывать символ (rate limiting)
func (sm *StateManager) ShouldProcessSymbol(symbol string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	data, exists := sm.symbolData[symbol]
	if !exists {
		return true
	}

	minInterval := time.Second * 30
	return time.Since(data.LastSignalTime) > minInterval
}

// GetSymbolStats - возвращает статистику по символу
func (sm *StateManager) GetSymbolStats(symbol string) map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	data, exists := sm.symbolData[symbol]
	if !exists {
		return map[string]interface{}{
			"signal_count":        0,
			"last_signal_time":    nil,
			"last_growth_percent": 0.0,
			"last_confidence":     0.0,
		}
	}

	return map[string]interface{}{
		"signal_count":        data.SignalCount,
		"last_signal_time":    data.LastSignalTime,
		"last_growth_percent": data.LastGrowthPercent,
		"last_confidence":     data.LastConfidence,
		"recent_signals":      len(data.GrowthSignals),
	}
}

// Reset - сбрасывает состояние
func (sm *StateManager) Reset() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.stats = GrowthStats{}
	sm.startTime = time.Now()
	sm.symbolData = make(map[string]*symbolGrowthData)
}

// GetUptime - возвращает время работы
func (sm *StateManager) GetUptime() time.Duration {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return time.Since(sm.startTime)
}
