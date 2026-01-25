// /internal/core/domain/signals/detectors/counter/methods.go
package counter

import (
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// getOIAndDelta получает OI и Volume Delta
func (a *CounterAnalyzer) GetOIAndDelta(symbol string) (float64, float64) {
	var oi, volumeDelta float64

	// Получаем OI из storage
	if a.storage != nil {
		if snapshot, exists := a.storage.GetCurrentSnapshot(symbol); exists {
			oi = snapshot.GetOpenInterest()
		}
	}

	// TODO: Реальная реализация Volume Delta
	// Пока эмуляция
	volumeDelta = 1000000.0 // $1M

	return oi, volumeDelta
}

// analyzeCandle анализирует свечу
func (a *CounterAnalyzer) AnalyzeCandle(symbol, period string, oi, volumeDelta float64) (*analysis.Signal, error) {
	if a.candleSystem == nil {
		return nil, fmt.Errorf("свечная система не инициализирована")
	}

	// Получаем свечу
	candleData, err := a.candleSystem.GetCandle(symbol, period)
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
	signal := a.CreateSignal(symbol, period, direction, changePercent, candleData, oi, volumeDelta)

	return &signal, nil
}

// createSignal создает сигнал
func (a *CounterAnalyzer) CreateSignal(symbol, period, direction string, changePercent float64,
	candleData *redis_storage.Candle, oi, volumeDelta float64) analysis.Signal {

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

	signal := analysis.Signal{
		ID:            uuid.New().String(),
		Symbol:        symbol,
		Type:          "counter_candle",
		Direction:     direction,
		ChangePercent: changePercent,
		Period:        GetPeriodMinutes(period),
		Confidence:    confidence,
		DataPoints:    2,
		StartPrice:    candleData.Open,
		EndPrice:      candleData.Close,
		Volume:        candleData.VolumeUSD,
		Timestamp:     time.Now(),
		Metadata: analysis.Metadata{
			Strategy: "counter_candle_analyzer",
			Tags:     []string{"candle_analysis", period},
			Custom:   make(map[string]interface{}), // Пустой, только для служебных данных
		},
		Progress: nil,
	}

	return signal
}

// publishRawCounterSignal публикует сигнал (только отправка)
func (a *CounterAnalyzer) PublishRawCounterSignal(signal analysis.Signal, period string, oi, volumeDelta float64) {
	if a.eventBus == nil {
		logger.Error("❌ EventBus не инициализирован")
		return
	}

	// Создаем данные через отдельный метод
	eventData := a.CreateCounterEventData(signal, period, oi, volumeDelta)

	// Создаем и отправляем событие
	event := types.Event{
		Type:      types.EventCounterSignalDetected,
		Source:    "counter_analyzer_raw",
		Data:      eventData,
		Timestamp: time.Now(),
	}

	if err := a.eventBus.Publish(event); err != nil {
		logger.Error("❌ Ошибка публикации сигнала %s: %v", signal.Symbol, err)
	} else {
		logger.Debug("✅ Сигнал опубликован: %s %s %.2f%%",
			signal.Symbol, signal.Direction, signal.ChangePercent)
	}
}

// createCounterEventData создает плоский map с 17 полями для контроллера
func (a *CounterAnalyzer) CreateCounterEventData(signal analysis.Signal, period string, oi, volumeDelta float64) map[string]interface{} {
	eventData := make(map[string]interface{})

	// 1. Базовые поля из Signal (5 полей)
	eventData["symbol"] = signal.Symbol
	eventData["direction"] = signal.Direction
	eventData["change_percent"] = signal.ChangePercent
	eventData["period"] = period // ТОЛЬКО period
	eventData["timestamp"] = signal.Timestamp

	// 2. Подтверждения (1 поле) - заглушка
	eventData["confirmations"] = 3

	// 3. Данные из indicators (8 полей) - flat map
	eventData["current_price"] = signal.EndPrice
	eventData["volume_24h"] = signal.Volume * 10 // Заглушка
	eventData["open_interest"] = oi
	eventData["funding_rate"] = 0.001 // Заглушка
	eventData["rsi"] = 55.0           // Заглушка
	eventData["macd_signal"] = 0.01   // Заглушка
	eventData["volume_delta"] = volumeDelta
	eventData["volume_delta_percent"] = 2.0 // Заглушка

	// 4. Данные прогресса (3 поля) - вложенные в progress map
	eventData["progress"] = map[string]interface{}{
		"filled_groups": 3,    // Заглушка
		"total_groups":  6,    // Заглушка
		"percentage":    50.0, // Заглушка
	}

	return eventData
}
