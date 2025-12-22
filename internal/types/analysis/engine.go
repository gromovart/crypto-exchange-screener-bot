// internal/types/analysis/engine.go
package analysis

import (
	"time"
)

// EngineConfig - конфигурация движка анализа
type EngineConfig struct {
	MaxConcurrentAnalyses int              `json:"max_concurrent_analyses"`
	AnalysisTimeout       time.Duration    `json:"analysis_timeout"`
	EnabledAnalyzers      []string         `json:"enabled_analyzers"`
	FilterConfig          FilterConfig     `json:"filter_config"`
	GlobalThresholds      GlobalThresholds `json:"global_thresholds"`
}

// FilterConfig - конфигурация фильтров
type FilterConfig struct {
	MinConfidence       float64 `json:"min_confidence"`
	MinVolume           float64 `json:"min_volume"`
	MaxSignalsPerMinute int     `json:"max_signals_per_minute"`
	EnableRateLimiting  bool    `json:"enable_rate_limiting"`
}

// GlobalThresholds - глобальные пороги
type GlobalThresholds struct {
	GrowthThreshold   float64 `json:"growth_threshold"`
	FallThreshold     float64 `json:"fall_threshold"`
	VolumeSpikeFactor float64 `json:"volume_spike_factor"`
	MinDataPoints     int     `json:"min_data_points"`
}

// EngineStats - статистика движка
type EngineStats struct {
	TotalAnalyses       int           `json:"total_analyses"`
	TotalSignals        int           `json:"total_signals"`
	FilteredSignals     int           `json:"filtered_signals"`
	AverageAnalysisTime time.Duration `json:"average_analysis_time"`
	ActiveAnalyses      int           `json:"active_analyses"`
	LastAnalysisTime    time.Time     `json:"last_analysis_time"`
}
