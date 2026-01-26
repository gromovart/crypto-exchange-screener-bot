// internal/core/domain/signals/detectors/fall_analyzer/calculator/fall_calculator.go
package calculator

import (
	"math"
	"time"

	"crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	"crypto-exchange-screener-bot/pkg/logger"
)

// FallCalculator - калькулятор для анализа падений
type FallCalculator struct{}

// NewFallCalculator создает новый калькулятор падений
func NewFallCalculator() *FallCalculator {
	return &FallCalculator{}
}

// FallConfigForCalculator - конфигурация для расчета падений
type FallConfigForCalculator struct {
	MinConfidence       float64
	MinFall             float64
	ContinuityThreshold float64
	VolumeWeight        float64
}

// FallResult - результат расчета падений
type FallResult struct {
	Symbol        string
	Type          string
	Direction     string
	ChangePercent float64
	Confidence    float64
	Period        int
	DataPoints    int
	StartPrice    float64
	EndPrice      float64
	Volume        float64
	IsContinuous  bool
	Indicators    map[string]float64
}

// AnalyzeFalls анализирует падения в данных
func (fc *FallCalculator) AnalyzeFalls(data []redis_storage.PriceData, config FallConfigForCalculator) []*FallResult {
	if len(data) < 2 {
		logger.Debug("📭 FallCalculator: недостаточно точек для анализа падений (%d < 2)", len(data))
		return nil
	}

	var results []*FallResult

	// 1. Анализ последовательных падений
	if singleFalls := fc.analyzeSingleFalls(data, config); singleFalls != nil {
		results = append(results, singleFalls...)
	}

	// 2. Анализ интервальных падений
	if len(data) >= 3 {
		if intervalFalls := fc.analyzeIntervalFalls(data, config); intervalFalls != nil {
			results = append(results, intervalFalls...)
		}
	}

	// 3. Анализ непрерывных падений
	if len(data) >= 3 {
		if continuousFalls := fc.analyzeContinuousFalls(data, config); continuousFalls != nil {
			results = append(results, continuousFalls...)
		}
	}

	return results
}

// analyzeSingleFalls анализирует последовательные падения
func (fc *FallCalculator) analyzeSingleFalls(data []redis_storage.PriceData, config FallConfigForCalculator) []*FallResult {
	var results []*FallResult

	for i := 1; i < len(data); i++ {
		change := ((data[i].Price - data[i-1].Price) / data[i-1].Price) * 100

		// Проверяем падение
		if change < 0 && math.Abs(change) >= config.MinFall {
			confidence := fc.calculateSingleFallConfidence(data[i-1:i+1], math.Abs(change), config)

			if confidence >= config.MinConfidence {
				result := &FallResult{
					Symbol:        data[0].Symbol,
					Type:          "single_fall",
					Direction:     "down",
					ChangePercent: change,
					Confidence:    confidence,
					Period:        int(data[i].Timestamp.Sub(data[i-1].Timestamp).Minutes()),
					DataPoints:    2,
					StartPrice:    data[i-1].Price,
					EndPrice:      data[i].Price,
					Volume:        (data[i-1].Volume24h + data[i].Volume24h) / 2,
					IsContinuous:  false,
					Indicators:    make(map[string]float64),
				}

				// Заполняем индикаторы
				result.Indicators["price_change"] = change
				result.Indicators["fall_value"] = math.Abs(change)
				result.Indicators["duration_minutes"] = data[i].Timestamp.Sub(data[i-1].Timestamp).Minutes()
				result.Indicators["volume_avg"] = result.Volume
				result.Indicators["trend_strength"] = fc.calculateTrendStrength(data[i-1 : i+1])
				result.Indicators["volatility"] = fc.calculateVolatility(data[i-1 : i+1])

				results = append(results, result)
			}
		}
	}

	return results
}

// analyzeIntervalFalls анализирует интервальные падения
func (fc *FallCalculator) analyzeIntervalFalls(data []redis_storage.PriceData, config FallConfigForCalculator) []*FallResult {
	var results []*FallResult

	// Находим максимальные падения между любыми точками
	for i := 0; i < len(data); i++ {
		for j := i + 1; j < len(data); j++ {
			change := ((data[j].Price - data[i].Price) / data[i].Price) * 100
			fallValue := math.Abs(change)

			if change < 0 && fallValue >= config.MinFall {
				// Проверяем уникальность (не должно быть более короткого интервала с таким же падением)
				isUnique := true
				for k := i + 1; k < j; k++ {
					partialChange := ((data[j].Price - data[k].Price) / data[k].Price) * 100
					if math.Abs(partialChange) >= fallValue*0.9 {
						isUnique = false
						break
					}
				}

				if isUnique {
					confidence := fc.calculateIntervalConfidence(data[i:j+1], fallValue, config)

					if confidence >= config.MinConfidence {
						// Проверяем непрерывность
						isContinuous := fc.checkContinuity(data[i:j+1], config.ContinuityThreshold)

						result := &FallResult{
							Symbol:        data[0].Symbol,
							Type:          "interval_fall",
							Direction:     "down",
							ChangePercent: change,
							Confidence:    confidence,
							Period:        int(data[j].Timestamp.Sub(data[i].Timestamp).Minutes()),
							DataPoints:    j - i + 1,
							StartPrice:    data[i].Price,
							EndPrice:      data[j].Price,
							Volume:        fc.calculateAverageVolume(data[i : j+1]),
							IsContinuous:  isContinuous,
							Indicators:    make(map[string]float64),
						}

						// Заполняем индикаторы
						result.Indicators["price_change"] = change
						result.Indicators["fall_value"] = fallValue
						result.Indicators["duration_minutes"] = data[j].Timestamp.Sub(data[i].Timestamp).Minutes()
						result.Indicators["data_points"] = float64(j - i + 1)
						result.Indicators["volume_avg"] = result.Volume
						result.Indicators["trend_strength"] = fc.calculateTrendStrength(data[i : j+1])
						result.Indicators["volatility"] = fc.calculateVolatility(data[i : j+1])
						result.Indicators["continuity_ratio"] = fc.calculateContinuityRatio(data[i : j+1])
						result.Indicators["is_continuous"] = 0
						if isContinuous {
							result.Indicators["is_continuous"] = 1
						}

						results = append(results, result)
					}
				}
			}
		}
	}

	return results
}

// analyzeContinuousFalls анализирует непрерывные падения
func (fc *FallCalculator) analyzeContinuousFalls(data []redis_storage.PriceData, config FallConfigForCalculator) []*FallResult {
	var results []*FallResult

	// Ищем последовательности непрерывных падений
	for i := 0; i < len(data)-2; i++ {
		for j := i + 2; j < len(data); j++ {
			segment := data[i : j+1]

			// Проверяем непрерывность
			if fc.checkContinuity(segment, config.ContinuityThreshold) {
				// Рассчитываем общее падение в сегменте
				totalChange := ((segment[len(segment)-1].Price - segment[0].Price) / segment[0].Price) * 100

				if totalChange < 0 && math.Abs(totalChange) >= config.MinFall {
					confidence := fc.calculateContinuousFallConfidence(segment, math.Abs(totalChange), config)

					if confidence >= config.MinConfidence {
						result := &FallResult{
							Symbol:        data[0].Symbol,
							Type:          "continuous_fall",
							Direction:     "down",
							ChangePercent: totalChange,
							Confidence:    confidence,
							Period:        int(segment[len(segment)-1].Timestamp.Sub(segment[0].Timestamp).Minutes()),
							DataPoints:    len(segment),
							StartPrice:    segment[0].Price,
							EndPrice:      segment[len(segment)-1].Price,
							Volume:        fc.calculateAverageVolume(segment),
							IsContinuous:  true,
							Indicators:    make(map[string]float64),
						}

						// Заполняем индикаторы
						result.Indicators["price_change"] = totalChange
						result.Indicators["fall_value"] = math.Abs(totalChange)
						result.Indicators["duration_minutes"] = segment[len(segment)-1].Timestamp.Sub(segment[0].Timestamp).Minutes()
						result.Indicators["data_points"] = float64(len(segment))
						result.Indicators["continuous_points"] = float64(fc.countContinuousPoints(segment))
						result.Indicators["volume_avg"] = result.Volume
						result.Indicators["trend_strength"] = fc.calculateTrendStrength(segment)
						result.Indicators["volatility"] = fc.calculateVolatility(segment)
						result.Indicators["continuity_ratio"] = fc.calculateContinuityRatio(segment)

						results = append(results, result)

						// Пропускаем часть сегмента чтобы избежать дублирования
						i = j
						break
					}
				}
			}
		}
	}

	return results
}

// calculateSingleFallConfidence рассчитывает уверенность для одиночного падения
func (fc *FallCalculator) calculateSingleFallConfidence(data []redis_storage.PriceData, fallPercent float64, config FallConfigForCalculator) float64 {
	if len(data) < 2 {
		return 0.0
	}

	baseConfidence := math.Min(fallPercent*10, 70)

	// Фактор объема
	volumeFactor := fc.calculateVolumeFactor(data, config.VolumeWeight)

	// Фактор времени
	timeFactor := fc.calculateTimeFactor(data[0].Timestamp, data[1].Timestamp)

	// Фактор волатильности
	volatility := fc.calculateVolatility(data)
	volatilityFactor := 0.0
	if volatility < 2.0 {
		volatilityFactor = 10.0
	} else if volatility > 5.0 {
		volatilityFactor = -5.0
	}

	confidence := baseConfidence + volumeFactor + timeFactor + volatilityFactor
	return math.Max(0, math.Min(100, confidence))
}

// calculateIntervalConfidence рассчитывает уверенность для интервального падения
func (fc *FallCalculator) calculateIntervalConfidence(data []redis_storage.PriceData, fallPercent float64, config FallConfigForCalculator) float64 {
	if len(data) < 2 {
		return 0.0
	}

	baseConfidence := math.Min(fallPercent*8, 80)

	// Бонус за непрерывность
	continuityBonus := 0.0
	if fc.checkContinuity(data, config.ContinuityThreshold) {
		continuityBonus = 20.0
	}

	// Фактор силы тренда
	trendStrength := fc.calculateTrendStrength(data)
	trendFactor := math.Min(trendStrength/2, 10)

	// Фактор объема
	volumeFactor := fc.calculateVolumeFactor(data, config.VolumeWeight)

	confidence := baseConfidence + continuityBonus + trendFactor + volumeFactor
	return math.Max(0, math.Min(100, confidence))
}

// calculateContinuousFallConfidence рассчитывает уверенность для непрерывного падения
func (fc *FallCalculator) calculateContinuousFallConfidence(data []redis_storage.PriceData, fallPercent float64, config FallConfigForCalculator) float64 {
	if len(data) < 2 {
		return 0.0
	}

	baseConfidence := math.Min(fallPercent*12, 90)

	// Бонус за количество непрерывных точек
	continuousPoints := fc.countContinuousPoints(data)
	pointsBonus := math.Min(float64(continuousPoints)*3, 15)

	// Фактор силы тренда
	trendStrength := fc.calculateTrendStrength(data)
	trendFactor := math.Min(trendStrength/2, 10)

	// Фактор объема
	volumeFactor := fc.calculateVolumeFactor(data, config.VolumeWeight)

	confidence := baseConfidence + pointsBonus + trendFactor + volumeFactor
	return math.Max(0, math.Min(100, confidence))
}

// calculateVolumeFactor рассчитывает фактор объема
func (fc *FallCalculator) calculateVolumeFactor(data []redis_storage.PriceData, volumeWeight float64) float64 {
	if len(data) == 0 || volumeWeight <= 0 {
		return 0.0
	}

	avgVolume := fc.calculateAverageVolume(data)
	volumeFactor := 0.0

	if avgVolume > 1000000 {
		volumeFactor = 10.0
	} else if avgVolume < 100000 {
		volumeFactor = -5.0
	}

	return volumeFactor * volumeWeight
}

// calculateTimeFactor рассчитывает фактор времени
func (fc *FallCalculator) calculateTimeFactor(startTime, endTime time.Time) float64 {
	timeDiff := endTime.Sub(startTime).Minutes()

	if timeDiff < 5 {
		return 15.0
	} else if timeDiff > 30 {
		return -10.0
	}

	return 0.0
}

// calculateTrendStrength рассчитывает силу тренда
func (fc *FallCalculator) calculateTrendStrength(data []redis_storage.PriceData) float64 {
	if len(data) < 2 {
		return 0.0
	}

	var sumX, sumY, sumXY, sumX2 float64
	n := float64(len(data))

	for i, point := range data {
		x := float64(i)
		y := point.Price
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	b := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	trendStrength := math.Abs(b) * 1000
	return math.Max(0, math.Min(100, trendStrength))
}

// calculateVolatility рассчитывает волатильность
func (fc *FallCalculator) calculateVolatility(data []redis_storage.PriceData) float64 {
	if len(data) < 2 {
		return 0.0
	}

	var sum float64
	for _, point := range data {
		sum += point.Price
	}
	mean := sum / float64(len(data))

	var variance float64
	for _, point := range data {
		diff := point.Price - mean
		variance += diff * diff
	}
	variance /= float64(len(data))

	return math.Sqrt(variance) / mean * 100
}

// calculateAverageVolume рассчитывает средний объем
func (fc *FallCalculator) calculateAverageVolume(data []redis_storage.PriceData) float64 {
	if len(data) == 0 {
		return 0.0
	}

	var sum float64
	for _, point := range data {
		sum += point.Volume24h
	}
	return sum / float64(len(data))
}

// checkContinuity проверяет непрерывность падения
func (fc *FallCalculator) checkContinuity(data []redis_storage.PriceData, threshold float64) bool {
	if len(data) < 2 {
		return false
	}

	continuousPoints := 0
	totalPoints := len(data) - 1

	for i := 1; i < len(data); i++ {
		if data[i].Price < data[i-1].Price {
			continuousPoints++
		}
	}

	if totalPoints == 0 {
		return false
	}

	continuousRatio := float64(continuousPoints) / float64(totalPoints)
	return continuousRatio > threshold
}

// calculateContinuityRatio рассчитывает коэффициент непрерывности
func (fc *FallCalculator) calculateContinuityRatio(data []redis_storage.PriceData) float64 {
	if len(data) < 2 {
		return 0.0
	}

	continuousPoints := 0
	totalPoints := len(data) - 1

	for i := 1; i < len(data); i++ {
		if data[i].Price < data[i-1].Price {
			continuousPoints++
		}
	}

	if totalPoints == 0 {
		return 0.0
	}

	return float64(continuousPoints) / float64(totalPoints)
}

// countContinuousPoints подсчитывает количество непрерывных падений
func (fc *FallCalculator) countContinuousPoints(data []redis_storage.PriceData) int {
	if len(data) < 2 {
		return 0
	}

	count := 0
	for i := 1; i < len(data); i++ {
		if data[i].Price < data[i-1].Price {
			count++
		}
	}
	return count
}
