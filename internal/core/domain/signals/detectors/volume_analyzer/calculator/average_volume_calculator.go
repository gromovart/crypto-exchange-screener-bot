// internal/core/domain/signals/detectors/volume_analyzer/calculator/average_volume_calculator.go
package calculator

import (
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/common"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	"time"
)

// AverageVolumeCalculator калькулятор среднего объема
type AverageVolumeCalculator struct {
	config         common.AnalyzerConfig
	confidenceCalc *ConfidenceCalculator
}

// NewAverageVolumeCalculator создает новый калькулятор среднего объема
func NewAverageVolumeCalculator(config common.AnalyzerConfig) *AverageVolumeCalculator {
	return &AverageVolumeCalculator{
		config:         config,
		confidenceCalc: NewConfidenceCalculator(config),
	}
}

// Calculate вычисляет сигнал на основе среднего объема
func (c *AverageVolumeCalculator) Calculate(data []redis_storage.PriceData) *analysis.Signal {
	if len(data) == 0 {
		return nil
	}

	var totalVolume float64
	validPoints := 0

	for _, point := range data {
		if point.Volume24h > 0 {
			totalVolume += point.Volume24h
			validPoints++
		}
	}

	if validPoints == 0 {
		return nil
	}

	avgVolume := totalVolume / float64(validPoints)
	minVolume := c.getMinVolume()

	if avgVolume < minVolume {
		return nil
	}

	confidence := c.confidenceCalc.CalculateVolumeConfidence(avgVolume, minVolume)

	if confidence < c.config.MinConfidence {
		return nil
	}

	return &analysis.Signal{
		Symbol:        data[0].Symbol,
		Type:          "high_volume",
		Direction:     "neutral",
		ChangePercent: 0,
		Confidence:    confidence,
		DataPoints:    validPoints,
		StartPrice:    data[0].Price,
		EndPrice:      data[len(data)-1].Price,
		Timestamp:     time.Now(),
		Metadata: analysis.Metadata{
			Strategy: "average_volume",
			Tags:     []string{"volume", "liquidity", "high_volume"},
			Indicators: map[string]float64{
				"avg_volume":   avgVolume,
				"min_volume":   minVolume,
				"volume_ratio": avgVolume / minVolume,
			},
		},
	}
}

// UpdateConfig обновляет конфигурацию
func (c *AverageVolumeCalculator) UpdateConfig(config common.AnalyzerConfig) {
	c.config = config
	c.confidenceCalc.UpdateConfig(config)
}

// GetName возвращает имя калькулятора
func (c *AverageVolumeCalculator) GetName() string {
	return "average_volume_calculator"
}

// getMinVolume возвращает минимальный объем из конфигурации
func (c *AverageVolumeCalculator) getMinVolume() float64 {
	if minVolume, ok := c.config.CustomSettings["min_volume"].(float64); ok {
		return minVolume
	}
	return 100000.0 // значение по умолчанию
}
