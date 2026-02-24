// internal/delivery/telegram/controllers/counter/convertors.go
package counter

import (
	counterService "crypto-exchange-screener-bot/internal/delivery/telegram/services/counter"
	"crypto-exchange-screener-bot/internal/types"
	periodPkg "crypto-exchange-screener-bot/pkg/period"
	"fmt"
	"time"
)

// convertEventToParams преобразует событие в параметры сервиса
func convertEventToParams(event types.Event) (counterService.CounterParams, error) {
	dataMap, ok := event.Data.(map[string]interface{})
	if !ok {
		return counterService.CounterParams{}, fmt.Errorf("неверный формат данных события")
	}

	// Получаем Timestamp из события
	var timestamp time.Time
	if ts, ok := dataMap["timestamp"]; ok {
		switch v := ts.(type) {
		case time.Time:
			timestamp = v
		case string:
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

	// Получаем и нормализуем период
	period := getString(dataMap, "period")
	if !periodPkg.IsValidPeriod(period) {
		// Если период невалидный, логируем и используем дефолтный
		period = periodPkg.DefaultPeriod
	}

	params := counterService.CounterParams{
		// Базовые поля
		Symbol:        getString(dataMap, "symbol"),
		Direction:     getString(dataMap, "direction"),
		ChangePercent: getFloat64(dataMap, "change_percent"),
		Period:        period, // Используем нормализованный период
		Timestamp:     timestamp,
	}

	// Confirmations из верхнего уровня
	if confirmations, ok := dataMap["confirmations"]; ok {
		switch v := confirmations.(type) {
		case int:
			params.Confirmations = v
		case float64:
			params.Confirmations = int(v)
		}
	}

	// ВСЕ индикаторы из верхнего уровня flat map
	params.CurrentPrice = getFloat64(dataMap, "current_price")
	params.Volume24h = getFloat64(dataMap, "volume_24h")
	params.OpenInterest = getFloat64(dataMap, "open_interest")
	params.FundingRate = getFloat64(dataMap, "funding_rate")
	params.RSI = getFloat64(dataMap, "rsi")
	params.MACDSignal = getFloat64(dataMap, "macd_signal")
	params.VolumeDelta = getFloat64(dataMap, "volume_delta")
	params.VolumeDeltaPercent = getFloat64(dataMap, "volume_delta_percent")

	// Прогресс - вложенный в "progress"
	if progress, ok := dataMap["progress"].(map[string]interface{}); ok {
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
	}

	// Зоны S/R
	params.SRSupportPrice = getFloat64(dataMap, "sr_support_price")
	params.SRSupportStrength = getFloat64(dataMap, "sr_support_strength")
	params.SRSupportDistPct = getFloat64(dataMap, "sr_support_dist_pct")
	params.SRSupportHasWall = getBool(dataMap, "sr_support_has_wall")
	params.SRSupportWallUSD = getFloat64(dataMap, "sr_support_wall_usd")
	params.SRResistancePrice = getFloat64(dataMap, "sr_resistance_price")
	params.SRResistanceStrength = getFloat64(dataMap, "sr_resistance_strength")
	params.SRResistanceDistPct = getFloat64(dataMap, "sr_resistance_dist_pct")
	params.SRResistanceHasWall = getBool(dataMap, "sr_resistance_has_wall")
	params.SRResistanceWallUSD = getFloat64(dataMap, "sr_resistance_wall_usd")

	return params, nil
}
