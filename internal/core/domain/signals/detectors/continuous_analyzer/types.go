// internal/core/domain/signals/detectors/continuous_analyzer/types.go
package continuous_analyzer

import (
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/common"
	"crypto-exchange-screener-bot/internal/types"
)

// ContinuousCalculator интерфейс для всех калькуляторов непрерывности
type ContinuousCalculator interface {
	// Calculate вычисляет сигналы на основе данных
	Calculate(data []types.PriceData, minPoints int) []analysis.Signal

	// UpdateConfig обновляет конфигурацию калькулятора
	UpdateConfig(config common.AnalyzerConfig)

	// GetName возвращает имя калькулятора
	GetName() string
}

// ContinuousConfig расширенная конфигурация для анализатора непрерывности
type ContinuousConfig struct {
	common.AnalyzerConfig

	// Специфичные настройки для непрерывности
	MinContinuousPoints int     `json:"min_continuous_points"`
	MaxGapRatio         float64 `json:"max_gap_ratio"`
	RequireConfirmation bool    `json:"require_confirmation"`
}

// ContinuousStats статистика анализатора непрерывности
type ContinuousStats struct {
	common.AnalyzerStats

	// Специфичная статистика
	GrowthSequencesFound int64 `json:"growth_sequences_found"`
	FallSequencesFound   int64 `json:"fall_sequences_found"`
	BestSequencesFound   int64 `json:"best_sequences_found"`
	InvalidSequences     int64 `json:"invalid_sequences"`

	// Метрики качества
	AvgSequenceLength  float64 `json:"avg_sequence_length"`
	MaxSequenceLength  int     `json:"max_sequence_length"`
	AvgConfidenceScore float64 `json:"avg_confidence_score"`
}

// SequenceMetrics метрики последовательности
type SequenceMetrics struct {
	Symbol    string `json:"symbol"`
	Timestamp int64  `json:"timestamp"`

	// Основные метрики
	SequenceLength int     `json:"sequence_length"`
	Direction      string  `json:"direction"`
	TotalChange    float64 `json:"total_change"`
	AverageChange  float64 `json:"average_change"`
	AverageGap     float64 `json:"average_gap"`

	// Качество последовательности
	IsContinuous    bool    `json:"is_continuous"`
	HasConfirmation bool    `json:"has_confirmation"`
	Confidence      float64 `json:"confidence"`
	QualityScore    float64 `json:"quality_score"`

	// Дополнительно
	StartPrice float64 `json:"start_price"`
	EndPrice   float64 `json:"end_price"`
	StartIdx   int     `json:"start_idx"`
	EndIdx     int     `json:"end_idx"`
}

// SequenceType типы последовательностей
type SequenceType string

const (
	SequenceTypeGrowth SequenceType = "growth"
	SequenceTypeFall   SequenceType = "fall"
	SequenceTypeBest   SequenceType = "best"
	SequenceTypeMixed  SequenceType = "mixed"
)

// SequenceAlgorithm конфигурация алгоритма поиска
type SequenceAlgorithm struct {
	Type       SequenceType       `json:"type"`
	Enabled    bool               `json:"enabled"`
	MinPoints  int                `json:"min_points"`
	Parameters map[string]float64 `json:"parameters"`
}


