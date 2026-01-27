// internal/core/domain/signals/detectors/analyzers/factory.go
package analyzers

import (
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/common"
	"fmt"
)

// AnalyzerFactory - фабрика для создания анализаторов (обновленная)
type AnalyzerFactory struct{}

// NewAnalyzerFactory создает новую фабрику анализаторов
func NewAnalyzerFactory() *AnalyzerFactory {
	return &AnalyzerFactory{}
}

// CreateAnalyzer создает анализатор по имени
func (f *AnalyzerFactory) CreateAnalyzer(name string, config common.AnalyzerConfig) common.Analyzer {
	switch name {
	case "counter_analyzer":
		// CounterAnalyzer требует дополнительные параметры
		// Он должен создаваться отдельно через NewCounterAnalyzer
		// с передачей storage и telegramBot
		return nil
	default:
		// Возвращаем анализатор по умолчанию
		return nil
	}
}

// GetAllcommon.AnalyzerConfigs возвращает конфигурации всех анализаторов
func GetAllAnalyzerConfigs() map[string]common.AnalyzerConfig {
	return map[string]common.AnalyzerConfig{

		"counter_analyzer": DefaultCounterConfig,
	}
}

// Вспомогательные функции для получения конфигураций
func getContinuousConfig() common.AnalyzerConfig {
	return common.AnalyzerConfig{
		Enabled:       true,
		Weight:        0.8,
		MinConfidence: 70.0,
		MinDataPoints: 4,
		CustomSettings: map[string]interface{}{
			"min_continuous_points": 3,
			"max_gap_ratio":         0.3,
			"require_confirmation":  true,
		},
	}
}

// GetAnalyzerNames возвращает список всех доступных анализаторов
func GetAnalyzerNames() []string {
	return []string{
		"counter_analyzer",
	}
}

// GetEnabledAnalyzers возвращает список включенных анализаторов на основе конфигурации
func GetEnabledAnalyzers(configs map[string]common.AnalyzerConfig) []string {
	var enabled []string
	for name, config := range configs {
		if config.Enabled {
			enabled = append(enabled, name)
		}
	}
	return enabled
}

// CreateAllEnabledAnalyzers создает все включенные анализаторы
func (f *AnalyzerFactory) CreateAllEnabledAnalyzers(configs map[string]common.AnalyzerConfig) map[string]common.Analyzer {
	analyzers := make(map[string]common.Analyzer)

	for name, config := range configs {
		if config.Enabled {
			analyzer := f.CreateAnalyzer(name, config)
			if analyzer != nil {
				analyzers[name] = analyzer
			}
		}
	}

	return analyzers
}

// MergeConfigs объединяет пользовательские настройки с настройками по умолчанию
func MergeConfigs(customConfigs map[string]common.AnalyzerConfig) map[string]common.AnalyzerConfig {
	defaultConfigs := GetAllAnalyzerConfigs()
	result := make(map[string]common.AnalyzerConfig)

	for name, defaultConfig := range defaultConfigs {
		if customConfig, exists := customConfigs[name]; exists {
			// Объединяем настройки
			mergedConfig := defaultConfig
			mergedConfig.Enabled = customConfig.Enabled
			mergedConfig.Weight = customConfig.Weight
			mergedConfig.MinConfidence = customConfig.MinConfidence
			mergedConfig.MinDataPoints = customConfig.MinDataPoints

			// Объединяем CustomSettings
			if mergedConfig.CustomSettings == nil {
				mergedConfig.CustomSettings = make(map[string]interface{})
			}
			if customConfig.CustomSettings != nil {
				for key, value := range customConfig.CustomSettings {
					mergedConfig.CustomSettings[key] = value
				}
			}

			result[name] = mergedConfig
		} else {
			result[name] = defaultConfig
		}
	}

	return result
}

// ValidateConfig проверяет корректность конфигурации анализатора
func ValidateConfig(config common.AnalyzerConfig) error {
	if config.Weight < 0 || config.Weight > 1 {
		return fmt.Errorf("weight must be between 0 and 1")
	}

	if config.MinConfidence < 0 || config.MinConfidence > 100 {
		return fmt.Errorf("min_confidence must be between 0 and 100")
	}

	if config.MinDataPoints < 1 {
		return fmt.Errorf("min_data_points must be at least 1")
	}

	return nil
}

// GetDefaultConfig возвращает конфигурацию по умолчанию для анализатора
func GetDefaultConfig(analyzerName string) common.AnalyzerConfig {
	configs := GetAllAnalyzerConfigs()
	if config, exists := configs[analyzerName]; exists {
		return config
	}

	// Возвращаем конфигурацию по умолчанию для growth_analyzer
	return DefaultCounterConfig
}

// IsAnalyzerAvailable проверяет, доступен ли анализатор
func IsAnalyzerAvailable(analyzerName string) bool {
	availableAnalyzers := GetAnalyzerNames()
	for _, name := range availableAnalyzers {
		if name == analyzerName {
			return true
		}
	}
	return false
}

// DefaultCounterConfig - конфигурация по умолчанию для CounterAnalyzer
var DefaultCounterConfig = common.AnalyzerConfig{
	Enabled:       true,
	Weight:        0.7,
	MinConfidence: 10.0,
	MinDataPoints: 2,
	CustomSettings: map[string]interface{}{
		"base_period_minutes":    1,
		"analysis_period":        "15m",
		"growth_threshold":       0.1,
		"fall_threshold":         0.1,
		"track_growth":           true,
		"track_fall":             true,
		"notify_on_signal":       true,
		"notification_threshold": 1,
		"chart_provider":         "coinglass",
		"exchange":               "bybit",
		"include_oi":             true,
		"include_volume":         true,
		"include_funding":        true,
		"volume_delta_ttl":       30,
		"delta_fallback_enabled": true,
		"show_delta_source":      true,
	},
}
