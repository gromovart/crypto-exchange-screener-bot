// internal/core/domain/signals/detectors/volume_analyzer/types.go
package volume_analyzer

import (
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/common"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
)

// VolumeCalculator интерфейс для всех калькуляторов объема
type VolumeCalculator interface {
	// Calculate вычисляет сигналы на основе данных
	Calculate(data []redis_storage.PriceData) *analysis.Signal

	// UpdateConfig обновляет конфигурацию калькулятора
	UpdateConfig(config common.AnalyzerConfig)

	// GetName возвращает имя калькулятора
	GetName() string
}

// VolumeConfig расширенная конфигурация для анализатора объема
type VolumeConfig struct {
	common.AnalyzerConfig

	// Специфичные настройки для объема
	MinVolume             float64 `json:"min_volume"`
	VolumeChangeThreshold float64 `json:"volume_change_threshold"`
	SpikeMultiplier       float64 `json:"spike_multiplier"`       // Во сколько раз больше среднего считается всплеском
	ConfirmationThreshold float64 `json:"confirmation_threshold"` // Порог для согласованности
}

// VolumeStats статистика анализатора объема
type VolumeStats struct {
	common.AnalyzerStats

	// Специфичная статистика
	AverageVolumeChecks    int64 `json:"average_volume_checks"`
	SpikeDetections        int64 `json:"spike_detections"`
	ConfirmationDetections int64 `json:"confirmation_detections"`
	DivergenceDetections   int64 `json:"divergence_detections"`

	// Метрики качества
	AverageVolume      float64 `json:"average_volume"`
	MaxSpikeRatio      float64 `json:"max_spike_ratio"`
	AverageCorrelation float64 `json:"average_correlation"`
}

// VolumeMetrics метрики для одного расчета
type VolumeMetrics struct {
	Symbol    string `json:"symbol"`
	Timestamp int64  `json:"timestamp"`

	// Метрики объема
	CurrentVolume float64 `json:"current_volume"`
	AverageVolume float64 `json:"average_volume"`
	VolumeChange  float64 `json:"volume_change"`
	VolumeRatio   float64 `json:"volume_ratio"` // current/avg

	// Метрики цены
	PriceChange  float64 `json:"price_change"`
	CurrentPrice float64 `json:"current_price"`

	// Корреляция
	PriceVolumeCorrelation float64 `json:"price_volume_correlation"`

	// Флаги
	IsSpike          bool `json:"is_spike"`
	IsConfirmation   bool `json:"is_confirmation"`
	IsDivergence     bool `json:"is_divergence"`
	IsAboveMinVolume bool `json:"is_above_min_volume"`

	// Дополнительно
	DataPoints int     `json:"data_points"`
	Confidence float64 `json:"confidence"`
}

// VolumeAlgorithmType типы алгоритмов анализа объема
type VolumeAlgorithmType string

const (
	AlgorithmAverageVolume      VolumeAlgorithmType = "average_volume"
	AlgorithmVolumeSpike        VolumeAlgorithmType = "volume_spike"
	AlgorithmVolumeConfirmation VolumeAlgorithmType = "volume_confirmation"
	AlgorithmVolumeDivergence   VolumeAlgorithmType = "volume_divergence"
)

// AlgorithmConfig конфигурация для конкретного алгоритма
type AlgorithmConfig struct {
	Type       VolumeAlgorithmType `json:"type"`
	Enabled    bool                `json:"enabled"`
	Weight     float64             `json:"weight"`
	Threshold  float64             `json:"threshold"`
	Parameters map[string]float64  `json:"parameters"`
}
