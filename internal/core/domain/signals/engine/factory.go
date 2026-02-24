// internal/core/domain/signals/engine/factory.go
package engine

import (
	candle "crypto-exchange-screener-bot/internal/core/domain/candle"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/common"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/counter"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/counter/calculator"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	sr_storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage/sr_storage"
	events "crypto-exchange-screener-bot/internal/infrastructure/transport/event_bus"
	"crypto-exchange-screener-bot/pkg/logger"
	"log"
	"time"
)

type Factory struct {
	priceFetcher  interface{}
	candleSystem  *candle.CandleSystem
	srZoneStorage *sr_storage.SRZoneStorage
}

// NewFactory создает фабрику
func NewFactory(priceFetcher interface{}, candleSystem *candle.CandleSystem) *Factory {
	return &Factory{
		priceFetcher: priceFetcher,
		candleSystem: candleSystem,
	}
}

func (f *Factory) NewAnalysisEngineFromConfig(
	storage storage.PriceStorageInterface,
	eventBus *events.EventBus,
	cfg *config.Config,
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
			CounterAnalyzer: AnalyzerConfig{
				Enabled: analyzerConfigs.CounterAnalyzer.Enabled,
			},
		},
		// УДАЛЕНО: FilterConfigs - AnalysisEngine теперь только оркестратор
	}

	engine := NewAnalysisEngine(storage, eventBus, engineConfig)
	f.configureAnalyzers(engine, cfg)
	// УДАЛЕНО: f.configureFilters(engine, cfg) - фильтров больше нет
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
) {
	analyzerConfigs := cfg.AnalyzerConfigs

	// Оставляем только CounterAnalyzer если он включен
	if analyzerConfigs.CounterAnalyzer.Enabled {
		f.configureCounterAnalyzer(engine, cfg)
	}

	logger.Warn("ℹ️ Анализаторы отключены через фабрику: Growth, Fall, Continuous, Volume, OpenInterest")
	logger.Debug("ℹ️ Активные анализаторы: %s", func() string {
		if analyzerConfigs.CounterAnalyzer.Enabled {
			return "CounterAnalyzer"
		}
		return "нет"
	}())
}

func (f *Factory) configureCounterAnalyzer(
	engine *AnalysisEngine,
	cfg *config.Config,
) {
	logger.Info("🔧 Настройка CounterAnalyzer с CandleTracker...")
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

	// Здесь нужно получить RedisService для создания трекера
	// Пока создаем без трекера если не можем получить RedisService
	// В реальном коде нужно будет передать RedisService из зависимостей

	logger.Warn("⚠️ CandleTracker временно не используется, нужен RedisService")

	// Создаем зависимости
	deps := counter.Dependencies{
		Storage:          storage,
		EventBus:         engine.eventBus,
		CandleSystem:     f.candleSystem,
		MarketFetcher:    f.priceFetcher,
		VolumeCalculator: calculator.NewVolumeDeltaCalculator(f.priceFetcher, storage),
		SRZoneStorage:    f.srZoneStorage,
	}

	counterAnalyzer := counter.NewCounterAnalyzer(counterConfig, deps)

	if err := engine.RegisterAnalyzer(counterAnalyzer); err != nil {
		logger.Warn("⚠️ Не удалось зарегистрировать CounterAnalyzer: %v", err)
	} else {
		logger.Info("✅ CounterAnalyzer успешно добавлен в AnalysisEngine")
		logger.Info("   Storage: %v", storage != nil)
		logger.Info("   MarketFetcher: %v", f.priceFetcher != nil)
		logger.Info("   CandleSystem: %v", f.candleSystem != nil)
	}
}

// УДАЛЕНО: configureFilters метод - AnalysisEngine теперь только оркестратор

func (e *AnalysisEngine) GetStorage() storage.PriceStorageInterface {
	return e.storage
}

// SetCandleSystem устанавливает свечную систему (дополнительный метод)
func (f *Factory) SetCandleSystem(candleSystem *candle.CandleSystem) {
	f.candleSystem = candleSystem
	log.Printf("✅ Factory: свечная система установлена")
}

// SetSRZoneStorage устанавливает хранилище зон S/R для передачи в CounterAnalyzer
func (f *Factory) SetSRZoneStorage(storage *sr_storage.SRZoneStorage) {
	f.srZoneStorage = storage
}
