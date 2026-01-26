// internal/core/domain/signals/detectors/volume_analyzer/calculator/volume_price_confirmation_calculator.go
package calculator

import (
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/common"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	"math"
	"time"
)

// VolumePriceConfirmationCalculator калькулятор согласованности объема и цены
type VolumePriceConfirmationCalculator struct {
	config         common.AnalyzerConfig
	confidenceCalc *ConfidenceCalculator
}

// NewVolumePriceConfirmationCalculator создает новый калькулятор
func NewVolumePriceConfirmationCalculator(config common.AnalyzerConfig) *VolumePriceConfirmationCalculator {
	return &VolumePriceConfirmationCalculator{
		config:         config,
		confidenceCalc: NewConfidenceCalculator(config),
	}
}

// Calculate вычисляет сигнал согласованности объема и цены
func (c *VolumePriceConfirmationCalculator) Calculate(data []redis_storage.PriceData) *analysis.Signal {
	if len(data) < 2 {
		return nil
	}

	// Рассчитываем изменение цены
	priceChange := ((data[len(data)-1].Price - data[0].Price) / data[0].Price) * 100

	// Рассчитываем изменение объема
	var volumeChange float64
	if data[0].Volume24h > 0 {
		volumeChange = ((data[len(data)-1].Volume24h - data[0].Volume24h) / data[0].Volume24h) * 100
	} else {
		return nil
	}

	// Получаем порог из конфигурации
	confirmationThreshold := c.getConfirmationThreshold()

	// Проверяем согласованность
	if math.Abs(priceChange) < 0.1 || math.Abs(volumeChange) < confirmationThreshold {
		// Изменения слишком малы
		return nil
	}

	var signalType, direction string
	var confidence float64

	if priceChange > 0 && volumeChange > 0 {
		// Рост цены + рост объема = сильный бычий сигнал
		signalType = "volume_confirmation"
		direction = "up"
		confirmationStrength := math.Min(priceChange, volumeChange) / 2
		confidence = 50 + math.Min(confirmationStrength, 40) // 50-90%
	} else if priceChange < 0 && volumeChange > 0 {
		// Падение цены + рост объема = сильный медвежий сигнал
		signalType = "volume_confirmation"
		direction = "down"
		confirmationStrength := math.Min(math.Abs(priceChange), volumeChange) / 2
		confidence = 50 + math.Min(confirmationStrength, 40)
	} else if priceChange > 0 && volumeChange < -20 {
		// Рост цены + падение объема = бычья дивергенция (слабый сигнал)
		signalType = "volume_divergence"
		direction = "up"
		confidence = 30
	} else if priceChange < 0 && volumeChange < -20 {
		// Падение цены + падение объема = медвежья дивергенция (слабый сигнал)
		signalType = "volume_divergence"
		direction = "down"
		confidence = 30
	} else {
		// Нет значимой корреляции
		return nil
	}

	// Вычисляем корреляцию для уверенности
	correlation := c.calculateVolumePriceCorrelation(data)
	correlationConfidence := c.confidenceCalc.CalculateCorrelationConfidence(correlation)

	// Объединяем уверенности
	finalConfidence := (confidence + correlationConfidence) / 2

	if finalConfidence < c.config.MinConfidence {
		return nil
	}

	return &analysis.Signal{
		Symbol:        data[0].Symbol,
		Type:          signalType,
		Direction:     direction,
		ChangePercent: priceChange,
		Confidence:    finalConfidence,
		DataPoints:    len(data),
		StartPrice:    data[0].Price,
		EndPrice:      data[len(data)-1].Price,
		Timestamp:     time.Now(),
		Metadata: analysis.Metadata{
			Strategy: "volume_price_analysis",
			Tags:     []string{"volume", "confirmation", "divergence"},
			Indicators: map[string]float64{
				"price_change":          priceChange,
				"volume_change":         volumeChange,
				"correlation":           correlation,
				"confirmation_strength": math.Min(math.Abs(priceChange), math.Abs(volumeChange)),
			},
		},
	}
}

// UpdateConfig обновляет конфигурацию
func (c *VolumePriceConfirmationCalculator) UpdateConfig(config common.AnalyzerConfig) {
	c.config = config
	c.confidenceCalc.UpdateConfig(config)
}

// GetName возвращает имя калькулятора
func (c *VolumePriceConfirmationCalculator) GetName() string {
	return "volume_price_confirmation_calculator"
}

// getConfirmationThreshold возвращает порог для подтверждения
func (c *VolumePriceConfirmationCalculator) getConfirmationThreshold() float64 {
	if threshold, ok := c.config.CustomSettings["confirmation_threshold"].(float64); ok {
		return threshold
	}
	return 10.0 // значение по умолчанию
}

// calculateVolumePriceCorrelation вычисляет корреляцию между ценой и объемом
func (c *VolumePriceConfirmationCalculator) calculateVolumePriceCorrelation(data []redis_storage.PriceData) float64 {
	if len(data) < 2 {
		return 0
	}

	var priceChanges, volumeChanges []float64

	for i := 1; i < len(data); i++ {
		priceChange := ((data[i].Price - data[i-1].Price) / data[i-1].Price) * 100
		priceChanges = append(priceChanges, priceChange)

		if data[i-1].Volume24h > 0 {
			volumeChange := ((data[i].Volume24h - data[i-1].Volume24h) / data[i-1].Volume24h) * 100
			volumeChanges = append(volumeChanges, volumeChange)
		}
	}

	// Простая корреляция (чем больше, тем сильнее связь)
	if len(priceChanges) != len(volumeChanges) || len(priceChanges) == 0 {
		return 0
	}

	// Считаем сколько раз изменение цены и объема в одном направлении
	sameDirection := 0
	for i := 0; i < len(priceChanges); i++ {
		if (priceChanges[i] > 0 && volumeChanges[i] > 0) ||
			(priceChanges[i] < 0 && volumeChanges[i] < 0) {
			sameDirection++
		}
	}

	correlation := float64(sameDirection) / float64(len(priceChanges)) * 100
	return correlation
}
