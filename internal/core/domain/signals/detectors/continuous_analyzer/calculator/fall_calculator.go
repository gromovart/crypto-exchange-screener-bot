// internal/core/domain/signals/detectors/continuous_analyzer/calculator/fall_calculator.go
package calculator

import (
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/common"
	"crypto-exchange-screener-bot/internal/types"
	"math"
	"time"
)

// FallCalculator калькулятор непрерывных последовательностей падения
type FallCalculator struct {
	config         common.AnalyzerConfig
	confidenceCalc *ConfidenceCalculator
}

// NewFallCalculator создает новый калькулятор падения
func NewFallCalculator(config common.AnalyzerConfig) *FallCalculator {
	return &FallCalculator{
		config:         config,
		confidenceCalc: NewConfidenceCalculator(config),
	}
}

// Calculate вычисляет сигналы непрерывного падения
func (c *FallCalculator) Calculate(data []types.PriceData, minPoints int) []analysis.Signal {
	var signals []analysis.Signal
	symbol := data[0].Symbol

	maxGapRatio := c.getMaxGapRatio()

	for i := 0; i <= len(data)-minPoints; i++ {
		continuous := true
		totalChange := 0.0
		startPrice := data[i].Price

		// Проверяем minPoints подряд
		for j := i; j < i+minPoints-1; j++ {
			if j+1 >= len(data) {
				continuous = false
				break
			}

			prevPrice := data[j].Price
			currPrice := data[j+1].Price

			// Проверяем gap
			gap := c.calculateGap(prevPrice, currPrice)
			if gap > maxGapRatio {
				continuous = false
				break
			}

			change := ((currPrice - prevPrice) / prevPrice) * 100
			if change >= 0 { // Не падение
				continuous = false
				break
			}
			totalChange += change
		}

		if continuous {
			endPrice := data[i+minPoints-1].Price
			totalChangePercent := ((endPrice - startPrice) / startPrice) * 100

			signal := c.createSignal(symbol, "down", totalChangePercent, minPoints, i, i+minPoints-1)
			signals = append(signals, signal)
		}
	}

	return signals
}

// createSignal создает сигнал непрерывного падения
func (c *FallCalculator) createSignal(symbol, direction string, change float64, points, startIdx, endIdx int) analysis.Signal {
	confidence := c.confidenceCalc.Calculate(points, math.Abs(change))

	return analysis.Signal{
		Symbol:        symbol,
		Type:          "continuous_" + direction,
		Direction:     direction,
		ChangePercent: change,
		Confidence:    confidence,
		DataPoints:    points,
		StartPrice:    0, // Заполняется вызывающим кодом
		EndPrice:      0, // Заполняется вызывающим кодом
		Timestamp:     time.Now(),
		Metadata: analysis.Metadata{
			Strategy:       "continuous_analyzer",
			Tags:           []string{"continuous", direction, "fall", "sequence"},
			IsContinuous:   true,
			ContinuousFrom: startIdx,
			ContinuousTo:   endIdx,
			Indicators: map[string]float64{
				"continuous_points": float64(points),
				"total_change":      change,
				"avg_change":        change / float64(points),
			},
		},
	}
}

// getMaxGapRatio возвращает максимальный допустимый gap
func (c *FallCalculator) getMaxGapRatio() float64 {
	if c.config.CustomSettings == nil {
		return 0.3
	}

	val := c.config.CustomSettings["max_gap_ratio"]
	if val == nil {
		return 0.3
	}

	switch v := val.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case float32:
		return float64(v)
	default:
		return 0.3
	}
}

// calculateGap вычисляет относительный разрыв между ценами
func (c *FallCalculator) calculateGap(prev, curr float64) float64 {
	if prev == 0 {
		return 0
	}
	diff := math.Abs(curr - prev)
	return diff / prev
}

// UpdateConfig обновляет конфигурацию
func (c *FallCalculator) UpdateConfig(config common.AnalyzerConfig) {
	c.config = config
	c.confidenceCalc.UpdateConfig(config)
}

// GetName возвращает имя калькулятора
func (c *FallCalculator) GetName() string {
	return "fall_calculator"
}
