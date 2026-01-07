// internal/core/domain/signals/detectors/volume_analyzer/config/config.go
package config

import (
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/common"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/volume_analyzer"
)

// ConfigManager управляет конфигурацией VolumeAnalyzer
type ConfigManager struct {
	baseConfig common.AnalyzerConfig
}

// NewConfigManager создает новый менеджер конфигурации
func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		baseConfig: volume_analyzer.DefaultVolumeConfig(),
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

// GetMinVolume возвращает минимальный объем из конфигурации
func (cm *ConfigManager) GetMinVolume() float64 {
	if minVolume, ok := cm.baseConfig.CustomSettings["min_volume"].(float64); ok {
		return minVolume
	}
	return 100000.0
}

// GetSpikeMultiplier возвращает множитель для всплеска объема
func (cm *ConfigManager) GetSpikeMultiplier() float64 {
	if multiplier, ok := cm.baseConfig.CustomSettings["spike_multiplier"].(float64); ok {
		return multiplier
	}
	return 3.0
}

// GetConfirmationThreshold возвращает порог для подтверждения
func (cm *ConfigManager) GetConfirmationThreshold() float64 {
	if threshold, ok := cm.baseConfig.CustomSettings["confirmation_threshold"].(float64); ok {
		return threshold
	}
	return 10.0
}

// GetVolumeChangeThreshold возвращает порог изменения объема
func (cm *ConfigManager) GetVolumeChangeThreshold() float64 {
	if threshold, ok := cm.baseConfig.CustomSettings["volume_change_threshold"].(float64); ok {
		return threshold
	}
	return 50.0
}

// IsAlgorithmEnabled проверяет включен ли алгоритм
func (cm *ConfigManager) IsAlgorithmEnabled(algorithmType string) bool {
	if enabled, ok := cm.baseConfig.CustomSettings[algorithmType+"_enabled"].(bool); ok {
		return enabled
	}
	return true // по умолчанию все алгоритмы включены
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

	minVolume := cm.GetMinVolume()
	if minVolume < 0 {
		return ErrInvalidConfig("min_volume must be positive")
	}

	return nil
}

// ErrInvalidConfig ошибка невалидной конфигурации
type ErrInvalidConfig string

func (e ErrInvalidConfig) Error() string {
	return string(e)
}
