// internal/core/domain/signals/detectors/open_interest_analyzer/types.go
package oianalyzer

import (
	"time"

	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/types"

	"github.com/google/uuid"
)

// OISignalType - типы сигналов OI анализатора
type OISignalType string

const (
	OITypeGrowthWithPrice OISignalType = "oi_growth_with_price"
	OITypeGrowthWithFall  OISignalType = "oi_growth_with_fall"
	OITypeExtreme         OISignalType = "extreme_oi"
	OITypeBullishDiv      OISignalType = "bullish_oi_divergence"
	OITypeBearishDiv      OISignalType = "bearish_oi_divergence"
)

// OIAnalysisResult - результат анализа OI
type OIAnalysisResult struct {
	Symbol    string
	Signals   []*OISignal
	Timestamp time.Time
	Duration  time.Duration
}

// OISignal - сигнал OI анализатора
type OISignal struct {
	ID            string
	Symbol        string
	Type          OISignalType
	Direction     string
	ChangePercent float64
	Confidence    float64
	DataPoints    int
	StartPrice    float64
	EndPrice      float64
	StartOI       float64
	EndOI         float64
	Timestamp     time.Time
	Metadata      OIMetadata
}

// OIMetadata - метаданные OI сигнала
type OIMetadata struct {
	Strategy       string
	Tags           []string
	Indicators     map[string]float64
	IsExtreme      bool
	ExtremeType    string // "high", "low"
	DivergenceType string // "bullish", "bearish"
	Patterns       []string
}

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

// OIState - состояние OI для символа
type OIState struct {
	Symbol       string
	CurrentOI    float64
	AvgOI        float64
	OIRatio      float64 // отношение текущего OI к среднему
	PriceChange  float64 // изменение цены в %
	OIChange     float64 // изменение OI в %
	LastUpdated  time.Time
	History      []OIDataPoint // история OI для анализа дивергенций
	ExtremeFlag  bool          // флаг экстремального значения
	ExtremeSince time.Time     // время с начала экстремального состояния
}

// OIDataPoint - точка данных OI
type OIDataPoint struct {
	Timestamp   time.Time
	Price       float64
	OI          float64
	Volume      float64
	PriceChange float64 // изменение цены с предыдущей точки
	OIChange    float64 // изменение OI с предыдущей точки
}

// OIAnalysisParams - параметры анализа OI
type OIAnalysisParams struct {
	Symbol          string
	Data            []types.PriceData
	Config          OIConfig
	CurrentState    *OIState
	GenerateSignal  bool // генерировать ли сигнал
	IncludeMetadata bool // включать ли метаданные
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

// NewOISignal создает новый OI сигнал
func NewOISignal(symbol string, signalType OISignalType, direction string, changePercent, confidence float64) *OISignal {
	return &OISignal{
		ID:            uuid.New().String(),
		Symbol:        symbol,
		Type:          signalType,
		Direction:     direction,
		ChangePercent: changePercent,
		Confidence:    confidence,
		Timestamp:     time.Now(),
		Metadata: OIMetadata{
			Strategy:   string(signalType),
			Tags:       []string{"open_interest"},
			Indicators: make(map[string]float64),
		},
	}
}

// ConvertToAnalysisSignal конвертирует OI сигнал в общий сигнал
func (s *OISignal) ConvertToAnalysisSignal() analysis.Signal {
	tags := append(s.Metadata.Tags, string(s.Type))
	if s.Metadata.ExtremeType != "" {
		tags = append(tags, "extreme_"+s.Metadata.ExtremeType)
	}
	if s.Metadata.DivergenceType != "" {
		tags = append(tags, s.Metadata.DivergenceType+"_divergence")
	}

	return analysis.Signal{
		ID:            s.ID,
		Symbol:        s.Symbol,
		Type:          string(s.Type),
		Direction:     s.Direction,
		ChangePercent: s.ChangePercent,
		Confidence:    s.Confidence,
		DataPoints:    s.DataPoints,
		StartPrice:    s.StartPrice,
		EndPrice:      s.EndPrice,
		Timestamp:     s.Timestamp,
		Metadata: analysis.Metadata{
			Strategy:     s.Metadata.Strategy,
			Tags:         tags,
			Indicators:   s.Metadata.Indicators,
			IsContinuous: false,
			Patterns:     s.Metadata.Patterns,
		},
	}
}

// AnalyzerConfigCopy - копия AnalyzerConfig без импорта пакета analyzers
type AnalyzerConfigCopy struct {
	Enabled        bool
	Weight         float64
	MinConfidence  float64
	MinDataPoints  int
	CustomSettings map[string]interface{}
}
