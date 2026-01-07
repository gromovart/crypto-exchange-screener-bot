// internal/core/domain/signals/detectors/growth_analyzer/config/config.go
package growth_analyzer

import "crypto-exchange-screener-bot/internal/core/domain/signals/detectors/common"

// DefaultGrowthConfig - конфигурация по умолчанию для анализатора роста
var DefaultGrowthConfig = common.AnalyzerConfig{
	Enabled:       true,
	Weight:        1.0,
	MinConfidence: 60.0,
	MinDataPoints: 3,
	CustomSettings: map[string]interface{}{
		"min_growth_percent":     2.0,
		"continuity_threshold":   0.7,
		"acceleration_threshold": 0.5,
		"volume_weight":          0.2,
		"trend_strength_weight":  0.4,
		"volatility_weight":      0.2,
	},
}

// NewGrowthConfig - создает новую конфигурацию с настройками по умолчанию
func NewGrowthConfig() common.AnalyzerConfig {
	// Создаем копию, чтобы избежать модификации оригинальной конфигурации
	config := DefaultGrowthConfig
	config.CustomSettings = make(map[string]interface{})

	for k, v := range DefaultGrowthConfig.CustomSettings {
		config.CustomSettings[k] = v
	}

	return config
}

// GetGrowthConfig - преобразует общую конфигурацию в конфигурацию роста
func GetGrowthConfig(config common.AnalyzerConfig) GrowthConfig {
	return GrowthConfig{
		AnalyzerConfig:        config,
		MinGrowthPercent:      getFloatSetting(config.CustomSettings, "min_growth_percent", 2.0),
		ContinuityThreshold:   getFloatSetting(config.CustomSettings, "continuity_threshold", 0.7),
		AccelerationThreshold: getFloatSetting(config.CustomSettings, "acceleration_threshold", 0.5),
		VolumeWeight:          getFloatSetting(config.CustomSettings, "volume_weight", 0.2),
		TrendStrengthWeight:   getFloatSetting(config.CustomSettings, "trend_strength_weight", 0.4),
		VolatilityWeight:      getFloatSetting(config.CustomSettings, "volatility_weight", 0.2),
	}
}

// getFloatSetting - вспомогательная функция для получения float настройки
func getFloatSetting(settings map[string]interface{}, key string, defaultValue float64) float64 {
	if value, ok := settings[key]; ok {
		if floatValue, ok := value.(float64); ok {
			return floatValue
		}
	}
	return defaultValue
}

// ValidateGrowthConfig - валидирует конфигурацию анализатора роста
func ValidateGrowthConfig(config common.AnalyzerConfig) error {
	if config.MinDataPoints < 2 {
		return ErrInvalidConfig("min_data_points must be at least 2")
	}

	if config.MinConfidence < 0 || config.MinConfidence > 100 {
		return ErrInvalidConfig("min_confidence must be between 0 and 100")
	}

	growthConfig := GetGrowthConfig(config)

	if growthConfig.MinGrowthPercent < 0 {
		return ErrInvalidConfig("min_growth_percent must be positive")
	}

	if growthConfig.ContinuityThreshold < 0 || growthConfig.ContinuityThreshold > 1 {
		return ErrInvalidConfig("continuity_threshold must be between 0 and 1")
	}

	if growthConfig.AccelerationThreshold < 0 || growthConfig.AccelerationThreshold > 1 {
		return ErrInvalidConfig("acceleration_threshold must be between 0 and 1")
	}

	// Проверяем веса
	totalWeight := growthConfig.VolumeWeight + growthConfig.TrendStrengthWeight + growthConfig.VolatilityWeight
	if totalWeight > 1.0 {
		return ErrInvalidConfig("sum of weights must not exceed 1.0")
	}

	return nil
}

// ErrInvalidConfig - ошибка невалидной конфигурации
type ErrInvalidConfig string

func (e ErrInvalidConfig) Error() string {
	return string(e)
}
