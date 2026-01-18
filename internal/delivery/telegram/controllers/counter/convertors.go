// internal/delivery/telegram/controllers/counter/convertors.go
package counter

import (
	counterService "crypto-exchange-screener-bot/internal/delivery/telegram/services/counter"
	"crypto-exchange-screener-bot/internal/types"
	"fmt"
	"time"
)

// convertEventToParams преобразует событие в параметры сервиса
func convertEventToParams(event types.Event) (counterService.CounterParams, error) {
	dataMap, ok := event.Data.(map[string]interface{})
	if !ok {
		return counterService.CounterParams{}, fmt.Errorf("неверный формат данных события")
	}

	// Получаем Timestamp из события, если есть
	var timestamp time.Time
	if ts, ok := dataMap["timestamp"]; ok {
		switch v := ts.(type) {
		case time.Time:
			timestamp = v
		case string:
			// Пробуем распарсить строку
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				timestamp = t
			} else {
				timestamp = time.Now()
			}
		default:
			timestamp = time.Now()
		}
	} else {
		timestamp = time.Now()
	}

	params := counterService.CounterParams{
		// Базовые поля
		Symbol:        getString(dataMap, "symbol"),
		Direction:     getString(dataMap, "direction"),
		ChangePercent: getFloat64(dataMap, "change_percent"),
		Period:        getString(dataMap, "period_string"),
		Timestamp:     timestamp,
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
