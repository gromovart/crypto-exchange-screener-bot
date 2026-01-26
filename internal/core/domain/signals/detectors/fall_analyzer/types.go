// internal/core/domain/signals/detectors/fall_analyzer/types.go
package fallanalyzer

import (
	"time"

	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"

	"github.com/google/uuid"
)

// FallSignalType - тип сигнала падения
type FallSignalType string

const (
	FallTypeSingle     FallSignalType = "single_fall"
	FallTypeInterval   FallSignalType = "interval_fall"
	FallTypeContinuous FallSignalType = "continuous_fall"
)

// FallAnalysisResult - результат анализа падений
type FallAnalysisResult struct {
	Symbol    string
	Signals   []*FallSignal
	Timestamp time.Time
	Duration  time.Duration
}

// FallSignal - сигнал падения
type FallSignal struct {
	ID            string
	Symbol        string
	Type          FallSignalType
	Direction     string
	ChangePercent float64
	Confidence    float64
	Period        int
	DataPoints    int
	StartPrice    float64
	EndPrice      float64
	Volume        float64
	Timestamp     time.Time
	Metadata      FallMetadata
}

// FallMetadata - метаданные сигнала падения
type FallMetadata struct {
	Strategy       string
	Tags           []string
	Indicators     map[string]float64
	IsContinuous   bool
	ContinuityFrom int
	ContinuityTo   int
	Patterns       []string
}

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

// FallState - состояние анализа падений для символа
type FallState struct {
	Symbol       string
	CurrentPrice float64
	PriceChange  float64         // изменение цены в %
	LastFallTime time.Time       // время последнего падения
	FallCount    int             // счетчик падений
	IsFalling    bool            // флаг текущего падения
	FallSince    time.Time       // время с начала падения
	History      []FallDataPoint // история для анализа непрерывности
}

// FallDataPoint - точка данных для анализа падений
type FallDataPoint struct {
	Timestamp   time.Time
	Price       float64
	Volume      float64
	PriceChange float64 // изменение цены с предыдущей точки
	IsFall      bool    // является ли падением
}

// FallAnalysisParams - параметры анализа падений
type FallAnalysisParams struct {
	Symbol          string
	Data            []redis_storage.PriceData
	Config          FallConfig
	CurrentState    *FallState
	GenerateSignal  bool // генерировать ли сигнал
	IncludeMetadata bool // включать ли метаданные
}

// FallAlgorithm - алгоритм анализа падений
type FallAlgorithm string

const (
	AlgorithmSingleFall     FallAlgorithm = "single_fall"
	AlgorithmIntervalFall   FallAlgorithm = "interval_fall"
	AlgorithmContinuousFall FallAlgorithm = "continuous_fall"
	AlgorithmAllFall        FallAlgorithm = "all"
)

// NewFallSignal создает новый сигнал падения
func NewFallSignal(symbol string, signalType FallSignalType, direction string,
	changePercent, confidence float64, period int) *FallSignal {

	return &FallSignal{
		ID:            uuid.New().String(),
		Symbol:        symbol,
		Type:          signalType,
		Direction:     direction,
		ChangePercent: changePercent,
		Confidence:    confidence,
		Period:        period,
		Timestamp:     time.Now(),
		Metadata: FallMetadata{
			Strategy:   string(signalType),
			Tags:       []string{"fall", "bearish"},
			Indicators: make(map[string]float64),
		},
	}
}

// ConvertToAnalysisSignal конвертирует сигнал падения в общий сигнал
func (s *FallSignal) ConvertToAnalysisSignal() analysis.Signal {
	tags := append(s.Metadata.Tags, string(s.Type))
	if s.Metadata.IsContinuous {
		tags = append(tags, "continuous")
	}

	return analysis.Signal{
		ID:            s.ID,
		Symbol:        s.Symbol,
		Type:          string(s.Type),
		Direction:     s.Direction,
		ChangePercent: s.ChangePercent,
		Period:        s.Period,
		Confidence:    s.Confidence,
		DataPoints:    s.DataPoints,
		StartPrice:    s.StartPrice,
		EndPrice:      s.EndPrice,
		Volume:        s.Volume,
		Timestamp:     s.Timestamp,
		Metadata: analysis.Metadata{
			Strategy:       s.Metadata.Strategy,
			Tags:           tags,
			Indicators:     s.Metadata.Indicators,
			IsContinuous:   s.Metadata.IsContinuous,
			ContinuousFrom: s.Metadata.ContinuityFrom,
			ContinuousTo:   s.Metadata.ContinuityTo,
			Patterns:       s.Metadata.Patterns,
		},
	}
}
