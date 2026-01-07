// internal/core/domain/signals/detectors/volume_analyzer/calculator/confidence_calculator.go
package calculator

import (
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/common"
	"math"
)

// ConfidenceCalculator калькулятор уверенности для анализа объема
type ConfidenceCalculator struct {
	config common.AnalyzerConfig
}

// NewConfidenceCalculator создает новый калькулятор уверенности
func NewConfidenceCalculator(config common.AnalyzerConfig) *ConfidenceCalculator {
	return &ConfidenceCalculator{
		config: config,
	}
}

// CalculateVolumeConfidence вычисляет уверенность на основе объема
func (c *ConfidenceCalculator) CalculateVolumeConfidence(volume, minVolume float64) float64 {
	if volume < minVolume {
		return 0
	}

	// Нормализация уверенности на основе объема
	ratio := volume / minVolume

	if ratio > 10 {
		return 90.0
	} else if ratio > 5 {
		return 70.0
	} else if ratio > 2 {
		return 50.0
	}
	return 30.0
}

// CalculateSpikeConfidence вычисляет уверенность для всплеска объема
func (c *ConfidenceCalculator) CalculateSpikeConfidence(spikeRatio float64) float64 {
	// Ограничиваем уверенность до 90%
	return math.Min(spikeRatio*15, 90)
}

// CalculateCorrelationConfidence вычисляет уверенность на основе корреляции
func (c *ConfidenceCalculator) CalculateCorrelationConfidence(correlation float64) float64 {
	if correlation > 80 {
		return 90.0
	} else if correlation > 60 {
		return 70.0
	} else if correlation > 40 {
		return 50.0
	} else if correlation > 20 {
		return 30.0
	}
	return 10.0
}

// UpdateConfig обновляет конфигурацию
func (c *ConfidenceCalculator) UpdateConfig(config common.AnalyzerConfig) {
	c.config = config
}

// GetName возвращает имя калькулятора
func (c *ConfidenceCalculator) GetName() string {
	return "confidence_calculator"
}
