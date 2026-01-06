// internal/core/domain/signals/detectors/open_interest_analyzer/calculator/divergence_calculator.go
package calculator

import (
	"math"

	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
)

// DivergenceCalculator - калькулятор для анализа дивергенций OI-цена
type DivergenceCalculator struct{}

// NewDivergenceCalculator создает новый калькулятор дивергенций
func NewDivergenceCalculator() *DivergenceCalculator {
	return &DivergenceCalculator{}
}

// OIConfigForDivergence - упрощенная версия конфигурации для дивергенций
type OIConfigForDivergence struct {
	MinConfidence       float64
	DivergenceMinPoints int
}

// OISignalForDivergence - упрощенная версия сигнала для дивергенций
type OISignalForDivergence struct {
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
	Metadata      DivergenceMetadata
}

// DivergenceMetadata - метаданные для дивергенций
type DivergenceMetadata struct {
	Strategy       string
	Tags           []string
	Indicators     map[string]float64
	DivergenceType string
	Patterns       []string
}

// AnalyzeDivergence анализирует дивергенции между OI и ценой
func (dc *DivergenceCalculator) AnalyzeDivergence(data []types.PriceData, config OIConfigForDivergence) *OISignalForDivergence {
	minPoints := config.DivergenceMinPoints
	if minPoints < 4 {
		minPoints = 4
	}

	if len(data) < minPoints {
		logger.Debug("📭 DivergenceCalculator: недостаточно точек для анализа дивергенции (%d < %d)", len(data), minPoints)
		return nil
	}

	// Собираем цены и OI
	var prices, oiValues []float64
	var priceChanges, oiChanges []float64

	for i, point := range data {
		if point.OpenInterest > 0 {
			prices = append(prices, point.Price)
			oiValues = append(oiValues, point.OpenInterest)
		}

		// Рассчитываем изменения
		if i > 0 && i < len(data) {
			if data[i].OpenInterest > 0 && data[i-1].OpenInterest > 0 {
				prevOI := data[i-1].OpenInterest
				currOI := data[i].OpenInterest

				priceChange := ((data[i].Price - data[i-1].Price) / data[i-1].Price) * 100
				oiChange := ((currOI - prevOI) / prevOI) * 100

				priceChanges = append(priceChanges, priceChange)
				oiChanges = append(oiChanges, oiChange)
			}
		}
	}

	if len(priceChanges) < 3 || len(oiChanges) < 3 {
		logger.Debug("📭 DivergenceCalculator: недостаточно изменений для дивергенции (цена:%d, OI:%d)",
			len(priceChanges), len(oiChanges))
		return nil
	}

	// Ищем дивергенции
	divergenceType := dc.findDivergence(priceChanges, oiChanges)

	if divergenceType != "" {
		priceChange := dc.calculatePriceChange(data)
		confidence := dc.calculateDivergenceConfidence(divergenceType, priceChanges, oiChanges)

		if confidence >= config.MinConfidence {
			var direction, signalType string
			if divergenceType == "bullish" {
				direction = "up"
				signalType = "bullish_oi_divergence"
			} else {
				direction = "down"
				signalType = "bearish_oi_divergence"
			}

			logger.Debug("🔀 DivergenceCalculator: %s - ДИВЕРГЕНЦИЯ %s! уверенность=%.1f%%, цена=%.2f%%",
				data[0].Symbol, divergenceType, confidence, priceChange)

			signal := &OISignalForDivergence{
				Symbol:        data[0].Symbol,
				Type:          signalType,
				Direction:     direction,
				ChangePercent: priceChange,
				Confidence:    confidence,
				DataPoints:    len(data),
				StartPrice:    data[0].Price,
				EndPrice:      data[len(data)-1].Price,
				StartOI:       data[0].OpenInterest,
				EndOI:         data[len(data)-1].OpenInterest,
				Metadata: DivergenceMetadata{
					DivergenceType: divergenceType,
					Patterns:       []string{"oi_price_divergence", divergenceType + "_divergence"},
					Indicators:     make(map[string]float64),
				},
			}

			// Заполняем индикаторы
			if divergenceType == "bullish" {
				signal.Metadata.Indicators["divergence_type"] = 1.0
			} else {
				signal.Metadata.Indicators["divergence_type"] = -1.0
			}
			signal.Metadata.Indicators["price_change"] = priceChange
			signal.Metadata.Indicators["avg_price_change"] = dc.calculateAverage(priceChanges)
			signal.Metadata.Indicators["avg_oi_change"] = dc.calculateAverage(oiChanges)
			signal.Metadata.Indicators["divergence_strength"] = confidence / 100
			signal.Metadata.Indicators["price_volatility"] = dc.calculateVolatility(prices)
			signal.Metadata.Indicators["oi_volatility"] = dc.calculateVolatility(oiValues)
			signal.Metadata.Indicators["correlation"] = dc.calculateCorrelation(priceChanges, oiChanges)

			return signal
		}
	}

	return nil
}

// findDivergence ищет дивергенции между ценами и OI
func (dc *DivergenceCalculator) findDivergence(priceChanges, oiChanges []float64) string {
	if len(priceChanges) < 3 || len(oiChanges) < 3 {
		return ""
	}

	// Простая логика дивергенции:
	// Бычья дивергенция: цена делает новые минимумы, а OI растет
	// Медвежья дивергенция: цена делает новые максимумы, а OI падает

	// Проверяем последние 3 точки
	lastPrice1 := priceChanges[len(priceChanges)-3]
	lastPrice2 := priceChanges[len(priceChanges)-2]
	lastPrice3 := priceChanges[len(priceChanges)-1]

	lastOI1 := oiChanges[len(oiChanges)-3]
	lastOI2 := oiChanges[len(oiChanges)-2]
	lastOI3 := oiChanges[len(oiChanges)-1]

	logger.Debug("🔍 DivergenceCalculator: проверка дивергенции - цена: [%.2f, %.2f, %.2f], OI: [%.2f, %.2f, %.2f]",
		lastPrice1, lastPrice2, lastPrice3, lastOI1, lastOI2, lastOI3)

	// Бычья дивергенция (цена делает higher low, OI делает lower high)
	if lastPrice1 > lastPrice2 && lastPrice2 < lastPrice3 && // цена делает higher low
		lastOI1 < lastOI2 && lastOI2 > lastOI3 { // OI делает lower high
		logger.Debug("✅ DivergenceCalculator: обнаружена БЫЧЬЯ дивергенция")
		return "bullish"
	}

	// Медвежья дивергенция (цена делает lower high, OI делает higher low)
	if lastPrice1 < lastPrice2 && lastPrice2 > lastPrice3 && // цена делает lower high
		lastOI1 > lastOI2 && lastOI2 < lastOI3 { // OI делает higher low
		logger.Debug("✅ DivergenceCalculator: обнаружена МЕДВЕЖЬЯ дивергенция")
		return "bearish"
	}

	// Дополнительная проверка: классическая дивергенция
	if dc.checkClassicalDivergence(priceChanges, oiChanges) {
		// Определяем тип по последним значениям
		if priceChanges[len(priceChanges)-1] > 0 && oiChanges[len(oiChanges)-1] < 0 {
			return "bearish"
		} else if priceChanges[len(priceChanges)-1] < 0 && oiChanges[len(oiChanges)-1] > 0 {
			return "bullish"
		}
	}

	return ""
}

// checkClassicalDivergence проверяет классическую дивергенцию
func (dc *DivergenceCalculator) checkClassicalDivergence(priceChanges, oiChanges []float64) bool {
	if len(priceChanges) < 5 || len(oiChanges) < 5 {
		return false
	}

	// Ищем расхождения в направлениях движения
	priceDirection := dc.getDirection(priceChanges[len(priceChanges)-5:])
	oiDirection := dc.getDirection(oiChanges[len(oiChanges)-5:])

	// Дивергенция есть, если направления противоположны
	return (priceDirection == "up" && oiDirection == "down") ||
		(priceDirection == "down" && oiDirection == "up")
}

// getDirection определяет общее направление движения
func (dc *DivergenceCalculator) getDirection(changes []float64) string {
	if len(changes) == 0 {
		return "neutral"
	}

	// Считаем положительные и отрицательные изменения
	positive := 0
	negative := 0

	for _, change := range changes {
		if change > 0 {
			positive++
		} else if change < 0 {
			negative++
		}
	}

	if positive > negative {
		return "up"
	} else if negative > positive {
		return "down"
	}
	return "neutral"
}

// calculatePriceChange рассчитывает общее изменение цены
func (dc *DivergenceCalculator) calculatePriceChange(data []types.PriceData) float64 {
	if len(data) < 2 {
		return 0
	}

	firstPrice := data[0].Price
	lastPrice := data[len(data)-1].Price
	return ((lastPrice - firstPrice) / firstPrice) * 100
}

// calculateDivergenceConfidence рассчитывает уверенность в дивергенции
func (dc *DivergenceCalculator) calculateDivergenceConfidence(divergenceType string, priceChanges, oiChanges []float64) float64 {
	if len(priceChanges) < 3 {
		return 0
	}

	// Рассчитываем силу дивергенции
	var divergenceStrength float64

	if divergenceType == "bullish" {
		// Для бычьей дивергенции: чем ниже цена и выше OI, тем сильнее
		priceDecrease := math.Abs(priceChanges[len(priceChanges)-2]) // самый низкий
		oiIncrease := oiChanges[len(oiChanges)-2]                    // самый высокий
		divergenceStrength = priceDecrease + oiIncrease
	} else {
		// Для медвежьей дивергенции: чем выше цена и ниже OI, тем сильнее
		priceIncrease := priceChanges[len(priceChanges)-2]  // самый высокий
		oiDecrease := math.Abs(oiChanges[len(oiChanges)-2]) // самый низкий
		divergenceStrength = priceIncrease + oiDecrease
	}

	// Нормализуем до 0-100%
	confidence := math.Min(divergenceStrength*10, 80)

	// Добавляем бонус за количество точек
	if len(priceChanges) >= 10 {
		confidence += 15
	} else if len(priceChanges) >= 7 {
		confidence += 10
	} else if len(priceChanges) >= 5 {
		confidence += 5
	}

	// Добавляем бонус за ясность паттерна
	clarityBonus := dc.calculatePatternClarity(priceChanges, oiChanges)
	confidence += clarityBonus

	logger.Debug("📊 DivergenceCalculator: уверенность дивергенции %s = %.1f%% (сила=%.2f, точек=%d, ясность=%.1f)",
		divergenceType, confidence, divergenceStrength, len(priceChanges), clarityBonus)

	return math.Min(confidence, 100)
}

// calculatePatternClarity рассчитывает ясность паттерна дивергенции
func (dc *DivergenceCalculator) calculatePatternClarity(priceChanges, oiChanges []float64) float64 {
	if len(priceChanges) < 4 {
		return 0
	}

	// Проверяем четкость паттерна
	clarity := 0.0

	// Проверяем последовательность минимумов/максимумов
	if dc.checkClearPattern(priceChanges) && dc.checkClearPattern(oiChanges) {
		clarity += 10
	}

	// Проверяем амплитуду движений
	priceAmplitude := dc.calculateAmplitude(priceChanges)
	oiAmplitude := dc.calculateAmplitude(oiChanges)

	if priceAmplitude > 2.0 && oiAmplitude > 2.0 {
		clarity += 5
	}

	// Проверяем количество точек в одном направлении
	if dc.countConsecutiveDirection(priceChanges) >= 3 ||
		dc.countConsecutiveDirection(oiChanges) >= 3 {
		clarity += 5
	}

	return clarity
}

// checkClearPattern проверяет четкость паттерна
func (dc *DivergenceCalculator) checkClearPattern(changes []float64) bool {
	if len(changes) < 3 {
		return false
	}

	// Проверяем наличие четкого минимума или максимума
	hasExtreme := false
	for i := 1; i < len(changes)-1; i++ {
		// Проверяем локальный минимум
		if changes[i-1] > changes[i] && changes[i] < changes[i+1] {
			hasExtreme = true
			break
		}
		// Проверяем локальный максимум
		if changes[i-1] < changes[i] && changes[i] > changes[i+1] {
			hasExtreme = true
			break
		}
	}

	return hasExtreme
}

// calculateAmplitude рассчитывает амплитуду движений
func (dc *DivergenceCalculator) calculateAmplitude(changes []float64) float64 {
	if len(changes) == 0 {
		return 0
	}

	min := changes[0]
	max := changes[0]

	for _, v := range changes {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	return math.Abs(max - min)
}

// countConsecutiveDirection считает последовательные движения в одном направлении
func (dc *DivergenceCalculator) countConsecutiveDirection(changes []float64) int {
	if len(changes) < 2 {
		return 0
	}

	maxCount := 0
	currentCount := 0
	currentSign := 0.0

	for _, change := range changes {
		if change > 0 {
			if currentSign > 0 {
				currentCount++
			} else {
				currentSign = 1
				currentCount = 1
			}
		} else if change < 0 {
			if currentSign < 0 {
				currentCount++
			} else {
				currentSign = -1
				currentCount = 1
			}
		} else {
			currentSign = 0
			currentCount = 0
		}

		if currentCount > maxCount {
			maxCount = currentCount
		}
	}

	return maxCount
}

// calculateAverage рассчитывает среднее значение
func (dc *DivergenceCalculator) calculateAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// calculateVolatility рассчитывает волатильность
func (dc *DivergenceCalculator) calculateVolatility(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}

	mean := dc.calculateAverage(values)
	var variance float64
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(values))

	return math.Sqrt(variance) / mean * 100 // возвращаем в процентах
}

// calculateCorrelation рассчитывает корреляцию между двумя наборами данных
func (dc *DivergenceCalculator) calculateCorrelation(x, y []float64) float64 {
	if len(x) != len(y) || len(x) < 2 {
		return 0
	}

	meanX := dc.calculateAverage(x)
	meanY := dc.calculateAverage(y)

	var numerator, denomX, denomY float64
	for i := 0; i < len(x); i++ {
		dx := x[i] - meanX
		dy := y[i] - meanY
		numerator += dx * dy
		denomX += dx * dx
		denomY += dy * dy
	}

	if denomX == 0 || denomY == 0 {
		return 0
	}

	return numerator / math.Sqrt(denomX*denomY)
}
