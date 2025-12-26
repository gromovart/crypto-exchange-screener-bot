package engine

import (
	"crypto-exchange-screener-bot/internal/analysis/analyzers"
	"crypto-exchange-screener-bot/internal/analysis/filters"
	"crypto-exchange-screener-bot/internal/config"
	"crypto-exchange-screener-bot/internal/delivery/telegram"
	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/in_memory_storage"
	events "crypto-exchange-screener-bot/internal/infrastructure/transport/event_bus"
	"log"
	"time"
)

// Factory - фабрика для создания AnalysisEngine
type Factory struct{}

// NewAnalysisEngineFromConfig создает AnalysisEngine из конфигурации
func (f *Factory) NewAnalysisEngineFromConfig(
	storage storage.PriceStorage,
	eventBus *events.EventBus,
	cfg *config.Config,
	telegramBot *telegram.TelegramBot, // ПЕРЕДАЕМ БОТА ЧЕРЕЗ DI
) *AnalysisEngine {

	// Конвертируем периоды
	var periods []time.Duration
	for _, period := range cfg.AnalysisEngine.AnalysisPeriods {
		periods = append(periods, time.Duration(period)*time.Minute)
	}

	// Создаем конфигурацию движка
	engineConfig := EngineConfig{
		UpdateInterval:   time.Duration(cfg.AnalysisEngine.UpdateInterval) * time.Second,
		AnalysisPeriods:  periods,
		MinVolumeFilter:  cfg.MinVolumeFilter,
		MaxSymbolsPerRun: cfg.AnalysisEngine.MaxSymbolsPerRun,
		EnableParallel:   cfg.AnalysisEngine.EnableParallel,
		MaxWorkers:       cfg.AnalysisEngine.MaxWorkers,
		SignalThreshold:  cfg.AnalysisEngine.SignalThreshold,
		RetentionPeriod:  time.Duration(cfg.AnalysisEngine.RetentionPeriod) * time.Hour,
		EnableCache:      cfg.AnalysisEngine.EnableCache,
		MinDataPoints:    3,

		// Конфигурация анализаторов
		AnalyzerConfigs: AnalyzerConfigs{
			GrowthAnalyzer: AnalyzerConfig{
				Enabled:       cfg.Analyzers.GrowthAnalyzer.Enabled,
				MinConfidence: cfg.Analyzers.GrowthAnalyzer.MinConfidence,
				MinGrowth:     cfg.Analyzers.GrowthAnalyzer.MinGrowth,
			},
			FallAnalyzer: AnalyzerConfig{
				Enabled:       cfg.Analyzers.FallAnalyzer.Enabled,
				MinConfidence: cfg.Analyzers.FallAnalyzer.MinConfidence,
				MinFall:       cfg.Analyzers.FallAnalyzer.MinFall,
			},
			ContinuousAnalyzer: AnalyzerConfig{
				Enabled: cfg.Analyzers.ContinuousAnalyzer.Enabled,
			},
		},

		// Конфигурация фильтров
		FilterConfigs: FilterConfigs{
			SignalFilters: SignalFilterConfig{
				Enabled:          cfg.SignalFilters.Enabled,
				MinConfidence:    cfg.SignalFilters.MinConfidence,
				MaxSignalsPerMin: cfg.SignalFilters.MaxSignalsPerMin,
			},
		},
	}

	// Создаем движок
	engine := NewAnalysisEngine(storage, eventBus, engineConfig)

	// Настраиваем анализаторы и фильтры
	f.configureAnalyzers(engine, cfg, telegramBot) // ПЕРЕДАЕМ БОТА
	f.configureFilters(engine, cfg)

	return engine
}

// configureAnalyzers настраивает анализаторы
func (f *Factory) configureAnalyzers(
	engine *AnalysisEngine,
	cfg *config.Config,
	telegramBot *telegram.TelegramBot, // ПЕРЕДАЕМ БОТА
) {
	minDataPoints := 3

	// Настраиваем GrowthAnalyzer
	if cfg.Analyzers.GrowthAnalyzer.Enabled {
		growthConfig := analyzers.AnalyzerConfig{
			Enabled:       true,
			Weight:        1.0,
			MinConfidence: cfg.Analyzers.GrowthAnalyzer.MinConfidence,
			MinDataPoints: minDataPoints,
			CustomSettings: map[string]interface{}{
				"min_growth":           cfg.Analyzers.GrowthAnalyzer.MinGrowth,
				"continuity_threshold": cfg.Analyzers.GrowthAnalyzer.ContinuityThreshold,
				"volume_weight":        0.2,
			},
		}

		growthAnalyzer := analyzers.NewGrowthAnalyzer(growthConfig)
		engine.RegisterAnalyzer(growthAnalyzer)
	}

	// Настраиваем FallAnalyzer
	if cfg.Analyzers.FallAnalyzer.Enabled {
		fallConfig := analyzers.AnalyzerConfig{
			Enabled:       true,
			Weight:        1.0,
			MinConfidence: cfg.Analyzers.FallAnalyzer.MinConfidence,
			MinDataPoints: minDataPoints,
			CustomSettings: map[string]interface{}{
				"min_fall":             cfg.Analyzers.FallAnalyzer.MinFall,
				"continuity_threshold": cfg.Analyzers.FallAnalyzer.ContinuityThreshold,
				"volume_weight":        0.2,
			},
		}

		fallAnalyzer := analyzers.NewFallAnalyzer(fallConfig)
		engine.RegisterAnalyzer(fallAnalyzer)
	}

	// ContinuousAnalyzer если включена проверка непрерывности
	if cfg.Analyzers.ContinuousAnalyzer.Enabled {
		continuousConfig := analyzers.AnalyzerConfig{
			Enabled:       true,
			Weight:        0.8,
			MinConfidence: cfg.Analyzers.GrowthAnalyzer.MinConfidence,
			MinDataPoints: minDataPoints,
			CustomSettings: map[string]interface{}{
				"min_continuous_points": cfg.Analyzers.ContinuousAnalyzer.MinContinuousPoints,
				"max_gap_ratio":         0.3,
			},
		}

		continuousAnalyzer := analyzers.NewContinuousAnalyzer(continuousConfig)
		engine.RegisterAnalyzer(continuousAnalyzer)
	}

	// CounterAnalyzer если включен
	if cfg.CounterAnalyzer.Enabled {
		f.configureCounterAnalyzer(engine, cfg, telegramBot) // ПЕРЕДАЕМ БОТА
	}
}

// configureCounterAnalyzer настраивает CounterAnalyzer
func (f *Factory) configureCounterAnalyzer(
	engine *AnalysisEngine,
	cfg *config.Config,
	telegramBot *telegram.TelegramBot, // ИСПОЛЬЗУЕМ ПЕРЕДАННОГО БОТА
) {
	log.Println("🔧 Настройка CounterAnalyzer с переданным Telegram ботом")

	// НЕ СОЗДАЕМ НОВОГО БОТА, ИСПОЛЬЗУЕМ ПЕРЕДАННОГО
	// var tgBot *telegram.TelegramBot - УДАЛЯЕМ ЭТУ СТРОКУ

	// Проверяем, передан ли бот
	if cfg.TelegramEnabled && telegramBot == nil {
		log.Println("⚠️ Telegram включен в конфигурации, но бот не передан в CounterAnalyzer")
	}

	// Настройки CounterAnalyzer из конфигурации
	counterConfig := analyzers.AnalyzerConfig{
		Enabled:       true,
		Weight:        0.7,
		MinConfidence: 10.0,
		MinDataPoints: 2,
		CustomSettings: map[string]interface{}{
			"base_period_minutes":    cfg.CounterAnalyzer.BasePeriodMinutes,
			"analysis_period":        cfg.CounterAnalyzer.DefaultPeriod,
			"growth_threshold":       cfg.CounterAnalyzer.GrowthThreshold,
			"fall_threshold":         cfg.CounterAnalyzer.FallThreshold,
			"track_growth":           cfg.CounterAnalyzer.TrackGrowth,
			"track_fall":             cfg.CounterAnalyzer.TrackFall,
			"notify_on_signal":       cfg.CounterAnalyzer.NotifyOnSignal,
			"notification_threshold": cfg.CounterAnalyzer.NotificationThreshold,
			"chart_provider":         cfg.CounterAnalyzer.ChartProvider,
		},
	}

	// Создаем CounterAnalyzer с переданным ботом
	storage := engine.GetStorage()
	counterAnalyzer := analyzers.NewCounterAnalyzer(counterConfig, storage, telegramBot) // ИСПОЛЬЗУЕМ ПЕРЕДАННОГО БОТА

	// Регистрируем анализатор
	if err := engine.RegisterAnalyzer(counterAnalyzer); err != nil {
		log.Printf("⚠️ Не удалось зарегистрировать CounterAnalyzer: %v", err)
	} else {
		log.Printf("✅ CounterAnalyzer успешно добавлен в AnalysisEngine (Telegram бот: %v)",
			telegramBot != nil)
	}
}

// configureFilters настраивает фильтры (без изменений)
func (f *Factory) configureFilters(engine *AnalysisEngine, cfg *config.Config) {
	// ConfidenceFilter
	if cfg.SignalFilters.Enabled && cfg.SignalFilters.MinConfidence > 0 {
		confidenceFilter := filters.NewConfidenceFilter(cfg.SignalFilters.MinConfidence)
		engine.AddFilter(confidenceFilter)
	}

	// VolumeFilter
	if cfg.MinVolumeFilter > 0 {
		volumeFilter := filters.NewVolumeFilter(cfg.MinVolumeFilter)
		engine.AddFilter(volumeFilter)
	}

	// RateLimitFilter
	if cfg.SignalFilters.Enabled && cfg.SignalFilters.MaxSignalsPerMin > 0 {
		minDelay := time.Minute / time.Duration(cfg.SignalFilters.MaxSignalsPerMin)
		rateLimitFilter := filters.NewRateLimitFilter(minDelay)
		engine.AddFilter(rateLimitFilter)
	}
}

// GetStorage возвращает хранилище из движка
func (e *AnalysisEngine) GetStorage() storage.PriceStorage {
	return e.storage
}
