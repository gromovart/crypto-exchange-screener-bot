// internal/core/domain/signals/detectors/counter/analyzer.go
package counter

import (
	"fmt"
	"math"
	"sync"
	"time"

	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/common"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/counter/calculator"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/counter/manager"
	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/in_memory_storage"
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"

	"github.com/google/uuid"
)

// CounterAnalyzer - анализатор счетчика сигналов
type CounterAnalyzer struct {
	config        common.AnalyzerConfig
	stats         common.AnalyzerStats
	marketFetcher interface{}
	storage       storage.PriceStorage
	eventBus      types.EventBus

	// Компоненты
	counterManager    *manager.CounterManager
	periodManager     *manager.PeriodManager
	volumeCalculator  *calculator.VolumeDeltaCalculator
	metricsCalculator *calculator.MarketMetricsCalculator
	techCalculator    *calculator.TechnicalCalculator

	// НОВОЕ: Менеджер подтверждений для анализатора
	confirmationManager *ConfirmationManager

	mu                  sync.RWMutex
	notificationEnabled bool
	chartProvider       string

	// НОВОЕ: Базовый порог для всех (из конфига)
	baseThreshold float64
}

// NewCounterAnalyzer создает новый анализатор счетчика
func NewCounterAnalyzer(
	config common.AnalyzerConfig,
	storage storage.PriceStorage,
	eventBus types.EventBus,
	marketFetcher interface{},
) *CounterAnalyzer {
	chartProvider := "coinglass"
	if custom, ok := config.CustomSettings["chart_provider"].(string); ok {
		chartProvider = custom
	}

	// Базовый порог из конфига (по умолчанию 0.1%)
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

	// НОВОЕ: Создаем менеджер подтверждений
	confirmationManager := NewConfirmationManager()

	// Создаем анализатор
	analyzer := &CounterAnalyzer{
		config:              config,
		marketFetcher:       marketFetcher,
		storage:             storage,
		eventBus:            eventBus, // ✅ УСТАНОВЛЕНО
		counterManager:      counterManager,
		periodManager:       periodManager,
		volumeCalculator:    volumeCalculator,
		metricsCalculator:   metricsCalculator,
		techCalculator:      techCalculator,
		confirmationManager: confirmationManager, // НОВОЕ
		notificationEnabled: true,
		chartProvider:       chartProvider,
		baseThreshold:       baseThreshold, // НОВОЕ
		stats:               common.AnalyzerStats{},
	}

	return analyzer
}

// AnalyzeAllSymbols анализирует все символы каждую минуту
// НОВЫЙ МЕТОД: Вместо старого Analyze
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
	change float64,
	confirmations int,
	data []types.PriceData,
) analysis.Signal {
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

	rsi := a.techCalculator.CalculateRSI(data)

	// НОВОЕ: Получаем все компоненты MACD вместо одного значения
	macdLine, signalLine, histogram := a.techCalculator.CalculateMACD(data)
	// Используем MACD линию как основной сигнал (для обратной совместимости)
	macdSignal := macdLine

	periodMinutes := getPeriodMinutes(period)

	// СОЗДАЕМ Custom map
	customMap := make(map[string]interface{})
	customMap["delta_source"] = deltaSource
	customMap["period_string"] = period
	customMap["period_minutes"] = periodMinutes
	customMap["base_threshold"] = a.baseThreshold
	customMap["change_percent"] = change
	customMap["symbol"] = symbol
	customMap["confirmations"] = confirmations
	customMap["required_confirmations"] = GetRequiredConfirmations(period)

	// НОВОЕ: Добавляем MACD компоненты в custom
	customMap["macd_line"] = macdLine
	customMap["macd_signal_line"] = signalLine
	customMap["macd_histogram"] = histogram

	return analysis.Signal{
		ID:            uuid.New().String(),
		Symbol:        symbol,
		Type:          "counter_raw",
		Direction:     a.getDirection(change),
		ChangePercent: change,
		Period:        periodMinutes,
		Confidence:    float64(confirmations),
		DataPoints:    len(data),
		StartPrice:    data[0].Price,
		EndPrice:      latestData.Price,
		Timestamp:     time.Now(),
		Metadata: analysis.Metadata{
			Strategy: "counter_analyzer_raw",
			Tags: []string{
				"counter_raw",
				a.getDirection(change),
				period,
				fmt.Sprintf("confirmations_%d", confirmations),
			},
			Indicators: map[string]float64{
				"period":                 float64(periodMinutes),
				"confirmations":          float64(confirmations),
				"required_confirmations": float64(GetRequiredConfirmations(period)),
				"volume_24h":             latestData.Volume24h,
				"open_interest":          latestData.OpenInterest,
				"funding_rate":           latestData.FundingRate,
				"current_price":          latestData.Price,
				"volume_delta":           volumeDelta,
				"volume_delta_percent":   volumeDeltaPercent,
				"rsi":                    rsi,
				"macd_signal":            macdSignal, // Для обратной совместимости
				"macd_line":              macdLine,   // НОВОЕ
				"macd_signal_line":       signalLine, // НОВОЕ
				"macd_histogram":         histogram,  // НОВОЕ
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

// getDataForPeriod получает данные за указанный период
func (a *CounterAnalyzer) getDataForPeriod(symbol, period string) ([]types.PriceData, error) {
	if a.storage == nil {
		logger.Error("⚠️ Storage не инициализирован для %s\n", symbol)
		return a.getFallbackData(symbol, period)
	}

	// Получаем длительность периода
	periodDuration := getPeriodDuration(period)
	endTime := time.Now()
	startTime := endTime.Add(-periodDuration)

	logger.Debug("🔍 getDataForPeriod: %s за %s (%s - %s)\n",
		symbol, period, startTime.Format("15:04:05"), endTime.Format("15:04:05"))

	// Пробуем получить историю цен за период
	priceHistory, err := a.storage.GetPriceHistoryRange(symbol, startTime, endTime)
	if err != nil {
		logger.Warn("⚠️ Ошибка получения истории для %s: %v\n", symbol, err)

		// Fallback: получаем последние N точек
		priceHistory, err = a.storage.GetPriceHistory(symbol, 10)
		if err != nil {
			logger.Error("❌ Не удалось получить данные для %s: %v\n", symbol, err)
			return a.getFallbackData(symbol, period)
		}
	}

	if len(priceHistory) < 2 {
		logger.Warn("⚠️ Недостаточно данных для %s: %d точек\n", symbol, len(priceHistory))
		return a.getFallbackData(symbol, period)
	}

	// Конвертируем storage.PriceData в types.PriceData
	var result []types.PriceData
	for _, priceData := range priceHistory {
		result = append(result, types.PriceData{
			Symbol:       priceData.Symbol,
			Price:        priceData.Price,
			Volume24h:    priceData.Volume24h,
			OpenInterest: priceData.OpenInterest,
			FundingRate:  priceData.FundingRate,
			Timestamp:    priceData.Timestamp,
			Change24h:    priceData.Change24h,
			High24h:      priceData.High24h,
			Low24h:       priceData.Low24h,
		})
	}

	logger.Info("✅ Получено %d точек данных для %s за %s\n",
		len(result), symbol, period)

	return result, nil
}

// getFallbackData возвращает заглушку если нет реальных данных
func (a *CounterAnalyzer) getFallbackData(symbol, period string) ([]types.PriceData, error) {
	logger.Warn("⚠️ Использую fallback данные для %s\n", symbol)

	// Пробуем получить текущий снапшот
	var currentPrice, volume24h, openInterest, fundingRate float64

	if a.storage != nil {
		if snapshot, exists := a.storage.GetCurrentSnapshot(symbol); exists {
			currentPrice = snapshot.Price
			volume24h = snapshot.Volume24h
			openInterest = snapshot.OpenInterest
			fundingRate = snapshot.FundingRate

			logger.Debug("   Найден снапшот: цена=%.4f, объем=%.0f, OI=%.0f\n",
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
	startPrice := data[0].Price
	endPrice := data[len(data)-1].Price
	return ((endPrice - startPrice) / startPrice) * 100
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
