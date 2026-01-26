// internal/core/domain/signals/detectors/counter/methods.go
package counter

import (
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/counter/calculator"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
	periodPkg "crypto-exchange-screener-bot/pkg/period"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// GetOI получает Open Interest
func (a *CounterAnalyzer) GetOI(symbol string) float64 {
	if a.deps.Storage != nil {
		if snapshot, exists := a.deps.Storage.GetCurrentSnapshot(symbol); exists {
			return snapshot.GetOpenInterest()
		}
	}
	return 0
}

// GetVolumeDelta получает дельту объема
func (a *CounterAnalyzer) GetVolumeDelta(symbol, direction string) *types.VolumeDeltaData {
	// ✅ Используем общий калькулятор из зависимостей
	if a.deps.VolumeCalculator == nil {
		// Создаем временно, если не передан в зависимостях
		logger.Debug("⚠️ Создаем временный VolumeDeltaCalculator для %s", symbol)
		tempCalculator := calculator.NewVolumeDeltaCalculator(a.deps.MarketFetcher, a.deps.Storage)
		defer tempCalculator.Stop() // ✅ ВАЖНО: останавливаем временный калькулятор

		return tempCalculator.CalculateWithFallback(symbol, direction)
	}

	return a.deps.VolumeCalculator.CalculateWithFallback(symbol, direction)
}

// AnalyzeCandle анализирует свечу
func (a *CounterAnalyzer) AnalyzeCandle(symbol, period string) (*analysis.Signal, error) {
	if a.deps.CandleSystem == nil {
		return nil, fmt.Errorf("свечная система не инициализирована")
	}

	// Валидируем период
	if !periodPkg.IsValidPeriod(period) {
		logger.Warn("⚠️ Невалидный период '%s' для символа %s, используем %s",
			period, symbol, periodPkg.DefaultPeriod)
		period = periodPkg.DefaultPeriod
	}

	// Получаем свечу
	candleData, err := a.deps.CandleSystem.GetCandle(symbol, period)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения свечи %s/%s: %w", symbol, period, err)
	}

	if candleData == nil {
		return nil, nil
	}

	if !candleData.IsRealFlag || candleData.Open == 0 {
		return nil, nil
	}

	// Рассчитываем изменение
	changePercent := ((candleData.Close - candleData.Open) / candleData.Open) * 100

	// Определяем направление
	direction := "growth"
	if changePercent < 0 {
		direction = "fall"
	}

	// Проверяем пороги
	growthThreshold := SafeGetFloat(a.config.CustomSettings, "growth_threshold", 0.1)
	fallThreshold := SafeGetFloat(a.config.CustomSettings, "fall_threshold", 0.1)

	var shouldCreateSignal bool
	if direction == "growth" && changePercent >= growthThreshold {
		shouldCreateSignal = SafeGetBool(a.config.CustomSettings, "track_growth", true)
	} else if direction == "fall" && changePercent <= -fallThreshold {
		shouldCreateSignal = SafeGetBool(a.config.CustomSettings, "track_fall", true)
	}

	if !shouldCreateSignal {
		return nil, nil
	}

	// Создаем сигнал
	signal := a.CreateSignal(symbol, period, direction, changePercent, candleData)

	return &signal, nil
}

// CreateSignal создает сигнал
func (a *CounterAnalyzer) CreateSignal(symbol, period, direction string, changePercent float64,
	candleData *redis_storage.Candle) analysis.Signal {

	// Упрощенный расчет уверенности
	confidence := 50.0
	if changePercent > 5 {
		confidence = 80
	} else if changePercent > 2 {
		confidence = 65
	} else if changePercent < -5 {
		confidence = 80
	} else if changePercent < -2 {
		confidence = 65
	}

	// Конвертируем период в минуты
	periodMinutes, err := periodPkg.StringToMinutes(period)
	if err != nil {
		logger.Warn("⚠️ Ошибка конвертации периода '%s', используем дефолтный: %s",
			period, periodPkg.DefaultPeriod)
		periodMinutes = periodPkg.DefaultMinutes
	}

	signal := analysis.Signal{
		ID:            uuid.New().String(),
		Symbol:        symbol,
		Type:          "counter_candle",
		Direction:     direction,
		ChangePercent: changePercent,
		Period:        periodMinutes, // Используем конвертированные минуты
		Confidence:    confidence,
		DataPoints:    2,
		StartPrice:    candleData.Open,
		EndPrice:      candleData.Close,
		Volume:        candleData.VolumeUSD,
		Timestamp:     time.Now(),
		Metadata: analysis.Metadata{
			Strategy: "counter_candle_analyzer",
			Tags:     []string{"candle_analysis", period},
			Custom: map[string]interface{}{
				"period_minutes": periodMinutes, // Добавляем минуты
				"period_string":  period,
			},
		},
		Progress: nil,
	}

	return signal
}

// PublishRawCounterSignal публикует сигнал (только отправка)
func (a *CounterAnalyzer) PublishRawCounterSignal(signal analysis.Signal, period string) {
	if a.deps.EventBus == nil {
		logger.Error("❌ EventBus не инициализирован")
		return
	}

	// Валидируем период перед отправкой
	if !periodPkg.IsValidPeriod(period) {
		logger.Warn("⚠️ Невалидный период '%s' для публикации сигнала %s, используем %s",
			period, signal.Symbol, periodPkg.DefaultPeriod)
		period = periodPkg.DefaultPeriod
	}

	// Создаем данные через отдельный метод
	eventData := a.CreateCounterEventData(signal, period)

	// Создаем и отправляем событие
	event := types.Event{
		Type:      types.EventCounterSignalDetected,
		Source:    "counter_analyzer_raw",
		Data:      eventData,
		Timestamp: time.Now(),
	}

	if err := a.deps.EventBus.Publish(event); err != nil {
		logger.Error("❌ Ошибка публикации сигнала %s: %v", signal.Symbol, err)
	} else {
		logger.Debug("✅ Сигнал опубликован: %s %s %.2f%% (%s)",
			signal.Symbol, signal.Direction, signal.ChangePercent, period)
	}
}

// CreateCounterEventData создает плоский map с 17 полями для контроллера
func (a *CounterAnalyzer) CreateCounterEventData(signal analysis.Signal, period string) map[string]interface{} {
	eventData := make(map[string]interface{})

	// 1. Базовые поля из Signal (5 полей)
	eventData["symbol"] = signal.Symbol
	eventData["direction"] = signal.Direction
	eventData["change_percent"] = signal.ChangePercent

	// Нормализуем период
	normalizedPeriod := period
	if !periodPkg.IsValidPeriod(period) {
		normalizedPeriod = periodPkg.DefaultPeriod
		logger.Debug("⚠️ Нормализован период для %s: %s → %s",
			signal.Symbol, period, normalizedPeriod)
	}
	eventData["period"] = normalizedPeriod

	eventData["timestamp"] = signal.Timestamp

	// 2. Подтверждения (1 поле) - заглушка
	eventData["confirmations"] = 3

	// 3. Данные из indicators (8 полей) - flat map
	eventData["current_price"] = signal.EndPrice

	// Получаем реальный объем 24ч из storage
	volume24h := 0.0
	if a.deps.Storage != nil {
		if snapshot, exists := a.deps.Storage.GetCurrentSnapshot(signal.Symbol); exists {
			volume24h = snapshot.GetVolume24h()
		}
	}
	eventData["volume_24h"] = volume24h

	// Получаем OI
	oi := a.GetOI(signal.Symbol)

	eventData["open_interest"] = oi
	eventData["funding_rate"] = 0.001 // Заглушка

	// Получаем реальную дельту и процент через новый метод
	deltaData := a.GetVolumeDelta(signal.Symbol, signal.Direction)

	eventData["rsi"] = 55.0         // Заглушка
	eventData["macd_signal"] = 0.01 // Заглушка
	eventData["volume_delta"] = deltaData.Delta
	eventData["volume_delta_percent"] = deltaData.DeltaPercent

	// 4. Данные прогресса (3 поля) - вложенные в progress map
	eventData["progress"] = map[string]interface{}{
		"filled_groups": 3,    // Заглушка
		"total_groups":  6,    // Заглушка
		"percentage":    50.0, // Заглушка
	}

	return eventData
}
