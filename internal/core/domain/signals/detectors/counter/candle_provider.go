// internal/core/domain/signals/detectors/counter/candle_provider.go
package counter

import (
	"fmt"
	"math"
	"time"

	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/counter/confirmation"
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"

	"github.com/google/uuid"
)

// analyzeSymbolPeriod анализирует конкретный символ и период
func (a *CounterAnalyzer) analyzeSymbolPeriod(symbol, period string, data []types.PriceData) (*analysis.Signal, error) {
	if len(data) < 2 {
		logger.Debug("⚠️ Недостаточно данных для %s период %s (%d точек)",
			symbol, period, len(data))
		return nil, fmt.Errorf("недостаточно данных")
	}

	// Проверяем, что у нас есть открытие и закрытие свечи
	if len(data) != 2 {
		logger.Warn("⚠️ Для %s %s получено %d точек, ожидается 2 (открытие/закрытие)",
			symbol, period, len(data))

		// Если точек много, берем первую и последнюю как приближение
		if len(data) > 2 {
			startPrice := data[0].Price
			endPrice := data[len(data)-1].Price
			logger.Warn("   • Используем приближение: %.6f → %.6f", startPrice, endPrice)

			// Создаем упрощенные данные
			data = []types.PriceData{data[0], data[len(data)-1]}
		}
	}

	// Рассчитываем изменение свечи (открытие → закрытие)
	change := a.calculateCandleChange(data, period)

	// Проверяем базовый порог (0.1% по умолчанию)
	if math.Abs(change) < a.baseThreshold {
		logger.Debug("📊 %s %s: изменение %.4f%% < порога %.4f%%, пропускаем",
			symbol, period, change, a.baseThreshold)
		return nil, nil
	}

	logger.Info("🎯 %s %s: значительное изменение %.4f%%", symbol, period, change)

	// Добавляем подтверждение в менеджер
	isReady, confirmations := a.confirmationManager.AddConfirmation(symbol, period)

	if isReady {
		// Создаем сырой сигнал
		signal := a.createRawSignal(symbol, period, change, confirmations, data)

		logger.Info("🚀 Сигнал для %s %s:", symbol, period)
		logger.Info("   • Изменение: %.4f%%", change)
		logger.Info("   • Подтверждений: %d/%d",
			confirmations, confirmation.GetRequiredConfirmations(period))
		logger.Info("   • Направление: %s", signal.Direction)

		// Публикуем в EventBus
		a.publishRawCounterSignal(signal)

		// Сбрасываем счетчик подтверждений
		a.confirmationManager.Reset(symbol, period)

		return &signal, nil
	} else {
		logger.Debug("⏳ %s %s: подтверждений %d/%d, ждем еще",
			symbol, period, confirmations, confirmation.GetRequiredConfirmations(period))
	}

	return nil, nil
}

// calculateCandleChange рассчитывает изменение свечи с проверкой корректности
func (a *CounterAnalyzer) calculateCandleChange(data []types.PriceData, period string) float64 {
	if len(data) < 2 {
		return 0
	}

	// Берем первую точку как открытие, последнюю как закрытие
	openPrice := data[0].Price
	closePrice := data[len(data)-1].Price
	openTime := data[0].Timestamp
	closeTime := data[len(data)-1].Timestamp

	// Рассчитываем изменение
	change := ((closePrice - openPrice) / openPrice) * 100

	// Проверяем длительность для определения корректности данных
	actualDuration := closeTime.Sub(openTime)
	expectedDuration := getPeriodDuration(period)
	coverageRatio := actualDuration.Seconds() / expectedDuration.Seconds()

	// Логируем детали
	logger.Debug("📐 Расчет изменения свечи %s:", data[0].Symbol)
	logger.Debug("   • Открытие: %.6f в %s", openPrice, openTime.Format("15:04:05"))
	logger.Debug("   • Закрытие: %.6f в %s", closePrice, closeTime.Format("15:04:05"))
	logger.Debug("   • Изменение: %.4f%%", change)
	logger.Debug("   • Длительность: %v (ожидается: %v)",
		actualDuration, expectedDuration)
	logger.Debug("   • Покрытие: %.1f%%", coverageRatio*100)

	// Проверяем корректность данных
	if coverageRatio < 0.5 {
		logger.Warn("⚠️ Низкое покрытие данных для %s %s: %.1f%% периода",
			data[0].Symbol, period, coverageRatio*100)
		logger.Warn("   • Могут быть расхождения с реальными свечами Bybit")
	}

	return change
}

// createRawSignal создает сырой сигнал (без user_id)
func (a *CounterAnalyzer) createRawSignal(
	symbol, period string,
	change float64, // Изменение свечи (открытие → закрытие)
	confirmations int,
	data []types.PriceData, // Все данные для индикаторов
) analysis.Signal {
	if len(data) == 0 {
		return analysis.Signal{} // Возвращаем пустой сигнал
	}

	// Берем данные для свечи
	openPrice := data[0].Price
	closePrice := data[len(data)-1].Price
	openTime := data[0].Timestamp
	closeTime := data[len(data)-1].Timestamp

	// Получаем текущие метрики для расчета индикаторов
	latestData := data[len(data)-1]

	// Рассчитываем дополнительные метрики
	var volumeDelta, volumeDeltaPercent float64
	var deltaSource string
	if a.volumeCalculator != nil {
		direction := "growth"
		if change < 0 {
			direction = "fall"
		}
		deltaData := a.volumeCalculator.CalculateWithFallback(symbol, direction)
		if deltaData != nil {
			volumeDelta = deltaData.Delta
			volumeDeltaPercent = deltaData.DeltaPercent
			deltaSource = string(deltaData.Source)
		}
	}

	// Технические индикаторы на основе ВСЕХ данных
	rsi := a.techCalculator.CalculateRSI(data)
	macdLine, signalLine, histogram := a.techCalculator.CalculateMACD(data)
	// Для обратной совместимости используем MACD линию
	macdSignal := macdLine

	periodMinutes := getPeriodMinutes(period)

	// Детальное логирование свечи
	logger.Info("📈 Создание сигнала для %s %s:", symbol, period)
	logger.Info("   • Свеча: %.6f → %.6f (изменение: %.2f%%)",
		openPrice, closePrice, change)
	logger.Info("   • Время: %s → %s",
		openTime.Format("15:04:05"), closeTime.Format("15:04:05"))
	logger.Info("   • Подтверждений: %d/%d",
		confirmations, confirmation.GetRequiredConfirmations(period))
	logger.Info("   • Индикаторы: RSI=%.1f, MACD=%.4f", rsi, macdLine)

	// СОЗДАЕМ Custom map с деталями свечи
	customMap := make(map[string]interface{})
	customMap["delta_source"] = deltaSource
	customMap["period_string"] = period
	customMap["period_minutes"] = periodMinutes
	customMap["base_threshold"] = a.baseThreshold
	customMap["change_percent"] = change
	customMap["symbol"] = symbol
	customMap["confirmations"] = confirmations
	customMap["required_confirmations"] = confirmation.GetRequiredConfirmations(period)

	// Данные свечи
	customMap["candle_open_price"] = openPrice
	customMap["candle_close_price"] = closePrice
	customMap["candle_open_time"] = openTime
	customMap["candle_close_time"] = closeTime
	customMap["candle_duration_minutes"] = closeTime.Sub(openTime).Minutes()
	customMap["candle_data_points"] = len(data)

	// MACD компоненты
	customMap["macd_line"] = macdLine
	customMap["macd_signal_line"] = signalLine
	customMap["macd_histogram"] = histogram

	// Определяем направление на основе изменения свечи
	direction := a.getDirection(change)

	return analysis.Signal{
		ID:            uuid.New().String(),
		Symbol:        symbol,
		Type:          "counter_raw",
		Direction:     direction,
		ChangePercent: change, // Изменение свечи (открытие → закрытие)
		Period:        periodMinutes,
		Confidence:    float64(confirmations),
		DataPoints:    len(data),
		StartPrice:    openPrice,  // Цена открытия свечи
		EndPrice:      closePrice, // Цена закрытия свечи
		Timestamp:     time.Now(),
		Metadata: analysis.Metadata{
			Strategy: "counter_analyzer_candle",
			Tags: []string{
				"counter_raw",
				direction,
				period,
				fmt.Sprintf("confirmations_%d", confirmations),
				"candle_based",
			},
			Indicators: map[string]float64{
				// Основные метрики
				"period":                 float64(periodMinutes),
				"confirmations":          float64(confirmations),
				"required_confirmations": float64(confirmation.GetRequiredConfirmations(period)),

				// Рыночные данные
				"volume_24h":           latestData.Volume24h,
				"open_interest":        latestData.OpenInterest,
				"funding_rate":         latestData.FundingRate,
				"current_price":        latestData.Price,
				"volume_delta":         volumeDelta,
				"volume_delta_percent": volumeDeltaPercent,

				// Технические индикаторы
				"rsi":              rsi,
				"macd_signal":      macdSignal, // Для обратной совместимости
				"macd_line":        macdLine,
				"macd_signal_line": signalLine,
				"macd_histogram":   histogram,

				// Данные свечи
				"candle_open_price":     openPrice,
				"candle_close_price":    closePrice,
				"candle_change_percent": change, // Дублируем для ясности
			},
			Custom: customMap,
		},
	}
}

// publishRawCounterSignal публикует сырой Counter сигнал в EventBus
func (a *CounterAnalyzer) publishRawCounterSignal(signal analysis.Signal) {
	if a.eventBus == nil {
		logger.Error("❌ EventBus НЕ ИНИЦИАЛИЗИРОВАН в CounterAnalyzer!\n")
		return
	}

	// Создаем событие с сырыми данными
	event := types.Event{
		Type:      types.EventCounterSignalDetected,
		Source:    "counter_analyzer_raw",
		Data:      signal.ToMap(),
		Timestamp: time.Now(),
	}

	if err := a.eventBus.Publish(event); err != nil {
		logger.Error("❌ Ошибка публикации сырого Counter сигнала для %s: %v\n",
			signal.Symbol, err)
	} else {
		logger.Debug("✅ Сырой Counter сигнал опубликован: %s %s %.2f%% (период: %s)\n",
			signal.Symbol, signal.Direction, signal.ChangePercent,
			signal.Metadata.Custom["period_string"])
	}
}

// getDirection возвращает направление изменения
func (a *CounterAnalyzer) getDirection(change float64) string {
	if change >= 0 {
		return "growth"
	}
	return "fall"
}

// getPeriodDuration возвращает длительность периода
func getPeriodDuration(period string) time.Duration {
	switch period {
	case "5m":
		return 5 * time.Minute
	case "15m":
		return 15 * time.Minute
	case "30m":
		return 30 * time.Minute
	case "1h":
		return 1 * time.Hour
	case "4h":
		return 4 * time.Hour
	case "1d":
		return 24 * time.Hour
	default:
		return 15 * time.Minute
	}
}

// getPeriodMinutes возвращает период в минутах
func getPeriodMinutes(period string) int {
	switch period {
	case "5m":
		return 5
	case "15m":
		return 15
	case "30m":
		return 30
	case "1h":
		return 60
	case "4h":
		return 240
	case "1d":
		return 1440
	default:
		return 15
	}
}
