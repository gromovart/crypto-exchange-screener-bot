// internal/core/domain/signals/engine/engine.go
package engine

import (
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	analyzers "crypto-exchange-screener-bot/internal/core/domain/signals/detectors"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/common"
	"crypto-exchange-screener-bot/internal/core/domain/signals/filters"
	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/in_memory_storage"
	events "crypto-exchange-screener-bot/internal/infrastructure/transport/event_bus"
	"crypto-exchange-screener-bot/internal/types"
	"fmt"
	"log"
	"sort"
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

// AnalysisEngine - основной движок анализа
type AnalysisEngine struct {
	mu           sync.RWMutex
	analyzers    map[string]common.Analyzer
	filters      *FilterChain
	storage      storage.PriceStorage
	eventBus     *events.EventBus
	config       EngineConfig
	stats        EngineStats
	lastAnalysis map[string]time.Time
	stopChan     chan struct{}
	wg           sync.WaitGroup
	running      bool
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

	// Добавляем поля для анализаторов и фильтров
	AnalyzerConfigs AnalyzerConfigs `json:"analyzer_configs"`
	FilterConfigs   FilterConfigs   `json:"filter_configs"`
}

// FilterConfigs - конфигурация фильтров
type FilterConfigs struct {
	SignalFilters SignalFilterConfig `json:"signal_filters"`
}

// SignalFilterConfig - конфигурация фильтров сигналов
type SignalFilterConfig struct {
	Enabled          bool    `json:"enabled"`
	MinConfidence    float64 `json:"min_confidence"`
	MaxSignalsPerMin int     `json:"max_signals_per_min"`
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
	UpdateInterval:   10 * time.Second,
	AnalysisPeriods:  []time.Duration{5 * time.Minute, 15 * time.Minute, 30 * time.Minute, 60 * time.Minute},
	MinVolumeFilter:  100000,
	MaxSymbolsPerRun: 100,
	EnableParallel:   true,
	MaxWorkers:       5,
	SignalThreshold:  2.0,
	RetentionPeriod:  24 * time.Hour,
	EnableCache:      true,
}

// NewAnalysisEngine создает новый движок анализа
func NewAnalysisEngine(storage storage.PriceStorage, eventBus *events.EventBus, config ...EngineConfig) *AnalysisEngine {
	cfg := DefaultConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	engine := &AnalysisEngine{
		analyzers: make(map[string]common.Analyzer),
		filters:   NewFilterChain(),
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
	}

	// Регистрируем стандартные анализаторы
	engine.registerDefaultAnalyzers()

	// Настраиваем стандартные фильтры
	engine.setupDefaultFilters()

	return engine
}

// Start запускает движок анализа
func (e *AnalysisEngine) Start() error {
	if e.running {
		return fmt.Errorf("analysis engine already running")
	}

	e.running = true

	// Запускаем периодический анализ
	e.wg.Add(1)
	go e.analysisLoop()

	// Подписываемся на события
	e.subscribeToEvents()

	log.Printf("🚀 AnalysisEngine запущен с %d анализаторами", len(e.analyzers))
	return nil
}

// Stop останавливает движок анализа
func (e *AnalysisEngine) Stop() error {
	if !e.running {
		return nil
	}

	e.running = false
	close(e.stopChan)
	e.wg.Wait()

	// Сохраняем статистику
	e.saveStats()

	log.Println("🛑 AnalysisEngine остановлен")
	return nil
}

// RegisterAnalyzer регистрирует анализатор
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

	log.Printf("✅ Зарегистрирован анализатор: %s v%s", name, analyzer.Version())
	return nil
}

// UnregisterAnalyzer удаляет анализатор
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

// AddFilter добавляет фильтр в цепочку
func (e *AnalysisEngine) AddFilter(filter filters.Filter) {
	e.filters.Add(filter)
	log.Printf("➕ Добавлен фильтр: %s", filter.Name())
}

// AnalyzeSymbol анализирует конкретный символ
func (e *AnalysisEngine) AnalyzeSymbol(symbol string, periods []time.Duration) (*analysis.AnalysisResult, error) {
	startTime := time.Now()

	// Проверяем объем символа
	if !e.passesVolumeFilter(symbol) {
		return nil, fmt.Errorf("symbol %s doesn't pass volume filter", symbol)
	}

	var allSignals []analysis.Signal

	// Анализируем для каждого периода
	for _, period := range periods {
		signals, err := e.analyzePeriod(symbol, period)
		if err != nil {
			continue
		}
		allSignals = append(allSignals, signals...)
	}

	// Применяем фильтры
	filteredSignals := e.filters.Apply(allSignals)

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

// AnalyzeAll анализирует все символы
func (e *AnalysisEngine) AnalyzeAll() (map[string]*analysis.AnalysisResult, error) {
	startTime := time.Now()

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

// analyzePeriod анализирует символ за конкретный период
func (e *AnalysisEngine) analyzePeriod(symbol string, period time.Duration) ([]analysis.Signal, error) {
	// Получаем данные за период
	endTime := time.Now()
	startTime := endTime.Add(-period)

	priceData, err := e.storage.GetPriceHistoryRange(symbol, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get price history for %s: %w", symbol, err)
	}

	if len(priceData) < 2 {
		return nil, fmt.Errorf("insufficient data for %s", symbol)
	}

	// Конвертируем в формат анализа
	data := convertToPriceData(priceData)

	// Запускаем все анализаторы
	var allSignals []analysis.Signal

	e.mu.RLock()
	analyzersList := make([]common.Analyzer, 0, len(e.analyzers))
	for _, analyzer := range e.analyzers {
		if analyzer.Supports(symbol) {
			analyzersList = append(analyzersList, analyzer)
		}
	}
	e.mu.RUnlock()

	for _, analyzer := range analyzersList {
		signals, err := analyzer.Analyze(data, analyzer.GetConfig())
		if err != nil {
			log.Printf("⚠️ Ошибка анализа %s анализатором %s: %v", symbol, analyzer.Name(), err)
			continue
		}

		// Добавляем метаданные
		for i := range signals {
			signals[i].Symbol = symbol
			signals[i].Period = int(period.Minutes())
			signals[i].Timestamp = time.Now()
			signals[i].ID = uuid.New().String()
		}

		allSignals = append(allSignals, signals...)
	}

	return allSignals, nil
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
				log.Printf("⚠️ Ошибка анализа %s: %v", s, err)
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
			log.Printf("⚠️ Ошибка анализа %s: %v", symbol, err)
			continue
		}

		results[symbol] = result
	}

	return results
}

// getSymbolsToAnalyze возвращает символы для анализа
func (e *AnalysisEngine) getSymbolsToAnalyze() []string {
	allSymbols := e.storage.GetSymbols()

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
			sv = append(sv, symbolVolume{symbol, snapshot.Volume24h})
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
		return snapshot.Volume24h >= e.config.MinVolumeFilter
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
	for _, signal := range signals {
		e.eventBus.Publish(events.Event{
			Type:   events.EventSignalDetected,
			Source: "analysis_engine",
			Data:   signal,
			Metadata: events.Metadata{
				CorrelationID: signal.ID,
				Priority:      int(signal.Confidence / 10),
				Tags:          signal.Metadata.Tags,
			},
		})

		// Логируем сигнал
		log.Printf("📈 Обнаружен сигнал: %s %s %.2f%% (уверенность: %.0f%%)",
			signal.Symbol, signal.Direction, signal.ChangePercent, signal.Confidence)
	}
}

// publishAnalysisComplete публикует событие завершения анализа
func (e *AnalysisEngine) publishAnalysisComplete(results map[string]*analysis.AnalysisResult, duration time.Duration) {
	totalSignals := 0
	for _, result := range results {
		totalSignals += len(result.Signals)
	}

	e.eventBus.Publish(events.Event{
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

// subscribeToEvents подписывается на события EventBus
func (e *AnalysisEngine) subscribeToEvents() {
	subscriber := events.NewBaseSubscriber(
		"analysis_engine",
		[]events.EventType{
			events.EventPriceUpdated,
			"analysis_request",
		},
		e.handleEvent,
	)

	e.eventBus.Subscribe(events.EventPriceUpdated, subscriber)
	e.eventBus.Subscribe("analysis_request", subscriber)
}

// handleEvent обрабатывает события EventBus
func (e *AnalysisEngine) handleEvent(event events.Event) error {
	switch event.Type {
	case events.EventPriceUpdated:
		// Можно добавить реактивный анализ при обновлении цен
		// Например, анализировать только обновленный символ
		if data, ok := event.Data.(map[string]interface{}); ok {
			if symbol, ok := data["symbol"].(string); ok {
				e.AnalyzeSymbol(symbol, e.config.AnalysisPeriods)
			}
		}

	case "analysis_request":
		// Обработка запроса на анализ
		if request, ok := event.Data.(analysis.AnalysisRequest); ok {
			e.AnalyzeSymbol(request.Symbol, []time.Duration{request.Period})
		}
	}

	return nil
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
	log.Printf("💾 Сохранение статистики AnalysisEngine")
}

// registerDefaultAnalyzers регистрирует стандартные анализаторы
func (e *AnalysisEngine) registerDefaultAnalyzers() {
	e.mu.Lock()
	e.analyzers = make(map[string]common.Analyzer)
	e.stats.AnalyzerStats = make(map[string]common.AnalyzerStats)
	e.stats.ActiveAnalyzers = 0
	e.mu.Unlock()

	// GrowthAnalyzer - анализатор роста
	if e.config.AnalyzerConfigs.GrowthAnalyzer.Enabled {
		growthConfig := common.AnalyzerConfig{
			Enabled:       true,
			Weight:        1.0,
			MinConfidence: e.config.AnalyzerConfigs.GrowthAnalyzer.MinConfidence,
			MinDataPoints: e.config.MinDataPoints,
			CustomSettings: map[string]interface{}{
				"min_growth":           e.config.AnalyzerConfigs.GrowthAnalyzer.MinGrowth,
				"continuity_threshold": 0.7,
				"volume_weight":        0.2,
			},
		}
		growthAnalyzer := analyzers.NewGrowthAnalyzer(growthConfig)
		e.RegisterAnalyzer(growthAnalyzer)
	}

	// FallAnalyzer - анализатор падения
	if e.config.AnalyzerConfigs.FallAnalyzer.Enabled {
		fallConfig := common.AnalyzerConfig{
			Enabled:       true,
			Weight:        1.0,
			MinConfidence: e.config.AnalyzerConfigs.FallAnalyzer.MinConfidence,
			MinDataPoints: e.config.MinDataPoints,
			CustomSettings: map[string]interface{}{
				"min_fall":             e.config.AnalyzerConfigs.FallAnalyzer.MinFall,
				"continuity_threshold": 0.7,
				"volume_weight":        0.2,
			},
		}
		fallAnalyzer := analyzers.NewFallAnalyzer(fallConfig)
		e.RegisterAnalyzer(fallAnalyzer)
	}

	// VolumeAnalyzer - анализатор объема
	if e.config.AnalyzerConfigs.VolumeAnalyzer.Enabled {
		volumeConfig := analyzers.DefaultVolumeConfig
		volumeConfig.MinDataPoints = e.config.MinDataPoints
		volumeConfig.MinConfidence = e.config.AnalyzerConfigs.VolumeAnalyzer.MinConfidence
		volumeAnalyzer := analyzers.NewVolumeAnalyzer(volumeConfig)
		e.RegisterAnalyzer(volumeAnalyzer)
	}

	// ContinuousAnalyzer - анализатор непрерывности
	if e.config.AnalyzerConfigs.ContinuousAnalyzer.Enabled {
		continuousConfig := analyzers.DefaultContinuousConfig
		continuousConfig.MinDataPoints = e.config.MinDataPoints
		continuousConfig.MinConfidence = e.config.AnalyzerConfigs.ContinuousAnalyzer.MinConfidence
		continuousAnalyzer := analyzers.NewContinuousAnalyzer(continuousConfig)
		e.RegisterAnalyzer(continuousAnalyzer)
	}

	// OpenInterestAnalyzer - анализатор открытого интереса (НОВЫЙ)
	if e.config.AnalyzerConfigs.OpenInterestAnalyzer.Enabled {
		openInterestConfig := analyzers.DefaultOpenInterestConfig
		openInterestConfig.MinDataPoints = e.config.MinDataPoints
		openInterestConfig.MinConfidence = e.config.AnalyzerConfigs.OpenInterestAnalyzer.MinConfidence

		// Копируем пользовательские настройки если они есть
		if e.config.AnalyzerConfigs.OpenInterestAnalyzer.CustomSettings != nil {
			openInterestConfig.CustomSettings = make(map[string]interface{})
			for k, v := range e.config.AnalyzerConfigs.OpenInterestAnalyzer.CustomSettings {
				openInterestConfig.CustomSettings[k] = v
			}
		}

		openInterestAnalyzer := analyzers.NewOpenInterestAnalyzer(openInterestConfig)
		e.RegisterAnalyzer(openInterestAnalyzer)
		log.Printf("✅ OpenInterestAnalyzer инициализирован")
	}
}

// setupDefaultFilters настраивает стандартные фильтры
func (e *AnalysisEngine) setupDefaultFilters() {
	// Очищаем цепочку фильтров
	e.filters = NewFilterChain()

	// ConfidenceFilter - фильтр по уверенности
	if e.config.FilterConfigs.SignalFilters.Enabled && e.config.FilterConfigs.SignalFilters.MinConfidence > 0 {
		confidenceFilter := filters.NewConfidenceFilter(e.config.FilterConfigs.SignalFilters.MinConfidence)
		e.AddFilter(confidenceFilter)
	}

	// VolumeFilter - фильтр по объему
	if e.config.MinVolumeFilter > 0 {
		volumeFilter := filters.NewVolumeFilter(e.config.MinVolumeFilter)
		e.AddFilter(volumeFilter)
	}

	// RateLimitFilter - фильтр частоты
	if e.config.FilterConfigs.SignalFilters.Enabled && e.config.FilterConfigs.SignalFilters.MaxSignalsPerMin > 0 {
		minDelay := time.Minute / time.Duration(e.config.FilterConfigs.SignalFilters.MaxSignalsPerMin)
		rateLimitFilter := filters.NewRateLimitFilter(minDelay)
		e.AddFilter(rateLimitFilter)
	}
}

// convertToPriceData конвертирует данные хранилища в формат анализа
func convertToPriceData(storageData []storage.PriceData) []types.PriceData {
	result := make([]types.PriceData, len(storageData))

	for i, data := range storageData {
		result[i] = types.PriceData{
			Symbol:       data.Symbol,
			Price:        data.Price,
			Volume24h:    data.Volume24h,
			Timestamp:    data.Timestamp,
			OpenInterest: data.OpenInterest, // ✅ Добавляем Open Interest
			FundingRate:  data.FundingRate,  // ✅ Добавляем Funding Rate
			Change24h:    data.Change24h,    // ✅ Добавляем Change 24h
			High24h:      data.High24h,      // ✅ Добавляем High 24h
			Low24h:       data.Low24h,       // ✅ Добавляем Low 24h
		}
		// Логируем для отладки
		if data.OpenInterest > 0 {
			log.Printf("🔍 Engine.convertToPriceData: %s OI=%.0f, Funding=%.4f%%, Change24h=%.2f%%",
				data.Symbol, data.OpenInterest, data.FundingRate*100, data.Change24h)
		}
	}

	return result
}

// FilterChain - цепочка фильтров
type FilterChain struct {
	filters []filters.Filter
	mu      sync.RWMutex
}

// NewFilterChain создает новую цепочку фильтров
func NewFilterChain() *FilterChain {
	return &FilterChain{
		filters: make([]filters.Filter, 0),
	}
}

// Add добавляет фильтр в цепочку
func (fc *FilterChain) Add(filter filters.Filter) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.filters = append(fc.filters, filter)
}

// Apply применяет все фильтры к сигналам
func (fc *FilterChain) Apply(signals []analysis.Signal) []analysis.Signal {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	if len(fc.filters) == 0 {
		return signals
	}

	var filtered []analysis.Signal
	for _, signal := range signals {
		passed := true
		for _, filter := range fc.filters {
			if !filter.Apply(signal) {
				passed = false
				break
			}
		}
		if passed {
			filtered = append(filtered, signal)
		}
	}

	return filtered
}

// GetFilterStats возвращает статистику по всем фильтрам
func (e *AnalysisEngine) GetFilterStats() map[string]filters.FilterStats {
	stats := make(map[string]filters.FilterStats)

	e.filters.mu.RLock()
	defer e.filters.mu.RUnlock()

	for _, filter := range e.filters.filters {
		stats[filter.Name()] = filter.GetStats()
	}

	return stats
}
