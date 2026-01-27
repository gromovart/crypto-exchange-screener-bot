// internal/core/domain/signals/engine/engine.go
package engine

import (
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/common"
	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	events "crypto-exchange-screener-bot/internal/infrastructure/transport/event_bus"
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// AnalyzerConfigs - конфигурация анализаторов
type AnalyzerConfigs struct {
	GrowthAnalyzer       AnalyzerConfig `json:"growth_analyzer"`
	FallAnalyzer         AnalyzerConfig `json:"fall_analyzer"`
	ContinuousAnalyzer   AnalyzerConfig `json:"continuous_analyzer"`
	VolumeAnalyzer       AnalyzerConfig `json:"volume_analyzer"`
	OpenInterestAnalyzer AnalyzerConfig `json:"open_interest_analyzer"`
	CounterAnalyzer      AnalyzerConfig `json:"counter_analyzer"`
}

// AnalysisEngine - основной движок анализа (оркестратор)
type AnalysisEngine struct {
	mu           sync.RWMutex
	analyzers    map[string]common.Analyzer
	storage      storage.PriceStorageInterface
	eventBus     *events.EventBus
	config       EngineConfig
	stats        EngineStats
	lastAnalysis map[string]time.Time
	stopChan     chan struct{}
	wg           sync.WaitGroup
	running      bool

	// Накопление статистики для логирования
	logStatsMu       sync.RWMutex
	logStats         map[int]*periodStats // период → статистика
	logLastFlush     time.Time
	logFlushInterval time.Duration
}

// Структура для накопления статистики по периоду
type periodStats struct {
	growthCount int
	fallCount   int
	symbols     map[string]bool
}

// EngineConfig - конфигурация движка
type EngineConfig struct {
	UpdateInterval   time.Duration   `json:"update_interval"`
	AnalysisPeriods  []time.Duration `json:"analysis_periods"`
	MinVolumeFilter  float64         `json:"min_volume_filter"`
	MaxSymbolsPerRun int             `json:"max_symbols_per_run"`
	EnableParallel   bool            `json:"enable_parallel"`
	MaxWorkers       int             `json:"max_workers"`
	SignalThreshold  float64         `json:"signal_threshold"`
	RetentionPeriod  time.Duration   `json:"retention_period"`
	EnableCache      bool            `json:"enable_cache"`
	MinDataPoints    int             `json:"min_data_points"`

	// Добавляем поля для анализаторов
	AnalyzerConfigs AnalyzerConfigs `json:"analyzer_configs"`
}

// AnalyzerConfig - конфигурация анализатора
type AnalyzerConfig struct {
	Enabled        bool                   `json:"enabled"`
	MinConfidence  float64                `json:"min_confidence"`
	MinGrowth      float64                `json:"min_growth"`
	MinFall        float64                `json:"min_fall"`
	CustomSettings map[string]interface{} `json:"custom_settings,omitempty"`
}

// EngineStats - статистика движка
type EngineStats struct {
	TotalAnalyses   int64                           `json:"total_analyses"`
	TotalSignals    int64                           `json:"total_signals"`
	AnalysisTime    time.Duration                   `json:"analysis_time"`
	ActiveAnalyzers int                             `json:"active_analyzers"`
	LastRunTime     time.Time                       `json:"last_run_time"`
	SymbolsAnalyzed map[string]int64                `json:"symbols_analyzed"`
	AnalyzerStats   map[string]common.AnalyzerStats `json:"analyzer_stats"`
}

// DefaultConfig - конфигурация по умолчанию
var DefaultConfig = EngineConfig{
	UpdateInterval:   30 * time.Second,
	AnalysisPeriods:  []time.Duration{5 * time.Minute, 15 * time.Minute, 30 * time.Minute, 60 * time.Minute, 240 * time.Minute, 1440 * time.Minute},
	MinVolumeFilter:  100000,
	MaxSymbolsPerRun: 100,
	EnableParallel:   true,
	MaxWorkers:       5,
	SignalThreshold:  2.0,
	RetentionPeriod:  24 * time.Hour,
	EnableCache:      true,
}

// NewAnalysisEngine создает новый движок анализа (оркестратор)
func NewAnalysisEngine(storage storage.PriceStorageInterface, eventBus *events.EventBus, config ...EngineConfig) *AnalysisEngine {
	cfg := DefaultConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	engine := &AnalysisEngine{
		analyzers: make(map[string]common.Analyzer),
		storage:   storage,
		eventBus:  eventBus,
		config:    cfg,
		stats: EngineStats{
			SymbolsAnalyzed: make(map[string]int64),
			AnalyzerStats:   make(map[string]common.AnalyzerStats),
		},
		lastAnalysis: make(map[string]time.Time),
		stopChan:     make(chan struct{}),
		running:      false,

		// Накопление статистики
		logStats:         make(map[int]*periodStats),
		logLastFlush:     time.Now(),
		logFlushInterval: 10 * time.Second,
	}

	// НЕ регистрируем стандартные анализаторы здесь
	// Они будут созданы через фабрику с реальными зависимостями
	logger.Warn("ℹ️ AnalysisEngine создан без анализаторов")
	logger.Info("ℹ️ Анализаторы будут созданы через фабрику с реальными зависимостями")

	// УДАЛЕНО: setupDefaultFilters() - AnalysisEngine теперь только оркестратор

	logger.Info("✅ AnalysisEngine создан как оркестратор анализаторов")
	return engine
}

// Start запускает движок анализа (оркестратор)
func (e *AnalysisEngine) Start() error {
	if e.running {
		return fmt.Errorf("analysis engine already running")
	}

	e.running = true

	// Запускаем периодический анализ
	e.wg.Add(1)
	go e.analysisLoop()

	logger.Info("🚀 AnalysisEngine запущен как оркестратор с %d анализаторами", len(e.analyzers))
	return nil
}

// Stop останавливает движок анализа
func (e *AnalysisEngine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return
	}

	e.running = false
	close(e.stopChan)

	logger.Info("🛑 Останавливаем AnalysisEngine...")

	// ✅ ОСТАНАВЛИВАЕМ ВСЕ АНАЛИЗАТОРЫ
	if e.analyzers != nil {
		logger.Info("🔧 Останавливаем анализаторы...")
		for name, analyzer := range e.analyzers {
			logger.Debug("🛑 Останавливаем анализатор: %s", name)

			// Пробуем разные сигнатуры Stop()
			if stopper, ok := analyzer.(interface{ Stop() error }); ok {
				if err := stopper.Stop(); err != nil {
					logger.Warn("⚠️ Ошибка остановки анализатора %s: %v", name, err)
				} else {
					logger.Debug("✅ Анализатор %s остановлен", name)
				}
			} else if stopper, ok := analyzer.(interface{ Stop() }); ok {
				stopper.Stop()
				logger.Debug("✅ Анализатор %s остановлен", name)
			} else {
				logger.Debug("⚠️ Анализатор %s не имеет метода Stop()", name)
			}
		}
	}

	// Ждем завершения горутин анализа
	e.wg.Wait()

	// Сохраняем статистику при остановке
	e.saveStats()

	logger.Info("✅ AnalysisEngine остановлен")
}

// RegisterAnalyzer регистрирует анализатор в оркестраторе
func (e *AnalysisEngine) RegisterAnalyzer(analyzer common.Analyzer) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	name := analyzer.Name()
	if _, exists := e.analyzers[name]; exists {
		return fmt.Errorf("analyzer %s already registered", name)
	}

	e.analyzers[name] = analyzer

	// Инициализируем статистику
	e.stats.AnalyzerStats[name] = common.AnalyzerStats{}
	e.stats.ActiveAnalyzers++

	logger.Info("✅ Зарегистрирован анализатор: %s v%s", name, analyzer.Version())
	return nil
}

// UnregisterAnalyzer удаляет анализатор из оркестратора
func (e *AnalysisEngine) UnregisterAnalyzer(name string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.analyzers[name]; !exists {
		return fmt.Errorf("analyzer %s not found", name)
	}

	delete(e.analyzers, name)
	delete(e.stats.AnalyzerStats, name)
	e.stats.ActiveAnalyzers--

	log.Printf("❌ Удален анализатор: %s", name)
	return nil
}

// AnalyzeSymbol анализирует конкретный символ через зарегистрированные анализаторы
func (e *AnalysisEngine) AnalyzeSymbol(symbol string, periods []time.Duration) (*analysis.AnalysisResult, error) {
	startTime := time.Now()

	// Проверяем объем символа
	if !e.passesVolumeFilter(symbol) {
		return nil, fmt.Errorf("symbol %s doesn't pass volume filter", symbol)
	}

	var allSignals []analysis.Signal

	// ЗАПУСКАЕМ ВСЕ ЗАРЕГИСТРИРОВАННЫЕ АНАЛИЗАТОРЫ
	e.mu.RLock()
	analyzersList := make([]common.Analyzer, 0, len(e.analyzers))
	for _, analyzer := range e.analyzers {
		if analyzer.Supports(symbol) {
			analyzersList = append(analyzersList, analyzer)
		}
	}
	e.mu.RUnlock()

	// Получаем текущие данные для символа
	if snapshot, exists := e.storage.GetCurrentSnapshot(symbol); exists {
		// Создаем массив с одной точкой данных
		data := []storage.PriceDataInterface{snapshot}

		// Запускаем каждый анализатор
		for _, analyzer := range analyzersList {
			signals, err := analyzer.Analyze(data, analyzer.GetConfig())
			if err != nil {
				logger.Debug("⚠️ Ошибка анализа %s анализатором %s: %v",
					symbol, analyzer.Name(), err)
				continue
			}

			// Добавляем метаданные
			for i := range signals {
				signals[i].Symbol = symbol
				signals[i].Timestamp = time.Now()
				signals[i].ID = uuid.New().String()
			}

			allSignals = append(allSignals, signals...)
		}
	}

	// УДАЛЕНО: Применение фильтров
	// AnalysisEngine теперь только оркестратор, не фильтрует сигналы

	// Используем все сигналы (без фильтрации)
	filteredSignals := allSignals

	// Обновляем статистику
	e.updateStats(symbol, len(allSignals), len(filteredSignals), time.Since(startTime))

	result := &analysis.AnalysisResult{
		Symbol:    symbol,
		Signals:   filteredSignals,
		Timestamp: time.Now(),
		Duration:  time.Since(startTime),
	}

	// Публикуем событие если есть сигналы
	if len(filteredSignals) > 0 {
		e.publishSignals(filteredSignals)
	}

	return result, nil
}

// AnalyzeAll анализирует все символы через анализаторы
func (e *AnalysisEngine) AnalyzeAll() (map[string]*analysis.AnalysisResult, error) {
	startTime := time.Now()
	logger.Info("✅ AnalysisEngine: цикл периодического анализа с интервалом %v", e.config.UpdateInterval)

	// Получаем символы для анализа
	symbols := e.getSymbolsToAnalyze()

	results := make(map[string]*analysis.AnalysisResult)

	if e.config.EnableParallel {
		results = e.analyzeParallel(symbols)
	} else {
		results = e.analyzeSequential(symbols)
	}

	// Публикуем общую статистику
	e.publishAnalysisComplete(results, time.Since(startTime))

	return results, nil
}

// analyzeParallel анализирует символы параллельно
func (e *AnalysisEngine) analyzeParallel(symbols []string) map[string]*analysis.AnalysisResult {
	results := make(map[string]*analysis.AnalysisResult)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Ограничиваем количество одновременных горутин
	workerPool := make(chan struct{}, e.config.MaxWorkers)

	for _, symbol := range symbols {
		wg.Add(1)
		workerPool <- struct{}{}

		go func(s string) {
			defer wg.Done()
			defer func() { <-workerPool }()

			result, err := e.AnalyzeSymbol(s, e.config.AnalysisPeriods)
			if err != nil {
				logger.Debug("⚠️ Ошибка анализа %s: %v", s, err)
				return
			}

			mu.Lock()
			results[s] = result
			mu.Unlock()
		}(symbol)
	}

	wg.Wait()
	return results
}

// analyzeSequential анализирует символы последовательно
func (e *AnalysisEngine) analyzeSequential(symbols []string) map[string]*analysis.AnalysisResult {
	results := make(map[string]*analysis.AnalysisResult)

	for _, symbol := range symbols {
		result, err := e.AnalyzeSymbol(symbol, e.config.AnalysisPeriods)
		if err != nil {
			logger.Debug("⚠️ Ошибка анализа %s: %v", symbol, err)
			continue
		}

		results[symbol] = result
	}

	return results
}

// getSymbolsToAnalyze возвращает символы для анализа
func (e *AnalysisEngine) getSymbolsToAnalyze() []string {
	allSymbols := e.storage.GetSymbols()

	logger.Debug("🔍 AnalysisEngine: всего символов в хранилище: %d", len(allSymbols))

	// Фильтруем по объему
	var filtered []string
	for _, symbol := range allSymbols {
		if e.passesVolumeFilter(symbol) {
			filtered = append(filtered, symbol)
		}
	}

	// Ограничиваем количество
	if len(filtered) > e.config.MaxSymbolsPerRun {
		// Сортируем по объему (по убыванию)
		sorted := e.sortByVolume(filtered)
		filtered = sorted[:e.config.MaxSymbolsPerRun]
	}

	// Сортируем по алфавиту для детерминированности
	sort.Strings(filtered)
	logger.Debug("✅ AnalysisEngine: символов после фильтрации: %d", len(filtered))

	return filtered
}

// sortByVolume сортирует символы по объему
func (e *AnalysisEngine) sortByVolume(symbols []string) []string {
	type symbolVolume struct {
		symbol string
		volume float64
	}

	var sv []symbolVolume
	for _, symbol := range symbols {
		if snapshot, exists := e.storage.GetCurrentSnapshot(symbol); exists {
			// Используем геттер вместо прямого доступа к полю
			sv = append(sv, symbolVolume{symbol, snapshot.GetVolumeUSD()})
		}
	}

	// Сортируем по убыванию объема
	sort.Slice(sv, func(i, j int) bool {
		return sv[i].volume > sv[j].volume
	})

	result := make([]string, len(sv))
	for i, item := range sv {
		result[i] = item.symbol
	}

	return result
}

// passesVolumeFilter проверяет фильтр объема
func (e *AnalysisEngine) passesVolumeFilter(symbol string) bool {
	if e.config.MinVolumeFilter <= 0 {
		return true
	}

	if snapshot, exists := e.storage.GetCurrentSnapshot(symbol); exists {
		// Используем геттер вместо прямого доступа к полю
		return snapshot.GetVolumeUSD() >= e.config.MinVolumeFilter
	}

	return false
}

// updateStats обновляет статистику
func (e *AnalysisEngine) updateStats(symbol string, totalSignals, filteredSignals int, duration time.Duration) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.stats.TotalAnalyses++
	e.stats.TotalSignals += int64(filteredSignals)
	e.stats.AnalysisTime += duration
	e.stats.LastRunTime = time.Now()
	e.stats.SymbolsAnalyzed[symbol]++
}

// publishSignals публикует сигналы в EventBus
func (e *AnalysisEngine) publishSignals(signals []analysis.Signal) {
	if len(signals) == 0 {
		return
	}

	// Накопление статистики
	e.logStatsMu.Lock()
	defer e.logStatsMu.Unlock()

	for _, signal := range signals {
		// Публикуем в EventBus
		e.eventBus.Publish(types.Event{
			Type:   types.EventSignalDetected,
			Source: "analysis_engine",
			Data:   signal,
			Metadata: types.Metadata{
				CorrelationID: signal.ID,
				Priority:      int(signal.Confidence / 10),
				Tags:          signal.Metadata.Tags,
			},
		})

		// Накопление статистики
		period := signal.Period
		if _, exists := e.logStats[period]; !exists {
			e.logStats[period] = &periodStats{
				growthCount: 0,
				fallCount:   0,
				symbols:     make(map[string]bool),
			}
		}

		stats := e.logStats[period]
		if signal.Direction == "growth" {
			stats.growthCount++
		} else {
			stats.fallCount++
		}
		stats.symbols[signal.Symbol] = true
	}

	// Проверяем, пора ли выводить статистику
	if time.Since(e.logLastFlush) >= e.logFlushInterval {
		e.flushLogStats()
	}
}

// flushLogStats выводит накопленную статистику одним сообщением
func (e *AnalysisEngine) flushLogStats() {
	if len(e.logStats) == 0 {
		e.logLastFlush = time.Now()
		return
	}

	// Собираем статистику
	var totalSignals, totalGrowth, totalFall int
	allSymbols := make(map[string]bool)

	// Сортируем периоды
	var periods []int
	for period := range e.logStats {
		periods = append(periods, period)
	}
	sort.Ints(periods)

	// Собираем данные по периодам
	type periodData struct {
		growth      int
		fall        int
		symbols     map[string]bool
		signalCount int
	}

	periodDataMap := make(map[int]*periodData)

	for _, period := range periods {
		stats := e.logStats[period]
		signalCount := stats.growthCount + stats.fallCount
		totalSignals += signalCount
		totalGrowth += stats.growthCount
		totalFall += stats.fallCount

		for symbol := range stats.symbols {
			allSymbols[symbol] = true
		}

		periodDataMap[period] = &periodData{
			growth:      stats.growthCount,
			fall:        stats.fallCount,
			symbols:     stats.symbols,
			signalCount: signalCount,
		}
	}

	if totalSignals == 0 {
		e.logStats = make(map[int]*periodStats)
		e.logLastFlush = time.Now()
		return
	}

	// Рассчитываем метрики
	elapsed := e.logFlushInterval.Seconds()
	calls := e.stats.TotalAnalyses
	symbolsProcessed := len(allSymbols)

	var avgTimePerCall time.Duration
	if calls > 0 {
		avgTimePerCall = e.stats.AnalysisTime / time.Duration(calls)
	}

	var avgSignalsPerSymbol float64
	if symbolsProcessed > 0 {
		avgSignalsPerSymbol = float64(totalSignals) / float64(symbolsProcessed)
	}

	speed := float64(symbolsProcessed) / elapsed

	// Форматируем как CounterAnalyzer (фиксированная ширина 4 символа)
	formatNumFixed := func(n interface{}) string {
		var str string
		switch v := n.(type) {
		case int:
			str = fmt.Sprintf("%d", v)
			// Выравниваем вправо до 4 символов
			if len(str) < 4 {
				str = strings.Repeat(" ", 4-len(str)) + str
			}
		case float64:
			str = fmt.Sprintf("%.1f", v)
			if len(str) < 4 {
				str = strings.Repeat(" ", 4-len(str)) + str
			}
		case time.Duration:
			str = fmt.Sprintf("%v", v.Round(time.Millisecond))
		default:
			str = fmt.Sprintf("%v", v)
		}
		return str
	}

	// ✅ ВЫВОДИМ КАЖДУЮ СТРОКУ ЧЕРЕЗ LOGGER.WARN() КАК В COUNTERANALYZER
	logger.Warn("📊 [AnalysisEngine] Статистика за последние %.0fs:", elapsed)
	logger.Warn("   📞 Вызовов AnalyzeAll: %s", formatNumFixed(calls))
	logger.Warn("   📍 Обработано символов: %s", formatNumFixed(symbolsProcessed))
	logger.Warn("   ⏱️  Среднее время: %s", avgTimePerCall.Round(time.Millisecond))
	logger.Warn("   📈 Среднее сигналов/символ: %s", formatNumFixed(avgSignalsPerSymbol))
	logger.Warn("   ⚡ Скорость: %s символов/сек", formatNumFixed(speed))

	// ✅ СИГНАЛЫ ПО ПЕРИОДАМ (как у CounterAnalyzer)
	logger.Warn("   📊 СИГНАЛЫ по периодам:")

	for _, period := range periods {
		data := periodDataMap[period]
		if data.signalCount > 0 {
			// Форматируем как у CounterAnalyzer
			logger.Warn("      • %s минут: рост=%s, падение=%s, символов=%s",
				formatNumFixed(period),
				formatNumFixed(data.growth),
				formatNumFixed(data.fall),
				formatNumFixed(len(data.symbols)))
		}
	}

	// ✅ ИТОГИ (как у CounterAnalyzer)
	logger.Warn("   📊 ИТОГО: сигналов=%s, рост=%s, падение=%s, уникальных символов=%s",
		formatNumFixed(totalSignals),
		formatNumFixed(totalGrowth),
		formatNumFixed(totalFall),
		formatNumFixed(len(allSymbols)))

	// Сбрасываем статистику
	e.logStats = make(map[int]*periodStats)
	e.logLastFlush = time.Now()

	// Сбрасываем счетчик вызовов для следующего интервала
	e.stats.TotalAnalyses = 0
	e.stats.AnalysisTime = 0
}

// logGroupedSignalStats логирует сгруппированную статистику сигналов
func (e *AnalysisEngine) logGroupedSignalStats(periodStats map[int]struct {
	growth  int
	fall    int
	symbols map[string]bool
}, totalSignals int) {

	// Сортируем периоды
	var periods []int
	for period := range periodStats {
		periods = append(periods, period)
	}
	sort.Ints(periods)

	// Логируем заголовок
	logger.Info("📊 AnalysisEngine: найдено %d сигналов", totalSignals)

	var totalGrowth, totalFall int
	var allSymbols = make(map[string]bool)

	// Логируем по периодам
	for _, period := range periods {
		stats := periodStats[period]
		uniqueSymbols := len(stats.symbols)
		totalGrowth += stats.growth
		totalFall += stats.fall

		// Добавляем символы в общий список
		for symbol := range stats.symbols {
			allSymbols[symbol] = true
		}

		logger.Info("   📈 %s минут: рост=%d, падение=%d, символов=%d",
			strconv.Itoa(period), stats.growth, stats.fall, uniqueSymbols)
	}

	// Логируем итого
	logger.Info("   📊 ИТОГО: рост=%d, падение=%d, уникальных символов=%d",
		totalGrowth, totalFall, len(allSymbols))
}

// publishAnalysisComplete публикует событие завершения анализа
func (e *AnalysisEngine) publishAnalysisComplete(results map[string]*analysis.AnalysisResult, duration time.Duration) {
	// Считаем общее количество сигналов
	totalSignals := 0
	for _, result := range results {
		totalSignals += len(result.Signals)
	}

	e.eventBus.Publish(types.Event{
		Type:   "analysis_complete",
		Source: "analysis_engine",
		Data: map[string]interface{}{
			"symbols_analyzed": len(results),
			"total_signals":    totalSignals,
			"duration":         duration.String(),
			"timestamp":        time.Now(),
		},
	})

}

// analysisLoop цикл периодического анализа
func (e *AnalysisEngine) analysisLoop() {
	defer e.wg.Done()

	ticker := time.NewTicker(e.config.UpdateInterval)
	defer ticker.Stop()

	// Первоначальный анализ
	e.AnalyzeAll()

	for {
		select {
		case <-ticker.C:
			e.AnalyzeAll()
		case <-e.stopChan:
			return
		}
	}
}

// GetStats возвращает статистику движка
func (e *AnalysisEngine) GetStats() EngineStats {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.stats
}

// GetAnalyzers возвращает список зарегистрированных анализаторов
func (e *AnalysisEngine) GetAnalyzers() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	names := make([]string, 0, len(e.analyzers))
	for name := range e.analyzers {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

// saveStats сохраняет статистику (заглушка)
func (e *AnalysisEngine) saveStats() {
	// В будущем можно сохранять в файл или базу данных
	logger.Info("💾 Сохранение статистики AnalysisEngine")
}

// УДАЛЕНО: setupDefaultFilters() - AnalysisEngine теперь только оркестратор

// УДАЛЕНО: subscribeToEvents() - AnalysisEngine не слушает события

// УДАЛЕНО: handleEvent() - AnalysisEngine не обрабатывает события

// УДАЛЕНО: FilterChain и все связанное с фильтрами

// Name возвращает имя сервиса
func (e *AnalysisEngine) Name() string {
	return "AnalysisEngine"
}

// State возвращает состояние сервиса
func (e *AnalysisEngine) State() string {
	if e.running {
		return "running"
	}
	return "stopped"
}

// IsRunning возвращает true если сервис запущен
func (e *AnalysisEngine) IsRunning() bool {
	return e.running
}

// HealthCheck проверяет здоровье сервиса
func (e *AnalysisEngine) HealthCheck() bool {
	if !e.running {
		return false
	}

	// Проверяем наличие хранилища
	if e.storage == nil {
		return false
	}

	// Проверяем наличие анализаторов
	if len(e.analyzers) == 0 {
		return false
	}

	return true
}

// GetStatus возвращает подробный статус
func (e *AnalysisEngine) GetStatus() map[string]interface{} {
	stats := e.GetStats()

	status := map[string]interface{}{
		"name":        e.Name(),
		"running":     e.running,
		"state":       e.State(),
		"healthy":     e.HealthCheck(),
		"analyzers":   e.GetAnalyzers(),
		"total_stats": stats,
	}

	// УДАЛЕНО: filter_stats - AnalysisEngine теперь только оркестратор

	// Информация о конфигурации
	status["config"] = map[string]interface{}{
		"parallel_analysis":   e.config.EnableParallel,
		"max_workers":         e.config.MaxWorkers,
		"analysis_interval":   e.config.UpdateInterval.String(),
		"min_volume":          e.config.MinVolumeFilter,
		"sort_by_volume":      true,
		"update_interval":     e.config.UpdateInterval.String(),
		"analysis_periods":    e.config.AnalysisPeriods,
		"max_symbols_per_run": e.config.MaxSymbolsPerRun,
		"signal_threshold":    e.config.SignalThreshold,
		"retention_period":    e.config.RetentionPeriod.String(),
		"enable_cache":        e.config.EnableCache,
		"min_data_points":     e.config.MinDataPoints,
	}

	return status
}
