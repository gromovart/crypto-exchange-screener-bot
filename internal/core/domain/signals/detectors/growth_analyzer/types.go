// internal/core/domain/signals/detectors/growth_analyzer/types.go
package growth_analyzer

import (
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/common"
	"crypto-exchange-screener-bot/internal/types"
	"time"
)

// GrowthSignalType - тип сигнала роста
type GrowthSignalType string

const (
	SignalTypeContinuousGrowth  GrowthSignalType = "continuous_growth"
	SignalTypeAcceleratedGrowth GrowthSignalType = "accelerated_growth"
	SignalTypeBreakoutGrowth    GrowthSignalType = "breakout_growth"
)

// GrowthConfig - конфигурация анализатора роста
type GrowthConfig struct {
	common.AnalyzerConfig
	MinGrowthPercent      float64 `json:"min_growth_percent"`
	ContinuityThreshold   float64 `json:"continuity_threshold"`
	AccelerationThreshold float64 `json:"acceleration_threshold"`
	VolumeWeight          float64 `json:"volume_weight"`
	TrendStrengthWeight   float64 `json:"trend_strength_weight"`
	VolatilityWeight      float64 `json:"volatility_weight"`
}

// GrowthStats - статистика анализатора роста
type GrowthStats struct {
	common.AnalyzerStats
	TotalGrowthSignals    int     `json:"total_growth_signals"`
	AverageGrowthPercent  float64 `json:"average_growth_percent"`
	MaxGrowthPercent      float64 `json:"max_growth_percent"`
	ContinuousGrowthCount int     `json:"continuous_growth_count"`
}

// GrowthAnalysisResult - результат анализа роста
type GrowthAnalysisResult struct {
	Symbol        string            `json:"symbol"`
	GrowthPercent float64           `json:"growth_percent"`
	SignalType    GrowthSignalType  `json:"signal_type"`
	Confidence    float64           `json:"confidence"`
	IsContinuous  bool              `json:"is_continuous"`
	TrendStrength float64           `json:"trend_strength"`
	Volatility    float64           `json:"volatility"`
	DataPoints    int               `json:"data_points"`
	StartPrice    float64           `json:"start_price"`
	EndPrice      float64           `json:"end_price"`
	Timestamp     time.Time         `json:"timestamp"`
	RawData       []types.PriceData `json:"-"`
}

// ToSignal - преобразует результат в общий сигнал
func (r *GrowthAnalysisResult) ToSignal() analysis.Signal {
	tags := []string{"growth", "bullish"}
	if r.IsContinuous {
		tags = append(tags, "continuous")
	}
	if string(r.SignalType) != "" {
		tags = append(tags, string(r.SignalType))
	}

	return analysis.Signal{
		Symbol:        r.Symbol,
		Type:          "growth",
		Direction:     "up",
		ChangePercent: r.GrowthPercent,
		Confidence:    r.Confidence,
		DataPoints:    r.DataPoints,
		StartPrice:    r.StartPrice,
		EndPrice:      r.EndPrice,
		Timestamp:     r.Timestamp,
		Metadata: analysis.Metadata{
			Strategy:     "growth_detection",
			Tags:         tags,
			IsContinuous: r.IsContinuous,
			Indicators: map[string]float64{
				"trend_strength": r.TrendStrength,
				"volatility":     r.Volatility,
				"growth_type":    mapGrowthSignalTypeToValue(r.SignalType),
			},
		},
	}
}

// mapGrowthSignalTypeToValue - маппинг типа сигнала в числовое значение
func mapGrowthSignalTypeToValue(signalType GrowthSignalType) float64 {
	switch signalType {
	case SignalTypeContinuousGrowth:
		return 1.0
	case SignalTypeAcceleratedGrowth:
		return 2.0
	case SignalTypeBreakoutGrowth:
		return 3.0
	default:
		return 0.0
	}
}

// GrowthCalculatorInput - входные данные для калькулятора роста
type GrowthCalculatorInput struct {
	PriceData   []types.PriceData `json:"price_data"`
	Config      GrowthConfig      `json:"config"`
	CurrentTime time.Time         `json:"current_time"`
}

// GrowthCalculatorOutput - выходные данные калькулятора роста
type GrowthCalculatorOutput struct {
	GrowthPercent   float64          `json:"growth_percent"`
	IsContinuous    bool             `json:"is_continuous"`
	ContinuityScore float64          `json:"continuity_score"`
	Acceleration    float64          `json:"acceleration"`
	TrendStrength   float64          `json:"trend_strength"`
	Volatility      float64          `json:"volatility"`
	SignalType      GrowthSignalType `json:"signal_type"`
	ConfidenceScore float64          `json:"confidence_score"`
	Recommendation  string           `json:"recommendation"`
}
