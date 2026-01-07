// internal/core/domain/signals/detectors/continuous_analyzer/config/config.go
package config

import (
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/common"
)

// ConfigManager управляет конфигурацией ContinuousAnalyzer
type ConfigManager struct {
	baseConfig common.AnalyzerConfig
}

// NewConfigManager создает новый менеджер конфигурации
func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		baseConfig: DefaultContinuousConfig(),
	}
}

// GetConfig возвращает текущую конфигурацию
func (cm *ConfigManager) GetConfig() common.AnalyzerConfig {
	return cm.baseConfig
}

// UpdateConfig обновляет конфигурацию
func (cm *ConfigManager) UpdateConfig(config common.AnalyzerConfig) {
	cm.baseConfig = config
}

// GetMinContinuousPoints возвращает минимальное количество непрерывных точек
func (cm *ConfigManager) GetMinContinuousPoints() int {
	if minPoints, ok := cm.baseConfig.CustomSettings["min_continuous_points"].(int); ok {
		return minPoints
	}
	if minPoints, ok := cm.baseConfig.CustomSettings["min_continuous_points"].(float64); ok {
		return int(minPoints)
	}
	return 3
}

// GetMaxGapRatio возвращает максимальный допустимый gap
func (cm *ConfigManager) GetMaxGapRatio() float64 {
	if maxGap, ok := cm.baseConfig.CustomSettings["max_gap_ratio"].(float64); ok {
		return maxGap
	}
	return 0.3
}

// GetRequireConfirmation возвращает требование подтверждения
func (cm *ConfigManager) GetRequireConfirmation() bool {
	if require, ok := cm.baseConfig.CustomSettings["require_confirmation"].(bool); ok {
		return require
	}
	return true
}

// IsAlgorithmEnabled проверяет включен ли алгоритм
func (cm *ConfigManager) IsAlgorithmEnabled(algorithmType string) bool {
	if enabled, ok := cm.baseConfig.CustomSettings[algorithmType+"_enabled"].(bool); ok {
		return enabled
	}
	return true
}

// SetAlgorithmEnabled включает или выключает алгоритм
func (cm *ConfigManager) SetAlgorithmEnabled(algorithmType string, enabled bool) {
	if cm.baseConfig.CustomSettings == nil {
		cm.baseConfig.CustomSettings = make(map[string]interface{})
	}
	cm.baseConfig.CustomSettings[algorithmType+"_enabled"] = enabled
}

// ValidateConfig проверяет валидность конфигурации
func (cm *ConfigManager) ValidateConfig() error {
	if cm.baseConfig.MinDataPoints < 1 {
		return ErrInvalidConfig("MinDataPoints must be at least 1")
	}
	if cm.baseConfig.MinConfidence < 0 || cm.baseConfig.MinConfidence > 100 {
		return ErrInvalidConfig("MinConfidence must be between 0 and 100")
	}
	if cm.baseConfig.Weight < 0 || cm.baseConfig.Weight > 1 {
		return ErrInvalidConfig("Weight must be between 0 and 1")
	}

	minPoints := cm.GetMinContinuousPoints()
	if minPoints < 2 {
		return ErrInvalidConfig("min_continuous_points must be at least 2")
	}

	maxGap := cm.GetMaxGapRatio()
	if maxGap < 0 || maxGap > 1 {
		return ErrInvalidConfig("max_gap_ratio must be between 0 and 1")
	}

	return nil
}

// ErrInvalidConfig ошибка невалидной конфигурации
type ErrInvalidConfig string

func (e ErrInvalidConfig) Error() string {
	return string(e)
}
func DefaultContinuousConfig() common.AnalyzerConfig {
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
