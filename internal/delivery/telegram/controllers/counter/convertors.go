// internal/delivery/telegram/controllers/counter/convertors.go
package counter

import (
	counterService "crypto-exchange-screener-bot/internal/delivery/telegram/services/counter"
	"crypto-exchange-screener-bot/internal/types"
	"fmt"
	"log"
	"time"
)

// convertEventToParams преобразует событие в параметры сервиса
func convertEventToParams(event types.Event) (counterService.CounterParams, error) {
	log.Printf("🔍 convertEventToParams: преобразование события")

	dataMap, ok := event.Data.(map[string]interface{})
	if !ok {
		log.Printf("❌ convertEventToParams: не map[string]interface{}, а %T", event.Data)
		return counterService.CounterParams{}, fmt.Errorf("неверный формат данных события")
	}

	params := counterService.CounterParams{
		Event: event,
		// Базовые поля
		Symbol:        getString(dataMap, "symbol"),
		Direction:     getString(dataMap, "direction"),
		ChangePercent: getFloat64(dataMap, "change_percent"),
		Period:        getString(dataMap, "period_string"),
		Timestamp:     time.Now(),
	}

	// Опциональные поля
	if confirmations, ok := dataMap["confirmations"]; ok {
		switch v := confirmations.(type) {
		case int:
			params.Confirmations = v
		case float64:
			params.Confirmations = int(v)
		}
	}

	// Поля из indicators - поддерживаем ДВА формата:
	// 1. map[string]float64 (актуальный из логов)
	// 2. map[string]interface{} (для обратной совместимости)

	// Попробуем как map[string]float64
	if indicators, ok := dataMap["indicators"].(map[string]float64); ok {
		log.Printf("🔍 convertEventToParams: indicators тип: %T, значение: %v", indicators, indicators)
		params.CurrentPrice = getFloat64FromFloatMap(indicators, "current_price")
		params.Volume24h = getFloat64FromFloatMap(indicators, "volume_24h")
		params.OpenInterest = getFloat64FromFloatMap(indicators, "open_interest")
		params.FundingRate = getFloat64FromFloatMap(indicators, "funding_rate")
		params.RSI = getFloat64FromFloatMap(indicators, "rsi")
		params.MACDSignal = getFloat64FromFloatMap(indicators, "macd_signal")
		params.VolumeDelta = getFloat64FromFloatMap(indicators, "volume_delta")
		params.VolumeDeltaPercent = getFloat64FromFloatMap(indicators, "volume_delta_percent")
	} else if indicators, ok := dataMap["indicators"].(map[string]interface{}); ok {
		// Для обратной совместимости
		params.CurrentPrice = getFloat64FromMap(indicators, "current_price")
		params.Volume24h = getFloat64FromMap(indicators, "volume_24h")
		params.OpenInterest = getFloat64FromMap(indicators, "open_interest")
		params.FundingRate = getFloat64FromMap(indicators, "funding_rate")
		params.RSI = getFloat64FromMap(indicators, "rsi")
		params.MACDSignal = getFloat64FromMap(indicators, "macd_signal")
		params.VolumeDelta = getFloat64FromMap(indicators, "volume_delta")
		params.VolumeDeltaPercent = getFloat64FromMap(indicators, "volume_delta_percent")
	}

	return params, nil
}
