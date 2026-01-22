// internal/core/domain/signals/detectors/counter/analyzer.go
package counter

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	candle "crypto-exchange-screener-bot/internal/core/domain/candle"
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/common"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/counter/calculator"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/counter/manager"
	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"

	"github.com/google/uuid"
)

// CounterAnalyzer - анализатор счетчика сигналов (обновлен с поддержкой свечного движка)
type CounterAnalyzer struct {
	config        common.AnalyzerConfig
	stats         common.AnalyzerStats
	marketFetcher interface{}
	storage       storage.PriceStorageInterface
	eventBus      types.EventBus
	candleSystem  *candle.CandleSystem // НОВОЕ: Свечная система

	// Компоненты
	counterManager      *manager.CounterManager
	periodManager       *manager.PeriodManager
	volumeCalculator    *calculator.VolumeDeltaCalculator
	metricsCalculator   *calculator.MarketMetricsCalculator
	techCalculator      *calculator.TechnicalCalculator
	confirmationManager *ConfirmationManager

	mu                  sync.RWMutex
	notificationEnabled bool
	chartProvider       string
	baseThreshold       float64
}

// NewCounterAnalyzer создает новый анализатор счетчика (обновленный конструктор)
func NewCounterAnalyzer(
	config common.AnalyzerConfig,
	storage storage.PriceStorageInterface,
	eventBus types.EventBus,
	marketFetcher interface{},
	candleSystem *candle.CandleSystem, // НОВЫЙ параметр
) *CounterAnalyzer {
	chartProvider := "coinglass"
	if custom, ok := config.CustomSettings["chart_provider"].(string); ok {
		chartProvider = custom
	}

	baseThreshold := 0.1
	if val, ok := config.CustomSettings["base_threshold"].(float64); ok {
		baseThreshold = val
	}

	// Создаем компоненты
	counterManager := manager.NewCounterManager()
	periodManager := manager.NewPeriodManager()
	volumeCalculator := calculator.NewVolumeDeltaCalculator(marketFetcher, storage)
	metricsCalculator := calculator.NewMarketMetricsCalculator(marketFetcher, storage)
	techCalculator := calculator.NewTechnicalCalculator()
	confirmationManager := NewConfirmationManager()

	// Создаем анализатор
	analyzer := &CounterAnalyzer{
		config:              config,
		marketFetcher:       marketFetcher,
		storage:             storage,
		eventBus:            eventBus,
		candleSystem:        candleSystem, // НОВОЕ
		counterManager:      counterManager,
		periodManager:       periodManager,
		volumeCalculator:    volumeCalculator,
		metricsCalculator:   metricsCalculator,
		techCalculator:      techCalculator,
		confirmationManager: confirmationManager,
		notificationEnabled: true,
		chartProvider:       chartProvider,
		baseThreshold:       baseThreshold,
		stats:               common.AnalyzerStats{},
	}

	logger.Info("✅ CounterAnalyzer создан с поддержкой свечного движка")
	return analyzer
}

// AnalyzeAllSymbols анализирует все символы каждую минуту
func (a *CounterAnalyzer) AnalyzeAllSymbols(symbols []string) error {
	startTime := time.Now()
	var signals []analysis.Signal

	// Определяем все периоды для анализа
	periods := []string{"5m", "15m", "30m", "1h", "4h", "1d"}

	// Для каждого символа
	for _, symbol := range symbols {
		// Для каждого периода
		for _, period := range periods {
			// Получаем данные за период
			data, err := a.getDataForPeriod(symbol, period)
			if err != nil {
				// Пропускаем если нет данных
				continue
			}

			// Анализируем
			signal, err := a.analyzeSymbolPeriod(symbol, period, data)
			if err != nil {
				continue
			}

			if signal != nil {
				signals = append(signals, *signal)
			}
		}
	}

	// Отправляем статистику
	a.updateStats(time.Since(startTime), len(signals) > 0)

	return nil
}

// analyzeSymbolPeriod анализирует конкретный символ и период
func (a *CounterAnalyzer) analyzeSymbolPeriod(symbol, period string, data []types.PriceData) (*analysis.Signal, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("insufficient data for %s period %s", symbol, period)
	}

	// Рассчитываем изменение за весь период
	change := a.calculateChangeOverPeriod(data)

	// Проверяем базовый порог (0.1% по умолчанию)
	if math.Abs(change) < a.baseThreshold {
		// Изменение слишком маленькое, пропускаем
		return nil, nil
	}

	// Добавляем подтверждение в менеджер
	isReady, confirmations := a.confirmationManager.AddConfirmation(symbol, period)

	if isReady {
		// Создаем сырой сигнал
		signal := a.createRawSignal(symbol, period, change, confirmations, data)

		// Публикуем в EventBus
		a.publishRawCounterSignal(signal)

		// Сбрасываем счетчик подтверждений
		a.confirmationManager.Reset(symbol, period)

		return &signal, nil
	}

	return nil, nil
}

// Analyze - совместимый метод для AnalysisEngine
func (a *CounterAnalyzer) Analyze(data []types.PriceData, cfg common.AnalyzerConfig) ([]analysis.Signal, error) {
	// ВРЕМЕННОЕ РЕШЕНИЕ для совместимости с AnalysisEngine

	if len(data) < 2 {
		return nil, fmt.Errorf("insufficient data points")
	}

	symbol := data[0].Symbol

	// Рассчитываем изменение
	change := a.calculateChangeOverPeriod(data)

	// Используем период из конфига или дефолтный
	period := "15m"
	if customPeriod, ok := cfg.CustomSettings["analysis_period"].(string); ok {
		period = customPeriod
	}

	// Проверяем порог
	if math.Abs(change) < a.baseThreshold {
		return nil, nil
	}

	// Добавляем подтверждение
	isReady, confirmations := a.confirmationManager.AddConfirmation(symbol, period)

	if !isReady {
		// Еще не готов, ждем больше подтверждений
		return nil, nil
	}

	// Создаем сигнал через новую систему
	signal := a.createRawSignal(symbol, period, change, confirmations, data)

	// Публикуем в EventBus
	a.publishRawCounterSignal(signal)

	// Сбрасываем счетчик подтверждений
	a.confirmationManager.Reset(symbol, period)

	return []analysis.Signal{signal}, nil
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

	latestData := data[len(data)-1]
	candleStartPrice := data[0].Price
	candleEndPrice := latestData.Price
	candleStartTime := data[0].Timestamp
	candleEndTime := latestData.Timestamp

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
		candleStartPrice, candleEndPrice, change)
	logger.Info("   • Время: %s → %s",
		candleStartTime.Format("15:04:05"), candleEndTime.Format("15:04:05"))
	logger.Info("   • Подтверждений: %d/%d",
		confirmations, GetRequiredConfirmations(period))
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
	customMap["required_confirmations"] = GetRequiredConfirmations(period)

	// Данные свечи
	customMap["candle_open_price"] = candleStartPrice
	customMap["candle_close_price"] = candleEndPrice
	customMap["candle_open_time"] = candleStartTime
	customMap["candle_close_time"] = candleEndTime
	customMap["candle_duration_minutes"] = candleEndTime.Sub(candleStartTime).Minutes()
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
		StartPrice:    candleStartPrice, // Цена открытия свечи
		EndPrice:      candleEndPrice,   // Цена закрытия свечи
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
				"required_confirmations": float64(GetRequiredConfirmations(period)),

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
				"candle_open_price":     candleStartPrice,
				"candle_close_price":    candleEndPrice,
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

	// Проверяем ToMap()
	signalMap := signal.ToMap()
	logger.Debug("   ToMap() результат (важные поля):\n")
	for key, value := range signalMap {
		if key == "change_percent" || key == "period" || key == "custom" ||
			key == "period_string" || key == "symbol" || key == "direction" {
			logger.Debug("      %s: %v (тип: %T)\n", key, value, value)
		}
	}

	// Создаем событие с сырыми данными
	event := types.Event{
		Type:      types.EventCounterSignalDetected,
		Source:    "counter_analyzer_raw",
		Data:      signalMap,
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

// getDataForPeriod получает данные за указанный период (обновлен с использованием свечного движка)
func (a *CounterAnalyzer) getDataForPeriod(symbol, period string) ([]types.PriceData, error) {
	if a.candleSystem != nil {
		// НОВОЕ: Используем свечной движок для получения свечи
		candleData, err := a.getCandleData(symbol, period)
		if err == nil {
			return candleData, nil
		}
		logger.Debug("⚠️ Не удалось получить свечу из движка: %v, используем старый метод", err)
	}

	// Старый метод как fallback
	return a.getDataForPeriodLegacy(symbol, period)
}

// getCandleData получает данные из свечного движка
func (a *CounterAnalyzer) getCandleData(symbol, period string) ([]types.PriceData, error) {
	// Получаем свечу из движка
	candle, err := a.candleSystem.GetCandle(symbol, period)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения свечи: %w", err)
	}

	if candle == nil || !candle.IsReal {
		return nil, fmt.Errorf("свеча не содержит реальных данных")
	}

	// Получаем историю цен для этой свечи
	prices, err := a.storage.GetPriceHistoryRange(symbol, candle.StartTime, candle.EndTime)
	if err != nil {
		// Если не можем получить историю, используем OHLC данные свечи
		return a.convertCandleToPriceData(candle), nil
	}

	// Конвертируем в types.PriceData
	return convertStoragePricesToTypes(prices), nil
}

// convertCandleToPriceData конвертирует свечу в массив PriceData
func (a *CounterAnalyzer) convertCandleToPriceData(c *candle.Candle) []types.PriceData {
	// Создаем две точки: открытие и закрытие свечи
	openData := types.PriceData{
		Symbol:    c.Symbol,
		Price:     c.Open,
		Timestamp: c.StartTime,
	}

	closeData := types.PriceData{
		Symbol:    c.Symbol,
		Price:     c.Close,
		Timestamp: c.EndTime,
	}

	// Получаем текущие метрики для символа
	if metrics, exists := a.storage.GetSymbolMetrics(c.Symbol); exists {
		openData.Volume24h = metrics.Volume24h
		openData.OpenInterest = metrics.OpenInterest
		openData.FundingRate = metrics.FundingRate
		openData.Change24h = metrics.Change24h

		closeData.Volume24h = metrics.Volume24h
		closeData.OpenInterest = metrics.OpenInterest
		closeData.FundingRate = metrics.FundingRate
		closeData.Change24h = metrics.Change24h
	}

	return []types.PriceData{openData, closeData}
}

// convertStoragePricesToTypes конвертирует storage.PriceData в types.PriceData
func convertStoragePricesToTypes(prices []storage.PriceData) []types.PriceData {
	var result []types.PriceData
	for _, price := range prices {
		result = append(result, types.PriceData{
			Symbol:       price.Symbol,
			Price:        price.Price,
			Volume24h:    price.Volume24h,
			VolumeUSD:    price.VolumeUSD,
			Timestamp:    price.Timestamp,
			OpenInterest: price.OpenInterest,
			FundingRate:  price.FundingRate,
			Change24h:    price.Change24h,
			High24h:      price.High24h,
			Low24h:       price.Low24h,
		})
	}
	return result
}

// getDataForPeriodLegacy старый метод получения данных (для совместимости)
func (a *CounterAnalyzer) getDataForPeriodLegacy(symbol, period string) ([]types.PriceData, error) {
	if a.storage == nil {
		logger.Error("⚠️ Storage не инициализирован для %s", symbol)
		return a.getFallbackData(symbol, period)
	}

	// Определяем длительность периода
	periodDuration := getPeriodDuration(period)
	endTime := time.Now()
	startTime := endTime.Add(-periodDuration)

	logger.Info("🔍 getDataForPeriodLegacy: %s за %s (%s - %s)",
		symbol, period, startTime.Format("15:04:05"), endTime.Format("15:04:05"))

	// Получаем историю цен за период
	priceHistory, err := a.storage.GetPriceHistoryRange(symbol, startTime, endTime)
	if err != nil {
		logger.Warn("⚠️ Ошибка получения истории для %s: %v", symbol, err)
		return a.getFallbackData(symbol, period)
	}

	if len(priceHistory) == 0 {
		logger.Warn("⚠️ Нет данных для %s за %s", symbol, period)
		return a.getFallbackData(symbol, period)
	}

	// Сортируем по времени
	sort.Slice(priceHistory, func(i, j int) bool {
		return priceHistory[i].Timestamp.Before(priceHistory[j].Timestamp)
	})

	// Конвертируем в types.PriceData
	return convertStoragePricesToTypes(priceHistory), nil
}

// getFallbackData возвращает заглушку если нет реальных данных
func (a *CounterAnalyzer) getFallbackData(symbol, period string) ([]types.PriceData, error) {
	logger.Warn("⚠️ Использую fallback данные для %s", symbol)

	// Пробуем получить текущий снапшот
	var currentPrice, volume24h, openInterest, fundingRate float64

	if a.storage != nil {
		if snapshot, exists := a.storage.GetCurrentSnapshot(symbol); exists {
			currentPrice = snapshot.Price
			volume24h = snapshot.Volume24h
			openInterest = snapshot.OpenInterest
			fundingRate = snapshot.FundingRate

			logger.Debug("   Найден снапшот: цена=%.4f, объем=%.0f, OI=%.0f",
				currentPrice, volume24h, openInterest)
		}
	}

	// Если нет снапшота, используем дефолтные значения
	if currentPrice == 0 {
		currentPrice = 1.0
		volume24h = 1000000
		openInterest = 500000
		fundingRate = 0.0001
	}

	// Создаем две точки данных с небольшим изменением
	startTime := time.Now().Add(-getPeriodDuration(period))

	// Небольшое случайное изменение (±0.5%)
	changePercent := (float64(time.Now().UnixNano()%100) - 50) / 10000 // ±0.5%
	startPrice := currentPrice / (1 + changePercent/100)

	return []types.PriceData{
		{
			Symbol:       symbol,
			Price:        startPrice,
			Volume24h:    volume24h,
			OpenInterest: openInterest,
			FundingRate:  fundingRate,
			Timestamp:    startTime,
		},
		{
			Symbol:       symbol,
			Price:        currentPrice,
			Volume24h:    volume24h,
			OpenInterest: openInterest,
			FundingRate:  fundingRate,
			Timestamp:    time.Now(),
		},
	}, nil
}

// calculateChangeOverPeriod рассчитывает изменение за период
func (a *CounterAnalyzer) calculateChangeOverPeriod(data []types.PriceData) float64 {
	if len(data) < 2 {
		return 0
	}

	// Для свечи: берем первую и последнюю точку (открытие и закрытие)
	startPrice := data[0].Price
	endPrice := data[len(data)-1].Price

	// Строгий расчет как у свечи
	change := ((endPrice - startPrice) / startPrice) * 100

	// Дополнительная проверка: время должно соответствовать периоду
	startTime := data[0].Timestamp
	endTime := data[len(data)-1].Timestamp
	actualDuration := endTime.Sub(startTime)
	expectedDuration := getPeriodDurationFromData(data)

	// Если данные покрывают менее 50% периода, результат ненадежен
	coverageRatio := actualDuration.Seconds() / expectedDuration.Seconds()
	if coverageRatio < 0.5 {
		logger.Debug("⚠️ Малое покрытие данных для %s: %.0f%% периода",
			data[0].Symbol, coverageRatio*100)
		// Можно скорректировать изменение пропорционально покрытию
		change = change * coverageRatio
	}

	logger.Info("📊 Изменение свечи %s: %.6f → %.6f = %.2f%% (покрытие: %.0f%%)",
		data[0].Symbol, startPrice, endPrice, change, coverageRatio*100)

	return change
}

// getPeriodDurationFromData определяет период на основе данных
func getPeriodDurationFromData(data []types.PriceData) time.Duration {
	if len(data) < 2 {
		return 15 * time.Minute // дефолтный период
	}

	// Пытаемся определить период по разнице времени
	timeDiffs := make([]time.Duration, 0)
	for i := 1; i < len(data); i++ {
		diff := data[i].Timestamp.Sub(data[i-1].Timestamp)
		if diff > 0 {
			timeDiffs = append(timeDiffs, diff)
		}
	}

	if len(timeDiffs) == 0 {
		return 15 * time.Minute
	}

	// Находим наиболее частый интервал
	freq := make(map[time.Duration]int)
	for _, diff := range timeDiffs {
		// Округляем до ближайшей минуты
		rounded := diff.Round(time.Minute)
		freq[rounded]++
	}

	var mostCommon time.Duration
	maxCount := 0
	for period, count := range freq {
		if count > maxCount {
			maxCount = count
			mostCommon = period
		}
	}

	// Стандартные периоды
	standardPeriods := []time.Duration{
		5 * time.Minute,
		15 * time.Minute,
		30 * time.Minute,
		1 * time.Hour,
		4 * time.Hour,
		24 * time.Hour,
	}

	// Находим ближайший стандартный период
	var closestPeriod time.Duration
	minDiff := time.Duration(1<<63 - 1)
	for _, std := range standardPeriods {
		diff := mostCommon - std
		if diff < 0 {
			diff = -diff
		}
		if diff < minDiff {
			minDiff = diff
			closestPeriod = std
		}
	}

	return closestPeriod
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
			result += fmt.Sprintf("✅ %s: %.6f → %.6f (%.2f%%)",
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
	storageStats := stats["storage_stats"].(candle.CandleStats)
	result += fmt.Sprintf("\n📊 Статистика системы:\n")
	result += fmt.Sprintf("• Активных свечей: %d\n", storageStats.ActiveCandles)
	result += fmt.Sprintf("• Всего свечей: %d\n", storageStats.TotalCandles)
	result += fmt.Sprintf("• Символов: %d\n", storageStats.SymbolsCount)

	return result
}

// getHistoryFromCandles получает историю свечей для анализа
func (a *CounterAnalyzer) getHistoryFromCandles(symbol, period string, limit int) ([]*candle.Candle, error) {
	if a.candleSystem == nil {
		return nil, fmt.Errorf("свечная система не инициализирована")
	}

	return a.candleSystem.GetHistory(symbol, period, limit)
}

// Старые методы для обратной совместимости
func (a *CounterAnalyzer) Name() string                { return "counter_analyzer" }
func (a *CounterAnalyzer) Version() string             { return "2.5.0" }
func (a *CounterAnalyzer) Supports(symbol string) bool { return true }

func (a *CounterAnalyzer) GetConfig() common.AnalyzerConfig { return a.config }
func (a *CounterAnalyzer) GetStats() common.AnalyzerStats {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.stats
}

func (a *CounterAnalyzer) updateStats(duration time.Duration, success bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.stats.TotalCalls++
	a.stats.TotalTime += duration
	a.stats.LastCallTime = time.Now()

	if success {
		a.stats.SuccessCount++
	} else {
		a.stats.ErrorCount++
	}

	if a.stats.TotalCalls > 0 {
		a.stats.AverageTime = time.Duration(
			int64(a.stats.TotalTime) / int64(a.stats.TotalCalls),
		)
	}
}

// Методы для обратной совместимости
func (a *CounterAnalyzer) SetNotificationEnabled(enabled bool) {
	a.notificationEnabled = enabled
}

func (a *CounterAnalyzer) SetChartProvider(provider string) {
	a.chartProvider = provider
}

func (a *CounterAnalyzer) SetAnalysisPeriod(period string) {
	custom := make(map[string]interface{})
	for k, v := range a.config.CustomSettings {
		custom[k] = v
	}
	custom["analysis_period"] = period
	a.config.CustomSettings = custom
	a.counterManager.ResetAllCounters(period)
}

func (a *CounterAnalyzer) GetAllCounters() map[string]manager.SignalCounter {
	return a.counterManager.GetAllCounters()
}

func (a *CounterAnalyzer) GetCounterStats(symbol string) (manager.SignalCounter, bool) {
	return a.counterManager.GetCounterStats(symbol)
}

func (a *CounterAnalyzer) SetTrackingOptions(symbol string, trackGrowth, trackFall bool) error {
	counter, exists := a.counterManager.GetCounter(symbol)
	if !exists {
		return fmt.Errorf("counter for symbol %s not found", symbol)
	}

	counter.Lock()
	counter.Settings.TrackGrowth = trackGrowth
	counter.Settings.TrackFall = trackFall
	counter.Unlock()
	return nil
}

// TestVolumeDeltaConnection тестирует подключение к API дельты
func (a *CounterAnalyzer) TestVolumeDeltaConnection(symbol string) error {
	if a.volumeCalculator == nil {
		return fmt.Errorf("volume calculator not initialized")
	}
	return a.volumeCalculator.TestConnection(symbol)
}

// GetVolumeDeltaCacheInfo возвращает информацию о кэше дельты
func (a *CounterAnalyzer) GetVolumeDeltaCacheInfo() map[string]interface{} {
	if a.volumeCalculator == nil {
		return map[string]interface{}{"error": "volume calculator not initialized"}
	}
	return a.volumeCalculator.GetCacheInfo()
}

// ClearVolumeDeltaCache очищает кэш дельты
func (a *CounterAnalyzer) ClearVolumeDeltaCache() {
	if a.volumeCalculator != nil {
		a.volumeCalculator.ClearCache()
	}
}

// TestNotification отправляет тестовое уведомление через EventBus
func (a *CounterAnalyzer) TestNotification(symbol string) error {
	if a.eventBus == nil {
		return fmt.Errorf("eventBus not initialized")
	}

	// Создаем тестовый Counter сигнал
	testData := map[string]interface{}{
		"symbol":        symbol,
		"direction":     "growth",
		"change":        2.5,
		"signal_count":  1,
		"max_signals":   5,
		"current_price": 100.0,
		"volume_24h":    1000000.0,
		"open_interest": 500000.0,
		"funding_rate":  0.0005,
		"period":        "15 минут",
		"timestamp":     time.Now(),
	}

	event := types.Event{
		Type:      types.EventCounterSignalDetected,
		Source:    "counter_analyzer",
		Data:      testData,
		Timestamp: time.Now(),
	}

	return a.eventBus.Publish(event)
}

// GetNotifierStats возвращает статистику нотификатора (теперь через EventBus)
func (a *CounterAnalyzer) GetNotifierStats() map[string]interface{} {
	if a.eventBus == nil {
		return map[string]interface{}{"error": "eventBus not initialized"}
	}

	// Получаем метрики EventBus
	metrics := a.eventBus.GetMetrics()

	return map[string]interface{}{
		"event_bus_metrics": map[string]interface{}{
			"events_published": metrics.EventsPublished,
			"events_processed": metrics.EventsProcessed,
			"events_failed":    metrics.EventsFailed,
		},
		"notification_enabled": a.notificationEnabled,
		"chart_provider":       a.chartProvider,
	}
}

// Вспомогательные методы для получения настроек
func (a *CounterAnalyzer) getBasePeriodMinutes(cfg common.AnalyzerConfig) int {
	if val, ok := cfg.CustomSettings["base_period_minutes"].(int); ok {
		return val
	}
	return 1
}

func (a *CounterAnalyzer) getCurrentPeriod(cfg common.AnalyzerConfig) string {
	if val, ok := cfg.CustomSettings["analysis_period"].(string); ok {
		return val
	}
	return "15m"
}

// TestDeltaConnection тестирует подключение к API дельты
func (a *CounterAnalyzer) TestDeltaConnection(symbol string) string {
	if a.volumeCalculator == nil {
		return "❌ VolumeCalculator не инициализирован"
	}
	err := a.volumeCalculator.TestConnection(symbol)
	if err != nil {
		return fmt.Sprintf("❌ Ошибка тестирования дельты для %s:\n%s", symbol, err.Error())
	}
	cacheInfo := a.volumeCalculator.GetCacheInfo()
	cacheSize := cacheInfo["cache_size"].(int)
	return fmt.Sprintf("✅ Тест дельты для %s пройден!\n📦 Размер кэша: %d", symbol, cacheSize)
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

// getRequiredPointsForPeriod возвращает необходимое количество точек для периода
func (a *CounterAnalyzer) getRequiredPointsForPeriod(period string) int {
	switch period {
	case "5m":
		return 6 // для RSI(14) нужно минимум 14, но для 5м берем меньше
	case "15m":
		return 10
	case "30m":
		return 15
	case "1h":
		return 20
	case "4h":
		return 25
	case "1d":
		return 30
	default:
		return 15
	}
}

// getInterpolatedData создает интерполированные данные если недостаточно точек
func (a *CounterAnalyzer) getInterpolatedData(symbol, period string,
	existingData []storage.PriceData, requiredPoints int) ([]types.PriceData, error) {

	if len(existingData) == 0 {
		return a.getFallbackData(symbol, period)
	}

	// Если есть только 1 точка, создаем небольшой тренд
	if len(existingData) == 1 {
		var result []types.PriceData
		point := existingData[0]
		// Создаем небольшой восходящий тренд (+0.01% на точку)
		trendPercent := 0.0001 // +0.01% на точку

		for i := 0; i < requiredPoints; i++ {
			// Рассчитываем цену с небольшим трендом
			priceMultiplier := 1.0 + (float64(i) * trendPercent)
			// Добавляем небольшой случайный шум (±0.005%)
			noise := (float64(time.Now().UnixNano()%100) - 50.0) / 1000000.0 // ±0.005%

			result = append(result, types.PriceData{
				Symbol:       symbol,
				Price:        point.Price*priceMultiplier + noise,
				Volume24h:    point.Volume24h,
				OpenInterest: point.OpenInterest,
				FundingRate:  point.FundingRate,
				Timestamp:    point.Timestamp.Add(time.Duration(i) * time.Minute),
				Change24h:    point.Change24h,
				High24h:      point.High24h * priceMultiplier,
				Low24h:       point.Low24h * priceMultiplier,
			})
		}
		logger.Warn("⚠️ Интерполяция %s: 1 точка → %d точек", symbol, requiredPoints)
		return result, nil
	}

	// Линейная интерполяция между существующими точками
	var result []types.PriceData

	// Сортируем по времени
	sort.Slice(existingData, func(i, j int) bool {
		return existingData[i].Timestamp.Before(existingData[j].Timestamp)
	})

	// Временной диапазон существующих данных
	timeRange := existingData[len(existingData)-1].Timestamp.Sub(existingData[0].Timestamp)
	if timeRange <= 0 {
		timeRange = time.Duration(requiredPoints) * time.Minute
	}

	// Время между интерполированными точками
	timeStep := timeRange / time.Duration(requiredPoints-1)

	// Интерполяция
	for i := 0; i < requiredPoints; i++ {
		currentTime := existingData[0].Timestamp.Add(timeStep * time.Duration(i))

		// Находим две ближайшие точки для интерполяции
		var prev, next *storage.PriceData
		for j := 0; j < len(existingData)-1; j++ {
			if !existingData[j].Timestamp.After(currentTime) && existingData[j+1].Timestamp.After(currentTime) {
				prev = &existingData[j]
				next = &existingData[j+1]
				break
			}
		}

		var price, volume, oi, funding float64
		var timestamp time.Time

		if prev != nil && next != nil {
			// Линейная интерполяция
			timeRatio := float64(currentTime.Sub(prev.Timestamp)) / float64(next.Timestamp.Sub(prev.Timestamp))
			price = prev.Price + (next.Price-prev.Price)*timeRatio
			volume = prev.Volume24h + (next.Volume24h-prev.Volume24h)*timeRatio
			oi = prev.OpenInterest + (next.OpenInterest-prev.OpenInterest)*timeRatio
			funding = prev.FundingRate + (next.FundingRate-prev.FundingRate)*timeRatio
			timestamp = currentTime
		} else {
			// Используем ближайшую точку
			if i == 0 {
				price = existingData[0].Price
				timestamp = existingData[0].Timestamp
			} else {
				price = existingData[len(existingData)-1].Price
				timestamp = existingData[len(existingData)-1].Timestamp
			}
			volume = existingData[0].Volume24h
			oi = existingData[0].OpenInterest
			funding = existingData[0].FundingRate
		}

		result = append(result, types.PriceData{
			Symbol:       symbol,
			Price:        price,
			Volume24h:    volume,
			OpenInterest: oi,
			FundingRate:  funding,
			Timestamp:    timestamp,
			Change24h:    existingData[0].Change24h,
			High24h:      existingData[0].High24h,
			Low24h:       existingData[0].Low24h,
		})
	}

	logger.Warn("⚠️ Интерполяция %s: %d точек → %d точек",
		symbol, len(existingData), requiredPoints)
	return result, nil
}
