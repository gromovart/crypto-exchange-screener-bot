// internal/core/domain/signals/detectors/counter/analyzer.go
package counter

import (
	"fmt"
	"math"
	"sync"
	"time"

	candle "crypto-exchange-screener-bot/internal/core/domain/candle"
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/common"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/counter/calculator"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/counter/confirmation"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/counter/manager"
	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
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
	confirmationManager *confirmation.ConfirmationManager

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
	confirmationManager := confirmation.NewConfirmationManager()

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
	totalSignals := 0

	logger.Info("🔍 Начало анализа %d символов", len(symbols))

	// Определяем все периоды для анализа
	periods := []string{"5m", "15m", "30m", "1h", "4h", "1d"}

	// Для каждого символа
	for i, symbol := range symbols {
		logger.Debug("  [%d/%d] Анализ %s", i+1, len(symbols), symbol)
		symbolSignals := 0

		// Для каждого периода
		for _, period := range periods {
			// Получаем данные за период
			data, err := a.getDataForPeriod(symbol, period)
			if err != nil {
				logger.Debug("    ⚠️ %s: %v", period, err)
				continue
			}

			// Анализируем
			signal, err := a.analyzeSymbolPeriod(symbol, period, data)
			if err != nil {
				logger.Debug("    ⚠️ %s: ошибка анализа - %v", period, err)
				continue
			}

			if signal != nil {
				totalSignals++
				symbolSignals++
				logger.Info("    🚀 %s: сигнал обнаружен (%.2f%%)",
					period, signal.ChangePercent)
			} else {
				logger.Debug("    📊 %s: сигнал не обнаружен", period)
			}
		}

		if symbolSignals > 0 {
			logger.Info("  📈 %s: найдено %d сигналов", symbol, symbolSignals)
		}
	}

	// Отправляем статистику
	a.updateStats(time.Since(startTime), totalSignals > 0)

	logger.Info("✅ Анализ завершен: %d символов, %d сигналов, время: %v",
		len(symbols), totalSignals, time.Since(startTime))

	return nil
}

// Analyze - совместимый метод для AnalysisEngine
func (a *CounterAnalyzer) Analyze(data []types.PriceData, cfg common.AnalyzerConfig) ([]analysis.Signal, error) {
	// ВРЕМЕННОЕ РЕШЕНИЕ для совместимости с AnalysisEngine

	if len(data) < 2 {
		return nil, fmt.Errorf("недостаточно точек данных")
	}

	symbol := data[0].Symbol

	// Рассчитываем изменение
	change := a.calculateCandleChange(data, "15m") // Используем дефолтный период

	// Используем период из конфига или дефолтный
	period := "15m"
	if customPeriod, ok := cfg.CustomSettings["analysis_period"].(string); ok {
		period = customPeriod
	}

	// Проверяем порог
	if math.Abs(change) < a.baseThreshold {
		return nil, nil
	}

	// Определяем направление на основе изменения
	direction := "growth"
	if change < 0 {
		direction = "fall"
	}

	// Добавляем подтверждение с направлением
	isReady, confirmations := a.confirmationManager.AddConfirmation(symbol, period, direction)

	if !isReady {
		// Еще не готов, ждем больше подтверждений
		logger.Debug("⏳ %s %s: подтверждений %d, ждем сигнала (направление: %s)",
			symbol, period, confirmations, direction)
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
