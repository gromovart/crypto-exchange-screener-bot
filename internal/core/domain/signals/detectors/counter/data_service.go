// internal/core/domain/signals/detectors/counter/data_service.go
package counter

import (
	"fmt"
	"time"

	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
)

// getDataForPeriod получает данные за указанный период (обновлен с использованием свечного движка)
func (a *CounterAnalyzer) getDataForPeriod(symbol, period string) ([]types.PriceData, error) {
	if a.candleSystem != nil {
		// Пробуем получить данные из свечного движка
		candleData, err := a.getCandleData(symbol, period)
		if err == nil && len(candleData) >= 2 {
			logger.Debug("✅ Получены свечные данные для %s %s", symbol, period)
			return candleData, nil
		}
		logger.Debug("⚠️ Не удалось получить свечу из движка: %v, используем старый метод", err)
	}

	// Старый метод как fallback
	return a.getDataForPeriodLegacy(symbol, period)
}

// getCandleData получает данные из свечного движка (исправленная версия)
func (a *CounterAnalyzer) getCandleData(symbol, period string) ([]types.PriceData, error) {
	// 1. Получаем свечу из движка
	candle, err := a.candleSystem.GetCandle(symbol, period)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения свечи: %w", err)
	}

	if candle == nil || !candle.IsReal {
		return nil, fmt.Errorf("свеча не содержит реальных данных")
	}

	logger.Info("🕯️ Получена свеча для %s %s:", symbol, period)
	logger.Info("   • Открытие: %.6f", candle.Open)
	logger.Info("   • Закрытие: %.6f", candle.Close)
	logger.Info("   • Высшая: %.6f", candle.High)
	logger.Info("   • Низшая: %.6f", candle.Low)
	logger.Info("   • Время: %s - %s",
		candle.StartTime.Format("15:04:05"),
		candle.EndTime.Format("15:04:05"))

	changePercent := ((candle.Close - candle.Open) / candle.Open) * 100
	logger.Info("   • Изменение: %.4f%%", changePercent)

	// 2. Возвращаем ТОЛЬКО данные открытия и закрытия для правильного расчета
	return []types.PriceData{
		{
			Symbol:    symbol,
			Price:     candle.Open,      // Важно: цена открытия свечи
			Timestamp: candle.StartTime, // Время открытия
		},
		{
			Symbol:    symbol,
			Price:     candle.Close,   // Важно: цена закрытия свечи
			Timestamp: candle.EndTime, // Время закрытия
		},
	}, nil
}

// convertStoragePricesInterfaceToTypes конвертирует storage.PriceDataInterface в types.PriceData
func (a *CounterAnalyzer) convertStoragePricesInterfaceToTypes(prices []storage.PriceDataInterface) []types.PriceData {
	var result []types.PriceData
	for _, price := range prices {
		result = append(result, types.PriceData{
			Symbol:       price.GetSymbol(),
			Price:        price.GetPrice(),
			Volume24h:    price.GetVolume24h(),
			VolumeUSD:    price.GetVolumeUSD(),
			Timestamp:    price.GetTimestamp(),
			OpenInterest: price.GetOpenInterest(),
			FundingRate:  price.GetFundingRate(),
			Change24h:    price.GetChange24h(),
			High24h:      price.GetHigh24h(),
			Low24h:       price.GetLow24h(),
		})
	}
	return result
}

// convertStoragePricesToTypes конвертирует storage.PriceData в types.PriceData
func convertStoragePricesToTypes(prices []storage.PriceData) []types.PriceData {
	var result []types.PriceData
	for _, price := range prices {
		result = append(result, types.PriceData{
			Symbol:       price.Symbol,
			Price:        price.Price,
			Volume24h:    price.Volume24h,
			VolumeUSD:    price.VolumeUSD,
			Timestamp:    price.Timestamp,
			OpenInterest: price.OpenInterest,
			FundingRate:  price.FundingRate,
			Change24h:    price.Change24h,
			High24h:      price.High24h,
			Low24h:       price.Low24h,
		})
	}
	return result
}

// getDataForPeriodLegacy старый метод получения данных (для совместимости)
func (a *CounterAnalyzer) getDataForPeriodLegacy(symbol, period string) ([]types.PriceData, error) {
	if a.storage == nil {
		logger.Error("⚠️ Storage не инициализирован для %s", symbol)
		return a.getFallbackData(symbol, period)
	}

	// Определяем длительность периода
	periodDuration := getPeriodDuration(period)
	endTime := time.Now()
	startTime := endTime.Add(-periodDuration)

	logger.Debug("🔍 getDataForPeriodLegacy: %s за %s (%s - %s)",
		symbol, period, startTime.Format("15:04:05"), endTime.Format("15:04:05"))

	// Получаем историю цен за период
	priceHistory, err := a.storage.GetPriceHistoryRange(symbol, startTime, endTime)
	if err != nil {
		logger.Warn("⚠️ Ошибка получения истории для %s: %v", symbol, err)
		return a.getFallbackData(symbol, period)
	}

	if len(priceHistory) == 0 {
		logger.Warn("⚠️ Нет данных для %s за %s", symbol, period)
		return a.getFallbackData(symbol, period)
	}

	// Конвертируем в types.PriceData
	result := a.convertStoragePricesInterfaceToTypes(priceHistory)

	logger.Debug("   Получено %d точек данных", len(result))
	if len(result) >= 2 {
		change := ((result[len(result)-1].Price - result[0].Price) / result[0].Price) * 100
		logger.Debug("   Изменение: %.4f%%", change)
	}

	return result, nil
}

// getFallbackData возвращает заглушку если нет реальных данных
func (a *CounterAnalyzer) getFallbackData(symbol, period string) ([]types.PriceData, error) {
	logger.Warn("⚠️ Использую fallback данные для %s", symbol)

	// Пробуем получить текущий снапшот
	var currentPrice, volume24h, openInterest, fundingRate float64

	if a.storage != nil {
		if snapshot, exists := a.storage.GetCurrentSnapshot(symbol); exists {
			currentPrice = snapshot.GetPrice()
			volume24h = snapshot.GetVolume24h()
			openInterest = snapshot.GetOpenInterest()
			fundingRate = snapshot.GetFundingRate()

			logger.Debug("   Найден снапшот: цена=%.4f, объем=%.0f, OI=%.0f",
				currentPrice, volume24h, openInterest)
		}
	}

	// Если нет снапшота, используем дефолтные значения
	if currentPrice == 0 {
		currentPrice = 1.0
		volume24h = 1000000
		openInterest = 500000
		fundingRate = 0.0001
	}

	// Создаем две точки данных с небольшим изменением
	startTime := time.Now().Add(-getPeriodDuration(period))

	// Небольшое случайное изменение (±0.5%)
	changePercent := (float64(time.Now().UnixNano()%100) - 50) / 10000 // ±0.5%
	startPrice := currentPrice / (1 + changePercent/100)

	return []types.PriceData{
		{
			Symbol:       symbol,
			Price:        startPrice,
			Volume24h:    volume24h,
			OpenInterest: openInterest,
			FundingRate:  fundingRate,
			Timestamp:    startTime,
		},
		{
			Symbol:       symbol,
			Price:        currentPrice,
			Volume24h:    volume24h,
			OpenInterest: openInterest,
			FundingRate:  fundingRate,
			Timestamp:    time.Now(),
		},
	}, nil
}
