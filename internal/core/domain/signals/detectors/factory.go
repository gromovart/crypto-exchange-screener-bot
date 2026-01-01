// internal/core/domain/signals/detectors/factory.go
package analyzers

import (
	"fmt"
	"time"
)

// NewGrowthAnalyzer создает анализатор роста
func NewGrowthAnalyzer(config AnalyzerConfig) *GrowthAnalyzer {
	return &GrowthAnalyzer{
		config: config,
		stats: AnalyzerStats{
			TotalCalls:   0,
			TotalTime:    0,
			SuccessCount: 0,
			ErrorCount:   0,
			LastCallTime: time.Time{},
			AverageTime:  0,
		},
	}
}

// NewFallAnalyzer создает анализатор падения
func NewFallAnalyzer(config AnalyzerConfig) *FallAnalyzer {
	return &FallAnalyzer{
		config: config,
		stats: AnalyzerStats{
			TotalCalls:   0,
			TotalTime:    0,
			SuccessCount: 0,
			ErrorCount:   0,
			LastCallTime: time.Time{},
			AverageTime:  0,
		},
	}
}

// NewVolumeAnalyzer создает анализатор объема
func NewVolumeAnalyzer(config AnalyzerConfig) *VolumeAnalyzer {
	return &VolumeAnalyzer{
		config: config,
		stats: AnalyzerStats{
			TotalCalls:   0,
			TotalTime:    0,
			SuccessCount: 0,
			ErrorCount:   0,
			LastCallTime: time.Time{},
			AverageTime:  0,
		},
	}
}

// NewContinuousAnalyzer создает анализатор непрерывности
func NewContinuousAnalyzer(config AnalyzerConfig) *ContinuousAnalyzer {
	// Гарантируем наличие необходимых настроек
	if config.CustomSettings == nil {
		config.CustomSettings = make(map[string]interface{})
	}

	// Устанавливаем значения по умолчанию
	defaults := map[string]interface{}{
		"min_continuous_points": 3,
		"max_gap_ratio":         0.3,
	}

	for key, defaultValue := range defaults {
		if _, ok := config.CustomSettings[key]; !ok {
			config.CustomSettings[key] = defaultValue
		}
	}

	return &ContinuousAnalyzer{
		config: config,
		stats:  AnalyzerStats{},
	}
}

// NewOpenInterestAnalyzer создает анализатор открытого интереса
func NewOpenInterestAnalyzer(config AnalyzerConfig) *OpenInterestAnalyzer {
	// Гарантируем наличие необходимых настроек
	if config.CustomSettings == nil {
		config.CustomSettings = make(map[string]interface{})
	}

	// Устанавливаем значения по умолчанию из DefaultOpenInterestConfig
	defaults := map[string]interface{}{
		"min_price_change":      1.0,
		"min_price_fall":        1.0,
		"min_oi_change":         5.0,
		"extreme_oi_threshold":  1.5,
		"divergence_min_points": 4,
		"volume_weight":         0.3,
	}

	for key, defaultValue := range defaults {
		if _, ok := config.CustomSettings[key]; !ok {
			config.CustomSettings[key] = defaultValue
		}
	}

	return &OpenInterestAnalyzer{
		config: config,
		stats: AnalyzerStats{
			TotalCalls:   0,
			TotalTime:    0,
			SuccessCount: 0,
			ErrorCount:   0,
			LastCallTime: time.Time{},
			AverageTime:  0,
		},
	}
}

// AnalyzerFactory - фабрика для создания анализаторов (обновленная)
type AnalyzerFactory struct{}

// NewAnalyzerFactory создает новую фабрику анализаторов
func NewAnalyzerFactory() *AnalyzerFactory {
	return &AnalyzerFactory{}
}

// CreateAnalyzer создает анализатор по имени
func (f *AnalyzerFactory) CreateAnalyzer(name string, config AnalyzerConfig) Analyzer {
	switch name {
	case "growth_analyzer":
		return NewGrowthAnalyzer(config)
	case "fall_analyzer":
		return NewFallAnalyzer(config)
	case "continuous_analyzer":
		return NewContinuousAnalyzer(config)
	case "volume_analyzer":
		return NewVolumeAnalyzer(config)
	case "open_interest_analyzer":
		return NewOpenInterestAnalyzer(config)
	case "counter_analyzer":
		// CounterAnalyzer требует дополнительные параметры
		// Он должен создаваться отдельно через NewCounterAnalyzer
		// с передачей storage и telegramBot
		return nil
	default:
		// Возвращаем анализатор по умолчанию
		return NewGrowthAnalyzer(config)
	}
}

// GetAllAnalyzerConfigs возвращает конфигурации всех анализаторов
func GetAllAnalyzerConfigs() map[string]AnalyzerConfig {
	return map[string]AnalyzerConfig{
		"growth_analyzer":        DefaultGrowthConfig,
		"fall_analyzer":          DefaultFallConfig,
		"continuous_analyzer":    DefaultContinuousConfig,
		"volume_analyzer":        DefaultVolumeConfig,
		"open_interest_analyzer": DefaultOpenInterestConfig,
		"counter_analyzer":       DefaultCounterConfig,
	}
}

// GetAnalyzerNames возвращает список всех доступных анализаторов
func GetAnalyzerNames() []string {
	return []string{
		"growth_analyzer",
		"fall_analyzer",
		"continuous_analyzer",
		"volume_analyzer",
		"open_interest_analyzer",
		"counter_analyzer",
	}
}

// GetEnabledAnalyzers возвращает список включенных анализаторов на основе конфигурации
func GetEnabledAnalyzers(configs map[string]AnalyzerConfig) []string {
	var enabled []string
	for name, config := range configs {
		if config.Enabled {
			enabled = append(enabled, name)
		}
	}
	return enabled
}

// CreateAllEnabledAnalyzers создает все включенные анализаторы
func (f *AnalyzerFactory) CreateAllEnabledAnalyzers(configs map[string]AnalyzerConfig) map[string]Analyzer {
	analyzers := make(map[string]Analyzer)

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
func MergeConfigs(customConfigs map[string]AnalyzerConfig) map[string]AnalyzerConfig {
	defaultConfigs := GetAllAnalyzerConfigs()
	result := make(map[string]AnalyzerConfig)

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
func ValidateConfig(config AnalyzerConfig) error {
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
func GetDefaultConfig(analyzerName string) AnalyzerConfig {
	configs := GetAllAnalyzerConfigs()
	if config, exists := configs[analyzerName]; exists {
		return config
	}

	// Возвращаем конфигурацию по умолчанию для growth_analyzer
	return DefaultGrowthConfig
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
