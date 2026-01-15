// internal/core/domain/signals/engine/factory.go
package engine

import (
	"crypto-exchange-screener-bot/internal/adapters/notification"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/common"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/counter"
	"crypto-exchange-screener-bot/internal/core/domain/signals/filters"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/in_memory_storage"
	events "crypto-exchange-screener-bot/internal/infrastructure/transport/event_bus"
	"log"
	"time"
)

type Factory struct {
	priceFetcher interface{}
}

func NewFactory(priceFetcher interface{}) *Factory {
	return &Factory{
		priceFetcher: priceFetcher,
	}
}

func (f *Factory) NewAnalysisEngineFromConfig(
	storage storage.PriceStorage,
	eventBus *events.EventBus,
	cfg *config.Config,
	notifier *notification.TelegramNotifier,
) *AnalysisEngine {

	var periods []time.Duration
	for _, period := range cfg.AnalysisEngine.AnalysisPeriods {
		periods = append(periods, time.Duration(period)*time.Minute)
	}

	analyzerConfigs := cfg.AnalyzerConfigs

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
		AnalyzerConfigs: AnalyzerConfigs{
			GrowthAnalyzer: AnalyzerConfig{
				Enabled:       analyzerConfigs.GrowthAnalyzer.Enabled,
				MinConfidence: analyzerConfigs.GrowthAnalyzer.MinConfidence,
				MinGrowth:     analyzerConfigs.GrowthAnalyzer.MinGrowth,
				CustomSettings: map[string]interface{}{
					"continuity_threshold": getFloatFromCustomSettings(analyzerConfigs.GrowthAnalyzer.CustomSettings, "continuity_threshold", 0.7),
				},
			},
			FallAnalyzer: AnalyzerConfig{
				Enabled:       analyzerConfigs.FallAnalyzer.Enabled,
				MinConfidence: analyzerConfigs.FallAnalyzer.MinConfidence,
				MinFall:       analyzerConfigs.FallAnalyzer.MinFall,
				CustomSettings: map[string]interface{}{
					"continuity_threshold": getFloatFromCustomSettings(analyzerConfigs.FallAnalyzer.CustomSettings, "continuity_threshold", 0.7),
				},
			},
			ContinuousAnalyzer: AnalyzerConfig{
				Enabled: analyzerConfigs.ContinuousAnalyzer.Enabled,
			},
			VolumeAnalyzer: AnalyzerConfig{
				Enabled:       analyzerConfigs.VolumeAnalyzer.Enabled,
				MinConfidence: analyzerConfigs.VolumeAnalyzer.MinConfidence,
			},
			OpenInterestAnalyzer: AnalyzerConfig{
				Enabled:       analyzerConfigs.OpenInterestAnalyzer.Enabled,
				MinConfidence: analyzerConfigs.OpenInterestAnalyzer.MinConfidence,
			},
			CounterAnalyzer: AnalyzerConfig{
				Enabled: analyzerConfigs.CounterAnalyzer.Enabled,
			},
		},
		FilterConfigs: FilterConfigs{
			SignalFilters: SignalFilterConfig{
				Enabled:          cfg.SignalFilters.Enabled,
				MinConfidence:    cfg.SignalFilters.MinConfidence,
				MaxSignalsPerMin: cfg.SignalFilters.MaxSignalsPerMin,
			},
		},
	}

	engine := NewAnalysisEngine(storage, eventBus, engineConfig)
	f.configureAnalyzers(engine, cfg, notifier)
	f.configureFilters(engine, cfg)
	return engine
}

func getFloatFromCustomSettings(customSettings map[string]interface{}, key string, defaultValue float64) float64 {
	if customSettings == nil {
		return defaultValue
	}
	if val, ok := customSettings[key].(float64); ok {
		return val
	}
	if val, ok := customSettings[key].(int); ok {
		return float64(val)
	}
	return defaultValue
}

func getBoolFromCustomSettings(customSettings map[string]interface{}, key string, defaultValue bool) bool {
	if customSettings == nil {
		return defaultValue
	}
	if val, ok := customSettings[key].(bool); ok {
		return val
	}
	return defaultValue
}

func getStringFromCustomSettings(customSettings map[string]interface{}, key string, defaultValue string) string {
	if customSettings == nil {
		return defaultValue
	}
	if val, ok := customSettings[key].(string); ok {
		return val
	}
	return defaultValue
}

func getIntFromCustomSettings(customSettings map[string]interface{}, key string, defaultValue int) int {
	if customSettings == nil {
		return defaultValue
	}
	if val, ok := customSettings[key].(int); ok {
		return val
	}
	if val, ok := customSettings[key].(float64); ok {
		return int(val)
	}
	return defaultValue
}

func (f *Factory) configureAnalyzers(
	engine *AnalysisEngine,
	cfg *config.Config,
	notifier *notification.TelegramNotifier,
) {
	// minDataPoints := cfg.AnalysisEngine.MinDataPoints
	analyzerConfigs := cfg.AnalyzerConfigs

	// ОТКЛЮЧАЕМ АНАЛИЗАТОРЫ:
	// GrowthAnalyzer - ОТКЛЮЧЕН
	// FallAnalyzer - ОТКЛЮЧЕН
	// ContinuousAnalyzer - ОТКЛЮЧЕН
	// VolumeAnalyzer - ОТКЛЮЧЕН
	// OpenInterestAnalyzer - ОТКЛЮЧЕН

	// Оставляем только CounterAnalyzer если он включен
	if analyzerConfigs.CounterAnalyzer.Enabled {
		f.configureCounterAnalyzer(engine, cfg, notifier)
	}

	log.Printf("ℹ️ Анализаторы отключены через фабрику: Growth, Fall, Continuous, Volume, OpenInterest")
	log.Printf("ℹ️ Активные анализаторы: %s", func() string {
		if analyzerConfigs.CounterAnalyzer.Enabled {
			return "CounterAnalyzer"
		}
		return "нет"
	}())
}

func (f *Factory) configureCounterAnalyzer(
	engine *AnalysisEngine,
	cfg *config.Config,
	notifier *notification.TelegramNotifier,
) {
	log.Println("🔧 Настройка CounterAnalyzer с TelegramNotifier...")
	analyzerConfigs := cfg.AnalyzerConfigs
	customSettings := analyzerConfigs.CounterAnalyzer.CustomSettings

	counterConfig := common.AnalyzerConfig{
		Enabled:       true,
		Weight:        0.7,
		MinConfidence: 10.0,
		MinDataPoints: 2,
		CustomSettings: map[string]interface{}{
			"base_period_minutes":    getIntFromCustomSettings(customSettings, "base_period_minutes", 1),
			"analysis_period":        getStringFromCustomSettings(customSettings, "analysis_period", "15m"),
			"growth_threshold":       getFloatFromCustomSettings(customSettings, "growth_threshold", 0.1),
			"fall_threshold":         getFloatFromCustomSettings(customSettings, "fall_threshold", 0.1),
			"track_growth":           getBoolFromCustomSettings(customSettings, "track_growth", true),
			"track_fall":             getBoolFromCustomSettings(customSettings, "track_fall", true),
			"notify_on_signal":       getBoolFromCustomSettings(customSettings, "notify_on_signal", true),
			"notification_threshold": getIntFromCustomSettings(customSettings, "notification_threshold", 1),
			"chart_provider":         getStringFromCustomSettings(customSettings, "chart_provider", "coinglass"),
			"max_signals_5m":         getIntFromCustomSettings(customSettings, "max_signals_5m", 5),
			"max_signals_15m":        getIntFromCustomSettings(customSettings, "max_signals_15m", 8),
			"max_signals_30m":        getIntFromCustomSettings(customSettings, "max_signals_30m", 10),
			"max_signals_1h":         getIntFromCustomSettings(customSettings, "max_signals_1h", 12),
			"max_signals_4h":         getIntFromCustomSettings(customSettings, "max_signals_4h", 15),
			"max_signals_1d":         getIntFromCustomSettings(customSettings, "max_signals_1d", 20),
		},
	}

	storage := engine.GetStorage()
	counterAnalyzer := counter.NewCounterAnalyzer(counterConfig, storage, engine.eventBus, f.priceFetcher)

	if err := engine.RegisterAnalyzer(counterAnalyzer); err != nil {
		log.Printf("⚠️ Не удалось зарегистрировать CounterAnalyzer: %v", err)
	} else {
		log.Printf("✅ CounterAnalyzer успешно добавлен в AnalysisEngine")
		log.Printf("   TelegramNotifier: %v", notifier != nil)
		log.Printf("   Storage: %v", storage != nil)
		log.Printf("   MarketFetcher: %v", f.priceFetcher != nil)
	}
}

func (f *Factory) configureFilters(engine *AnalysisEngine, cfg *config.Config) {
	if cfg.SignalFilters.Enabled && cfg.SignalFilters.MinConfidence > 0 {
		confidenceFilter := filters.NewConfidenceFilter(cfg.SignalFilters.MinConfidence)
		engine.AddFilter(confidenceFilter)
	}
	if cfg.MinVolumeFilter > 0 {
		volumeFilter := filters.NewVolumeFilter(cfg.MinVolumeFilter)
		engine.AddFilter(volumeFilter)
	}
	if cfg.SignalFilters.Enabled && cfg.SignalFilters.MaxSignalsPerMin > 0 {
		minDelay := time.Minute / time.Duration(cfg.SignalFilters.MaxSignalsPerMin)
		rateLimitFilter := filters.NewRateLimitFilter(minDelay)
		engine.AddFilter(rateLimitFilter)
	}
}

func (e *AnalysisEngine) GetStorage() storage.PriceStorage {
	return e.storage
}
