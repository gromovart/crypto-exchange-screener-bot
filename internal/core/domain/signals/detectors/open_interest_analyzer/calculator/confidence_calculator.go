// internal/core/domain/signals/detectors/open_interest_analyzer/calculator/confidence_calculator.go
package calculator

import (
	"math"
	"time"

	"crypto-exchange-screener-bot/pkg/logger"
)

// ConfidenceCalculator - калькулятор для расчета уверенности в сигналах OI
type ConfidenceCalculator struct{}

// NewConfidenceCalculator создает новый калькулятор уверенности
func NewConfidenceCalculator() *ConfidenceCalculator {
	return &ConfidenceCalculator{}
}

// CalculateGrowthWithPriceConfidence рассчитывает уверенность для сигнала роста OI с ростом цены
func (cc *ConfidenceCalculator) CalculateGrowthWithPriceConfidence(priceChange, oiChange float64, duration time.Duration, dataPoints int) float64 {
	// Базовая уверенность на основе изменения цены (макс 40%)
	priceConfidence := math.Min(priceChange*2, 40)

	// Уверенность на основе изменения OI (макс 30%)
	oiConfidence := math.Min(oiChange/2, 30)

	// Дополнительный бонус за синхронность (макс 30%)
	syncBonus := cc.calculateSyncBonus(priceChange, oiChange)

	// Бонус за количество точек данных
	dataPointsBonus := cc.calculateDataPointsBonus(dataPoints)

	// Бонус за длительность периода
	durationBonus := cc.calculateDurationBonus(duration)

	totalConfidence := priceConfidence + oiConfidence + syncBonus + dataPointsBonus + durationBonus
	result := math.Min(totalConfidence, 100)

	logger.Debug("📊 ConfidenceCalculator: уверенность роста OI+цена = %.1f%% "+
		"(цена:%.1f%%, OI:%.1f%%, синхр:%.1f%%, точки:%.1f%%, время:%.1f%%)",
		result, priceConfidence, oiConfidence, syncBonus, dataPointsBonus, durationBonus)

	return result
}

// CalculateGrowthWithFallConfidence рассчитывает уверенность для сигнала роста OI при падении цены
func (cc *ConfidenceCalculator) CalculateGrowthWithFallConfidence(priceFall, oiGrowth float64, duration time.Duration, dataPoints int) float64 {
	// Чем сильнее падение цены при росте OI, тем увереннее сигнал
	baseConfidence := math.Min(priceFall*3, 60)
	oiConfidence := math.Min(oiGrowth, 30)

	// Бонус за контрастность (чем сильнее контраст, тем выше уверенность)
	contrastBonus := cc.calculateContrastBonus(priceFall, oiGrowth)

	// Бонус за количество точек данных
	dataPointsBonus := cc.calculateDataPointsBonus(dataPoints)

	// Бонус за длительность периода
	durationBonus := cc.calculateDurationBonus(duration)

	totalConfidence := baseConfidence + oiConfidence + contrastBonus + dataPointsBonus + durationBonus
	result := math.Min(totalConfidence, 100)

	logger.Debug("📊 ConfidenceCalculator: уверенность роста OI при падении = %.1f%% "+
		"(падение:%.1f%%, OI:%.1f%%, контраст:%.1f%%, точки:%.1f%%, время:%.1f%%)",
		result, baseConfidence, oiConfidence, contrastBonus, dataPointsBonus, durationBonus)

	return result
}

// calculateSyncBonus рассчитывает бонус за синхронность движений цены и OI
func (cc *ConfidenceCalculator) calculateSyncBonus(priceChange, oiChange float64) float64 {
	// Идеальная синхронность: OI изменяется пропорционально цене
	// но с некоторым лагом или усилением

	if oiChange <= 0 || priceChange <= 0 {
		return 0
	}

	ratio := oiChange / priceChange

	// Оптимальное соотношение: от 0.5 до 2.0
	if ratio >= 0.5 && ratio <= 2.0 {
		// Чем ближе к 1.0, тем лучше
		distanceFromOne := math.Abs(ratio - 1.0)
		if distanceFromOne <= 0.5 {
			syncBonus := (1.0 - distanceFromOne*2) * 30
			return math.Max(syncBonus, 0)
		}
	}

	return 0
}

// calculateContrastBonus рассчитывает бонус за контрастность (противоположные движения)
func (cc *ConfidenceCalculator) calculateContrastBonus(priceFall, oiGrowth float64) float64 {
	// Контрастность хороша, когда оба движения значительны
	minMovement := math.Min(math.Abs(priceFall), oiGrowth)
	if minMovement < 2.0 {
		return 0
	}

	// Чем сильнее оба движения, тем выше бонус
	contrastStrength := (math.Abs(priceFall) + oiGrowth) / 2
	return math.Min(contrastStrength*2, 15)
}

// calculateDataPointsBonus рассчитывает бонус за количество точек данных
func (cc *ConfidenceCalculator) calculateDataPointsBonus(dataPoints int) float64 {
	if dataPoints >= 20 {
		return 15
	} else if dataPoints >= 15 {
		return 12
	} else if dataPoints >= 10 {
		return 8
	} else if dataPoints >= 7 {
		return 5
	} else if dataPoints >= 5 {
		return 3
	} else if dataPoints >= 3 {
		return 1
	}
	return 0
}

// calculateDurationBonus рассчитывает бонус за длительность периода анализа
func (cc *ConfidenceCalculator) calculateDurationBonus(duration time.Duration) float64 {
	minutes := duration.Minutes()

	if minutes >= 30 {
		return 10
	} else if minutes >= 15 {
		return 7
	} else if minutes >= 10 {
		return 5
	} else if minutes >= 5 {
		return 3
	} else if minutes >= 2 {
		return 1
	}
	return 0
}

// CalculateCompositeConfidence рассчитывает комбинированную уверенность
func (cc *ConfidenceCalculator) CalculateCompositeConfidence(factors map[string]float64, weights map[string]float64) float64 {
	if len(factors) == 0 {
		return 0
	}

	var totalWeightedConfidence float64
	var totalWeight float64

	for factor, value := range factors {
		weight := 1.0
		if w, exists := weights[factor]; exists {
			weight = w
		}

		// Нормализуем значение фактора (предполагаем, что оно уже в диапазоне 0-1 или 0-100)
		normalizedValue := value
		if value > 1.0 && value <= 100.0 {
			normalizedValue = value / 100.0
		}

		totalWeightedConfidence += normalizedValue * weight
		totalWeight += weight
	}

	if totalWeight == 0 {
		return 0
	}

	averageConfidence := totalWeightedConfidence / totalWeight
	return math.Min(averageConfidence*100, 100)
}

// AdjustConfidenceForVolume корректирует уверенность с учетом объема
func (cc *ConfidenceCalculator) AdjustConfidenceForVolume(baseConfidence, volumeRatio float64, volumeWeight float64) float64 {
	// volumeRatio - отношение текущего объема к среднему
	// volumeWeight - вес объема в итоговой уверенности (0-1)

	if volumeWeight <= 0 {
		return baseConfidence
	}

	// Объем увеличивает уверенность, если он выше среднего
	volumeImpact := 0.0
	if volumeRatio > 1.0 {
		// Объем выше среднего: положительное влияние
		volumeImpact = math.Min((volumeRatio-1.0)*20, 20) * volumeWeight
	} else if volumeRatio < 0.5 {
		// Объем значительно ниже среднего: отрицательное влияние
		volumeImpact = -10 * volumeWeight
	}

	adjustedConfidence := baseConfidence + volumeImpact
	return math.Max(0, math.Min(adjustedConfidence, 100))
}

// CalculateTrendStrengthConfidence рассчитывает уверенность в силе тренда
func (cc *ConfidenceCalculator) CalculateTrendStrengthConfidence(priceChanges []float64, oiChanges []float64) float64 {
	if len(priceChanges) < 3 || len(oiChanges) < 3 {
		return 0
	}

	// Проверяем согласованность движений
	priceConsistency := cc.calculateConsistency(priceChanges)
	oiConsistency := cc.calculateConsistency(oiChanges)

	// Рассчитываем силу тренда
	trendStrength := (priceConsistency + oiConsistency) / 2

	// Преобразуем в уверенность (0-100%)
	confidence := trendStrength * 100
	return math.Min(confidence, 100)
}

// calculateConsistency рассчитывает согласованность последовательности значений
func (cc *ConfidenceCalculator) calculateConsistency(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}

	// Считаем количество движений в одном направлении
	sameDirectionCount := 0
	totalMovements := len(values) - 1

	firstSign := 0.0
	for i := 1; i < len(values); i++ {
		diff := values[i] - values[i-1]
		sign := 0.0
		if diff > 0 {
			sign = 1.0
		} else if diff < 0 {
			sign = -1.0
		}

		if i == 1 {
			firstSign = sign
		} else if sign == firstSign && sign != 0 {
			sameDirectionCount++
		}
	}

	if totalMovements == 0 {
		return 0
	}

	consistency := float64(sameDirectionCount) / float64(totalMovements)
	return consistency
}

// CalculateSignalQualityScore рассчитывает оценку качества сигнала
func (cc *ConfidenceCalculator) CalculateSignalQualityScore(
	confidence float64,
	dataPoints int,
	duration time.Duration,
	volatility float64,
	extremeDuration time.Duration,
) float64 {
	// Базовый показатель - уверенность
	score := confidence

	// Бонус за количество точек данных
	if dataPoints >= 10 {
		score += 5
	} else if dataPoints >= 5 {
		score += 2
	}

	// Бонус за длительность анализа
	minutes := duration.Minutes()
	if minutes >= 15 {
		score += 5
	} else if minutes >= 5 {
		score += 2
	}

	// Штраф за высокую волатильность (сигнал менее надежен)
	if volatility > 5.0 {
		score -= (volatility - 5.0) * 2
	}

	// Бонус за длительность экстремального состояния
	if extremeDuration > 0 {
		extremeMinutes := extremeDuration.Minutes()
		if extremeMinutes >= 30 {
			score += 10
		} else if extremeMinutes >= 15 {
			score += 5
		} else if extremeMinutes >= 5 {
			score += 2
		}
	}

	// Ограничиваем диапазон 0-100
	return math.Max(0, math.Min(score, 100))
}
