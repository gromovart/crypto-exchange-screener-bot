// internal/analysis/analyzers/factory.go
package analyzers

import "time"

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
