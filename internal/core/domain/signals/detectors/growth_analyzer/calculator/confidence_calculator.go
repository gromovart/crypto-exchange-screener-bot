// internal/core/domain/signals/detectors/growth_analyzer/calculator/confidence_calculator.go
package calculator

import (
	"crypto-exchange-screener-bot/internal/types"
	"math"
)

// CalculatorConfig - конфигурация для калькулятора (без импорта growth_analyzer)
type CalculatorConfig struct {
	MinGrowthPercent      float64
	ContinuityThreshold   float64
	AccelerationThreshold float64
	VolumeWeight          float64
	TrendStrengthWeight   float64
	VolatilityWeight      float64
}

// CalculatorResult - результат расчета для калькулятора
type CalculatorResult struct {
	GrowthPercent   float64
	IsContinuous    bool
	ContinuityScore float64
	SignalType      string
	TrendStrength   float64
	Volatility      float64
	RawData         []types.PriceData
}

// CalculateConfidence - специализированный калькулятор уверенности для роста
func CalculateConfidence(result CalculatorResult, config CalculatorConfig) float64 {
	baseConfidence := calculateBaseConfidenceFromResult(result, config)

	// Применяем дополнительные факторы
	volumeFactor := calculateVolumeFactor(result.RawData, config.VolumeWeight)
	trendFactor := calculateTrendFactor(result.TrendStrength, config.TrendStrengthWeight)
	volatilityFactor := calculateVolatilityFactor(result.Volatility, config.VolatilityWeight)

	// Суммируем с базовой уверенностью
	finalConfidence := baseConfidence + volumeFactor + trendFactor + volatilityFactor

	// Применяем корректировки на основе типа сигнала
	finalConfidence = applySignalTypeAdjustment(finalConfidence, result.SignalType)

	// Ограничиваем 0-100
	return math.Max(0, math.Min(finalConfidence, 100))
}

// calculateBaseConfidenceFromResult - рассчитывает базовую уверенность из результата
func calculateBaseConfidenceFromResult(result CalculatorResult, config CalculatorConfig) float64 {
	baseConfidence := 0.0

	// 1. Процент роста (до 40%)
	growthScore := math.Min(result.GrowthPercent*2, 40)
	baseConfidence += growthScore

	// 2. Непрерывность (до 30%)
	if result.IsContinuous {
		// Рассчитываем score непрерывности
		continuityScore := calculateContinuityScore(result.RawData, config.ContinuityThreshold)
		baseConfidence += continuityScore * 30
	}

	// 3. Количество точек данных (до 20%)
	dataPointsScore := math.Min(float64(len(result.RawData))/50.0*20, 20)
	baseConfidence += dataPointsScore

	// 4. Сила тренда (до 10%)
	trendScore := math.Min(math.Abs(result.TrendStrength)*2, 10)
	baseConfidence += trendScore

	return baseConfidence
}

// calculateContinuityScore - рассчитывает score непрерывности
func calculateContinuityScore(data []types.PriceData, threshold float64) float64 {
	if len(data) < 2 {
		return 0.0
	}

	continuousPoints := 0
	totalPoints := len(data) - 1

	for i := 1; i < len(data); i++ {
		if data[i].Price >= data[i-1].Price {
			continuousPoints++
		}
	}

	continuityRatio := float64(continuousPoints) / float64(totalPoints)

	// Нормализуем относительно порога
	if continuityRatio >= threshold {
		return 1.0 // Максимальный score
	}

	// Линейная интерполяция от 0 до threshold
	return continuityRatio / threshold
}

// calculateVolumeFactor - рассчитывает фактор объема
func calculateVolumeFactor(data []types.PriceData, volumeWeight float64) float64 {
	if len(data) == 0 || volumeWeight <= 0 {
		return 0.0
	}

	// Рассчитываем средний объем (используем Volume24h)
	var totalVolume float64
	for _, point := range data {
		totalVolume += point.Volume24h
	}
	averageVolume := totalVolume / float64(len(data))

	// Нормализуем объем (предполагаем, что объем > 1000 считается хорошим)
	volumeScore := math.Min(averageVolume/10000.0, 1.0)

	// Применяем вес
	return volumeScore * volumeWeight * 100
}

// calculateTrendFactor - рассчитывает фактор силы тренда
func calculateTrendFactor(trendStrength float64, trendWeight float64) float64 {
	if trendWeight <= 0 {
		return 0.0
	}

	// Сила тренда уже нормализована в процентах
	// Преобразуем в score от 0 до 1
	trendScore := math.Min(math.Abs(trendStrength)/10.0, 1.0)

	// Применяем вес
	return trendScore * trendWeight * 100
}

// calculateVolatilityFactor - рассчитывает фактор волатильности
func calculateVolatilityFactor(volatility float64, volatilityWeight float64) float64 {
	if volatilityWeight <= 0 {
		return 0.0
	}

	// Низкая волатильность лучше для роста
	// Преобразуем волатильность в score (ниже волатильность = выше score)
	var volatilityScore float64
	if volatility < 2.0 {
		volatilityScore = 1.0 // Отличная низкая волатильность
	} else if volatility < 5.0 {
		volatilityScore = 0.7 // Хорошая волатильность
	} else if volatility < 10.0 {
		volatilityScore = 0.4 // Средняя волатильность
	} else if volatility < 20.0 {
		volatilityScore = 0.1 // Высокая волатильность
	} else {
		volatilityScore = 0.0 // Очень высокая волатильность
	}

	// Применяем вес
	return volatilityScore * volatilityWeight * 100
}

// applySignalTypeAdjustment - применяет корректировки на основе типа сигнала
func applySignalTypeAdjustment(confidence float64, signalType string) float64 {
	switch signalType {
	case "accelerated_growth":
		// Ускоренный рост дает бонус +5%
		return confidence + 5.0
	case "breakout_growth":
		// Пробойной рост дает бонус +8%
		return confidence + 8.0
	case "continuous_growth":
		// Устойчивый рост дает бонус +3%
		return confidence + 3.0
	default:
		return confidence
	}
}

// ValidateConfidence - проверяет уверенность на валидность
func ValidateConfidence(confidence float64) bool {
	return confidence >= 0 && confidence <= 100
}

// ConfidenceLevel - возвращает уровень уверенности
func ConfidenceLevel(confidence float64) string {
	if confidence >= 80 {
		return "Очень высокая"
	} else if confidence >= 60 {
		return "Высокая"
	} else if confidence >= 40 {
		return "Средняя"
	} else if confidence >= 20 {
		return "Низкая"
	} else {
		return "Очень низкая"
	}
}
