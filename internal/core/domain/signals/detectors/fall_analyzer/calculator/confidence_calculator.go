// internal/core/domain/signals/detectors/fall_analyzer/calculator/confidence_calculator.go
package calculator

import (
	"math"
	"time"

	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
)

// ConfidenceCalculator - калькулятор для расчета уверенности в сигналах падения
type ConfidenceCalculator struct{}

// NewConfidenceCalculator создает новый калькулятор уверенности
func NewConfidenceCalculator() *ConfidenceCalculator {
	return &ConfidenceCalculator{}
}

// FallConfidenceParams - параметры для расчета уверенности в падении
type FallConfidenceParams struct {
	FallPercent      float64       // процент падения
	Duration         time.Duration // длительность падения
	DataPoints       int           // количество точек данных
	Volume           float64       // средний объем
	IsContinuous     bool          // является ли непрерывным
	ContinuousPoints int           // количество непрерывных точек
	TrendStrength    float64       // сила тренда (0-100)
	Volatility       float64       // волатильность (%)
	ContinuityRatio  float64       // коэффициент непрерывности (0-1)
}

// CalculateSingleFallConfidence рассчитывает уверенность для одиночного падения
func (cc *ConfidenceCalculator) CalculateSingleFallConfidence(params FallConfidenceParams) float64 {
	// Базовая уверенность на основе процента падения (макс 70%)
	fallConfidence := math.Min(params.FallPercent*10, 70)

	// Фактор объема
	volumeFactor := cc.calculateVolumeFactor(params.Volume)

	// Фактор времени
	timeFactor := cc.calculateTimeFactor(params.Duration)

	// Фактор волатильности
	volatilityFactor := cc.calculateVolatilityFactor(params.Volatility)

	// Фактор количества точек
	dataPointsFactor := cc.calculateDataPointsFactor(params.DataPoints)

	totalConfidence := fallConfidence + volumeFactor + timeFactor + volatilityFactor + dataPointsFactor
	result := math.Min(totalConfidence, 100)

	logger.Debug("📊 Fall ConfidenceCalculator: уверенность одиночного падения = %.1f%% "+
		"(падение:%.1f%%, объем:%.1f%%, время:%.1f%%, волатильность:%.1f%%, точки:%.1f%%)",
		result, fallConfidence, volumeFactor, timeFactor, volatilityFactor, dataPointsFactor)

	return math.Max(0, result)
}

// CalculateIntervalFallConfidence рассчитывает уверенность для интервального падения
func (cc *ConfidenceCalculator) CalculateIntervalFallConfidence(params FallConfidenceParams) float64 {
	// Базовая уверенность на основе процента падения (макс 80%)
	fallConfidence := math.Min(params.FallPercent*8, 80)

	// Бонус за непрерывность
	continuityBonus := cc.calculateContinuityBonus(params.IsContinuous, params.ContinuityRatio)

	// Фактор силы тренда
	trendFactor := cc.calculateTrendFactor(params.TrendStrength)

	// Фактор объема
	volumeFactor := cc.calculateVolumeFactor(params.Volume)

	// Фактор количества точек
	dataPointsFactor := cc.calculateDataPointsFactor(params.DataPoints)

	totalConfidence := fallConfidence + continuityBonus + trendFactor + volumeFactor + dataPointsFactor
	result := math.Min(totalConfidence, 100)

	logger.Debug("📊 Fall ConfidenceCalculator: уверенность интервального падения = %.1f%% "+
		"(падение:%.1f%%, непрерывность:%.1f%%, тренд:%.1f%%, объем:%.1f%%, точки:%.1f%%)",
		result, fallConfidence, continuityBonus, trendFactor, volumeFactor, dataPointsFactor)

	return math.Max(0, result)
}

// CalculateContinuousFallConfidence рассчитывает уверенность для непрерывного падения
func (cc *ConfidenceCalculator) CalculateContinuousFallConfidence(params FallConfidenceParams) float64 {
	// Базовая уверенность на основе процента падения (макс 90%)
	fallConfidence := math.Min(params.FallPercent*12, 90)

	// Бонус за количество непрерывных точек
	continuousPointsBonus := cc.calculateContinuousPointsBonus(params.ContinuousPoints)

	// Бонус за коэффициент непрерывности
	continuityRatioBonus := cc.calculateContinuityRatioBonus(params.ContinuityRatio)

	// Фактор силы тренда
	trendFactor := cc.calculateTrendFactor(params.TrendStrength)

	// Фактор объема
	volumeFactor := cc.calculateVolumeFactor(params.Volume)

	// Фактор количества точек данных
	dataPointsFactor := cc.calculateDataPointsFactor(params.DataPoints)

	totalConfidence := fallConfidence + continuousPointsBonus + continuityRatioBonus + trendFactor + volumeFactor + dataPointsFactor
	result := math.Min(totalConfidence, 100)

	logger.Debug("📊 Fall ConfidenceCalculator: уверенность непрерывного падения = %.1f%% "+
		"(падение:%.1f%%, непрер.точки:%.1f%%, коэф.непрер:%.1f%%, тренд:%.1f%%, объем:%.1f%%, точки:%.1f%%)",
		result, fallConfidence, continuousPointsBonus, continuityRatioBonus, trendFactor, volumeFactor, dataPointsFactor)

	return math.Max(0, result)
}

// calculateVolumeFactor рассчитывает фактор объема
func (cc *ConfidenceCalculator) calculateVolumeFactor(volume float64) float64 {
	if volume > 1000000 {
		return 10.0
	} else if volume > 500000 {
		return 7.0
	} else if volume > 100000 {
		return 3.0
	} else if volume < 50000 {
		return -5.0
	}
	return 0.0
}

// calculateTimeFactor рассчитывает фактор времени
func (cc *ConfidenceCalculator) calculateTimeFactor(duration time.Duration) float64 {
	minutes := duration.Minutes()

	if minutes < 5 {
		return 15.0
	} else if minutes < 10 {
		return 10.0
	} else if minutes < 30 {
		return 5.0
	} else if minutes > 60 {
		return -10.0
	}
	return 0.0
}

// calculateVolatilityFactor рассчитывает фактор волатильности
func (cc *ConfidenceCalculator) calculateVolatilityFactor(volatility float64) float64 {
	if volatility < 2.0 {
		return 10.0
	} else if volatility < 5.0 {
		return 5.0
	} else if volatility > 10.0 {
		return -10.0
	}
	return 0.0
}

// calculateDataPointsFactor рассчитывает фактор количества точек данных
func (cc *ConfidenceCalculator) calculateDataPointsFactor(dataPoints int) float64 {
	if dataPoints >= 10 {
		return 15.0
	} else if dataPoints >= 7 {
		return 10.0
	} else if dataPoints >= 5 {
		return 7.0
	} else if dataPoints >= 3 {
		return 3.0
	}
	return 0.0
}

// calculateContinuityBonus рассчитывает бонус за непрерывность
func (cc *ConfidenceCalculator) calculateContinuityBonus(isContinuous bool, continuityRatio float64) float64 {
	if !isContinuous {
		return 0.0
	}

	// Бонус зависит от коэффициента непрерывности
	if continuityRatio > 0.9 {
		return 25.0
	} else if continuityRatio > 0.8 {
		return 20.0
	} else if continuityRatio > 0.7 {
		return 15.0
	} else if continuityRatio > 0.6 {
		return 10.0
	}
	return 5.0
}

// calculateContinuousPointsBonus рассчитывает бонус за количество непрерывных точек
func (cc *ConfidenceCalculator) calculateContinuousPointsBonus(continuousPoints int) float64 {
	if continuousPoints >= 5 {
		return 15.0
	} else if continuousPoints >= 4 {
		return 12.0
	} else if continuousPoints >= 3 {
		return 8.0
	} else if continuousPoints >= 2 {
		return 5.0
	}
	return 0.0
}

// calculateContinuityRatioBonus рассчитывает бонус за коэффициент непрерывности
func (cc *ConfidenceCalculator) calculateContinuityRatioBonus(continuityRatio float64) float64 {
	if continuityRatio > 0.9 {
		return 15.0
	} else if continuityRatio > 0.8 {
		return 10.0
	} else if continuityRatio > 0.7 {
		return 7.0
	} else if continuityRatio > 0.6 {
		return 3.0
	}
	return 0.0
}

// calculateTrendFactor рассчитывает фактор силы тренда
func (cc *ConfidenceCalculator) calculateTrendFactor(trendStrength float64) float64 {
	// Сила тренда в диапазоне 0-100
	return math.Min(trendStrength/2, 10)
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

		// Нормализуем значение фактора
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

// AdjustConfidenceForMarketConditions корректирует уверенность с учетом рыночных условий
func (cc *ConfidenceCalculator) AdjustConfidenceForMarketConditions(baseConfidence float64, marketData []types.PriceData) float64 {
	if len(marketData) < 3 {
		return baseConfidence
	}

	// Рассчитываем общий рыночный тренд
	var marketAdjustment float64

	// Анализируем последние 3 точки
	recentData := marketData[len(marketData)-3:]

	// Проверяем, есть ли общий нисходящий тренд на рынке
	downCount := 0
	for i := 1; i < len(recentData); i++ {
		if recentData[i].Price < recentData[i-1].Price {
			downCount++
		}
	}

	// Если рынок в целом падает, увеличиваем уверенность
	if downCount >= 2 {
		marketAdjustment = 10.0
	} else if downCount == 0 {
		// Если рынок растет, уменьшаем уверенность
		marketAdjustment = -5.0
	}

	adjustedConfidence := baseConfidence + marketAdjustment
	return math.Max(0, math.Min(adjustedConfidence, 100))
}

// CalculateSignalQualityScore рассчитывает оценку качества сигнала падения
func (cc *ConfidenceCalculator) CalculateSignalQualityScore(
	confidence float64,
	fallPercent float64,
	duration time.Duration,
	volume float64,
	isContinuous bool,
	continuousPoints int,
) float64 {
	// Базовый показатель - уверенность
	score := confidence

	// Бонус за сильное падение
	if fallPercent > 5.0 {
		score += (fallPercent - 5.0) * 2
	}

	// Бонус за короткую длительность (быстрое падение)
	minutes := duration.Minutes()
	if minutes < 10 {
		score += (10 - minutes)
	}

	// Бонус за высокий объем
	if volume > 1000000 {
		score += 5
	} else if volume > 500000 {
		score += 3
	}

	// Бонус за непрерывность
	if isContinuous {
		score += 10
		if continuousPoints >= 3 {
			score += float64(continuousPoints) * 2
		}
	}

	// Ограничиваем диапазон 0-100
	return math.Max(0, math.Min(score, 100))
}
