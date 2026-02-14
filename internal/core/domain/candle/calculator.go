// internal/core/domain/candle/calculator.go
package candle

import (
	"sort"
	"time"

	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	"crypto-exchange-screener-bot/pkg/period"
)

// CandleCalculator - калькулятор для свечей
type CandleCalculator struct {
	storage storage.PriceStorageInterface
}

// NewCandleCalculator создает новый калькулятор свечей
func NewCandleCalculator(priceStorage storage.PriceStorageInterface) *CandleCalculator {
	return &CandleCalculator{
		storage: priceStorage,
	}
}

// BuildCandleFromHistory строит свечу из истории цен
func (cc *CandleCalculator) BuildCandleFromHistory(symbol, periodStr string) (*storage.Candle, error) {
	// Определяем период через универсальную функцию
	duration := period.PeriodToDuration(periodStr)
	endTime := time.Now()
	startTime := endTime.Add(-duration)

	// Получаем цены за период
	prices, err := cc.storage.GetPriceHistoryRange(symbol, startTime, endTime)
	if err != nil {
		return nil, err
	}

	if len(prices) == 0 {
		// Если нет данных, возвращаем пустую свечу
		return &storage.Candle{
			Symbol:       symbol,
			Period:       periodStr,
			StartTime:    startTime,
			EndTime:      endTime,
			IsClosedFlag: true,
			IsRealFlag:   false,
		}, nil
	}

	// Строим свечу
	return cc.buildCandleFromPriceData(symbol, periodStr, prices), nil
}

// buildCandleFromPriceData строит свечу из массива PriceDataInterface
func (cc *CandleCalculator) buildCandleFromPriceData(symbol, periodStr string,
	prices []storage.PriceDataInterface) *storage.Candle {

	// Сортируем цены по времени
	sortedPrices := sortPriceDataInterfaceByTime(prices)

	// Если после сортировки нет данных
	if len(sortedPrices) == 0 {
		return &storage.Candle{
			Symbol:       symbol,
			Period:       periodStr,
			IsClosedFlag: true,
			IsRealFlag:   false,
		}
	}

	// Начальные значения
	open := sortedPrices[0].GetPrice()
	close := sortedPrices[len(sortedPrices)-1].GetPrice()
	high := open
	low := open

	var volume, volumeUSD float64
	startTime := sortedPrices[0].GetTimestamp()
	endTime := sortedPrices[len(sortedPrices)-1].GetTimestamp()

	// Рассчитываем OHLCV
	for _, price := range sortedPrices {
		priceVal := price.GetPrice()
		if priceVal > high {
			high = priceVal
		}
		if priceVal < low {
			low = priceVal
		}
		volume += price.GetVolume24h()
		volumeUSD += price.GetVolumeUSD()

		// Обновляем временные границы
		timestamp := price.GetTimestamp()
		if timestamp.Before(startTime) {
			startTime = timestamp
		}
		if timestamp.After(endTime) {
			endTime = timestamp
		}
	}

	// Проверяем, покрывает ли данные весь период
	duration := period.PeriodToDuration(periodStr)
	minDuration := duration * 8 / 10 // 80% от периода
	coversFullPeriod := endTime.Sub(startTime) >= minDuration

	return &storage.Candle{
		Symbol:       symbol,
		Period:       periodStr,
		Open:         open,
		High:         high,
		Low:          low,
		Close:        close,
		Volume:       volume,
		VolumeUSD:    volumeUSD,
		Trades:       len(sortedPrices),
		StartTime:    startTime,
		EndTime:      endTime,
		IsClosedFlag: true,
		IsRealFlag:   coversFullPeriod,
	}
}

// CalculateChangePercent рассчитывает процент изменения свечи
func (cc *CandleCalculator) CalculateChangePercent(candle *storage.Candle) float64 {
	if candle.Open == 0 {
		return 0
	}
	return ((candle.Close - candle.Open) / candle.Open) * 100
}

// CalculateAverageCandle строит среднюю свечу из массива свечей
func (cc *CandleCalculator) CalculateAverageCandle(candles []*storage.Candle) *storage.Candle {
	if len(candles) == 0 {
		return nil
	}

	var totalOpen, totalHigh, totalLow, totalClose float64
	var totalVolume, totalVolumeUSD float64
	var totalTrades int

	for _, candle := range candles {
		totalOpen += candle.Open
		totalHigh += candle.High
		totalLow += candle.Low
		totalClose += candle.Close
		totalVolume += candle.Volume
		totalVolumeUSD += candle.VolumeUSD
		totalTrades += candle.Trades
	}

	count := float64(len(candles))

	return &storage.Candle{
		Symbol:       candles[0].Symbol,
		Period:       candles[0].Period,
		Open:         totalOpen / count,
		High:         totalHigh / count,
		Low:          totalLow / count,
		Close:        totalClose / count,
		Volume:       totalVolume / count,
		VolumeUSD:    totalVolumeUSD / count,
		Trades:       totalTrades / len(candles),
		StartTime:    candles[0].StartTime,
		EndTime:      candles[len(candles)-1].EndTime,
		IsClosedFlag: true,
		IsRealFlag:   true,
	}
}

// MergeCandles объединяет свечи в более крупный период
func (cc *CandleCalculator) MergeCandles(candles []*storage.Candle, targetPeriod string) *storage.Candle {
	if len(candles) == 0 {
		return nil
	}

	// Сортируем свечи по времени
	sort.Slice(candles, func(i, j int) bool {
		return candles[i].StartTime.Before(candles[j].StartTime)
	})

	// Определяем общие значения
	open := candles[0].Open
	close := candles[len(candles)-1].Close
	high := candles[0].High
	low := candles[0].Low

	var volume, volumeUSD float64
	var totalTrades int
	startTime := candles[0].StartTime
	endTime := candles[len(candles)-1].EndTime

	// Находим экстремумы и суммируем объемы
	for _, candle := range candles {
		if candle.High > high {
			high = candle.High
		}
		if candle.Low < low {
			low = candle.Low
		}
		volume += candle.Volume
		volumeUSD += candle.VolumeUSD
		totalTrades += candle.Trades
	}

	return &storage.Candle{
		Symbol:       candles[0].Symbol,
		Period:       targetPeriod,
		Open:         open,
		High:         high,
		Low:          low,
		Close:        close,
		Volume:       volume,
		VolumeUSD:    volumeUSD,
		Trades:       totalTrades,
		StartTime:    startTime,
		EndTime:      endTime,
		IsClosedFlag: true,
		IsRealFlag:   true,
	}
}

// AnalyzeCandleTrend анализирует тренд свечи
func (cc *CandleCalculator) AnalyzeCandleTrend(candle *storage.Candle) string {
	changePercent := cc.CalculateChangePercent(candle)

	if changePercent > 5.0 {
		return "strong_bullish"
	} else if changePercent > 1.0 {
		return "bullish"
	} else if changePercent < -5.0 {
		return "strong_bearish"
	} else if changePercent < -1.0 {
		return "bearish"
	} else {
		return "neutral"
	}
}

// sortPriceDataInterfaceByTime сортирует интерфейсы цен по времени
func sortPriceDataInterfaceByTime(prices []storage.PriceDataInterface) []storage.PriceDataInterface {
	sorted := make([]storage.PriceDataInterface, len(prices))
	copy(sorted, prices)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].GetTimestamp().Before(sorted[j].GetTimestamp())
	})

	return sorted
}
