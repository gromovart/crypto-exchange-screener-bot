// internal/core/domain/signals/detectors/continuous_analyzer/analyzer.go
package continuous_analyzer

import (
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/common"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/continuous_analyzer/calculator"
	"crypto-exchange-screener-bot/internal/types"
	"fmt"
	"sync"
	"time"
)

// ContinuousAnalyzer - анализатор непрерывности (модульный)
type ContinuousAnalyzer struct {
	config         common.AnalyzerConfig
	stats          common.AnalyzerStats
	mu             sync.RWMutex
	growthCalc     *calculator.GrowthCalculator
	fallCalc       *calculator.FallCalculator
	sequenceCalc   *calculator.SequenceFinder
	confidenceCalc *calculator.ConfidenceCalculator
}

// NewContinuousAnalyzer создает новый анализатор непрерывности
func NewContinuousAnalyzer(config common.AnalyzerConfig) *ContinuousAnalyzer {
	return &ContinuousAnalyzer{
		config:         config,
		stats:          common.AnalyzerStats{},
		growthCalc:     calculator.NewGrowthCalculator(config),
		fallCalc:       calculator.NewFallCalculator(config),
		sequenceCalc:   calculator.NewSequenceFinder(config),
		confidenceCalc: calculator.NewConfidenceCalculator(config),
	}
}

func (a *ContinuousAnalyzer) Name() string {
	return "continuous_analyzer"
}

func (a *ContinuousAnalyzer) Version() string {
	return "1.1.0" // Обновляем версию для модульной версии
}

func (a *ContinuousAnalyzer) Supports(symbol string) bool {
	return true
}

func (a *ContinuousAnalyzer) Analyze(data []types.PriceData, config common.AnalyzerConfig) ([]analysis.Signal, error) {
	startTime := time.Now()

	minPoints := a.getMinContinuousPoints()
	if len(data) < minPoints {
		a.updateStats(time.Since(startTime), false)
		return nil, fmt.Errorf("недостаточно точек данных: нужно минимум %d, получено %d", minPoints, len(data))
	}

	// Обновляем конфигурацию если передана новая
	if !isZeroConfig(config) {
		a.updateConfig(config)
	}

	var signals []analysis.Signal

	// 1. Ищем непрерывные последовательности роста
	if growthSignals := a.growthCalc.Calculate(data, minPoints); len(growthSignals) > 0 {
		signals = append(signals, growthSignals...)
	}

	// 2. Ищем непрерывные последовательности падения
	if fallSignals := a.fallCalc.Calculate(data, minPoints); len(fallSignals) > 0 {
		signals = append(signals, fallSignals...)
	}

	// 3. Если не нашли конкретных последовательностей, ищем общую тенденцию
	if len(signals) == 0 {
		if bestSequence := a.sequenceCalc.FindBestSequence(data); bestSequence.Length >= minPoints {
			signal := a.createSequenceSignal(data, bestSequence)
			signals = append(signals, signal)
		}
	}

	a.updateStats(time.Since(startTime), len(signals) > 0)
	return signals, nil
}

// createSequenceSignal создает сигнал на основе найденной последовательности
func (a *ContinuousAnalyzer) createSequenceSignal(data []types.PriceData, sequence calculator.SequenceInfo) analysis.Signal {
	symbol := data[0].Symbol
	startIdx := sequence.StartIdx
	endIdx := sequence.StartIdx + sequence.Length - 1

	startPrice := data[startIdx].Price
	endPrice := data[endIdx].Price
	change := ((endPrice - startPrice) / startPrice) * 100

	confidence := a.confidenceCalc.Calculate(sequence.Length, change)

	return analysis.Signal{
		Symbol:        symbol,
		Type:          "continuous_trend",
		Direction:     sequence.Direction,
		ChangePercent: change,
		Confidence:    confidence,
		DataPoints:    sequence.Length,
		StartPrice:    startPrice,
		EndPrice:      endPrice,
		Timestamp:     time.Now(),
		Metadata: analysis.Metadata{
			Strategy:       "continuous_analyzer",
			Tags:           []string{"continuous", "trend", sequence.Direction},
			IsContinuous:   true,
			ContinuousFrom: startIdx,
			ContinuousTo:   endIdx,
			Indicators: map[string]float64{
				"trend_length":     float64(sequence.Length),
				"avg_gap":          sequence.AvgGap,
				"avg_change":       sequence.AvgChange,
				"continuity_score": confidence,
			},
		},
	}
}

// getMinContinuousPoints получает минимальное количество непрерывных точек
func (a *ContinuousAnalyzer) getMinContinuousPoints() int {
	if a.config.CustomSettings == nil {
		return 3
	}

	val := a.config.CustomSettings["min_continuous_points"]
	if val == nil {
		return 3
	}

	switch v := val.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case float32:
		return int(v)
	default:
		return 3
	}
}

// updateStats обновляет статистику
func (a *ContinuousAnalyzer) updateStats(duration time.Duration, success bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.stats.TotalCalls++
	a.stats.TotalTime += duration
	a.stats.LastCallTime = time.Now()

	if success {
		a.stats.SuccessCount++
	} else {
		a.stats.ErrorCount++
	}

	if a.stats.TotalCalls > 0 {
		a.stats.AverageTime = time.Duration(
			int64(a.stats.TotalTime) / int64(a.stats.TotalCalls),
		)
	}
}

// updateConfig обновляет конфигурацию
func (a *ContinuousAnalyzer) updateConfig(config common.AnalyzerConfig) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.config = config
	// Обновляем конфигурацию калькуляторов
	a.growthCalc.UpdateConfig(config)
	a.fallCalc.UpdateConfig(config)
	a.sequenceCalc.UpdateConfig(config)
	a.confidenceCalc.UpdateConfig(config)
}

// GetConfig возвращает конфигурацию
func (a *ContinuousAnalyzer) GetConfig() common.AnalyzerConfig {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config
}

// GetStats возвращает статистику
func (a *ContinuousAnalyzer) GetStats() common.AnalyzerStats {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.stats
}

// isZeroConfig проверяет, является ли конфигурация нулевой
func isZeroConfig(config common.AnalyzerConfig) bool {
	return !config.Enabled &&
		config.Weight == 0 &&
		config.MinConfidence == 0 &&
		config.MinDataPoints == 0 &&
		len(config.CustomSettings) == 0
}

// DefaultContinuousConfig возвращает конфигурацию по умолчанию
func DefaultContinuousConfig() common.AnalyzerConfig {
	return common.AnalyzerConfig{
		Enabled:       true,
		Weight:        0.8,
		MinConfidence: 70.0,
		MinDataPoints: 4,
		CustomSettings: map[string]interface{}{
			"min_continuous_points": 3,
			"max_gap_ratio":         0.3,
			"require_confirmation":  true,
		},
	}
}
