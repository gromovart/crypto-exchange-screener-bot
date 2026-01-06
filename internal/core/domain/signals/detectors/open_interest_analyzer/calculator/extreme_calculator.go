// internal/core/domain/signals/detectors/open_interest_analyzer/calculator/extreme_calculator.go
package calculator

import (
	"math"
	"time"

	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
)

// ExtremeCalculator - калькулятор для анализа экстремальных значений OI
type ExtremeCalculator struct{}

// NewExtremeCalculator создает новый калькулятор экстремальных значений
func NewExtremeCalculator() *ExtremeCalculator {
	return &ExtremeCalculator{}
}

// OIConfigForExtreme - конфигурация для экстремального анализа
type OIConfigForExtreme struct {
	MinConfidence      float64
	ExtremeOIThreshold float64
}

// ExtremeResult - результат экстремального анализа
type ExtremeResult struct {
	Symbol        string
	Type          string
	Direction     string
	ChangePercent float64
	Confidence    float64
	DataPoints    int
	StartPrice    float64
	EndPrice      float64
	StartOI       float64
	EndOI         float64
	CurrentOI     float64
	AvgOI         float64
	OIRatio       float64
	IsExtreme     bool
	ExtremeType   string
	Indicators    map[string]float64
}

// AnalyzeExtremeOI анализирует экстремальные значения OI
func (ec *ExtremeCalculator) AnalyzeExtremeOI(data []types.PriceData, config OIConfigForExtreme) *ExtremeResult {
	if len(data) < 3 {
		logger.Debug("📭 ExtremeCalculator: недостаточно точек для экстремального анализа (%d < 3)", len(data))
		return nil
	}

	// Собираем все значения OI
	var oiValues []float64
	var totalOI float64
	validPoints := 0

	for _, point := range data {
		if point.OpenInterest > 0 {
			oiValues = append(oiValues, point.OpenInterest)
			totalOI += point.OpenInterest
			validPoints++
		}
	}

	if validPoints < 3 {
		logger.Debug("📭 ExtremeCalculator: недостаточно точек с OI для анализа (%d < 3)", validPoints)
		return nil
	}

	// Рассчитываем среднее OI
	avgOI := totalOI / float64(validPoints)

	// Находим последнее значение OI
	latest := data[len(data)-1]
	lastOI := latest.OpenInterest

	if lastOI <= 0 {
		return nil
	}

	// Рассчитываем, насколько последнее OI отличается от среднего
	oiRatio := lastOI / avgOI
	extremeThreshold := config.ExtremeOIThreshold

	logger.Debug("📊 ExtremeCalculator: %s - OI анализ: текущее=%.0f, среднее=%.0f, отношение=%.2f (порог=%.1f)",
		data[0].Symbol, lastOI, avgOI, oiRatio, extremeThreshold)

	// Проверяем экстремальное значение
	isExtreme := oiRatio > extremeThreshold
	if isExtreme {
		// Высокий OI относительно среднего
		confidence := ec.calculateExtremeConfidence(oiRatio, extremeThreshold, validPoints)

		if confidence >= config.MinConfidence {
			// Определяем направление по цене
			priceChange := ec.calculatePriceChange(data)
			direction := ec.determineDirection(priceChange)

			logger.Debug("⚠️  ExtremeCalculator: %s - ЭКСТРЕМАЛЬНЫЙ OI! отношение=%.2f, уверенность=%.1f%%, цена=%.2f%%",
				data[0].Symbol, oiRatio, confidence, priceChange)

			result := &ExtremeResult{
				Symbol:        data[0].Symbol,
				Type:          "extreme_oi",
				Direction:     direction,
				ChangePercent: priceChange,
				Confidence:    confidence,
				DataPoints:    validPoints,
				StartPrice:    data[0].Price,
				EndPrice:      latest.Price,
				StartOI:       data[0].OpenInterest,
				EndOI:         lastOI,
				CurrentOI:     lastOI,
				AvgOI:         avgOI,
				OIRatio:       oiRatio,
				IsExtreme:     true,
				ExtremeType:   "high",
				Indicators:    make(map[string]float64),
			}

			// Заполняем индикаторы
			result.Indicators["current_oi"] = lastOI
			result.Indicators["avg_oi"] = avgOI
			result.Indicators["oi_ratio"] = oiRatio
			result.Indicators["oi_deviation"] = (oiRatio - 1) * 100
			result.Indicators["price_change"] = priceChange
			result.Indicators["oi_values_count"] = float64(validPoints)
			result.Indicators["extreme_threshold"] = extremeThreshold
			result.Indicators["volatility"] = ec.calculateOIVolatility(oiValues)

			return result
		} else {
			logger.Debug("📉 ExtremeCalculator: %s - экстремальное значение есть, но уверенность низкая (%.1f%% < %.1f%%)",
				data[0].Symbol, confidence, config.MinConfidence)
		}
	}

	return nil
}

// calculatePriceChange рассчитывает изменение цены за период
func (ec *ExtremeCalculator) calculatePriceChange(data []types.PriceData) float64 {
	if len(data) < 2 {
		return 0
	}

	firstPrice := data[0].Price
	lastPrice := data[len(data)-1].Price
	return ((lastPrice - firstPrice) / firstPrice) * 100
}

// determineDirection определяет направление движения цены
func (ec *ExtremeCalculator) determineDirection(priceChange float64) string {
	if priceChange > 0 {
		return "up"
	} else if priceChange < 0 {
		return "down"
	}
	return "neutral"
}

// calculateExtremeConfidence рассчитывает уверенность в экстремальном значении
func (ec *ExtremeCalculator) calculateExtremeConfidence(oiRatio, threshold float64, dataPoints int) float64 {
	// Базовая уверенность на основе отклонения от порога
	baseConfidence := math.Min((oiRatio-threshold)*100, 50)

	// Бонус за количество точек данных
	dataPointsBonus := 0.0
	if dataPoints >= 10 {
		dataPointsBonus = 20
	} else if dataPoints >= 5 {
		dataPointsBonus = 10
	} else if dataPoints >= 3 {
		dataPointsBonus = 5
	}

	// Бонус за сильное отклонение
	deviationBonus := 0.0
	if oiRatio > threshold*1.5 {
		deviationBonus = 15
	} else if oiRatio > threshold*1.2 {
		deviationBonus = 8
	}

	totalConfidence := baseConfidence + dataPointsBonus + deviationBonus
	return math.Min(totalConfidence, 90)
}

// calculateOIVolatility рассчитывает волатильность OI
func (ec *ExtremeCalculator) calculateOIVolatility(oiValues []float64) float64 {
	if len(oiValues) < 2 {
		return 0
	}

	// Рассчитываем среднее
	var sum float64
	for _, v := range oiValues {
		sum += v
	}
	mean := sum / float64(len(oiValues))

	// Рассчитываем стандартное отклонение
	var variance float64
	for _, v := range oiValues {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(oiValues))
	stdDev := math.Sqrt(variance)

	// Возвращаем коэффициент вариации в процентах
	if mean > 0 {
		return (stdDev / mean) * 100
	}
	return 0
}

// IsExtremeOI проверяет, является ли значение OI экстремальным
func (ec *ExtremeCalculator) IsExtremeOI(currentOI, avgOI float64, threshold float64) bool {
	if avgOI <= 0 {
		return false
	}
	ratio := currentOI / avgOI
	return ratio > threshold
}

// GetExtremeLevel возвращает уровень экстремальности
func (ec *ExtremeCalculator) GetExtremeLevel(currentOI, avgOI float64, threshold float64) string {
	if avgOI <= 0 {
		return "normal"
	}

	ratio := currentOI / avgOI
	if ratio > threshold*2 {
		return "extreme_high"
	} else if ratio > threshold*1.5 {
		return "very_high"
	} else if ratio > threshold {
		return "high"
	}
	return "normal"
}

// CalculateExtremeDurationConfidence рассчитывает уверенность на основе длительности экстремального состояния
func (ec *ExtremeCalculator) CalculateExtremeDurationConfidence(duration time.Duration) float64 {
	// Чем дольше состояние сохраняется, тем выше уверенность
	if duration >= 30*time.Minute {
		return 15
	} else if duration >= 15*time.Minute {
		return 10
	} else if duration >= 5*time.Minute {
		return 5
	}
	return 0
}
