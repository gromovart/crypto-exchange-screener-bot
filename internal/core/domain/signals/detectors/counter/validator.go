// internal/core/domain/signals/detectors/counter/validator.go
package counter

import (
	"fmt"
	"math"
	"strings"

	candle "crypto-exchange-screener-bot/internal/core/domain/candle"
)

// TestCandleAccuracy тестирует точность свечей
func (a *CounterAnalyzer) TestCandleAccuracy(symbol string) string {
	if a.candleSystem == nil {
		return "❌ Свечная система не инициализирована"
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("📊 Тест точности свечей для %s:\n", symbol))

	// Тестируем разные периоды
	periods := []string{"5m", "15m", "30m", "1h", "4h", "1d"}

	for _, period := range periods {
		// Получаем свечу из системы
		candle, err := a.candleSystem.GetCandle(symbol, period)
		if err != nil {
			result.WriteString(fmt.Sprintf("⚠️ %s: ошибка - %s\n", period, err))
			continue
		}

		if candle == nil || !candle.IsReal {
			result.WriteString(fmt.Sprintf("⏳ %s: нет данных\n", period))
			continue
		}

		// Рассчитываем изменение из свечи
		candleChangePercent := ((candle.Close - candle.Open) / candle.Open) * 100

		// Получаем данные через наш метод
		data, err := a.getDataForPeriod(symbol, period)
		var ourChangePercent float64
		if err == nil && len(data) >= 2 {
			ourChangePercent = ((data[1].Price - data[0].Price) / data[0].Price) * 100
		}

		// Сравниваем
		result.WriteString(fmt.Sprintf("✅ %s:\n", period))
		result.WriteString(fmt.Sprintf("   • Bybit свеча: %.6f → %.6f (%.4f%%)\n",
			candle.Open, candle.Close, candleChangePercent))

		if err == nil && len(data) >= 2 {
			result.WriteString(fmt.Sprintf("   • Наш расчет: %.6f → %.6f (%.4f%%)\n",
				data[0].Price, data[1].Price, ourChangePercent))

			// Рассчитываем разницу
			diff := math.Abs(candleChangePercent - ourChangePercent)
			diffPriceOpen := math.Abs(candle.Open - data[0].Price)
			diffPriceClose := math.Abs(candle.Close - data[1].Price)

			result.WriteString(fmt.Sprintf("   • Разница цен: открытие=%.6f, закрытие=%.6f\n",
				diffPriceOpen, diffPriceClose))
			result.WriteString(fmt.Sprintf("   • Разница изменения: %.6f%%\n", diff))

			// Оценка точности
			if diff < 0.001 { // 0.001% разницы
				result.WriteString(fmt.Sprintf("   • ✓ Точность: отличная\n"))
			} else if diff < 0.01 { // 0.01% разницы
				result.WriteString(fmt.Sprintf("   • ✓ Точность: хорошая\n"))
			} else if diff < 0.1 { // 0.1% разницы
				result.WriteString(fmt.Sprintf("   • ⚠️ Точность: приемлемая\n"))
			} else {
				result.WriteString(fmt.Sprintf("   • ❌ Точность: низкая\n"))
			}

			// Проверяем временные метки
			if len(data) >= 2 {
				candleDuration := candle.EndTime.Sub(candle.StartTime)
				ourDuration := data[1].Timestamp.Sub(data[0].Timestamp)
				result.WriteString(fmt.Sprintf("   • Время свечи: %v (наше: %v)\n",
					candleDuration, ourDuration))
			}
		} else {
			result.WriteString(fmt.Sprintf("   • ❌ Не удалось получить данные для сравнения\n"))
		}
	}

	return result.String()
}

// VerifyCandleData проверяет корректность свечных данных
func (a *CounterAnalyzer) VerifyCandleData(symbol string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	if a.candleSystem == nil {
		result["error"] = "Свечная система не инициализирована"
		return result, nil
	}

	// Проверяем все периоды
	periods := []string{"5m", "15m", "30m", "1h", "4h", "1d"}
	periodData := make(map[string]interface{})

	for _, period := range periods {
		periodInfo := make(map[string]interface{})

		// Получаем свечу
		candle, err := a.candleSystem.GetCandle(symbol, period)
		if err != nil {
			periodInfo["status"] = "error"
			periodInfo["error"] = err.Error()
			periodData[period] = periodInfo
			continue
		}

		if candle == nil {
			periodInfo["status"] = "no_data"
			periodData[period] = periodInfo
			continue
		}

		// Проверяем свечу
		periodInfo["status"] = "ok"
		periodInfo["is_real"] = candle.IsReal
		periodInfo["is_closed"] = candle.IsClosed
		periodInfo["open"] = candle.Open
		periodInfo["close"] = candle.Close
		periodInfo["high"] = candle.High
		periodInfo["low"] = candle.Low
		periodInfo["volume_usd"] = candle.VolumeUSD
		periodInfo["start_time"] = candle.StartTime.Format("15:04:05")
		periodInfo["end_time"] = candle.EndTime.Format("15:04:05")

		// Рассчитываем изменение
		if candle.Open > 0 {
			change := ((candle.Close - candle.Open) / candle.Open) * 100
			periodInfo["change_percent"] = change
		}

		// Проверяем логику: закрытие должно быть между high и low
		if candle.Close < candle.Low || candle.Close > candle.High {
			periodInfo["warning"] = "Цена закрытия вне диапазона high/low"
		}

		// Проверяем временные метки
		if candle.StartTime.After(candle.EndTime) {
			periodInfo["error"] = "Некорректное время: начало позже окончания"
		}

		periodData[period] = periodInfo
	}

	result["periods"] = periodData

	// Получаем статистику системы
	stats := a.candleSystem.GetStats()
	result["system_stats"] = stats

	return result, nil
}

// GetCandleStats получает статистику свечей для символа
func (a *CounterAnalyzer) GetCandleStats(symbol string) (map[string]interface{}, error) {
	if a.candleSystem == nil {
		return nil, fmt.Errorf("свечная система не инициализирована")
	}

	stats := make(map[string]interface{})
	periods := []string{"5m", "15m", "30m", "1h", "4h", "1d"}

	for _, period := range periods {
		candle, err := a.candleSystem.GetCandle(symbol, period)
		if err != nil {
			stats[period] = map[string]interface{}{
				"error": err.Error(),
			}
			continue
		}

		if candle != nil {
			changePercent := ((candle.Close - candle.Open) / candle.Open) * 100
			stats[period] = map[string]interface{}{
				"open":           candle.Open,
				"high":           candle.High,
				"low":            candle.Low,
				"close":          candle.Close,
				"change_percent": changePercent,
				"volume_usd":     candle.VolumeUSD,
				"is_closed":      candle.IsClosed,
				"is_real":        candle.IsReal,
				"start_time":     candle.StartTime.Format("15:04:05"),
				"end_time":       candle.EndTime.Format("15:04:05"),
			}
		} else {
			stats[period] = map[string]interface{}{
				"status": "no_data",
			}
		}
	}

	return stats, nil
}

// TestCandleSystem тестирует свечную систему
func (a *CounterAnalyzer) TestCandleSystem(symbol string) string {
	if a.candleSystem == nil {
		return "❌ Свечная система не инициализирована"
	}

	var result string
	periods := []string{"5m", "15m", "30m", "1h", "4h", "1d"}

	for _, period := range periods {
		candle, err := a.candleSystem.GetCandle(symbol, period)
		if err != nil {
			result += fmt.Sprintf("⚠️ %s: ошибка - %s\n", period, err.Error())
			continue
		}

		if candle != nil && candle.IsReal {
			changePercent := ((candle.Close - candle.Open) / candle.Open) * 100
			result += fmt.Sprintf("✅ %s: %.6f → %.6f (%.4f%%)",
				period, candle.Open, candle.Close, changePercent)

			if !candle.IsClosed {
				result += " 🔄 активная"
			}
			result += "\n"
		} else {
			result += fmt.Sprintf("⏳ %s: нет данных\n", period)
		}
	}

	// Получаем статистику системы
	stats := a.candleSystem.GetStats()
	if storageStats, ok := stats["storage_stats"].(candle.CandleStats); ok {
		result += fmt.Sprintf("\n📊 Статистика системы:\n")
		result += fmt.Sprintf("• Активных свечей: %d\n", storageStats.ActiveCandles)
		result += fmt.Sprintf("• Всего свечей: %d\n", storageStats.TotalCandles)
		result += fmt.Sprintf("• Символов: %d\n", storageStats.SymbolsCount)
	}

	return result
}

// getHistoryFromCandles получает историю свечей для анализа
func (a *CounterAnalyzer) getHistoryFromCandles(symbol, period string, limit int) ([]*candle.Candle, error) {
	if a.candleSystem == nil {
		return nil, fmt.Errorf("свечная система не инициализирована")
	}

	return a.candleSystem.GetHistory(symbol, period, limit)
}
