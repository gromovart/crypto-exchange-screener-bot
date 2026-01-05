// internal/core/domain/signals/detectors/counter/config/config.go
package config

import (
	analyzers "crypto-exchange-screener-bot/internal/core/domain/signals/detectors"
)

// CounterConfig - конфигурация счетчика
type CounterConfig struct {
	Enabled       bool
	Weight        float64
	MinConfidence float64
	MinDataPoints int
	Settings      CounterSettings
}

// CounterSettings - настройки счетчика
type CounterSettings struct {
	BasePeriodMinutes     int
	AnalysisPeriod        string // Изменено с CounterPeriod на string
	GrowthThreshold       float64
	FallThreshold         float64
	TrackGrowth           bool
	TrackFall             bool
	NotifyOnSignal        bool
	NotificationThreshold int
	ChartProvider         string
	Exchange              string
	IncludeOI             bool
	IncludeVolume         bool
	IncludeFunding        bool
	VolumeDeltaTTL        int
	DeltaFallbackEnabled  bool
	ShowDeltaSource       bool
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() CounterConfig {
	return CounterConfig{
		Enabled:       true,
		Weight:        0.7,
		MinConfidence: 10.0,
		MinDataPoints: 2,
		Settings: CounterSettings{
			BasePeriodMinutes:     1,
			AnalysisPeriod:        "15m", // Строка вместо CounterPeriod
			GrowthThreshold:       0.1,
			FallThreshold:         0.1,
			TrackGrowth:           true,
			TrackFall:             true,
			NotifyOnSignal:        true,
			NotificationThreshold: 1,
			ChartProvider:         "coinglass",
			Exchange:              "bybit",
			IncludeOI:             true,
			IncludeVolume:         true,
			IncludeFunding:        true,
			VolumeDeltaTTL:        30,
			DeltaFallbackEnabled:  true,
			ShowDeltaSource:       true,
		},
	}
}

// ToAnalyzerConfig преобразует в формат анализатора
func (c *CounterConfig) ToAnalyzerConfig() analyzers.AnalyzerConfig {
	return analyzers.AnalyzerConfig{
		Enabled:       c.Enabled,
		Weight:        c.Weight,
		MinConfidence: c.MinConfidence,
		MinDataPoints: c.MinDataPoints,
		CustomSettings: map[string]interface{}{
			"base_period_minutes":    c.Settings.BasePeriodMinutes,
			"analysis_period":        c.Settings.AnalysisPeriod,
			"growth_threshold":       c.Settings.GrowthThreshold,
			"fall_threshold":         c.Settings.FallThreshold,
			"track_growth":           c.Settings.TrackGrowth,
			"track_fall":             c.Settings.TrackFall,
			"notify_on_signal":       c.Settings.NotifyOnSignal,
			"notification_threshold": c.Settings.NotificationThreshold,
			"chart_provider":         c.Settings.ChartProvider,
			"exchange":               c.Settings.Exchange,
			"include_oi":             c.Settings.IncludeOI,
			"include_volume":         c.Settings.IncludeVolume,
			"include_funding":        c.Settings.IncludeFunding,
			"volume_delta_ttl":       c.Settings.VolumeDeltaTTL,
			"delta_fallback_enabled": c.Settings.DeltaFallbackEnabled,
			"show_delta_source":      c.Settings.ShowDeltaSource,
		},
	}
}

// FromAnalyzerConfig создает конфигурацию из формата анализатора
func FromAnalyzerConfig(analyzerConfig analyzers.AnalyzerConfig) CounterConfig {
	custom := analyzerConfig.CustomSettings
	if custom == nil {
		custom = make(map[string]interface{})
	}

	return CounterConfig{
		Enabled:       analyzerConfig.Enabled,
		Weight:        analyzerConfig.Weight,
		MinConfidence: analyzerConfig.MinConfidence,
		MinDataPoints: analyzerConfig.MinDataPoints,
		Settings: CounterSettings{
			BasePeriodMinutes:     getInt(custom, "base_period_minutes", 1),
			AnalysisPeriod:        getString(custom, "analysis_period", "15m"),
			GrowthThreshold:       getFloat(custom, "growth_threshold", 0.1),
			FallThreshold:         getFloat(custom, "fall_threshold", 0.1),
			TrackGrowth:           getBool(custom, "track_growth", true),
			TrackFall:             getBool(custom, "track_fall", true),
			NotifyOnSignal:        getBool(custom, "notify_on_signal", true),
			NotificationThreshold: getInt(custom, "notification_threshold", 1),
			ChartProvider:         getString(custom, "chart_provider", "coinglass"),
			Exchange:              getString(custom, "exchange", "bybit"),
			IncludeOI:             getBool(custom, "include_oi", true),
			IncludeVolume:         getBool(custom, "include_volume", true),
			IncludeFunding:        getBool(custom, "include_funding", true),
			VolumeDeltaTTL:        getInt(custom, "volume_delta_ttl", 30),
			DeltaFallbackEnabled:  getBool(custom, "delta_fallback_enabled", true),
			ShowDeltaSource:       getBool(custom, "show_delta_source", true),
		},
	}
}

// Вспомогательные функции
func getInt(m map[string]interface{}, key string, defaultValue int) int {
	if val, ok := m[key]; ok {
		if v, ok := val.(int); ok {
			return v
		}
		if v, ok := val.(float64); ok {
			return int(v)
		}
	}
	return defaultValue
}

func getFloat(m map[string]interface{}, key string, defaultValue float64) float64 {
	if val, ok := m[key]; ok {
		if v, ok := val.(float64); ok {
			return v
		}
		if v, ok := val.(int); ok {
			return float64(v)
		}
	}
	return defaultValue
}

func getBool(m map[string]interface{}, key string, defaultValue bool) bool {
	if val, ok := m[key]; ok {
		if v, ok := val.(bool); ok {
			return v
		}
		if v, ok := val.(string); ok {
			return v == "true"
		}
	}
	return defaultValue
}

func getString(m map[string]interface{}, key string, defaultValue string) string {
	if val, ok := m[key]; ok {
		if v, ok := val.(string); ok {
			return v
		}
	}
	return defaultValue
}
