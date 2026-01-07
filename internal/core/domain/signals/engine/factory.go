// internal/core/domain/signals/engine/factory.go
package engine

import (
	analyzers "crypto-exchange-screener-bot/internal/core/domain/signals/detectors"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/common"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/counter"
	"crypto-exchange-screener-bot/internal/core/domain/signals/filters"
	"crypto-exchange-screener-bot/internal/delivery/telegram"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/in_memory_storage"
	events "crypto-exchange-screener-bot/internal/infrastructure/transport/event_bus"
	"log"
	"time"
)

// Factory - фабрика для создания AnalysisEngine
type Factory struct {
	priceFetcher interface{} // 🔴 ДОБАВЛЯЕМ ПОЛЕ ДЛЯ РЫНОЧНЫХ ДАННЫХ
}

// NewFactory создает фабрику с доступом к marketFetcher
func NewFactory(priceFetcher interface{}) *Factory {
	return &Factory{
		priceFetcher: priceFetcher,
	}
}

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

	// Получаем конфигурации анализаторов из новой структуры
	analyzerConfigs := cfg.AnalyzerConfigs

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

// getFloatFromCustomSettings получает float значение из CustomSettings
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

// getBoolFromCustomSettings получает bool значение из CustomSettings
func getBoolFromCustomSettings(customSettings map[string]interface{}, key string, defaultValue bool) bool {
	if customSettings == nil {
		return defaultValue
	}
	if val, ok := customSettings[key].(bool); ok {
		return val
	}
	return defaultValue
}

// getStringFromCustomSettings получает string значение из CustomSettings
func getStringFromCustomSettings(customSettings map[string]interface{}, key string, defaultValue string) string {
	if customSettings == nil {
		return defaultValue
	}
	if val, ok := customSettings[key].(string); ok {
		return val
	}
	return defaultValue
}

// getIntFromCustomSettings получает int значение из CustomSettings
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

// configureAnalyzers настраивает анализаторы
func (f *Factory) configureAnalyzers(
	engine *AnalysisEngine,
	cfg *config.Config,
	telegramBot *telegram.TelegramBot,
) {
	minDataPoints := cfg.AnalysisEngine.MinDataPoints
	analyzerConfigs := cfg.AnalyzerConfigs

	// Настраиваем GrowthAnalyzer
	if analyzerConfigs.GrowthAnalyzer.Enabled {
		growthConfig := common.AnalyzerConfig{
			Enabled:       true,
			Weight:        1.0,
			MinConfidence: analyzerConfigs.GrowthAnalyzer.MinConfidence,
			MinDataPoints: minDataPoints,
			CustomSettings: map[string]interface{}{
				"min_growth":           analyzerConfigs.GrowthAnalyzer.MinGrowth,
				"continuity_threshold": getFloatFromCustomSettings(analyzerConfigs.GrowthAnalyzer.CustomSettings, "continuity_threshold", 0.7),
				"volume_weight":        0.2,
			},
		}

		growthAnalyzer := analyzers.NewGrowthAnalyzer(growthConfig)
		engine.RegisterAnalyzer(growthAnalyzer)
	}

	// Настраиваем FallAnalyzer
	if analyzerConfigs.FallAnalyzer.Enabled {
		fallConfig := common.AnalyzerConfig{
			Enabled:       true,
			Weight:        1.0,
			MinConfidence: analyzerConfigs.FallAnalyzer.MinConfidence,
			MinDataPoints: minDataPoints,
			CustomSettings: map[string]interface{}{
				"min_fall":             analyzerConfigs.FallAnalyzer.MinFall,
				"continuity_threshold": getFloatFromCustomSettings(analyzerConfigs.FallAnalyzer.CustomSettings, "continuity_threshold", 0.7),
				"volume_weight":        0.2,
			},
		}

		fallAnalyzer := analyzers.NewFallAnalyzer(fallConfig)
		engine.RegisterAnalyzer(fallAnalyzer)
	}

	// VolumeAnalyzer
	if analyzerConfigs.VolumeAnalyzer.Enabled {
		volumeConfig := analyzers.DefaultVolumeConfig
		volumeConfig.MinDataPoints = minDataPoints
		volumeConfig.MinConfidence = analyzerConfigs.VolumeAnalyzer.MinConfidence

		// Получаем min_volume из кастомных настроек
		if minVolume := getFloatFromCustomSettings(analyzerConfigs.VolumeAnalyzer.CustomSettings, "min_volume", 100000.0); minVolume > 0 {
			volumeConfig.CustomSettings["min_volume"] = minVolume
		}

		volumeAnalyzer := analyzers.NewVolumeAnalyzer(volumeConfig)
		engine.RegisterAnalyzer(volumeAnalyzer)
		log.Printf("✅ VolumeAnalyzer настроен (мин. уверенность: %.0f%%)", analyzerConfigs.VolumeAnalyzer.MinConfidence)
	}

	// ContinuousAnalyzer
	if analyzerConfigs.ContinuousAnalyzer.Enabled {
		continuousConfig := common.AnalyzerConfig{
			Enabled:       true,
			Weight:        0.8,
			MinConfidence: analyzerConfigs.GrowthAnalyzer.MinConfidence,
			MinDataPoints: minDataPoints,
			CustomSettings: map[string]interface{}{
				"min_continuous_points": getIntFromCustomSettings(analyzerConfigs.ContinuousAnalyzer.CustomSettings, "min_continuous_points", 3),
				"max_gap_ratio":         0.3,
			},
		}

		continuousAnalyzer := analyzers.NewContinuousAnalyzer(continuousConfig)
		engine.RegisterAnalyzer(continuousAnalyzer)
		log.Printf("✅ ContinuousAnalyzer настроен")
	}

	// OpenInterestAnalyzer
	if analyzerConfigs.OpenInterestAnalyzer.Enabled {
		openInterestConfig := analyzers.DefaultOpenInterestConfig
		openInterestConfig.MinDataPoints = minDataPoints
		openInterestConfig.MinConfidence = analyzerConfigs.OpenInterestAnalyzer.MinConfidence

		// Добавляем пользовательские настройки из CustomSettings
		customSettings := analyzerConfigs.OpenInterestAnalyzer.CustomSettings
		if customSettings != nil {
			if minPriceChange := getFloatFromCustomSettings(customSettings, "min_price_change", 1.0); minPriceChange > 0 {
				openInterestConfig.CustomSettings["min_price_change"] = minPriceChange
			}
			if minPriceFall := getFloatFromCustomSettings(customSettings, "min_price_fall", 1.0); minPriceFall > 0 {
				openInterestConfig.CustomSettings["min_price_fall"] = minPriceFall
			}
			if minOIChange := getFloatFromCustomSettings(customSettings, "min_oi_change", 5.0); minOIChange > 0 {
				openInterestConfig.CustomSettings["min_oi_change"] = minOIChange
			}
			if extremeOIThreshold := getFloatFromCustomSettings(customSettings, "extreme_oi_threshold", 1.5); extremeOIThreshold > 0 {
				openInterestConfig.CustomSettings["extreme_oi_threshold"] = extremeOIThreshold
			}
			if analyzerWeight := getFloatFromCustomSettings(customSettings, "analyzer_weight", 0.6); analyzerWeight > 0 {
				openInterestConfig.CustomSettings["analyzer_weight"] = analyzerWeight
			}
		}

		openInterestAnalyzer := analyzers.NewOpenInterestAnalyzer(openInterestConfig)
		engine.RegisterAnalyzer(openInterestAnalyzer)
		log.Printf("✅ OpenInterestAnalyzer настроен")
	}

	// CounterAnalyzer если включен
	if analyzerConfigs.CounterAnalyzer.Enabled {
		f.configureCounterAnalyzer(engine, cfg, telegramBot)
	}
}

// configureCounterAnalyzer настраивает CounterAnalyzer
func (f *Factory) configureCounterAnalyzer(
	engine *AnalysisEngine,
	cfg *config.Config,
	telegramBot *telegram.TelegramBot,
) {
	log.Println("🔧 Настройка CounterAnalyzer с переданным Telegram ботом И marketFetcher...")

	// Получаем настройки CounterAnalyzer из конфигурации
	analyzerConfigs := cfg.AnalyzerConfigs
	customSettings := analyzerConfigs.CounterAnalyzer.CustomSettings

	// Настройки CounterAnalyzer из конфигурации
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

	// Создаем CounterAnalyzer с переданным ботом И marketFetcher
	storage := engine.GetStorage()

	// 🔴 ИСПРАВЛЕНИЕ: Используем priceFetcher из фабрики
	log.Printf("✅ Фабрика имеет доступ к marketFetcher: %v", f.priceFetcher != nil)

	counterAnalyzer := counter.NewCounterAnalyzer(counterConfig, storage, telegramBot, f.priceFetcher)

	// Регистрируем анализатор
	if err := engine.RegisterAnalyzer(counterAnalyzer); err != nil {
		log.Printf("⚠️ Не удалось зарегистрировать CounterAnalyzer: %v", err)
	} else {
		log.Printf("✅ CounterAnalyzer успешно добавлен в AnalysisEngine")
		log.Printf("   Telegram бот: %v", telegramBot != nil)
		log.Printf("   Storage: %v", storage != nil)
		log.Printf("   MarketFetcher: %v", f.priceFetcher != nil) // 🔴 Теперь показываем статус
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

// GetStorage возвращает хранилище из движка
func (e *AnalysisEngine) GetStorage() storage.PriceStorage {
	return e.storage
}
