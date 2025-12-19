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
	return &ContinuousAnalyzer{
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
