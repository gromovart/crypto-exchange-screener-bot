// internal/core/domain/signals/detectors/counter/methods.go
package counter

import (
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/counter/calculator"
	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
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
		logger.Warn("⚠️ Создаем временный VolumeDeltaCalculator для %s", symbol)
		tempCalculator := calculator.NewVolumeDeltaCalculator(a.deps.MarketFetcher, a.deps.Storage)
		defer tempCalculator.Stop() // ✅ ВАЖНО: останавливаем временный калькулятор

		return tempCalculator.CalculateWithFallback(symbol, direction)
	}

	return a.deps.VolumeCalculator.CalculateWithFallback(symbol, direction)
}

// AnalyzeCandle анализирует свечу
func (a *CounterAnalyzer) AnalyzeCandle(symbol, period string) (*analysis.Signal, error) {
	// ✅ ИНКРЕМЕНТИРУЕМ ОБЩИЙ СЧЕТЧИК
	a.candleStatsMu.Lock()
	a.candleStats.TotalCalls++
	a.candleStatsMu.Unlock()

	if a.deps.CandleSystem == nil {
		return nil, fmt.Errorf("свечная система не инициализирована")
	}

	// Валидируем период
	if !periodPkg.IsValidPeriod(period) {
		period = periodPkg.DefaultPeriod
	}

	// Получаем последнюю закрытую свечу
	candleData, err := a.deps.CandleSystem.GetLatestClosedCandle(symbol, period)
	if err != nil {
		// ✅ АГРЕГИРУЕМ ОШИБКУ ПОЛУЧЕНИЯ
		a.candleStatsMu.Lock()
		a.candleStats.GetCandleError++
		a.candleStatsMu.Unlock()
		return nil, fmt.Errorf("ошибка получения закрытой свечи %s/%s: %w", symbol, period, err)
	}

	if candleData == nil {
		// ✅ АГРЕГИРУЕМ ОТСУТСТВИЕ СВЕЧЕЙ
		a.candleStatsMu.Lock()
		a.candleStats.NoCandleData++
		a.candleStatsMu.Unlock()
		return nil, nil
	}

	if !candleData.IsRealFlag || candleData.Open == 0 {
		// ✅ АГРЕГИРУЕМ НЕРЕАЛЬНЫЕ СВЕЧИ
		a.candleStatsMu.Lock()
		a.candleStats.UnrealCandle++
		a.candleStatsMu.Unlock()
		return nil, nil
	}

	// Атомарная проверка и отметка свечи
	startTimeUnix := candleData.StartTime.Unix()
	marked, err := a.deps.CandleSystem.MarkCandleProcessedAtomically(symbol, period, startTimeUnix)
	if err != nil {
		// ✅ АГРЕГИРУЕМ ОШИБКИ ОТМЕТКИ
		a.candleStatsMu.Lock()
		a.candleStats.MarkCandleError++
		a.candleStatsMu.Unlock()
		return nil, fmt.Errorf("ошибка отметки свечи %s/%s: %w", symbol, period, err)
	}

	if !marked {
		// ✅ АГРЕГИРУЕМ УЖЕ ОБРАБОТАННЫЕ
		a.candleStatsMu.Lock()
		a.candleStats.AlreadyProcessed++
		a.candleStatsMu.Unlock()
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
	growthThreshold := SafeGetFloat(a.config.CustomSettings, "growth_threshold", 0.01) // 0.01%
	fallThreshold := SafeGetFloat(a.config.CustomSettings, "fall_threshold", 0.01)     // 0.01%

	var shouldCreateSignal bool
	if direction == "growth" && changePercent >= growthThreshold {
		shouldCreateSignal = SafeGetBool(a.config.CustomSettings, "track_growth", true)
		if shouldCreateSignal {
			// ✅ АГРЕГИРУЕМ РОСТОВЫЕ СИГНАЛЫ
			a.candleStatsMu.Lock()
			a.candleStats.GrowthSignal++
			a.candleStatsMu.Unlock()
		}
	} else if direction == "fall" && changePercent <= -fallThreshold {
		shouldCreateSignal = SafeGetBool(a.config.CustomSettings, "track_fall", true)
		if shouldCreateSignal {
			// ✅ АГРЕГИРУЕМ ПАДАЮЩИЕ СИГНАЛЫ
			a.candleStatsMu.Lock()
			a.candleStats.FallSignal++
			a.candleStatsMu.Unlock()
		}
	}

	if !shouldCreateSignal {
		// ✅ АГРЕГИРУЕМ НИЖЕ ПОРОГА
		a.candleStatsMu.Lock()
		a.candleStats.BelowThreshold++
		a.candleStatsMu.Unlock()
		return nil, nil
	}

	// Создаем сигнал
	signal := a.CreateSignal(symbol, period, direction, changePercent, candleData)
	return &signal, nil
}

// CreateSignal создает сигнал
func (a *CounterAnalyzer) CreateSignal(symbol, period, direction string, changePercent float64,
	candleData *storage.Candle) analysis.Signal {

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

// getPriceHistoryForAnalysis получает историю цен для технического анализа
func (a *CounterAnalyzer) getPriceHistoryForAnalysis(symbol, period string, limit int) ([]storage.PriceData, error) {
	if a.deps.Storage == nil {
		return nil, fmt.Errorf("хранилище не инициализировано")
	}

	// Получаем историю цен
	history, err := a.deps.Storage.GetPriceHistory(symbol, limit)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения истории цен для %s: %w", symbol, err)
	}

	// Конвертируем интерфейсы в PriceData
	var priceData []storage.PriceData
	for _, h := range history {
		priceData = append(priceData, storage.PriceData{
			Symbol:       h.GetSymbol(),
			Price:        h.GetPrice(),
			Volume24h:    h.GetVolume24h(),
			VolumeUSD:    h.GetVolumeUSD(),
			Timestamp:    h.GetTimestamp(),
			OpenInterest: h.GetOpenInterest(),
			FundingRate:  h.GetFundingRate(),
			Change24h:    h.GetChange24h(),
			High24h:      h.GetHigh24h(),
			Low24h:       h.GetLow24h(),
		})
	}

	return priceData, nil
}

// calculateRSI рассчитывает RSI для символа и периода
func (a *CounterAnalyzer) calculateRSI(symbol, period string) (float64, string) {
	if a.deps.TechnicalCalculator == nil {
		return 55.0, "нейтральный" // Заглушка если калькулятор не доступен
	}

	// Получаем историю цен (достаточно для RSI расчета)
	priceHistory, err := a.getPriceHistoryForAnalysis(symbol, period, 30) // 30 точек достаточно
	if err != nil {
		logger.Warn("⚠️ Не удалось получить историю для расчета RSI %s/%s: %v", symbol, period, err)
		return 55.0, "нейтральный"
	}

	if len(priceHistory) < 2 {
		return 50.0, "недостаточно данных"
	}

	// Рассчитываем RSI
	rsi := a.deps.TechnicalCalculator.CalculateRSI(priceHistory)
	status := a.deps.TechnicalCalculator.GetRSIStatus(rsi)

	return rsi, status
}

// calculateMACD рассчитывает MACD для символа и периода
func (a *CounterAnalyzer) calculateMACD(symbol, period string) (float64, string, string) {
	if a.deps.TechnicalCalculator == nil {
		logger.Warn("⚠️ CounterAnalyzer: TechnicalCalculator не доступен для %s/%s", symbol, period)
		return 0.01, "нейтральный", "⭕ калькулятор недоступен" // Заглушка
	}

	// Получаем историю цен (нужно больше точек для MACD)
	priceHistory, err := a.getPriceHistoryForAnalysis(symbol, period, 50) // 50 точек для MACD
	if err != nil {
		logger.Warn("⚠️ CounterAnalyzer: Не удалось получить историю для расчета MACD %s/%s: %v", symbol, period, err)
		return 0.01, "нейтральный", "⭕ недостаточно данных"
	}

	if len(priceHistory) < 2 {
		logger.Warn("⚠️ CounterAnalyzer: недостаточно данных для MACD %s/%s: %d точек",
			symbol, period, len(priceHistory))
		return 0.01, "нейтральный", "⭕ недостаточно данных"
	}

	// Рассчитываем MACD
	macdLine, _, _ := a.deps.TechnicalCalculator.CalculateMACD(priceHistory)
	status := a.deps.TechnicalCalculator.GetMACDStatus(priceHistory)
	description := a.deps.TechnicalCalculator.GetMACDDescription(priceHistory)

	return macdLine, status, description
}

// CreateCounterEventData создает плоский map с реальными данными RSI/MACD
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

	// 3. Данные из indicators (8 полей) - flat map с РЕАЛЬНЫМИ значениями
	eventData["current_price"] = signal.EndPrice

	// Получаем реальный объем 24ч из storage
	volume24h := 0.0
	if a.deps.Storage != nil {
		if snapshot, exists := a.deps.Storage.GetCurrentSnapshot(signal.Symbol); exists {
			volume24h = snapshot.GetVolume24h()
		}
	}
	eventData["volume_24h"] = volume24h

	// Получаем реальный OI
	oi := a.GetOI(signal.Symbol)
	eventData["open_interest"] = oi

	// Получаем реальную ставку фандинга
	fundingRate := 0.001 // Заглушка, можно доработать
	if a.deps.Storage != nil {
		if snapshot, exists := a.deps.Storage.GetCurrentSnapshot(signal.Symbol); exists {
			fundingRate = snapshot.GetFundingRate()
		}
	}
	eventData["funding_rate"] = fundingRate

	// ⭐ РЕАЛЬНЫЙ RSI
	rsi, rsiStatus := a.calculateRSI(signal.Symbol, period)
	eventData["rsi"] = rsi
	eventData["rsi_status"] = rsiStatus

	// ⭐ РЕАЛЬНЫЙ MACD
	macdSignal, macdStatus, macdDescription := a.calculateMACD(signal.Symbol, period)
	eventData["macd_signal"] = macdSignal
	eventData["macd_status"] = macdStatus
	eventData["macd_description"] = macdDescription

	// Получаем реальную дельту и процент
	deltaData := a.GetVolumeDelta(signal.Symbol, signal.Direction)
	eventData["volume_delta"] = deltaData.Delta
	eventData["volume_delta_percent"] = deltaData.DeltaPercent

	// 4. Данные прогресса (3 поля) - вложенные в progress map
	eventData["progress"] = map[string]interface{}{
		"filled_groups": 3,    // Заглушка
		"total_groups":  6,    // Заглушка
		"percentage":    50.0, // Заглушка
	}

	logger.Debug("📊 CounterAnalyzer: реальные индикаторы для %s/%s - RSI: %.1f (%s), MACD: %.4f (%s)",
		signal.Symbol, period, rsi, rsiStatus, macdSignal, macdStatus)

	return eventData
}

// isCandleAlreadyProcessed проверяет обрабатывали ли мы уже эту свечу
func (a *CounterAnalyzer) isCandleAlreadyProcessed(candleKey string) bool {
	if a.deps.CandleSystem == nil {
		logger.Warn("⚠️ CandleSystem не инициализирован")
		return false
	}

	// Парсим ключ свечи
	symbol, period, startTime, err := parseCandleKey(candleKey)
	if err != nil {
		logger.Warn("⚠️ Ошибка парсинга ключа свечи %s: %v", candleKey, err)
		return false
	}

	// Используем CandleSystem для проверки
	processed, err := a.deps.CandleSystem.IsCandleProcessed(symbol, period, startTime)
	if err != nil {
		// logger.Warn("⚠️ Ошибка проверки свечи %s через CandleSystem: %v", candleKey, err)
		return false
	}

	return processed
}

// markCandleAsProcessed помечает свечу как обработанную (через CandleSystem)
func (a *CounterAnalyzer) markCandleAsProcessed(candleKey string) bool {
	if a.deps.CandleSystem == nil {
		logger.Warn("⚠️ CandleSystem не инициализирован")
		return false
	}

	// Парсим ключ свечи
	symbol, period, startTime, err := parseCandleKey(candleKey)
	if err != nil {
		logger.Warn("⚠️ Ошибка парсинга ключа свечи %s: %v", candleKey, err)
		return false
	}

	// Используем CandleSystem для атомарной отметки
	marked, err := a.deps.CandleSystem.MarkCandleProcessedAtomically(symbol, period, startTime)
	if err != nil {
		logger.Warn("⚠️ Ошибка отметки свечи %s через CandleSystem: %v", candleKey, err)
		return false
	}

	return marked
}

// parseCandleKey парсит ключ свечи в формате "symbol:period:startTimeUnix"
func parseCandleKey(candleKey string) (symbol, period string, startTime int64, err error) {
	// Формат: symbol:period:startTimeUnix
	// Пример: BTCUSDT:5m:1737897000

	var startTimeInt int64
	n, scanErr := fmt.Sscanf(candleKey, "%s:%s:%d", &symbol, &period, &startTimeInt)
	if scanErr != nil {
		return "", "", 0, fmt.Errorf("ошибка парсинга ключа свечи: %w", scanErr)
	}
	if n != 3 {
		return "", "", 0, fmt.Errorf("неверный формат ключа свечи: %s", candleKey)
	}

	return symbol, period, startTimeInt, nil
}
