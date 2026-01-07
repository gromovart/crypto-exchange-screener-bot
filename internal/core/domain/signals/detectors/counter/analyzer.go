// internal/core/domain/signals/detectors/counter/analyzer.go
package counter

import (
	"fmt"
	"sync"
	"time"

	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/common"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/counter/calculator"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/counter/manager"
	"crypto-exchange-screener-bot/internal/types"
)

// CounterAnalyzer - анализатор счетчика сигналов
type CounterAnalyzer struct {
	config        common.AnalyzerConfig
	stats         common.AnalyzerStats
	marketFetcher interface{}
	storage       interface{}
	eventBus      types.EventBus // ✅ ДОБАВЛЕНО: EventBus для публикации событий

	// Компоненты
	counterManager    *manager.CounterManager
	periodManager     *manager.PeriodManager
	signalProcessor   *SignalProcessor
	volumeCalculator  *calculator.VolumeDeltaCalculator
	metricsCalculator *calculator.MarketMetricsCalculator
	techCalculator    *calculator.TechnicalCalculator

	mu                  sync.RWMutex
	notificationEnabled bool
	chartProvider       string
}

// NewCounterAnalyzer создает новый анализатор счетчика
func NewCounterAnalyzer(
	config common.AnalyzerConfig,
	storage interface{},
	eventBus types.EventBus,
	marketFetcher interface{},
) *CounterAnalyzer {
	chartProvider := "coinglass"
	if custom, ok := config.CustomSettings["chart_provider"].(string); ok {
		chartProvider = custom
	}

	// Создаем компоненты
	counterManager := manager.NewCounterManager()
	periodManager := manager.NewPeriodManager()
	volumeCalculator := calculator.NewVolumeDeltaCalculator(marketFetcher, storage)
	metricsCalculator := calculator.NewMarketMetricsCalculator(marketFetcher, storage)
	techCalculator := calculator.NewTechnicalCalculator()

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
		notificationEnabled: true,
		chartProvider:       chartProvider,
		stats:               common.AnalyzerStats{},
	}

	// Создаем процессор сигналов
	analyzer.signalProcessor = NewSignalProcessor(analyzer)
	return analyzer
}

func (a *CounterAnalyzer) Name() string                { return "counter_analyzer" }
func (a *CounterAnalyzer) Version() string             { return "2.5.0" }
func (a *CounterAnalyzer) Supports(symbol string) bool { return true }

func (a *CounterAnalyzer) Analyze(data []types.PriceData, cfg common.AnalyzerConfig) ([]analysis.Signal, error) {
	startTime := time.Now()

	signals, err := a.signalProcessor.Process(data, cfg)

	// Отправляем уведомления если есть сигналы
	if err == nil && len(signals) > 0 && a.notificationEnabled && a.eventBus != nil {
		for _, signal := range signals {
			a.publishCounterSignal(signal, data)
		}
	}

	a.updateStats(time.Since(startTime), err == nil && len(signals) > 0)
	return signals, err
}

// publishCounterSignal публикует Counter сигнал в EventBus
func (a *CounterAnalyzer) publishCounterSignal(signal analysis.Signal, priceData []types.PriceData) {
	if a.eventBus == nil {
		return
	}

	// Получаем дополнительные данные для Counter сигнала
	currentPrice := priceData[len(priceData)-1].Price
	volume24h := priceData[len(priceData)-1].Volume24h
	openInterest := priceData[len(priceData)-1].OpenInterest
	fundingRate := priceData[len(priceData)-1].FundingRate

	oiChange24h := a.metricsCalculator.CalculateOIChange24h(signal.Symbol)
	averageFunding := a.metricsCalculator.CalculateAverageFunding(getFundingRates(priceData))
	nextFundingTime := a.metricsCalculator.CalculateNextFundingTime()
	liquidationVolume, longLiqVolume, shortLiqVolume := a.metricsCalculator.GetLiquidationData(signal.Symbol)

	rsi := a.techCalculator.CalculateRSI(priceData)
	macdSignal := a.techCalculator.CalculateMACD(priceData)

	var volumeDelta, volumeDeltaPercent float64
	var deltaSource string
	if a.volumeCalculator != nil {
		direction := "growth"
		if signal.Type == "counter_fall" {
			direction = "fall"
		}
		deltaData := a.volumeCalculator.CalculateWithFallback(signal.Symbol, direction)
		if deltaData != nil {
			volumeDelta = deltaData.Delta
			volumeDeltaPercent = deltaData.DeltaPercent
			deltaSource = string(deltaData.Source)
		}
	}

	// Получаем данные счетчика
	counterStats, exists := a.counterManager.GetCounterStats(signal.Symbol)
	signalCount := 0
	maxSignals := 0
	if exists {
		if signal.Type == "counter_growth" {
			signalCount = counterStats.GrowthCount
			maxSignals = a.getMaxSignalsForPeriod()
		} else if signal.Type == "counter_fall" {
			signalCount = counterStats.FallCount
			maxSignals = a.getMaxSignalsForPeriod()
		}
	}

	// Создаем данные для Counter сигнала
	counterData := map[string]interface{}{
		"symbol":               signal.Symbol,
		"direction":            signal.Direction,
		"change":               signal.ChangePercent,
		"signal_count":         signalCount,
		"max_signals":          maxSignals,
		"current_price":        currentPrice,
		"volume_24h":           volume24h,
		"open_interest":        openInterest,
		"oi_change_24h":        oiChange24h,
		"funding_rate":         fundingRate,
		"average_funding":      averageFunding,
		"next_funding_time":    nextFundingTime,
		"liquidation_volume":   liquidationVolume,
		"long_liq_volume":      longLiqVolume,
		"short_liq_volume":     shortLiqVolume,
		"volume_delta":         volumeDelta,
		"volume_delta_percent": volumeDeltaPercent,
		"rsi":                  rsi,
		"macd_signal":          macdSignal,
		"delta_source":         deltaSource,
		"period":               a.getPeriodFromSignalCount(signalCount, maxSignals),
		"confidence":           signal.Confidence,
		"data_points":          signal.DataPoints,
		"timestamp":            signal.Timestamp,
	}

	// Публикуем событие Counter сигнала
	event := types.Event{
		Type:      types.EventCounterSignalDetected, // ✅ ПРАВИЛЬНЫЙ ТИП СОБЫТИЯ
		Source:    "counter_analyzer",
		Data:      counterData,
		Timestamp: time.Now(),
	}

	if err := a.eventBus.Publish(event); err != nil {
		fmt.Printf("⚠️ Ошибка публикации Counter сигнала для %s: %v\n", signal.Symbol, err)
	} else {
		fmt.Printf("✅ Counter сигнал опубликован: %s %s %.2f%%\n",
			signal.Symbol, signal.Direction, signal.ChangePercent)
	}
}

// getMaxSignalsForPeriod возвращает максимальное количество сигналов для текущего периода
func (a *CounterAnalyzer) getMaxSignalsForPeriod() int {
	period := a.getCurrentPeriod(a.config)
	switch period {
	case "5m":
		return 5
	case "15m":
		return 8
	case "30m":
		return 10
	case "1h":
		return 12
	case "4h":
		return 15
	case "1d":
		return 20
	default:
		return 8 // дефолтное значение для 15m
	}
}

// getPeriodFromSignalCount определяет период на основе счетчика сигналов
func (a *CounterAnalyzer) getPeriodFromSignalCount(signalCount, maxSignals int) string {
	percentage := float64(signalCount) / float64(maxSignals) * 100
	switch {
	case percentage < 20:
		return "5 минут"
	case percentage < 40:
		return "15 минут"
	case percentage < 60:
		return "30 минут"
	case percentage < 80:
		return "1 час"
	default:
		return "4 часа"
	}
}

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

// Вспомогательная функция для извлечения ставок фандинга
func getFundingRates(priceData []types.PriceData) []float64 {
	var rates []float64
	for _, data := range priceData {
		if data.FundingRate != 0 {
			rates = append(rates, data.FundingRate)
		}
	}
	return rates
}
