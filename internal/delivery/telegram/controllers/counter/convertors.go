// internal/delivery/telegram/controllers/counter/convertors.go
package counter

import (
	counterService "crypto-exchange-screener-bot/internal/delivery/telegram/services/counter"
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
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

	// НОВОЕ: Извлекаем данные прогресса если есть
	if progress, ok := dataMap["progress"].(map[string]interface{}); ok {
		// Извлекаем данные прогресса
		if filled, ok := progress["filled_groups"]; ok {
			switch v := filled.(type) {
			case int:
				params.ProgressFilledGroups = v
			case float64:
				params.ProgressFilledGroups = int(v)
			}
		}

		if total, ok := progress["total_groups"]; ok {
			switch v := total.(type) {
			case int:
				params.ProgressTotalGroups = v
			case float64:
				params.ProgressTotalGroups = int(v)
			}
		}

		if percent, ok := progress["percentage"]; ok {
			if v, ok := percent.(float64); ok {
				params.ProgressPercentage = v
			}
		}

		logger.Warn("📊 CounterController: Извлечены данные прогресса: заполнено %d из %d (%.0f%%)",
			params.ProgressFilledGroups, params.ProgressTotalGroups, params.ProgressPercentage)
	}

	// После извлечения прогресса добавить:
	if params.ProgressFilledGroups > 0 || params.ProgressTotalGroups > 0 {
		logger.Warn("📊 CounterController: Извлечен прогресс из события: %d/%d групп (%.0f%%)",
			params.ProgressFilledGroups, params.ProgressTotalGroups, params.ProgressPercentage)
	} else {
		logger.Warn("⚠️ CounterController: Данные прогресса НЕ извлечены из события")

		// Логируем структуру данных для отладки
		if progress, ok := dataMap["progress"]; ok {
			logger.Warn("ℹ️ Структура progress в событии: %T = %+v", progress, progress)

			// Детальное логирование структуры
			if progressMap, ok := progress.(map[string]interface{}); ok {
				for key, val := range progressMap {
					logger.Warn("   • %s: %T = %v", key, val, val)
				}
			}
		} else {
			logger.Warn("ℹ️ Поле 'progress' отсутствует в данных события")

			// Логируем все ключи для отладки
			logger.Warn("ℹ️ Доступные поля в событии:")
			for key := range dataMap {
				logger.Warn("   • %s", key)
			}
		}
	}

	return params, nil
}
