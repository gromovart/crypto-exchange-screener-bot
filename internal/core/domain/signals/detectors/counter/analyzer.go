// internal/core/domain/signals/detectors/counter/analyzer.go
package counter

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	candle "crypto-exchange-screener-bot/internal/core/domain/candle"
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/common"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/counter/calculator"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/counter/confirmation"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/counter/manager"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"

	"github.com/google/uuid"
)

// CounterAnalyzer - анализатор счетчика сигналов
type CounterAnalyzer struct {
	config        common.AnalyzerConfig
	stats         common.AnalyzerStats
	marketFetcher interface{}
	storage       storage.PriceStorageInterface
	eventBus      types.EventBus
	candleSystem  *candle.CandleSystem

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

	// Параллелизм и кэширование
	maxWorkers        int
	workerPool        chan struct{}
	cacheEnabled      bool
	cacheTTL          time.Duration
	lastAnalysis      map[string]time.Time
	analysisCacheMu   sync.RWMutex
	parallelThreshold int

	// НОВОЕ: Диагностика и мониторинг
	problematicSymbols map[string]*SymbolProblem
	candleCache        *CandleAvailabilityCache
	fallbackStats      map[string]int
	diagnosticsEnabled bool
}

type SymbolProblem struct {
	Symbol    string
	Period    string
	FirstSeen time.Time
	LastSeen  time.Time
	Count     int
	LastError string
}

type CandleAvailabilityCache struct {
	unavailableSymbols map[string]time.Time
	ttl                time.Duration
	mu                 sync.RWMutex
}

func (c *CandleAvailabilityCache) IsUnavailable(symbol, period string) bool {
	c.mu.RLock()
	key := symbol + ":" + period
	lastTime, exists := c.unavailableSymbols[key]
	c.mu.RUnlock()

	if !exists {
		return false
	}

	return time.Since(lastTime) < c.ttl
}

func (c *CandleAvailabilityCache) MarkUnavailable(symbol, period string) {
	c.mu.Lock()
	key := symbol + ":" + period
	c.unavailableSymbols[key] = time.Now()
	c.mu.Unlock()
}

func (c *CandleAvailabilityCache) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, lastTime := range c.unavailableSymbols {
		if now.Sub(lastTime) > c.ttl {
			delete(c.unavailableSymbols, key)
		}
	}
}

func NewCandleAvailabilityCache(ttl time.Duration) *CandleAvailabilityCache {
	return &CandleAvailabilityCache{
		unavailableSymbols: make(map[string]time.Time),
		ttl:                ttl,
	}
}

// NewCounterAnalyzer создает новый анализатор счетчика
func NewCounterAnalyzer(
	config common.AnalyzerConfig,
	storage storage.PriceStorageInterface,
	eventBus types.EventBus,
	marketFetcher interface{},
	candleSystem *candle.CandleSystem,
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

	// Создаем анализатор с новыми полями
	analyzer := &CounterAnalyzer{
		config:              config,
		marketFetcher:       marketFetcher,
		storage:             storage,
		eventBus:            eventBus,
		candleSystem:        candleSystem,
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

		// Параллелизм и кэширование
		maxWorkers:        10,
		workerPool:        make(chan struct{}, 10),
		cacheEnabled:      true,
		cacheTTL:          30 * time.Second,
		lastAnalysis:      make(map[string]time.Time),
		parallelThreshold: 100,

		// НОВОЕ: Диагностика и мониторинг
		problematicSymbols: make(map[string]*SymbolProblem),
		candleCache:        NewCandleAvailabilityCache(5 * time.Minute),
		fallbackStats:      make(map[string]int),
		diagnosticsEnabled: true,
	}

	// Запускаем очистку кэша
	go analyzer.cleanupCacheRoutine()

	logger.Info("✅ CounterAnalyzer создан с улучшенной диагностикой свечей")
	return analyzer
}

// AnalyzeAllSymbols анализирует все символы только по актуальным закрытым свечам
// Умный метод: если символов > parallelThreshold, использует параллельный анализ
func (a *CounterAnalyzer) AnalyzeAllSymbols(symbols []string) error {

	if len(symbols) == 0 {
		logger.Warn("⚠️ Нет символов для анализа")
		return nil
	}

	logger.Info("🔍 Начало анализа %d символов", len(symbols))

	// Выбираем стратегию анализа
	if len(symbols) > a.parallelThreshold {
		logger.Info("📊 Используется параллельный анализ (символов: %d > порога: %d)",
			len(symbols), a.parallelThreshold)
		return a.analyzeAllSymbolsParallel(symbols)
	} else {
		logger.Debug("📊 Используется последовательный анализ (символов: %d ≤ порога: %d)",
			len(symbols), a.parallelThreshold)
		return a.analyzeAllSymbolsSequential(symbols)
	}
}

// analyzeAllSymbolsParallel параллельный анализ символов
func (a *CounterAnalyzer) analyzeAllSymbolsParallel(symbols []string) error {
	startTime := time.Now()

	var wg sync.WaitGroup
	var mu sync.Mutex
	totalSignals := 0
	processedSymbols := 0
	skippedSymbols := 0

	logger.Info("⚡ Параллельный анализ %d символов (воркеров: %d)",
		len(symbols), a.maxWorkers)

	for _, symbol := range symbols {
		wg.Add(1)

		// Захватываем слот в пуле воркеров
		a.workerPool <- struct{}{}

		go func(s string) {
			defer wg.Done()
			defer func() { <-a.workerPool }()
			// Проверяем кэш
			if a.cacheEnabled && a.shouldSkipAnalysis(s) {
				mu.Lock()
				skippedSymbols++
				mu.Unlock()

				// Обновляем время в кэше
				a.analysisCacheMu.Lock()
				a.lastAnalysis[s] = time.Now()
				a.analysisCacheMu.Unlock()
				return
			}

			// Анализируем символ
			signalCount := a.analyzeSymbolParallel(s)

			mu.Lock()
			totalSignals += signalCount
			processedSymbols++

			// Обновляем кэш при успешном анализе
			if signalCount > 0 {
				a.analysisCacheMu.Lock()
				a.lastAnalysis[s] = time.Now()
				a.analysisCacheMu.Unlock()
			}
			mu.Unlock()

			// Логируем прогресс каждые 50 символов
			if processedSymbols%50 == 0 {
				mu.Lock()
				currentProcessed := processedSymbols
				mu.Unlock()

				logger.Info("📊 Обработано %d/%d символов (пропущено: %d)",
					currentProcessed, len(symbols), skippedSymbols)
			}
		}(symbol)
	}

	wg.Wait()

	duration := time.Since(startTime)
	logger.Info("✅ Параллельный анализ завершен: %d символов, %d сигналов, время: %v",
		len(symbols), totalSignals, duration)
	logger.Info("📊 Статистика: обработано %d, пропущено %d, скорость: %.1f символов/сек",
		processedSymbols, skippedSymbols, float64(len(symbols))/duration.Seconds())

	// Отправляем статистику
	a.updateStats(duration, totalSignals > 0)

	return nil
}

// analyzeAllSymbolsSequential последовательный анализ символов
func (a *CounterAnalyzer) analyzeAllSymbolsSequential(symbols []string) error {
	startTime := time.Now()
	totalSignals := 0
	processedSymbols := 0
	skippedSymbols := 0

	logger.Debug("🔄 Последовательный анализ %d символов", len(symbols))

	// Для каждого символа
	for i, symbol := range symbols {
		logger.Debug("  [%d/%d] Проверка %s", i+1, len(symbols), symbol)
		// Проверяем кэш
		if a.cacheEnabled && a.shouldSkipAnalysis(symbol) {
			skippedSymbols++
			logger.Debug("    📦 Пропуск (данные в кэше)")

			// Обновляем время в кэше
			a.analysisCacheMu.Lock()
			a.lastAnalysis[symbol] = time.Now()
			a.analysisCacheMu.Unlock()
			continue
		}

		// Получаем актуальные закрытые периоды
		relevantPeriods := a.getRelevantClosedPeriods(symbol)

		if len(relevantPeriods) == 0 {
			a.logCandleDiagnostics(symbol)
			continue
		}

		symbolSignals := 0

		// Для каждого актуального периода
		for _, period := range relevantPeriods {
			// Быстрый анализ с проверкой актуальности
			signal, err := a.analyzePeriodWithPriority(symbol, period)
			if err != nil {
				logger.Debug("    ⚠️ %s: %v", period, err)
				continue
			}

			if signal != nil {
				totalSignals++
				symbolSignals++
				logger.Debug("    🚀 %s: актуальный сигнал (%.2f%%)", period, signal.ChangePercent)
			}
		}

		processedSymbols++

		// Обновляем кэш при успешном анализе
		if symbolSignals > 0 {
			a.analysisCacheMu.Lock()
			a.lastAnalysis[symbol] = time.Now()
			a.analysisCacheMu.Unlock()
		}

		// Логируем прогресс каждые 50 символов
		if (i+1)%50 == 0 {
			logger.Info("📊 Обработано %d/%d символов (пропущено: %d)",
				i+1, len(symbols), skippedSymbols)
		}
	}

	duration := time.Since(startTime)
	logger.Info("✅ Последовательный анализ завершен: %d символов, %d сигналов, время: %v",
		len(symbols), totalSignals, duration)
	logger.Info("📊 Статистика: обработано %d, пропущено %d, скорость: %.1f символов/сек",
		processedSymbols, skippedSymbols, float64(len(symbols))/duration.Seconds())

	// Отправляем статистику
	a.updateStats(duration, totalSignals > 0)

	return nil
}

// logCandleDiagnostics логирует диагностику свечей
func (a *CounterAnalyzer) logCandleDiagnostics(symbol string) {
	if a.candleSystem == nil {
		logger.Debug("    ❌ CandleSystem не инициализирован")
		return
	}

	periods := []string{"5m", "15m", "30m", "1h", "4h", "1d"}

	for _, period := range periods {
		candle, err := a.candleSystem.GetCandle(symbol, period)
		if err != nil {
			logger.Debug("    ⚠️ %s: ошибка - %v", period, err)
			continue
		}

		if candle == nil {
			logger.Debug("    ❌ %s: свеча не найдена", period)
			continue
		}

		status := "❓ неизвестно"
		if candle.IsClosedFlag {
			if a.isCandleStillRelevant(candle, period) {
				status = "✅ готова"
			} else {
				status = "⏰ устарела"
			}
		} else {
			// Расчет процента завершения
			elapsed := time.Now().Sub(candle.StartTime)
			duration := getPeriodDuration(period)
			percent := float64(elapsed) / float64(duration) * 100
			status = fmt.Sprintf("⏳ %.0f%% завершено", percent)
		}

		change := ((candle.Close - candle.Open) / candle.Open) * 100
		logger.Debug("    📊 %s: %s (изменение: %.4f%%, реальная: %v)",
			period, status, change, candle.IsRealFlag)
	}
}

// shouldSkipAnalysis проверяет нужно ли пропускать анализ (по кэшу)
func (a *CounterAnalyzer) shouldSkipAnalysis(symbol string) bool {
	a.analysisCacheMu.RLock()
	lastTime, exists := a.lastAnalysis[symbol]
	a.analysisCacheMu.RUnlock()

	if !exists {
		return false // Никогда не анализировали
	}

	// Проверяем время с последнего анализа
	timeSinceLast := time.Since(lastTime)
	return timeSinceLast < a.cacheTTL
}

// analyzeSymbolParallel анализирует один символ (для параллельного использования)
func (a *CounterAnalyzer) analyzeSymbolParallel(symbol string) int {
	logger.Debug("  🔄 Параллельная проверка %s", symbol)

	relevantPeriods := a.getRelevantClosedPeriods(symbol)
	if len(relevantPeriods) == 0 {
		return 0
	}

	signalCount := 0
	for _, period := range relevantPeriods {
		signal, err := a.analyzePeriodWithPriority(symbol, period)
		if err == nil && signal != nil {
			signalCount++
		}
	}

	return signalCount
}

// getRelevantClosedPeriods возвращает периоды с актуальными закрытыми свечами
func (a *CounterAnalyzer) getRelevantClosedPeriods(symbol string) []string {
	allPeriods := []string{"5m", "15m", "30m", "1h", "4h", "1d"}
	var relevantPeriods []string
	now := time.Now()

	for _, period := range allPeriods {
		if a.candleSystem != nil {
			candle, err := a.candleSystem.GetCandle(symbol, period)
			if err == nil && candle != nil && candle.IsRealFlag {
				// Определяем, можно ли анализировать свечу
				canAnalyze := a.canAnalyzeCandle(candle, period, now)

				if canAnalyze {
					relevantPeriods = append(relevantPeriods, period)
					logger.Debug("    ✅ %s: свеча доступна для анализа", period)
				} else {
					logger.Debug("    ⏳ %s: свеча не готова для анализа", period)
				}
			} else {
				logger.Debug("    ⚠️ %s: не удалось получить свечу", period)
			}
		}
	}

	return relevantPeriods
}

// canAnalyzeCandle проверяет, можно ли анализировать свечу
func (a *CounterAnalyzer) canAnalyzeCandle(candle *redis_storage.Candle, period string, now time.Time) bool {
	// Базовые проверки
	if !candle.IsRealFlag {
		return false
	}

	if candle.Open <= 0 || candle.Close <= 0 {
		return false
	}

	if candle.StartTime.After(now) {
		return false
	}

	// Получаем длительность периода
	periodDuration := getPeriodDuration(period)
	if periodDuration <= 0 {
		return false
	}

	// 1. Если свеча закрыта
	if candle.IsClosedFlag {
		// Проверяем время с момента закрытия
		timeSinceClose := now.Sub(candle.EndTime)

		// Максимальное время актуальности после закрытия
		var maxTimeSinceClose time.Duration
		switch period {
		case "5m":
			maxTimeSinceClose = 1 * time.Minute
		case "15m":
			maxTimeSinceClose = 3 * time.Minute
		case "30m":
			maxTimeSinceClose = 5 * time.Minute
		case "1h":
			maxTimeSinceClose = 10 * time.Minute
		case "4h":
			maxTimeSinceClose = 30 * time.Minute
		case "1d":
			maxTimeSinceClose = 2 * time.Hour
		default:
			maxTimeSinceClose = 5 * time.Minute
		}

		return timeSinceClose <= maxTimeSinceClose
	}

	// 2. Если свеча не закрыта
	elapsed := now.Sub(candle.StartTime)
	completionPercent := float64(elapsed) / float64(periodDuration) * 100

	// Минимальный процент завершенности для анализа
	minCompletionPercent := 60.0
	maxCompletionPercent := 95.0

	if completionPercent < minCompletionPercent {
		return false
	}

	if completionPercent > maxCompletionPercent {
		return false
	}

	// Проверяем что свеча не слишком старая
	maxElapsed := periodDuration * 2
	if elapsed > maxElapsed {
		return false
	}

	// Проверяем изменение цены
	changePercent := math.Abs((candle.Close - candle.Open) / candle.Open * 100)
	return changePercent >= a.baseThreshold
}

// isCandleStillRelevant проверяет актуальность свечи
func (a *CounterAnalyzer) isCandleStillRelevant(candle *redis_storage.Candle, period string) bool {
	now := time.Now()
	return a.canAnalyzeCandle(candle, period, now)
}

// getCandleWithRetry получает свечу с повторными попытками
func (a *CounterAnalyzer) getCandleWithRetry(symbol, period string, maxRetries int, delay time.Duration) (*redis_storage.Candle, error) {
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Диагностическое логирование на первой и последней попытке
		if attempt == 1 || attempt == maxRetries {
			logger.Debug("🔍 Попытка %d/%d получить свечу %s %s, candleSystem=%v",
				attempt, maxRetries, symbol, period, a.candleSystem != nil)
		}

		candle, err := a.candleSystem.GetCandle(symbol, period)

		if err == nil && candle != nil {
			// Проверяем минимальную валидность свечи
			if candle.IsRealFlag && candle.Open > 0 && candle.Close > 0 {
				if attempt > 1 {
					logger.Info("✅ Получена свеча %s %s с попытки %d", symbol, period, attempt)
				}
				return candle, nil
			}

			// Свеча существует но невалидна
			if !candle.IsRealFlag {
				lastErr = fmt.Errorf("свеча нереальная (тестовая)")
			} else if candle.Open <= 0 || candle.Close <= 0 {
				lastErr = fmt.Errorf("некорректные цены (Open=%.6f, Close=%.6f)",
					candle.Open, candle.Close)
			} else {
				lastErr = fmt.Errorf("свеча невалидна (закрыта=%v)", candle.IsClosedFlag)
			}
		} else if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("свеча не найдена (nil)")
		}

		if attempt < maxRetries {
			time.Sleep(delay)
		}
	}

	return nil, fmt.Errorf("не удалось получить свечу после %d попыток: %v",
		maxRetries, lastErr)
}

// createSignalFromCandle создает сигнал из свечи
func (a *CounterAnalyzer) createSignalFromCandle(symbol, period string, candle *redis_storage.Candle, changePercent float64) *analysis.Signal {
	direction := "growth"
	if changePercent < 0 {
		direction = "fall"
		changePercent = math.Abs(changePercent)
	}

	// Получаем дополнительные данные если нужно
	var metadata analysis.Metadata
	if a.volumeCalculator != nil {
		// Можно добавить дельту объема и т.д.
		metadata.Tags = append(metadata.Tags, "from_candle")
	}

	// Получаем прогресс счетчика
	counter, _ := a.counterManager.GetCounterStats(symbol)
	confirmations, _ := a.confirmationManager.GetProgress(symbol, period)

	return &analysis.Signal{
		Symbol:        symbol,
		Period:        periodToMinutes(period),
		Direction:     direction,
		ChangePercent: changePercent,
		Confidence:    a.calculateConfidence(counter, confirmations),
		Timestamp:     time.Now(),
		ID:            uuid.New().String(),
		Metadata:      metadata,
	}
}

// analyzeWithCandle анализирует с использованием свечи
func (a *CounterAnalyzer) analyzeWithCandle(symbol, period string, candle *redis_storage.Candle) (*analysis.Signal, error) {
	// Проверяем актуальность
	if !a.isCandleStillRelevant(candle, period) {
		return nil, fmt.Errorf("свеча неактуальна")
	}

	// Рассчитываем изменение
	changePercent := ((candle.Close - candle.Open) / candle.Open) * 100

	// Проверяем порог
	if math.Abs(changePercent) < a.baseThreshold {
		return nil, fmt.Errorf("изменение (%.4f%%) ниже порога (%.4f%%)",
			math.Abs(changePercent), a.baseThreshold)
	}

	// Создаем сигнал
	return a.createSignalFromCandle(symbol, period, candle, changePercent), nil
}

// analyzePeriodWithPriority быстрый анализ с проверкой актуальности
func (a *CounterAnalyzer) analyzePeriodWithPriority(symbol, period string) (*analysis.Signal, error) {
	analysisStart := time.Now()

	// 1. Проверяем кэш недоступности
	if a.candleCache.IsUnavailable(symbol, period) {
		logger.Debug("⏭️ Пропускаем %s %s (временно недоступен в кэше)", symbol, period)
		return nil, fmt.Errorf("свеча временно недоступна (кэш)")
	}

	// 2. Получаем свечу с увеличенными ретраями
	candle, err := a.getCandleWithRetry(symbol, period, 5, 500*time.Millisecond)
	if err != nil {
		// Отмечаем в кэше недоступности
		a.candleCache.MarkUnavailable(symbol, period)

		// Трекаем проблему
		a.trackProblematicSymbol(symbol, period, err)

		// Fallback с улучшенной диагностикой
		logger.Warn("🔄 Использую fallback анализ для %s %s: %v", symbol, period, err)
		return a.fallbackWithDiagnostics(symbol, period, err)
	}

	// 3. Проверяем актуальность
	if !a.isCandleStillRelevant(candle, period) {
		return nil, fmt.Errorf("свеча потеряла актуальность (%v с момента закрытия)",
			time.Since(candle.EndTime).Round(time.Second))
	}

	// 4. Используем данные свечи если она доступна и валидна
	if candle.IsClosedFlag && candle.IsRealFlag && candle.Open > 0 && candle.Close > 0 {
		changePercent := ((candle.Close - candle.Open) / candle.Open) * 100

		// Проверяем порог (с учетом периода)
		threshold := a.getThresholdForPeriod(period)
		if math.Abs(changePercent) >= threshold {
			// Создаем сигнал на основе свечи
			signal := a.createSignalFromCandle(symbol, period, candle, changePercent)

			analysisTime := time.Since(analysisStart)
			logger.Debug("    ⏱️ Анализ %s %s занял %v (использована свеча)", symbol, period, analysisTime)

			// Сбрасываем счетчик проблем если успешно
			a.resetProblematicSymbol(symbol, period)

			return signal, nil
		}

		return nil, fmt.Errorf("изменение цены (%.4f%%) ниже порога (%.4f%%)",
			math.Abs(changePercent), threshold)
	}

	// 5. Fallback с диагностикой
	logger.Debug("🔄 Свеча %s %s невалидна (закрыта=%v, реальная=%v), использую fallback",
		symbol, period, candle.IsClosedFlag, candle.IsRealFlag)
	return a.fallbackWithDiagnostics(symbol, period,
		fmt.Errorf("свеча невалидна (закрыта=%v, реальная=%v)",
			candle.IsClosedFlag, candle.IsRealFlag))
}

// getThresholdForPeriod возвращает порог в зависимости от периода
func (a *CounterAnalyzer) getThresholdForPeriod(period string) float64 {
	switch period {
	case "1d":
		return a.baseThreshold * 0.1 // 0.01% для дневных свечей
	case "4h":
		return a.baseThreshold * 0.3 // 0.03%
	case "1h":
		return a.baseThreshold * 0.5 // 0.05%
	default:
		return a.baseThreshold // 0.1% для 5m, 15m, 30m
	}
}

// trackProblematicSymbol отслеживает проблемные символы
func (a *CounterAnalyzer) trackProblematicSymbol(symbol, period string, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	key := symbol + ":" + period
	now := time.Now()

	if problem, exists := a.problematicSymbols[key]; !exists {
		a.problematicSymbols[key] = &SymbolProblem{
			Symbol:    symbol,
			Period:    period,
			FirstSeen: now,
			LastSeen:  now,
			Count:     1,
			LastError: err.Error(),
		}

		// Логируем новую проблему
		logger.Warn("🔴 Новая проблема с символом %s %s: %v", symbol, period, err)
	} else {
		problem.Count++
		problem.LastSeen = now
		problem.LastError = err.Error()

		// Логируем если проблема повторяется
		if problem.Count%10 == 0 {
			logger.Error("🔴 Проблема с символом %s %s повторяется %d раз: %v",
				symbol, period, problem.Count, err)

			// Диагностика при частых проблемах
			if problem.Count >= 50 {
				a.runDeepDiagnostics(symbol, period)
			}
		}
	}
}

// resetProblematicSymbol сбрасывает счетчик проблем
func (a *CounterAnalyzer) resetProblematicSymbol(symbol, period string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	key := symbol + ":" + period
	if problem, exists := a.problematicSymbols[key]; exists && problem.Count > 0 {
		logger.Info("✅ Символ %s %s восстановлен после %d проблем",
			symbol, period, problem.Count)
		delete(a.problematicSymbols, key)
	}
}

// fallbackWithDiagnostics улучшенный fallback с диагностикой
func (a *CounterAnalyzer) fallbackWithDiagnostics(symbol, period string, originalErr error) (*analysis.Signal, error) {
	// Обновляем статистику fallback
	a.mu.Lock()
	a.fallbackStats[symbol]++
	a.mu.Unlock()

	// Диагностика если включена
	if a.diagnosticsEnabled {
		a.logCandleDiagnostics(symbol)

		// Проверяем доступность candleSystem
		if a.candleSystem == nil {
			logger.Error("❌ CandleSystem не инициализирован")
		} else {
			// Проверяем health check если доступен

		}
	}

	// Получаем исторические данные
	data, err := a.getCandleData(symbol, period)
	if err != nil {
		return nil, fmt.Errorf("fallback тоже не удался: %v (оригинальная ошибка: %v)", err, originalErr)
	}

	if len(data) < 2 {
		return nil, fmt.Errorf("недостаточно данных для fallback (%d точек, ожидается 2+)", len(data))
	}

	logger.Info("📊 Fallback использует %d точек для %s %s (ошибка свечи: %v)",
		len(data), symbol, period, originalErr)

	// Анализируем
	return a.analyzeWithPriceData(symbol, period, data)
}

// runDeepDiagnostics запускает глубокую диагностику
func (a *CounterAnalyzer) runDeepDiagnostics(symbol, period string) {
	logger.Info("🔍 Глубокая диагностика для %s %s", symbol, period)

	var result strings.Builder
	result.WriteString(fmt.Sprintf("🔍 ДИАГНОСТИКА %s %s:\n", symbol, period))
	result.WriteString(fmt.Sprintf("Время: %s\n", time.Now().Format("15:04:05")))

	// 1. Проверка candleSystem
	result.WriteString("\n1. CANDLE SYSTEM:\n")
	if a.candleSystem == nil {
		result.WriteString("   ❌ Не инициализирован\n")
	} else {
		result.WriteString("   ✅ Инициализирован\n")

		// Пробуем получить свечу напрямую
		candle, err := a.candleSystem.GetCandle(symbol, period)
		if err != nil {
			result.WriteString(fmt.Sprintf("   ❌ Ошибка GetCandle: %v\n", err))
		} else if candle == nil {
			result.WriteString("   ❌ GetCandle вернул nil\n")
		} else {
			result.WriteString(fmt.Sprintf("   ✅ Свеча получена: закрыта=%v, реальная=%v\n",
				candle.IsClosedFlag, candle.IsRealFlag))
			result.WriteString(fmt.Sprintf("   📊 Данные: Open=%.6f, Close=%.6f, Change=%.4f%%\n",
				candle.Open, candle.Close, ((candle.Close-candle.Open)/candle.Open)*100))
		}
	}

	// 2. Проверка storage
	result.WriteString("\n2. STORAGE:\n")
	if a.storage == nil {
		result.WriteString("   ❌ Не инициализирован\n")
	} else {
		result.WriteString("   ✅ Инициализирован\n")

		// Проверяем текущий снапшот
		if snapshot, exists := a.storage.GetCurrentSnapshot(symbol); exists {
			result.WriteString(fmt.Sprintf("   ✅ Снапшот: цена=%.6f, время=%s\n",
				snapshot.GetPrice(), snapshot.GetTimestamp().Format("15:04:05")))
		} else {
			result.WriteString("   ❌ Нет текущего снапшота\n")
		}
	}

	// 3. Проверка исторических данных
	result.WriteString("\n3. ИСТОРИЧЕСКИЕ ДАННЫЕ:\n")
	endTime := time.Now()
	startTime := endTime.Add(-getPeriodDuration(period) * 2)

	priceHistory, err := a.storage.GetPriceHistoryRange(symbol, startTime, endTime)
	if err != nil {
		result.WriteString(fmt.Sprintf("   ❌ Ошибка получения истории: %v\n", err))
	} else {
		result.WriteString(fmt.Sprintf("   ✅ Получено %d точек\n", len(priceHistory)))
		if len(priceHistory) > 0 {
			first := priceHistory[0].GetTimestamp()
			last := priceHistory[len(priceHistory)-1].GetTimestamp()
			result.WriteString(fmt.Sprintf("   🕐 Диапазон: %s - %s\n",
				first.Format("15:04:05"), last.Format("15:04:05")))
		}
	}

	// 4. Статистика проблем
	result.WriteString("\n4. СТАТИСТИКА ПРОБЛЕМ:\n")
	key := symbol + ":" + period
	if problem, exists := a.problematicSymbols[key]; exists {
		result.WriteString(fmt.Sprintf("   🔴 Проблем: %d\n", problem.Count))
		result.WriteString(fmt.Sprintf("   📅 Первая: %s\n", problem.FirstSeen.Format("15:04:05")))
		result.WriteString(fmt.Sprintf("   📅 Последняя: %s\n", problem.LastSeen.Format("15:04:05")))
		result.WriteString(fmt.Sprintf("   ❌ Последняя ошибка: %s\n", problem.LastError))
	} else {
		result.WriteString("   ✅ Нет записей о проблемах\n")
	}

	// 5. Fallback статистика
	result.WriteString("\n5. FALLBACK СТАТИСТИКА:\n")
	if count, exists := a.fallbackStats[symbol]; exists {
		result.WriteString(fmt.Sprintf("   🔄 Fallback использован: %d раз\n", count))
	} else {
		result.WriteString("   ✅ Fallback не использовался\n")
	}

	logger.Info(result.String())
}

// cleanupCacheRoutine очищает кэши
func (a *CounterAnalyzer) cleanupCacheRoutine() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.candleCache.Cleanup()

			// Очищаем старые проблемы (старше 1 часа)
			a.mu.Lock()
			now := time.Now()
			for key, problem := range a.problematicSymbols {
				if now.Sub(problem.LastSeen) > time.Hour {
					delete(a.problematicSymbols, key)
				}
			}
			a.mu.Unlock()
		}
	}
}

// Analyze - совместимый метод для AnalysisEngine
func (a *CounterAnalyzer) Analyze(data []types.PriceData, cfg common.AnalyzerConfig) ([]analysis.Signal, error) {
	// ВРЕМЕННОЕ РЕШЕНИЕ для совместимости с AnalysisEngine
	// Этот метод устарел, используйте AnalyzeAllSymbols

	if len(data) < 2 {
		return nil, fmt.Errorf("недостаточно точек данных")
	}

	symbol := data[0].Symbol
	period := "15m"
	if customPeriod, ok := cfg.CustomSettings["analysis_period"].(string); ok {
		period = customPeriod
	}

	// Используем новую логику с учетом свечей
	candle, err := a.getCandleWithRetry(symbol, period, 2, 50*time.Millisecond)
	if err == nil && candle != nil && a.isCandleStillRelevant(candle, period) {
		// Используем данные свечи напрямую
		if candle.IsClosedFlag && candle.IsRealFlag {
			changePercent := ((candle.Close - candle.Open) / candle.Open) * 100

			if math.Abs(changePercent) >= a.baseThreshold {
				signal := a.createSignalFromCandle(symbol, period, candle, changePercent)
				return []analysis.Signal{*signal}, nil
			}
			return nil, fmt.Errorf("изменение цены (%.4f%%) ниже порога (%.4f%%)",
				math.Abs(changePercent), a.baseThreshold)
		}
	}

	// Fallback: используем старые данные если свечи недоступны
	logger.Warn("⚠️ Использую fallback анализ для %s (свечи недоступны: %v)", symbol, err)
	signal, err := a.analyzeSymbolPeriod(symbol, period, data)
	if err != nil {
		return nil, err
	}

	if signal == nil {
		return nil, nil
	}

	return []analysis.Signal{*signal}, nil
}

// analyzeWithPriceData анализирует с использованием исторических данных
func (a *CounterAnalyzer) analyzeWithPriceData(symbol, period string, data []types.PriceData) (*analysis.Signal, error) {
	// Старая логика анализа
	if len(data) < 2 {
		return nil, fmt.Errorf("недостаточно данных для анализа")
	}

	// Рассчитываем изменение между первой и последней точкой
	firstPrice := data[0].Price
	lastPrice := data[len(data)-1].Price
	changePercent := ((lastPrice - firstPrice) / firstPrice) * 100

	// Проверяем порог
	if math.Abs(changePercent) < a.baseThreshold {
		return nil, fmt.Errorf("изменение (%.4f%%) ниже порога (%.4f%%)",
			math.Abs(changePercent), a.baseThreshold)
	}

	// Создаем базовый сигнал
	direction := "growth"
	if changePercent < 0 {
		direction = "fall"
		changePercent = math.Abs(changePercent)
	}

	return &analysis.Signal{
		Symbol:        symbol,
		Period:        periodToMinutes(period),
		Direction:     direction,
		ChangePercent: changePercent,
		Confidence:    50.0, // Базовая уверенность
		Timestamp:     time.Now(),
		ID:            uuid.New().String(),
		Metadata: analysis.Metadata{
			Tags: []string{"fallback", period},
		},
	}, nil
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

// GetNotifierStats возвращает статистику нотификатора
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

// DebugCandleStatus проверяет состояние свечей для диагностики
func (a *CounterAnalyzer) DebugCandleStatus(symbol string) string {
	if a.candleSystem == nil {
		return "❌ Свечная система не инициализирована"
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("🔍 Диагностика свечей для %s:\n", symbol))

	periods := []string{"5m", "15m", "30m", "1h", "4h", "1d"}
	now := time.Now()

	for _, period := range periods {
		candle, err := a.candleSystem.GetCandle(symbol, period)
		if err != nil {
			result.WriteString(fmt.Sprintf("⚠️ %s: ошибка - %s\n", period, err))
			continue
		}

		if candle == nil {
			result.WriteString(fmt.Sprintf("❌ %s: свеча не найдена\n", period))
			continue
		}

		// Проверяем статус
		isClosed := candle.IsClosedFlag || now.After(candle.EndTime)
		isRelevant := a.isCandleStillRelevant(candle, period)
		changePercent := ((candle.Close - candle.Open) / candle.Open) * 100

		result.WriteString(fmt.Sprintf("📊 %s:\n", period))
		result.WriteString(fmt.Sprintf("   • Время: %s - %s\n",
			candle.StartTime.Format("15:04:05"), candle.EndTime.Format("15:04:05")))
		result.WriteString(fmt.Sprintf("   • Цена: %.6f → %.6f (%.4f%%)\n",
			candle.Open, candle.Close, changePercent))
		result.WriteString(fmt.Sprintf("   • Закрыта: %v (IsClosedFlag: %v, now.After: %v)\n",
			isClosed, candle.IsClosedFlag, now.After(candle.EndTime)))
		result.WriteString(fmt.Sprintf("   • Актуальна: %v\n", isRelevant))
		result.WriteString(fmt.Sprintf("   • Реальная: %v\n", candle.IsRealFlag))

		if isClosed && isRelevant {
			result.WriteString("   • ✅ ГОТОВА для анализа\n")
		} else {
			result.WriteString("   • ⏳ НЕ ГОТОВА для анализа\n")
			if !isClosed {
				result.WriteString(fmt.Sprintf("     - До закрытия: %v\n",
					candle.EndTime.Sub(now).Round(time.Second)))
			}
			if !isRelevant {
				timeSinceClose := now.Sub(candle.EndTime)
				result.WriteString(fmt.Sprintf("     - С момента закрытия: %v\n",
					timeSinceClose.Round(time.Second)))
			}
		}
	}

	return result.String()
}

// CheckDataDepth проверяет глубину данных в Redis
func (a *CounterAnalyzer) CheckDataDepth(symbol string) string {
	if a.storage == nil {
		return "❌ Хранилище Redis не инициализировано"
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("📊 Проверка глубины данных для %s:\n", symbol))

	// Проверяем разные периоды
	periods := []string{"5m", "15m", "30m", "1h", "4h", "1d"}
	now := time.Now()

	for _, period := range periods {
		// Определяем начало периода
		periodDuration := getPeriodDuration(period)
		startTime := now.Add(-periodDuration * 2) // Берем 2 периода для проверки

		result.WriteString(fmt.Sprintf("\n🔍 Период %s:\n", period))
		result.WriteString(fmt.Sprintf("   • Ищем данные с: %s\n", startTime.Format("15:04:05")))
		result.WriteString(fmt.Sprintf("   • До: %s\n", now.Format("15:04:05")))
		result.WriteString(fmt.Sprintf("   • Длительность: %v\n", periodDuration))

		// Пробуем получить данные через storage
		priceHistory, err := a.storage.GetPriceHistoryRange(symbol, startTime, now)
		if err != nil {
			result.WriteString(fmt.Sprintf("   • ❌ Ошибка: %v\n", err))
			continue
		}

		// Анализируем полученные данные
		dataCount := len(priceHistory)
		result.WriteString(fmt.Sprintf("   • 📈 Найдено записей: %d\n", dataCount))

		if dataCount > 0 {
			// Определяем временной диапазон данных
			var earliest, latest time.Time
			for i, record := range priceHistory {
				timestamp := record.GetTimestamp()
				if i == 0 || timestamp.Before(earliest) {
					earliest = timestamp
				}
				if i == 0 || timestamp.After(latest) {
					latest = timestamp
				}
			}

			result.WriteString(fmt.Sprintf("   • 🕐 Диапазон данных: %s - %s\n",
				earliest.Format("15:04:05"), latest.Format("15:04:05")))
			result.WriteString(fmt.Sprintf("   • ⏱️  Возраст самой старой записи: %v\n",
				now.Sub(earliest).Round(time.Second)))
			result.WriteString(fmt.Sprintf("   • ⏱️  Возраст самой новой записи: %v\n",
				now.Sub(latest).Round(time.Second)))

			// Проверяем достаточно ли данных для анализа
			if dataCount >= 2 {
				// Получаем первую и последнюю цены
				firstPrice := priceHistory[0].GetPrice()
				lastPrice := priceHistory[dataCount-1].GetPrice()
				changePercent := ((lastPrice - firstPrice) / firstPrice) * 100

				result.WriteString(fmt.Sprintf("   • 💰 Изменение за период: %.4f%%\n", changePercent))
				result.WriteString(fmt.Sprintf("   • 📊 Открытие: %.6f, Закрытие: %.6f\n",
					firstPrice, lastPrice))

				// Проверяем интервалы между точками
				if dataCount > 2 {
					var intervals []time.Duration
					for i := 1; i < dataCount; i++ {
						interval := priceHistory[i].GetTimestamp().Sub(priceHistory[i-1].GetTimestamp())
						intervals = append(intervals, interval)
					}

					// Находим средний интервал
					var total time.Duration
					for _, interval := range intervals {
						total += interval
					}
					avgInterval := total / time.Duration(len(intervals))

					result.WriteString(fmt.Sprintf("   • ⏲️  Средний интервал между точками: %v\n", avgInterval))

					// Проверяем соответствует ли интервал периоду
					expectedInterval := periodDuration / 10 // Предполагаем 10 точек на период
					if avgInterval > expectedInterval*2 {
						result.WriteString(fmt.Sprintf("   • ⚠️  СЛИШКОМ РЕДКИЕ ДАННЫЕ! Ожидается ~%v\n", expectedInterval))
					}
				}

				result.WriteString("   • ✅ ДОСТАТОЧНО данных для анализа\n")
			} else {
				result.WriteString("   • ❌ НЕДОСТАТОЧНО данных для анализа (нужно минимум 2 точки)\n")
			}
		} else {
			result.WriteString("   • ❌ НЕТ ДАННЫХ в Redis для этого периода\n")

			// Проверяем текущий снапшот
			if snapshot, exists := a.storage.GetCurrentSnapshot(symbol); exists {
				result.WriteString(fmt.Sprintf("   • 📸 Есть текущий снапшот: цена=%.6f, время=%s\n",
					snapshot.GetPrice(), snapshot.GetTimestamp().Format("15:04:05")))
			} else {
				result.WriteString("   • ❌ Нет даже текущего снапшота\n")
			}
		}
	}

	// Дополнительная проверка: глубина истории
	result.WriteString("\n📈 ГЛУБИНА ИСТОРИИ:\n")

	// Проверяем как далеко назад есть данные
	timePoints := []time.Duration{
		5 * time.Minute,
		15 * time.Minute,
		30 * time.Minute,
		1 * time.Hour,
		2 * time.Hour,
		4 * time.Hour,
		24 * time.Hour,
	}

	for _, lookback := range timePoints {
		checkTime := now.Add(-lookback)
		priceHistory, err := a.storage.GetPriceHistoryRange(symbol, checkTime, now)

		if err == nil && len(priceHistory) > 0 {
			oldest := priceHistory[0].GetTimestamp()
			result.WriteString(fmt.Sprintf("   • %v назад: ЕСТЬ данные (самые старые от %s, разница: %v)\n",
				lookback.Round(time.Minute), oldest.Format("15:04:05"), now.Sub(oldest).Round(time.Second)))
		} else {
			result.WriteString(fmt.Sprintf("   • %v назад: ❌ НЕТ данных\n", lookback.Round(time.Minute)))
		}
	}

	// Проверяем статистику storage
	if storageWithStats, ok := a.storage.(interface{ GetStorageStats() map[string]interface{} }); ok {
		stats := storageWithStats.GetStorageStats()
		result.WriteString("\n📊 СТАТИСТИКА ХРАНИЛИЩА:\n")
		for key, value := range stats {
			result.WriteString(fmt.Sprintf("   • %s: %v\n", key, value))
		}
	}

	return result.String()
}

// RunDiagnostics запускает полную диагностику CounterAnalyzer
func (a *CounterAnalyzer) RunDiagnostics(symbol string) string {
	var result strings.Builder

	result.WriteString("🔍 ПОЛНАЯ ДИАГНОСТИКА COUNTER ANALYZER\n")
	result.WriteString("═" + strings.Repeat("═", 50) + "\n\n")

	// 1. Проверка свечей
	result.WriteString("1. ПРОВЕРКА СВЕЧЕЙ:\n")
	result.WriteString(a.DebugCandleStatus(symbol))
	result.WriteString("\n")

	// 2. Проверка данных в Redis
	result.WriteString("2. ПРОВЕРКА ДАННЫХ В REDIS:\n")
	result.WriteString(a.CheckDataDepth(symbol))
	result.WriteString("\n")

	// 3. Проверка настроек анализатора
	result.WriteString("3. НАСТРОЙКИ АНАЛИЗАТОРА:\n")
	result.WriteString(fmt.Sprintf("   • Базовый порог: %.4f%%\n", a.baseThreshold))
	result.WriteString(fmt.Sprintf("   • Провайдер графиков: %s\n", a.chartProvider))
	result.WriteString(fmt.Sprintf("   • Уведомления: %v\n", a.notificationEnabled))
	result.WriteString(fmt.Sprintf("   • Кэширование: %v (TTL: %v)\n", a.cacheEnabled, a.cacheTTL))
	result.WriteString(fmt.Sprintf("   • Параллелизм: %d воркеров\n", a.maxWorkers))
	result.WriteString(fmt.Sprintf("   • Порог параллельности: %d символов\n", a.parallelThreshold))

	// 4. Проверка компонентов
	result.WriteString("4. КОМПОНЕНТЫ:\n")
	result.WriteString(fmt.Sprintf("   • CandleSystem: %v\n", a.candleSystem != nil))
	result.WriteString(fmt.Sprintf("   • Storage: %v\n", a.storage != nil))
	result.WriteString(fmt.Sprintf("   • EventBus: %v\n", a.eventBus != nil))
	result.WriteString(fmt.Sprintf("   • VolumeCalculator: %v\n", a.volumeCalculator != nil))
	result.WriteString(fmt.Sprintf("   • ConfirmationManager: %v\n", a.confirmationManager != nil))

	// 5. Проверка статистики
	result.WriteString("5. СТАТИСТИКА АНАЛИЗАТОРА:\n")
	stats := a.GetStats()
	result.WriteString(fmt.Sprintf("   • Всего вызовов: %d\n", stats.TotalCalls))
	result.WriteString(fmt.Sprintf("   • Успешных: %d\n", stats.SuccessCount))
	result.WriteString(fmt.Sprintf("   • Ошибок: %d\n", stats.ErrorCount))
	result.WriteString(fmt.Sprintf("   • Среднее время: %v\n", stats.AverageTime))
	result.WriteString(fmt.Sprintf("   • Последний вызов: %v\n", stats.LastCallTime.Format("15:04:05")))

	// 6. Проверка подтверждений (если есть для символа)
	result.WriteString("6. СТАТУС ПОДТВЕРЖДЕНИЙ:\n")
	periods := []string{"5m", "15m", "30m", "1h", "4h", "1d"}
	for _, period := range periods {
		confirmations, direction := a.confirmationManager.GetProgress(symbol, period)
		required := confirmation.GetRequiredConfirmations(period)
		signalThreshold := confirmation.GetSignalThreshold()

		result.WriteString(fmt.Sprintf("   • %s: %d/%d подтверждений\n",
			period, confirmations, required))
		result.WriteString(fmt.Sprintf("     - Направление: %s\n", direction))
		result.WriteString(fmt.Sprintf("     - Порог сигнала: каждые %d подтверждений\n", signalThreshold))

		if confirmations >= signalThreshold {
			result.WriteString(fmt.Sprintf("     - ✅ Готов к сигналу (следующий через %d подтверждений)\n",
				signalThreshold-(confirmations%signalThreshold)))
		}
	}

	// 7. Проверка периодов для анализа
	result.WriteString("\n7. ПЕРИОДЫ ДЛЯ АНАЛИЗА:\n")
	relevantPeriods := a.getRelevantClosedPeriods(symbol)
	if len(relevantPeriods) > 0 {
		result.WriteString("   • ✅ Актуальные закрытые периоды:\n")
		for _, period := range relevantPeriods {
			result.WriteString(fmt.Sprintf("     - %s\n", period))
		}
	} else {
		result.WriteString("   • ❌ НЕТ актуальных закрытых периодов!\n")
		// Показываем почему
		result.WriteString("   • ПРИЧИНЫ:\n")
		now := time.Now()
		for _, period := range []string{"5m", "15m", "30m", "1h", "4h", "1d"} {
			if a.candleSystem != nil {
				candle, err := a.candleSystem.GetCandle(symbol, period)
				if err == nil && candle != nil {
					isClosed := candle.IsClosedFlag || now.After(candle.EndTime)
					isRelevant := a.isCandleStillRelevant(candle, period)

					if !isClosed {
						result.WriteString(fmt.Sprintf("     - %s: ❌ НЕ ЗАКРЫТ (закроется %s)\n",
							period, candle.EndTime.Format("15:04:05")))
					} else if !isRelevant {
						timeSinceClose := now.Sub(candle.EndTime)
						result.WriteString(fmt.Sprintf("     - %s: ⏰ НЕ АКТУАЛЕН (%v с момента закрытия, лимит: %v)\n",
							period, timeSinceClose.Round(time.Second), getMaxRelevanceTime(period)))
					}
				} else {
					result.WriteString(fmt.Sprintf("     - %s: ❌ СВЕЧА НЕ НАЙДЕНА\n", period))
				}
			} else {
				result.WriteString(fmt.Sprintf("     - %s: ❌ CANDLE SYSTEM НЕ ИНИЦИАЛИЗИРОВАН\n", period))
			}
		}
	}

	// 8. Предложения по исправлению
	result.WriteString("\n8. ПРЕДЛОЖЕНИЯ ПО ИСПРАВЛЕНИЮ:\n")
	if len(relevantPeriods) == 0 {
		result.WriteString("   • 🔧 Увеличить время актуальности в isCandleStillRelevant()\n")
		result.WriteString("   • 🔧 Проверить корректность закрытия свечей в CandleSystem\n")
		result.WriteString("   • 🔧 Увеличить частоту агрегации данных для длинных периодов\n")
	} else if len(relevantPeriods) == 1 && relevantPeriods[0] == "5m" {
		result.WriteString("   • ⚠️  Только 5m период доступен. Вероятные причины:\n")
		result.WriteString("     1. Данные для 15m+ периодов не агрегируются вовремя\n")
		result.WriteString("     2. Свечи закрываются но помечаются как неактуальные\n")
		result.WriteString("     3. В Redis недостаточно исторических данных\n")
		result.WriteString("   • 🛠️  Решения:\n")
		result.WriteString("     1. Увеличить maxRelevanceTime для всех периодов в 2-3 раза\n")
		result.WriteString("     2. Добавить fallback на использование незакрытых свечей\n")
		result.WriteString("     3. Проверить конфигурацию CandleSystem для агрегации\n")
	}

	result.WriteString("\n═" + strings.Repeat("═", 50) + "\n")
	result.WriteString("📊 ДИАГНОСТИКА ЗАВЕРШЕНА\n")

	return result.String()
}

// Вспомогательная функция для получения времени актуальности
func getMaxRelevanceTime(period string) time.Duration {
	switch period {
	case "5m":
		return 30 * time.Second
	case "15m":
		return 1 * time.Minute
	case "30m":
		return 2 * time.Minute
	case "1h":
		return 5 * time.Minute
	case "4h":
		return 15 * time.Minute
	case "1d":
		return 1 * time.Hour
	default:
		return 1 * time.Minute
	}
}

// Вспомогательная функция для конвертации периода в минуты
func periodToMinutes(period string) int {
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

// calculateConfidence рассчитывает уверенность сигнала
func (a *CounterAnalyzer) calculateConfidence(counter manager.SignalCounter, confirmations int) float64 {
	// Базовая уверенность
	confidence := 50.0

	// Увеличиваем уверенность на основе счетчика
	if counter.GrowthCount > 0 {
		confidence += float64(counter.GrowthCount) * 5
	}
	if counter.FallCount > 0 {
		confidence += float64(counter.FallCount) * 5
	}

	// Увеличиваем на основе подтверждений
	confidence += float64(confirmations) * 10

	// Ограничиваем 100%
	if confidence > 100.0 {
		confidence = 100.0
	}

	return confidence
}
