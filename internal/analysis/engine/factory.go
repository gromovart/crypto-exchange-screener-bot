// internal/analysis/engine/factory.go (дополненная версия)
package engine

import (
	"crypto-exchange-screener-bot/internal/analysis/analyzers"
	"crypto-exchange-screener-bot/internal/analysis/filters"
	"crypto-exchange-screener-bot/internal/config"
	"crypto-exchange-screener-bot/internal/events"
	"crypto-exchange-screener-bot/internal/storage"
	"crypto-exchange-screener-bot/internal/telegram"
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
) *AnalysisEngine {

	// Конвертируем периоды
	var periods []time.Duration
	for _, period := range cfg.AnalysisEngine.AnalysisPeriods {
		periods = append(periods, time.Duration(period)*time.Minute)
	}

	// Создаем конфигурацию движка с новыми полями
	engineConfig := EngineConfig{
		UpdateInterval:   time.Duration(cfg.AnalysisEngine.UpdateInterval) * time.Second,
		AnalysisPeriods:  periods,
		MinVolumeFilter:  cfg.MinVolumeFilter, // Из основной конфигурации
		MaxSymbolsPerRun: cfg.AnalysisEngine.MaxSymbolsPerRun,
		EnableParallel:   cfg.AnalysisEngine.EnableParallel,
		MaxWorkers:       cfg.AnalysisEngine.MaxWorkers,
		SignalThreshold:  cfg.AnalysisEngine.SignalThreshold,
		RetentionPeriod:  time.Duration(cfg.AnalysisEngine.RetentionPeriod) * time.Hour, // Конвертируем часы в duration
		EnableCache:      cfg.AnalysisEngine.EnableCache,
		MinDataPoints:    3, // Значение по умолчанию

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
	f.configureAnalyzers(engine, cfg)
	f.configureFilters(engine, cfg)

	return engine
}

// configureAnalyzers настраивает анализаторы
func (f *Factory) configureAnalyzers(engine *AnalysisEngine, cfg *config.Config) {
	// Значение по умолчанию для MinDataPoints
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

	// Добавляем ContinuousAnalyzer если включена проверка непрерывности
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

	// 🔴 ВАЖНО: Добавляем CounterAnalyzer если включен
	if cfg.CounterAnalyzer.Enabled {
		f.configureCounterAnalyzer(engine, cfg)
	}
}

// configureCounterAnalyzer настраивает CounterAnalyzer
func (f *Factory) configureCounterAnalyzer(engine *AnalysisEngine, cfg *config.Config) {
	// Создаем Telegram бота для CounterAnalyzer
	var tgBot *telegram.TelegramBot
	if cfg.TelegramEnabled {
		tgBot = telegram.NewTelegramBot(cfg)
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

	// Создаем CounterAnalyzer
	storage := engine.GetStorage()
	counterAnalyzer := analyzers.NewCounterAnalyzer(counterConfig, storage, tgBot)

	// Регистрируем анализатор
	if err := engine.RegisterAnalyzer(counterAnalyzer); err != nil {
		log.Printf("⚠️ Не удалось зарегистрировать CounterAnalyzer: %v", err)
	} else {
		log.Printf("✅ CounterAnalyzer успешно добавлен в AnalysisEngine")
	}
}

// configureFilters настраивает фильтры
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

// GetStorage возвращает хранилище из движка (нужно для CounterAnalyzer)
func (e *AnalysisEngine) GetStorage() storage.PriceStorage {
	return e.storage
}
