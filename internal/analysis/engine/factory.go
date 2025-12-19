// internal/analysis/engine/factory.go
package engine

import (
	"crypto-exchange-screener-bot/internal/analysis/analyzers"
	"crypto-exchange-screener-bot/internal/analysis/filters"
	"crypto-exchange-screener-bot/internal/config"
	"crypto-exchange-screener-bot/internal/events"
	"crypto-exchange-screener-bot/internal/storage"
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
		UpdateInterval:   time.Duration(cfg.UpdateInterval) * time.Second,
		AnalysisPeriods:  periods,
		MinVolumeFilter:  cfg.AnalysisEngine.MinVolumeFilter,
		MaxSymbolsPerRun: cfg.AnalysisEngine.MaxSymbolsPerRun,
		EnableParallel:   cfg.AnalysisEngine.EnableParallel,
		MaxWorkers:       cfg.AnalysisEngine.MaxWorkers,
		SignalThreshold:  cfg.AnalysisEngine.SignalThreshold,
		RetentionPeriod:  cfg.AnalysisEngine.RetentionPeriod,
		EnableCache:      cfg.AnalysisEngine.EnableCache,
		MinDataPoints:    cfg.AnalysisEngine.MinDataPoints,

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

	return engine
}

// configureAnalyzers настраивает анализаторы
func (f *Factory) configureAnalyzers(engine *AnalysisEngine, cfg *config.Config) {
	// Настраиваем GrowthAnalyzer
	growthConfig := analyzers.AnalyzerConfig{
		Enabled:       cfg.Analyzers.GrowthAnalyzer.Enabled,
		Weight:        1.0,
		MinConfidence: cfg.Analyzers.GrowthAnalyzer.MinConfidence,
		MinDataPoints: cfg.AnalysisEngine.MinDataPoints,
		CustomSettings: map[string]interface{}{
			"min_growth":           cfg.Analyzers.GrowthAnalyzer.MinGrowth,
			"continuity_threshold": cfg.Analyzers.GrowthAnalyzer.ContinuityThreshold,
			"volume_weight":        0.2,
		},
	}

	growthAnalyzer := analyzers.NewGrowthAnalyzer(growthConfig)
	engine.RegisterAnalyzer(growthAnalyzer)

	// Настраиваем FallAnalyzer
	fallConfig := analyzers.AnalyzerConfig{
		Enabled:       cfg.Analyzers.FallAnalyzer.Enabled,
		Weight:        1.0,
		MinConfidence: cfg.Analyzers.FallAnalyzer.MinConfidence,
		MinDataPoints: cfg.AnalysisEngine.MinDataPoints,
		CustomSettings: map[string]interface{}{
			"min_fall":             cfg.Analyzers.FallAnalyzer.MinFall,
			"continuity_threshold": cfg.Analyzers.FallAnalyzer.ContinuityThreshold,
			"volume_weight":        0.2,
		},
	}

	fallAnalyzer := analyzers.NewFallAnalyzer(fallConfig)
	engine.RegisterAnalyzer(fallAnalyzer)

	// Добавляем ContinuousAnalyzer если включена проверка непрерывности
	if cfg.Analyzers.ContinuousAnalyzer.Enabled {
		continuousConfig := analyzers.AnalyzerConfig{
			Enabled:       true,
			Weight:        0.8,
			MinConfidence: cfg.Analyzers.GrowthAnalyzer.MinConfidence, // Используем тот же MinConfidence
			MinDataPoints: cfg.AnalysisEngine.MinDataPoints,
			CustomSettings: map[string]interface{}{
				"min_continuous_points": cfg.Analyzers.ContinuousAnalyzer.MinContinuousPoints,
				"max_gap_ratio":         0.3,
			},
		}

		continuousAnalyzer := analyzers.NewContinuousAnalyzer(continuousConfig)
		engine.RegisterAnalyzer(continuousAnalyzer)
	}
}

// configureFilters настраивает фильтры
func (f *Factory) configureFilters(engine *AnalysisEngine, cfg *config.Config) {
	// ConfidenceFilter
	confidenceFilter := filters.NewConfidenceFilter(cfg.SignalFilters.MinConfidence)
	engine.AddFilter(confidenceFilter)

	// VolumeFilter
	volumeFilter := filters.NewVolumeFilter(cfg.MinVolumeFilter)
	engine.AddFilter(volumeFilter)

	// RateLimitFilter
	if cfg.SignalFilters.MaxSignalsPerMin > 0 {
		minDelay := time.Minute / time.Duration(cfg.SignalFilters.MaxSignalsPerMin)
		rateLimitFilter := filters.NewRateLimitFilter(minDelay)
		engine.AddFilter(rateLimitFilter)
	}
}
