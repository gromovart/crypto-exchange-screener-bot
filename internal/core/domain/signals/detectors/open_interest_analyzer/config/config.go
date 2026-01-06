// internal/core/domain/signals/detectors/open_interest_analyzer/config/config.go
package config

import (
	"fmt"
)

// OIConfig - конфигурация OI анализатора
type OIConfig struct {
	Enabled             bool
	Weight              float64
	MinConfidence       float64
	MinDataPoints       int
	MinPriceChange      float64 // минимальное изменение цены (%)
	MinPriceFall        float64 // минимальное падение цены (%)
	MinOIChange         float64 // минимальное изменение OI (%)
	ExtremeOIThreshold  float64 // порог экстремального OI (1.5 = на 50% выше среднего)
	DivergenceMinPoints int     // минимальное количество точек для дивергенции
	VolumeWeight        float64 // вес объема в расчетах
	CheckAllAlgorithms  bool    // проверять все алгоритмы
}

// DefaultConfig - конфигурация по умолчанию для OI анализатора
var DefaultConfig = OIConfig{
	Enabled:             true,
	Weight:              0.6,
	MinConfidence:       50.0,
	MinDataPoints:       3,
	MinPriceChange:      1.0,  // минимальное изменение цены для сигнала (%)
	MinPriceFall:        1.0,  // минимальное падение цены для сигнала (%)
	MinOIChange:         5.0,  // минимальное изменение OI для сигнала (%)
	ExtremeOIThreshold:  1.5,  // порог экстремального OI (1.5 = на 50% выше среднего)
	DivergenceMinPoints: 4,    // минимальное количество точек для дивергенции
	VolumeWeight:        0.3,  // вес объема в расчетах
	CheckAllAlgorithms:  true, // проверять все алгоритмы
}

// OIAlgorithm - алгоритм анализа OI
type OIAlgorithm string

const (
	AlgorithmGrowthWithPrice OIAlgorithm = "growth_with_price"
	AlgorithmGrowthWithFall  OIAlgorithm = "growth_with_fall"
	AlgorithmExtremeOI       OIAlgorithm = "extreme_oi"
	AlgorithmDivergence      OIAlgorithm = "divergence"
	AlgorithmAll             OIAlgorithm = "all"
)

// ConfigManager - менеджер конфигурации OI анализатора
type ConfigManager struct {
	config OIConfig
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
func (cm *ConfigManager) GetConfig() OIConfig {
	return cm.config
}

// UpdateConfig обновляет конфигурацию
func (cm *ConfigManager) UpdateConfig(newConfig OIConfig) {
	cm.config = newConfig
}

// IsAlgorithmEnabled проверяет, включен ли алгоритм
func (cm *ConfigManager) IsAlgorithmEnabled(algorithm OIAlgorithm) bool {
	if !cm.config.Enabled {
		return false
	}

	if cm.config.CheckAllAlgorithms {
		return true
	}

	// Здесь можно добавить логику выборочного включения алгоритмов
	// пока возвращаем true для всех, если анализатор включен
	return true
}

// GetThresholds возвращает пороговые значения
func (cm *ConfigManager) GetThresholds() map[string]float64 {
	return map[string]float64{
		"min_price_change":     cm.config.MinPriceChange,
		"min_price_fall":       cm.config.MinPriceFall,
		"min_oi_change":        cm.config.MinOIChange,
		"extreme_oi_threshold": cm.config.ExtremeOIThreshold,
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

	if cm.config.MinPriceChange < 0 {
		return fmt.Errorf("min_price_change должен быть >= 0")
	}

	if cm.config.MinPriceFall < 0 {
		return fmt.Errorf("min_price_fall должен быть >= 0")
	}

	if cm.config.MinOIChange < 0 {
		return fmt.Errorf("min_oi_change должен быть >= 0")
	}

	if cm.config.ExtremeOIThreshold < 1.0 {
		return fmt.Errorf("extreme_oi_threshold должен быть >= 1.0")
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
	case "min_price_change":
		if v, ok := value.(float64); ok {
			cm.config.MinPriceChange = v
		}
	case "min_price_fall":
		if v, ok := value.(float64); ok {
			cm.config.MinPriceFall = v
		}
	case "min_oi_change":
		if v, ok := value.(float64); ok {
			cm.config.MinOIChange = v
		}
	case "extreme_oi_threshold":
		if v, ok := value.(float64); ok {
			cm.config.ExtremeOIThreshold = v
		}
	case "divergence_min_points":
		if v, ok := value.(int); ok {
			cm.config.DivergenceMinPoints = v
		} else if v, ok := value.(float64); ok {
			cm.config.DivergenceMinPoints = int(v)
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
