// internal/core/domain/signals/detectors/growth_analyzer/utils.go
package growth_analyzer

import (
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	"fmt"
	"math"
	"sort"
	"time"
)

// ValidatePriceData - валидирует данные о цене
func ValidatePriceData(data []redis_storage.PriceData) error {
	if len(data) == 0 {
		return fmt.Errorf("empty price data")
	}

	for i, point := range data {
		if point.Price <= 0 {
			return fmt.Errorf("invalid price for symbol %s at index %d", point.Symbol, i)
		}
		if point.Timestamp.IsZero() {
			return fmt.Errorf("invalid timestamp for symbol %s at index %d", point.Symbol, i)
		}
	}

	return nil
}

// SortPriceDataByTime - сортирует данные по времени
func SortPriceDataByTime(data []redis_storage.PriceData) []redis_storage.PriceData {
	sortedData := make([]redis_storage.PriceData, len(data))
	copy(sortedData, data)

	sort.Slice(sortedData, func(i, j int) bool {
		return sortedData[i].Timestamp.Before(sortedData[j].Timestamp)
	})

	return sortedData
}

// CalculateMovingAverage - рассчитывает скользящую среднюю
func CalculateMovingAverage(data []redis_storage.PriceData, period int) []float64 {
	if len(data) < period || period <= 0 {
		return make([]float64, len(data))
	}

	ma := make([]float64, len(data))
	for i := 0; i < len(data); i++ {
		if i < period-1 {
			ma[i] = data[i].Price
			continue
		}

		var sum float64
		for j := 0; j < period; j++ {
			sum += data[i-j].Price
		}
		ma[i] = sum / float64(period)
	}

	return ma
}

// CalculateRSI - рассчитывает индекс относительной силы
func CalculateRSI(data []redis_storage.PriceData, period int) []float64 {
	if len(data) < period+1 || period <= 0 {
		return make([]float64, len(data))
	}

	gains := make([]float64, len(data))
	losses := make([]float64, len(data))

	// Рассчитываем изменения
	for i := 1; i < len(data); i++ {
		change := data[i].Price - data[i-1].Price
		if change > 0 {
			gains[i] = change
		} else {
			losses[i] = -change
		}
	}

	// Рассчитываем RSI
	rsi := make([]float64, len(data))
	for i := period; i < len(data); i++ {
		avgGain := calculateAverage(gains[i-period+1:i+1], period)
		avgLoss := calculateAverage(losses[i-period+1:i+1], period)

		if avgLoss == 0 {
			rsi[i] = 100
		} else {
			rs := avgGain / avgLoss
			rsi[i] = 100 - (100 / (1 + rs))
		}
	}

	// Заполняем начальные значения
	for i := 0; i < period; i++ {
		rsi[i] = 50 // нейтральное значение
	}

	return rsi
}

// CalculateStandardDeviation - рассчитывает стандартное отклонение
func CalculateStandardDeviation(data []redis_storage.PriceData) float64 {
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
		variance += (point.Price - mean) * (point.Price - mean)
	}
	variance /= float64(len(data))

	return math.Sqrt(variance)
}

// FilterOutliers - фильтрует выбросы в данных
func FilterOutliers(data []redis_storage.PriceData, threshold float64) []redis_storage.PriceData {
	if len(data) < 3 {
		return data
	}

	stdDev := CalculateStandardDeviation(data)
	mean := calculateMean(data)

	filtered := make([]redis_storage.PriceData, 0, len(data))
	for _, point := range data {
		if math.Abs(point.Price-mean) <= threshold*stdDev {
			filtered = append(filtered, point)
		}
	}

	return filtered
}

// CalculatePriceChange - рассчитывает изменение цены между двумя точками
func CalculatePriceChange(startPrice, endPrice float64) float64 {
	if startPrice == 0 {
		return 0.0
	}
	return ((endPrice - startPrice) / startPrice) * 100
}

// GroupDataByTimeInterval - группирует данные по временным интервалам
func GroupDataByTimeInterval(data []redis_storage.PriceData, interval time.Duration) [][]redis_storage.PriceData {
	if len(data) == 0 {
		return [][]redis_storage.PriceData{}
	}

	sortedData := SortPriceDataByTime(data)
	var groups [][]redis_storage.PriceData
	var currentGroup []redis_storage.PriceData
	var groupStartTime time.Time

	for i, point := range sortedData {
		if i == 0 {
			groupStartTime = point.Timestamp
			currentGroup = []redis_storage.PriceData{point}
			continue
		}

		if point.Timestamp.Sub(groupStartTime) <= interval {
			currentGroup = append(currentGroup, point)
		} else {
			groups = append(groups, currentGroup)
			groupStartTime = point.Timestamp
			currentGroup = []redis_storage.PriceData{point}
		}
	}

	// Добавляем последнюю группу
	if len(currentGroup) > 0 {
		groups = append(groups, currentGroup)
	}

	return groups
}

// CalculateVolumeWeightedAveragePrice - рассчитывает среднюю цену, взвешенную по объему
func CalculateVolumeWeightedAveragePrice(data []redis_storage.PriceData) float64 {
	if len(data) == 0 {
		return 0.0
	}

	var totalValue float64
	var totalVolume float64

	for _, point := range data {
		// Используем VolumeUSD если доступно, иначе Volume24h
		volume := point.VolumeUSD
		if volume == 0 {
			volume = point.Volume24h
		}

		totalValue += point.Price * volume
		totalVolume += volume
	}

	if totalVolume == 0 {
		// Если объемы отсутствуют, возвращаем простое среднее
		return calculateMean(data)
	}

	return totalValue / totalVolume
}

// Helper functions

func calculateAverage(values []float64, period int) float64 {
	var sum float64
	count := 0
	for _, v := range values {
		if v > 0 {
			sum += v
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return sum / float64(period)
}

func calculateMean(data []redis_storage.PriceData) float64 {
	var sum float64
	for _, point := range data {
		sum += point.Price
	}
	return sum / float64(len(data))
}
