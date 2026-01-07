// internal/core/domain/signals/detectors/volume_analyzer/calculator/volume_spike_calculator.go
package calculator

import (
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/common"
	"crypto-exchange-screener-bot/internal/types"
	"time"
)

// VolumeSpikeCalculator калькулятор всплесков объема
type VolumeSpikeCalculator struct {
	config         common.AnalyzerConfig
	confidenceCalc *ConfidenceCalculator
}

// NewVolumeSpikeCalculator создает новый калькулятор всплесков объема
func NewVolumeSpikeCalculator(config common.AnalyzerConfig) *VolumeSpikeCalculator {
	return &VolumeSpikeCalculator{
		config:         config,
		confidenceCalc: NewConfidenceCalculator(config),
	}
}

// Calculate вычисляет сигнал всплеска объема
func (c *VolumeSpikeCalculator) Calculate(data []types.PriceData) *analysis.Signal {
	if len(data) < 2 {
		return nil
	}

	// Находим максимальный объем
	maxVolume := 0.0
	maxIndex := 0

	for i, point := range data {
		if point.Volume24h > maxVolume {
			maxVolume = point.Volume24h
			maxIndex = i
		}
	}

	// Вычисляем средний объем без максимума
	var totalWithoutMax float64
	countWithoutMax := 0
	for i, point := range data {
		if i != maxIndex && point.Volume24h > 0 {
			totalWithoutMax += point.Volume24h
			countWithoutMax++
		}
	}

	if countWithoutMax == 0 {
		return nil
	}

	avgWithoutMax := totalWithoutMax / float64(countWithoutMax)

	// Получаем множитель для всплеска из конфигурации
	spikeMultiplier := c.getSpikeMultiplier()

	// Проверяем, является ли это скачком
	if avgWithoutMax > 0 && maxVolume > avgWithoutMax*spikeMultiplier {
		spikeRatio := maxVolume / avgWithoutMax
		confidence := c.confidenceCalc.CalculateSpikeConfidence(spikeRatio)

		if confidence < c.config.MinConfidence {
			return nil
		}

		return &analysis.Signal{
			Symbol:        data[0].Symbol,
			Type:          "volume_spike",
			Direction:     "neutral",
			ChangePercent: 0,
			Confidence:    confidence,
			DataPoints:    len(data),
			StartPrice:    data[0].Price,
			EndPrice:      data[len(data)-1].Price,
			Timestamp:     time.Now(),
			Metadata: analysis.Metadata{
				Strategy: "volume_spike_detection",
				Tags:     []string{"volume", "spike", "unusual"},
				Indicators: map[string]float64{
					"spike_volume":     maxVolume,
					"avg_volume":       avgWithoutMax,
					"spike_ratio":      spikeRatio,
					"spike_position":   float64(maxIndex),
					"spike_multiplier": spikeMultiplier,
				},
			},
		}
	}

	return nil
}

// UpdateConfig обновляет конфигурацию
func (c *VolumeSpikeCalculator) UpdateConfig(config common.AnalyzerConfig) {
	c.config = config
	c.confidenceCalc.UpdateConfig(config)
}

// GetName возвращает имя калькулятора
func (c *VolumeSpikeCalculator) GetName() string {
	return "volume_spike_calculator"
}

// getSpikeMultiplier возвращает множитель для всплеска из конфигурации
func (c *VolumeSpikeCalculator) getSpikeMultiplier() float64 {
	if multiplier, ok := c.config.CustomSettings["spike_multiplier"].(float64); ok {
		return multiplier
	}
	return 3.0 // значение по умолчанию
}
