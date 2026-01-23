// internal/core/domain/signals/detectors/counter/data_interpolator.go
package counter

import (
	"fmt"
	"sort"
	"time"

	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	"crypto-exchange-screener-bot/internal/types"
)

// GetInterpolatedData создает интерполированные данные если недостаточно точек
func GetInterpolatedData(symbol, period string,
	existingData []storage.PriceDataInterface, requiredPoints int) ([]types.PriceData, error) {

	if len(existingData) == 0 {
		return nil, fmt.Errorf("нет данных для интерполяции")
	}

	// Если есть только 1 точка, создаем небольшой тренд
	if len(existingData) == 1 {
		return createTrendFromSinglePoint(symbol, existingData[0], requiredPoints), nil
	}

	// Линейная интерполяция между существующими точками
	return linearInterpolation(symbol, existingData, requiredPoints), nil
}

// createTrendFromSinglePoint создает тренд из одной точки
func createTrendFromSinglePoint(symbol string, point storage.PriceDataInterface, requiredPoints int) []types.PriceData {
	var result []types.PriceData
	// Создаем небольшой восходящий тренд (+0.01% на точку)
	trendPercent := 0.0001

	for i := 0; i < requiredPoints; i++ {
		priceMultiplier := 1.0 + (float64(i) * trendPercent)
		noise := (float64(time.Now().UnixNano()%100) - 50.0) / 1000000.0

		result = append(result, types.PriceData{
			Symbol:       symbol,
			Price:        point.GetPrice()*priceMultiplier + noise,
			Volume24h:    point.GetVolume24h(),
			OpenInterest: point.GetOpenInterest(),
			FundingRate:  point.GetFundingRate(),
			Timestamp:    point.GetTimestamp().Add(time.Duration(i) * time.Minute),
			Change24h:    point.GetChange24h(),
			High24h:      point.GetHigh24h() * priceMultiplier,
			Low24h:       point.GetLow24h() * priceMultiplier,
		})
	}
	return result
}

// linearInterpolation выполняет линейную интерполяцию между точками
func linearInterpolation(symbol string, existingData []storage.PriceDataInterface, requiredPoints int) []types.PriceData {
	var result []types.PriceData

	// Сортируем по времени
	sort.Slice(existingData, func(i, j int) bool {
		return existingData[i].GetTimestamp().Before(existingData[j].GetTimestamp())
	})

	// Временной диапазон и шаг
	timeRange := existingData[len(existingData)-1].GetTimestamp().Sub(existingData[0].GetTimestamp())
	if timeRange <= 0 {
		timeRange = time.Duration(requiredPoints) * time.Minute
	}
	timeStep := timeRange / time.Duration(requiredPoints-1)

	// Интерполяция
	for i := 0; i < requiredPoints; i++ {
		currentTime := existingData[0].GetTimestamp().Add(timeStep * time.Duration(i))
		dataPoint := interpolatePoint(symbol, existingData, currentTime)
		result = append(result, dataPoint)
	}

	return result
}

// interpolatePoint интерполирует точку в заданное время
func interpolatePoint(symbol string, existingData []storage.PriceDataInterface, targetTime time.Time) types.PriceData {
	// Находим ближайшие точки
	prev, next := findNearestPoints(existingData, targetTime)

	var price, volume, oi, funding float64
	var timestamp time.Time

	if prev != nil && next != nil {
		// Линейная интерполяция
		timeRatio := float64(targetTime.Sub(prev.GetTimestamp())) / float64(next.GetTimestamp().Sub(prev.GetTimestamp()))
		price = prev.GetPrice() + (next.GetPrice()-prev.GetPrice())*timeRatio
		volume = prev.GetVolume24h() + (next.GetVolume24h()-prev.GetVolume24h())*timeRatio
		oi = prev.GetOpenInterest() + (next.GetOpenInterest()-prev.GetOpenInterest())*timeRatio
		funding = prev.GetFundingRate() + (next.GetFundingRate()-prev.GetFundingRate())*timeRatio
		timestamp = targetTime
	} else {
		// Используем ближайшую точку
		if targetTime.Before(existingData[0].GetTimestamp()) {
			point := existingData[0]
			price = point.GetPrice()
			timestamp = point.GetTimestamp()
		} else {
			point := existingData[len(existingData)-1]
			price = point.GetPrice()
			timestamp = point.GetTimestamp()
		}
		volume = existingData[0].GetVolume24h()
		oi = existingData[0].GetOpenInterest()
		funding = existingData[0].GetFundingRate()
	}

	return types.PriceData{
		Symbol:       symbol,
		Price:        price,
		Volume24h:    volume,
		OpenInterest: oi,
		FundingRate:  funding,
		Timestamp:    timestamp,
		Change24h:    existingData[0].GetChange24h(),
		High24h:      existingData[0].GetHigh24h(),
		Low24h:       existingData[0].GetLow24h(),
	}
}

// findNearestPoints находит две ближайшие точки для интерполяции
func findNearestPoints(data []storage.PriceDataInterface, targetTime time.Time) (storage.PriceDataInterface, storage.PriceDataInterface) {
	for j := 0; j < len(data)-1; j++ {
		if !data[j].GetTimestamp().After(targetTime) && data[j+1].GetTimestamp().After(targetTime) {
			return data[j], data[j+1]
		}
	}
	return nil, nil
}
