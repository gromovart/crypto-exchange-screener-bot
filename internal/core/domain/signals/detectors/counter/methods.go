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
	"math"
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
func (a *CounterAnalyzer) GetVolumeDelta(symbol, direction, period string) *types.VolumeDeltaData {
	// ✅ Используем общий калькулятор из зависимостей
	if a.deps.VolumeCalculator == nil {
		// Создаем временно, если не передан в зависимостях
		logger.Warn("⚠️ Создаем временный VolumeDeltaCalculator для %s", symbol)
		tempCalculator := calculator.NewVolumeDeltaCalculator(a.deps.MarketFetcher, a.deps.Storage)
		defer tempCalculator.Stop() // ✅ ВАЖНО: останавливаем временный калькулятор

		return tempCalculator.CalculateWithFallback(symbol, direction, period)
	}

	return a.deps.VolumeCalculator.CalculateWithFallback(symbol, direction, period)
}

// AnalyzeCandle анализирует свечу (закрытую или активную)
func (a *CounterAnalyzer) AnalyzeCandle(symbol, period string) (*analysis.Signal, error) {
	// ✅ СТАТИСТИКА: инкрементируем общий счетчик вызовов
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

	// 🟢 1. ПРОБУЕМ АНАЛИЗИРОВАТЬ ЗАКРЫТУЮ СВЕЧУ
	signal, err := a.analyzeClosedCandle(symbol, period)
	if err != nil {
		// Логируем ошибку, но продолжаем
		logger.Debug("⚠️ CounterAnalyzer: ошибка анализа закрытой свечи %s/%s: %v",
			symbol, period, err)
	}

	if signal != nil {
		// ✅ СТАТИСТИКА: успешный анализ закрытой свечи
		a.candleStatsMu.Lock()
		if signal.Direction == "growth" {
			a.candleStats.ClosedCandleStats.GrowthSignals++
			a.candleStats.IntervalStats.GrowthSignals++
		} else {
			a.candleStats.ClosedCandleStats.FallSignals++
			a.candleStats.IntervalStats.FallSignals++
		}
		a.candleStatsMu.Unlock()

		return signal, nil
	}

	// 🟡 2. ЕСЛИ НЕТ ЗАКРЫТОЙ - АНАЛИЗИРУЕМ АКТИВНУЮ СВЕЧУ
	signal, err = a.analyzeActiveCandle(symbol, period)
	if err != nil {
		logger.Debug("⚠️ CounterAnalyzer: ошибка анализа активной свечи %s/%s: %v",
			symbol, period, err)
	}

	if signal != nil {
		// ✅ СТАТИСТИКА: успешный анализ активной свечи
		a.candleStatsMu.Lock()
		if signal.Direction == "growth" {
			a.candleStats.ActiveCandleStats.GrowthSignals++
			a.candleStats.IntervalStats.GrowthSignals++
		} else {
			a.candleStats.ActiveCandleStats.FallSignals++
			a.candleStats.IntervalStats.FallSignals++
		}
		a.candleStatsMu.Unlock()
	}

	return signal, nil
}

// analyzeClosedCandle анализирует закрытую свечу
func (a *CounterAnalyzer) analyzeClosedCandle(symbol, period string) (*analysis.Signal, error) {
	// ✅ СТАТИСТИКА: инкрементируем попытки анализа закрытых свечей
	a.candleStatsMu.Lock()
	a.candleStats.ClosedCandleStats.Attempts++
	a.candleStatsMu.Unlock()

	candleData, err := a.deps.CandleSystem.GetLatestClosedCandle(symbol, period)
	if err != nil {
		a.candleStatsMu.Lock()
		a.candleStats.ClosedCandleStats.GetCandleError++
		a.candleStatsMu.Unlock()
		return nil, fmt.Errorf("ошибка получения закрытой свечи %s/%s: %w", symbol, period, err)
	}

	if candleData == nil {
		a.candleStatsMu.Lock()
		a.candleStats.ClosedCandleStats.NoData++
		a.candleStatsMu.Unlock()
		return nil, fmt.Errorf("нет закрытых свечей")
	}

	if !candleData.IsRealFlag || candleData.Open == 0 {
		a.candleStatsMu.Lock()
		a.candleStats.ClosedCandleStats.Unreal++
		a.candleStatsMu.Unlock()
		return nil, fmt.Errorf("нереальная свеча")
	}

	// Атомарная проверка и отметка свечи
	startTimeUnix := candleData.StartTime.Unix()
	marked, err := a.deps.CandleSystem.MarkCandleProcessedAtomically(symbol, period, startTimeUnix)
	if err != nil {
		a.candleStatsMu.Lock()
		a.candleStats.ClosedCandleStats.MarkCandleError++
		a.candleStatsMu.Unlock()
		return nil, fmt.Errorf("ошибка отметки свечи %s/%s: %w", symbol, period, err)
	}

	if !marked {
		a.candleStatsMu.Lock()
		a.candleStats.ClosedCandleStats.AlreadyProcessed++
		a.candleStatsMu.Unlock()
		return nil, fmt.Errorf("уже обработана")
	}

	// Рассчитываем изменение
	changePercent := ((candleData.Close - candleData.Open) / candleData.Open) * 100

	// Проверяем пороги
	growthThreshold := SafeGetFloat(a.config.CustomSettings, "growth_threshold", 0.01) // 0.01%
	fallThreshold := SafeGetFloat(a.config.CustomSettings, "fall_threshold", 0.01)     // 0.01%

	var shouldCreateSignal bool
	var direction string

	if changePercent >= growthThreshold {
		direction = "growth"
		shouldCreateSignal = SafeGetBool(a.config.CustomSettings, "track_growth", true)
	} else if changePercent <= -fallThreshold {
		direction = "fall"
		shouldCreateSignal = SafeGetBool(a.config.CustomSettings, "track_fall", true)
	}

	if !shouldCreateSignal {
		a.candleStatsMu.Lock()
		a.candleStats.ClosedCandleStats.BelowThreshold++
		a.candleStatsMu.Unlock()
		return nil, fmt.Errorf("ниже порога")
	}

	// Создаем сигнал
	signal := a.CreateSignal(symbol, period, direction, changePercent, candleData)
	signal.Metadata.Tags = append(signal.Metadata.Tags, "closed_candle")
	signal.Metadata.Custom["candle_type"] = "closed"

	// ✅ СТАТИСТИКА: успешный анализ закрытой свечи
	a.candleStatsMu.Lock()
	a.candleStats.ClosedCandleStats.Success++
	a.candleStats.IntervalStats.ClosedSignals++
	a.candleStats.IntervalStats.TotalSignals++
	a.candleStatsMu.Unlock()

	return &signal, nil
}

// analyzeActiveCandle анализирует активную свечу
func (a *CounterAnalyzer) analyzeActiveCandle(symbol, period string) (*analysis.Signal, error) {
	// ✅ СТАТИСТИКА: инкрементируем попытки анализа активных свечей
	a.candleStatsMu.Lock()
	a.candleStats.ActiveCandleStats.Attempts++
	a.candleStatsMu.Unlock()

	// Получаем активную свечу через хранилище
	if a.deps.CandleSystem == nil || a.deps.CandleSystem.Storage == nil {
		a.candleStatsMu.Lock()
		a.candleStats.ActiveCandleStats.GetCandleError++
		a.candleStatsMu.Unlock()
		return nil, fmt.Errorf("свечная система не инициализирована")
	}

	candleInterface, exists := a.deps.CandleSystem.Storage.GetActiveCandle(symbol, period)
	if !exists || candleInterface == nil {
		a.candleStatsMu.Lock()
		a.candleStats.ActiveCandleStats.NoActiveCandle++
		a.candleStatsMu.Unlock()
		return nil, fmt.Errorf("нет активной свечи")
	}

	// Конвертируем интерфейс в Candle
	var candle *storage.Candle
	if c, ok := candleInterface.(*storage.Candle); ok {
		candle = c
	} else {
		// Создаем из интерфейса
		candle = &storage.Candle{
			Symbol:       candleInterface.GetSymbol(),
			Period:       candleInterface.GetPeriod(),
			Open:         candleInterface.GetOpen(),
			High:         candleInterface.GetHigh(),
			Low:          candleInterface.GetLow(),
			Close:        candleInterface.GetClose(),
			Volume:       candleInterface.GetVolume(),
			VolumeUSD:    candleInterface.GetVolumeUSD(),
			Trades:       candleInterface.GetTrades(),
			StartTime:    candleInterface.GetStartTime(),
			EndTime:      candleInterface.GetEndTime(),
			IsClosedFlag: candleInterface.IsClosed(),
			IsRealFlag:   candleInterface.IsReal(),
		}
	}

	if !candle.IsRealFlag || candle.Open == 0 {
		a.candleStatsMu.Lock()
		a.candleStats.ActiveCandleStats.InsufficientData++
		a.candleStatsMu.Unlock()
		return nil, fmt.Errorf("недостаточно данных")
	}

	// Проверяем минимальное время свечи для анализа
	elapsed := time.Since(candle.StartTime)
	minTimePercent := SafeGetFloat(a.config.CustomSettings, "active_candle_min_time_percent", 0.3) // 30%

	expectedDuration := periodToDuration(period)
	minTime := expectedDuration * time.Duration(minTimePercent)

	if elapsed < minTime {
		a.candleStatsMu.Lock()
		a.candleStats.ActiveCandleStats.BelowMinTime++
		a.candleStatsMu.Unlock()
		return nil, fmt.Errorf("мало времени")
	}

	// Используем текущую цену из хранилища
	var currentPrice float64
	if snapshot, exists := a.deps.Storage.GetCurrentSnapshot(symbol); exists {
		currentPrice = snapshot.GetPrice()
	} else {
		currentPrice = candle.Close // Используем последнюю закрытую цену свечи
	}

	// Рассчитываем текущее изменение
	changePercent := ((currentPrice - candle.Open) / candle.Open) * 100

	// Более строгие пороги для активных свечей
	activeGrowthThreshold := SafeGetFloat(a.config.CustomSettings, "active_growth_threshold", 0.02) // 0.02%
	activeFallThreshold := SafeGetFloat(a.config.CustomSettings, "active_fall_threshold", 0.02)

	// Дополнительный критерий: объем должен быть значительным
	var volumeOK bool
	if snapshot, exists := a.deps.Storage.GetCurrentSnapshot(symbol); exists {
		minVolume := SafeGetFloat(a.config.CustomSettings, "active_min_volume", 100000) // $100k
		volumeOK = snapshot.GetVolumeUSD() >= minVolume
	} else {
		volumeOK = candle.VolumeUSD >= 100000 // Используем объем свечи
	}

	var shouldCreateSignal bool
	var direction string

	if changePercent >= activeGrowthThreshold && volumeOK {
		direction = "growth"
		shouldCreateSignal = SafeGetBool(a.config.CustomSettings, "track_active_growth", true)
	} else if changePercent <= -activeFallThreshold && volumeOK {
		direction = "fall"
		shouldCreateSignal = SafeGetBool(a.config.CustomSettings, "track_active_fall", true)
	}

	if !shouldCreateSignal {
		a.candleStatsMu.Lock()
		a.candleStats.ActiveCandleStats.BelowThreshold++
		a.candleStatsMu.Unlock()
		return nil, fmt.Errorf("ниже порога")
	}

	// Создаем сигнал с пометкой "active"
	signal := a.CreateSignal(symbol, period, direction, changePercent, candle)
	signal.Metadata.Tags = append(signal.Metadata.Tags, "active_candle")
	signal.Metadata.Custom["candle_type"] = "active"
	signal.Metadata.Custom["elapsed_percent"] = float64(elapsed) / float64(expectedDuration) * 100
	signal.Metadata.Custom["current_price"] = currentPrice
	signal.Metadata.Custom["active_threshold"] = activeGrowthThreshold

	// ✅ СТАТИСТИКА: успешный анализ активной свечи
	a.candleStatsMu.Lock()
	a.candleStats.ActiveCandleStats.Success++
	a.candleStats.IntervalStats.ActiveSignals++
	a.candleStats.IntervalStats.TotalSignals++
	a.candleStatsMu.Unlock()

	return &signal, nil
}

// periodToDuration конвертирует строковый период в time.Duration
func periodToDuration(period string) time.Duration {
	switch period {
	case "1m":
		return 1 * time.Minute
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

// CreateSignal создает сигнал
func (a *CounterAnalyzer) CreateSignal(symbol, period, direction string, changePercent float64,
	candleData *storage.Candle) analysis.Signal {

	// Упрощенный расчет уверенности
	confidence := 50.0
	if math.Abs(changePercent) > 5 {
		confidence = 80
	} else if math.Abs(changePercent) > 2 {
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
		Period:        periodMinutes,
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
				"period_minutes": periodMinutes,
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
	fundingRate := 0.0
	if a.deps.Storage != nil {
		if snapshot, exists := a.deps.Storage.GetCurrentSnapshot(signal.Symbol); exists {
			fundingRate = snapshot.GetFundingRate()
		}
	}
	eventData["funding_rate"] = fundingRate

	// ⭐ ДОБАВЛЯЕМ ВРЕМЯ СЛЕДУЮЩЕГО ФАНДИНГА (заглушка, нужно получить реальное)
	// В Bybit фандинг обычно каждые 8 часов: 00:00, 08:00, 16:00 UTC
	now := time.Now().UTC()
	nextFunding := time.Date(now.Year(), now.Month(), now.Day(),
		(now.Hour()/8+1)*8, 0, 0, 0, time.UTC)
	if nextFunding.Before(now) {
		nextFunding = nextFunding.Add(8 * time.Hour)
	}
	eventData["next_funding_time"] = nextFunding

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
	deltaData := a.GetVolumeDelta(signal.Symbol, signal.Direction, period)
	eventData["volume_delta"] = deltaData.Delta
	eventData["volume_delta_percent"] = deltaData.DeltaPercent
	eventData["delta_source"] = deltaData.Source

	// ⭐ ДОБАВЛЯЕМ ЛИКВИДАЦИИ
	liquidationVolume := 0.0
	longLiqVolume := 0.0
	shortLiqVolume := 0.0

	// Пробуем получить метрики ликвидаций через MarketFetcher
	if a.deps.MarketFetcher != nil {
		// Пытаемся привести к интерфейсу с методом GetLiquidationMetrics
		if fetcher, ok := a.deps.MarketFetcher.(interface {
			GetLiquidationMetrics(string) (interface{}, bool)
		}); ok {
			if metrics, exists := fetcher.GetLiquidationMetrics(signal.Symbol); exists {
				// Пытаемся извлечь данные из интерфейса
				if m, ok := metrics.(struct {
					TotalVolumeUSD float64
					LongLiqVolume  float64
					ShortLiqVolume float64
				}); ok {
					liquidationVolume = m.TotalVolumeUSD
					longLiqVolume = m.LongLiqVolume
					shortLiqVolume = m.ShortLiqVolume
				}
			}
		}
	}

	// Если не получили через MarketFetcher, пробуем через Storage
	if liquidationVolume == 0 && a.deps.Storage != nil {
		if _, exists := a.deps.Storage.GetCurrentSnapshot(signal.Symbol); exists {
			// Некоторые хранилища могут хранить ликвидации
			// Это зависит от реализации
		}
	}

	eventData["liquidation_volume"] = liquidationVolume
	eventData["long_liq_volume"] = longLiqVolume
	eventData["short_liq_volume"] = shortLiqVolume

	// 4. Данные прогресса (3 поля) - вложенные в progress map
	eventData["progress"] = map[string]interface{}{
		"filled_groups": 3,    // Заглушка
		"total_groups":  6,    // Заглушка
		"percentage":    50.0, // Заглушка
	}

	// 5. Зоны S/R (если хранилище доступно)
	// Используем fallback по более старшим периодам, если для текущего зон нет.
	// Причина: зоны пересчитываются только при закрытии свечи (EventCandleClosed),
	// а сигналы генерируются каждые 30 секунд — возникает временной разрыв.
	if a.deps.SRZoneStorage != nil && signal.EndPrice > 0 {
		fallback := srFallbackPeriods(normalizedPeriod)
		nearest, usedPeriod, err := a.deps.SRZoneStorage.GetNearestZonesWithFallback(
			signal.Symbol, normalizedPeriod, signal.EndPrice, fallback,
		)
		if err == nil {
			// Помечаем, какой период зон фактически использован (для отладки и аналитики)
			eventData["sr_zone_period"] = usedPeriod
			if usedPeriod != normalizedPeriod {
				eventData["sr_zone_fallback"] = true
			}
			if nearest.Support != nil {
				eventData["sr_support_price"] = nearest.Support.PriceCenter
				eventData["sr_support_strength"] = nearest.Support.Strength
				eventData["sr_support_dist_pct"] = nearest.DistToSupportPct
				eventData["sr_support_has_wall"] = nearest.Support.HasOrderWall
				eventData["sr_support_wall_usd"] = nearest.Support.OrderWallSizeUSD
			}
			if nearest.Resistance != nil {
				eventData["sr_resistance_price"] = nearest.Resistance.PriceCenter
				eventData["sr_resistance_strength"] = nearest.Resistance.Strength
				eventData["sr_resistance_dist_pct"] = nearest.DistToResistPct
				eventData["sr_resistance_has_wall"] = nearest.Resistance.HasOrderWall
				eventData["sr_resistance_wall_usd"] = nearest.Resistance.OrderWallSizeUSD
			}
		}
	}

	logger.Debug("📊 CounterAnalyzer: реальные индикаторы для %s/%s - RSI: %.1f (%s), MACD: %.4f (%s), ликвидации: $%.0f",
		signal.Symbol, period, rsi, rsiStatus, macdSignal, macdStatus, liquidationVolume)

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

// SafeGetFloat безопасно получает float из map
