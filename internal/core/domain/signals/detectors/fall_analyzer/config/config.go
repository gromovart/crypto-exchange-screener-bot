// internal/core/domain/signals/detectors/fall_analyzer/config/config.go
package config

import (
	"fmt"
)

// FallConfig - конфигурация анализатора падений
type FallConfig struct {
	Enabled             bool
	Weight              float64
	MinConfidence       float64
	MinDataPoints       int
	MinFall             float64 // минимальное падение для сигнала (%)
	ContinuityThreshold float64 // порог непрерывности (0-1)
	VolumeWeight        float64 // вес объема в расчетах
	CheckAllAlgorithms  bool    // проверять все алгоритмы
}

// FallAlgorithm - алгоритм анализа падений
type FallAlgorithm string

const (
	AlgorithmSingleFall     FallAlgorithm = "single_fall"
	AlgorithmIntervalFall   FallAlgorithm = "interval_fall"
	AlgorithmContinuousFall FallAlgorithm = "continuous_fall"
	AlgorithmAllFall        FallAlgorithm = "all"
)

// DefaultConfig - конфигурация по умолчанию для FallAnalyzer
var DefaultConfig = FallConfig{
	Enabled:             true,
	Weight:              1.0,
	MinConfidence:       60.0,
	MinDataPoints:       3,
	MinFall:             2.0,  // минимальное падение для сигнала (%)
	ContinuityThreshold: 0.7,  // порог непрерывности (0-1)
	VolumeWeight:        0.2,  // вес объема в расчетах
	CheckAllAlgorithms:  true, // проверять все алгоритмы
}

// ConfigManager - менеджер конфигурации FallAnalyzer
type ConfigManager struct {
	config FallConfig
}

// NewConfigManager создает новый менеджер конфигурации
func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		config: DefaultConfig,
	}
}

// WithCustomConfig настраивает кастомную конфигурацию
func (cm *ConfigManager) WithCustomConfig(custom map[string]interface{}) *ConfigManager {
	for key, value := range custom {
		cm.applySetting(key, value)
	}
	return cm
}

// GetConfig возвращает текущую конфигурацию
func (cm *ConfigManager) GetConfig() FallConfig {
	return cm.config
}

// UpdateConfig обновляет конфигурацию
func (cm *ConfigManager) UpdateConfig(newConfig FallConfig) {
	cm.config = newConfig
}

// IsAlgorithmEnabled проверяет, включен ли алгоритм
func (cm *ConfigManager) IsAlgorithmEnabled(algorithm FallAlgorithm) bool {
	if !cm.config.Enabled {
		return false
	}

	if cm.config.CheckAllAlgorithms {
		return true
	}

	// Здесь можно добавить логику выборочного включения алгоритмов
	return true
}

// GetThresholds возвращает пороговые значения
func (cm *ConfigManager) GetThresholds() map[string]float64 {
	return map[string]float64{
		"min_fall":             cm.config.MinFall,
		"continuity_threshold": cm.config.ContinuityThreshold,
		"volume_weight":        cm.config.VolumeWeight,
	}
}

// Validate проверяет валидность конфигурации
func (cm *ConfigManager) Validate() error {
	if cm.config.MinConfidence < 0 || cm.config.MinConfidence > 100 {
		return fmt.Errorf("min_confidence должен быть в диапазоне 0-100")
	}

	if cm.config.MinDataPoints < 2 {
		return fmt.Errorf("min_data_points должен быть >= 2")
	}

	if cm.config.MinFall < 0 {
		return fmt.Errorf("min_fall должен быть >= 0")
	}

	if cm.config.ContinuityThreshold < 0 || cm.config.ContinuityThreshold > 1 {
		return fmt.Errorf("continuity_threshold должен быть в диапазоне 0-1")
	}

	if cm.config.VolumeWeight < 0 || cm.config.VolumeWeight > 1 {
		return fmt.Errorf("volume_weight должен быть в диапазоне 0-1")
	}

	return nil
}

// applySetting применяет настройку из map к конфигурации
func (cm *ConfigManager) applySetting(key string, value interface{}) {
	switch key {
	case "enabled":
		if v, ok := value.(bool); ok {
			cm.config.Enabled = v
		}
	case "weight":
		if v, ok := value.(float64); ok {
			cm.config.Weight = v
		}
	case "min_confidence":
		if v, ok := value.(float64); ok {
			cm.config.MinConfidence = v
		}
	case "min_data_points":
		if v, ok := value.(int); ok {
			cm.config.MinDataPoints = v
		} else if v, ok := value.(float64); ok {
			cm.config.MinDataPoints = int(v)
		}
	case "min_fall":
		if v, ok := value.(float64); ok {
			cm.config.MinFall = v
		}
	case "continuity_threshold":
		if v, ok := value.(float64); ok {
			cm.config.ContinuityThreshold = v
		}
	case "volume_weight":
		if v, ok := value.(float64); ok {
			cm.config.VolumeWeight = v
		}
	case "check_all_algorithms":
		if v, ok := value.(bool); ok {
			cm.config.CheckAllAlgorithms = v
		}
	}
}
