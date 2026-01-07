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
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/counter/notification"
	"crypto-exchange-screener-bot/internal/delivery/telegram"
	"crypto-exchange-screener-bot/internal/types"
)

// CounterAnalyzer - анализатор счетчика сигналов
type CounterAnalyzer struct {
	config        common.AnalyzerConfig
	stats         common.AnalyzerStats
	telegramBot   interface{}
	marketFetcher interface{}
	storage       interface{}

	// Компоненты
	counterManager    *manager.CounterManager
	periodManager     *manager.PeriodManager
	signalProcessor   *SignalProcessor
	volumeCalculator  *calculator.VolumeDeltaCalculator
	metricsCalculator *calculator.MarketMetricsCalculator
	techCalculator    *calculator.TechnicalCalculator
	notifier          *notification.CounterNotifier

	mu                  sync.RWMutex
	notificationEnabled bool
	chartProvider       string
}

// NewCounterAnalyzer создает новый анализатор счетчика
func NewCounterAnalyzer(
	config common.AnalyzerConfig,
	storage interface{},
	tgBot interface{},
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
		telegramBot:         tgBot,
		marketFetcher:       marketFetcher,
		storage:             storage,
		counterManager:      counterManager,
		periodManager:       periodManager,
		volumeCalculator:    volumeCalculator,
		metricsCalculator:   metricsCalculator,
		techCalculator:      techCalculator,
		notificationEnabled: true,
		chartProvider:       chartProvider,
		stats:               common.AnalyzerStats{},
	}

	// Создаем нотификатор если есть Telegram бот
	analyzer.initNotifier(tgBot, metricsCalculator, techCalculator, volumeCalculator)

	// Создаем процессор сигналов
	analyzer.signalProcessor = NewSignalProcessor(analyzer)
	return analyzer
}

// initNotifier инициализирует нотификатор
func (a *CounterAnalyzer) initNotifier(
	tgBot interface{},
	metricsCalculator *calculator.MarketMetricsCalculator,
	techCalculator *calculator.TechnicalCalculator,
	volumeCalculator *calculator.VolumeDeltaCalculator,
) {
	if tgBot != nil {
		if telegramBot, ok := tgBot.(*telegram.TelegramBot); ok {
			a.notifier = notification.NewCounterNotifier(
				telegramBot,
				metricsCalculator,
				techCalculator,
				volumeCalculator, // Теперь передаем volumeCalculator
			)
		}
	}
}

func (a *CounterAnalyzer) Name() string                { return "counter_analyzer" }
func (a *CounterAnalyzer) Version() string             { return "2.5.0" }
func (a *CounterAnalyzer) Supports(symbol string) bool { return true }

func (a *CounterAnalyzer) Analyze(data []types.PriceData, cfg common.AnalyzerConfig) ([]analysis.Signal, error) {
	startTime := time.Now()

	signals, err := a.signalProcessor.Process(data, cfg)

	// Отправляем уведомления если есть сигналы
	if err == nil && len(signals) > 0 && a.notificationEnabled && a.notifier != nil {
		for _, signal := range signals {
			a.sendNotification(signal, data)
		}
	}

	a.updateStats(time.Since(startTime), err == nil && len(signals) > 0)
	return signals, err
}

// sendNotification отправляет уведомление о сигнале
func (a *CounterAnalyzer) sendNotification(signal analysis.Signal, priceData []types.PriceData) {
	if a.notifier == nil {
		return
	}

	direction := "growth"
	if signal.Type == "counter_fall" {
		direction = "fall"
	}

	// Получаем счетчик для символа
	counter, exists := a.counterManager.GetCounter(signal.Symbol)
	if !exists {
		return
	}

	counter.RLock()
	signalCount := counter.SignalCount
	counter.RUnlock()

	// Рассчитываем максимальное количество сигналов
	basePeriodMinutes := a.getBasePeriodMinutes(a.config)
	period := a.getCurrentPeriod(a.config)
	maxSignals := a.periodManager.CalculateMaxSignals(period, basePeriodMinutes)

	// Отправляем уведомление
	err := a.notifier.SendNotification(
		signal.Symbol,
		direction,
		signal.ChangePercent,
		signalCount,
		maxSignals,
		priceData,
	)

	if err != nil {
		// Логируем ошибку, но не прерываем выполнение
		fmt.Printf("⚠️ Ошибка отправки уведомления для %s: %v\n", signal.Symbol, err)
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
	if a.notifier != nil {
		a.notifier.SetEnabled(enabled)
	}
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

// TestNotification отправляет тестовое уведомление
func (a *CounterAnalyzer) TestNotification(symbol string) error {
	if a.notifier == nil {
		return fmt.Errorf("notifier not initialized")
	}
	return a.notifier.SendTestNotification(symbol)
}

// GetNotifierStats возвращает статистику нотификатора
func (a *CounterAnalyzer) GetNotifierStats() map[string]interface{} {
	if a.notifier == nil {
		return map[string]interface{}{"error": "notifier not initialized"}
	}
	return a.notifier.GetNotificationStats()
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

	// Тестируем подключение
	err := a.volumeCalculator.TestConnection(symbol)
	if err != nil {
		return fmt.Sprintf("❌ Ошибка тестирования дельты для %s:\n%s", symbol, err.Error())
	}

	// Получаем информацию о кэше
	cacheInfo := a.volumeCalculator.GetCacheInfo()
	cacheSize := cacheInfo["cache_size"].(int)

	return fmt.Sprintf("✅ Тест дельты для %s пройден!\n📦 Размер кэша: %d", symbol, cacheSize)
}
