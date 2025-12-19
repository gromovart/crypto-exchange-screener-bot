// internal/analysis/analyzers/types.go
package analyzers

import (
	"crypto-exchange-screener-bot/internal/analysis"
	"crypto-exchange-screener-bot/internal/types"
	"time"
)

// Analyzer - интерфейс анализатора
type Analyzer interface {
	Name() string
	Version() string
	Supports(symbol string) bool
	Analyze(data []types.PriceData, config AnalyzerConfig) ([]analysis.Signal, error)
	GetConfig() AnalyzerConfig
	GetStats() AnalyzerStats
}

// AnalyzerConfig - конфигурация анализатора
type AnalyzerConfig struct {
	Enabled        bool                   `json:"enabled"`
	Weight         float64                `json:"weight"`          // вес анализатора в общем результате
	MinConfidence  float64                `json:"min_confidence"`  // минимальная уверенность
	MinDataPoints  int                    `json:"min_data_points"` // минимальное количество точек
	CustomSettings map[string]interface{} `json:"custom_settings"`
}

// AnalyzerStats - статистика анализатора
type AnalyzerStats struct {
	TotalCalls   int64         `json:"total_calls"`
	TotalTime    time.Duration `json:"total_time"`
	SuccessCount int64         `json:"success_count"`
	ErrorCount   int64         `json:"error_count"`
	LastCallTime time.Time     `json:"last_call_time"`
	AverageTime  time.Duration `json:"average_time"`
}

// Signal - структура сигнала анализа
type Signal struct {
	ID            string    `json:"id"`
	Symbol        string    `json:"symbol"`
	Type          string    `json:"type"`           // "growth", "fall", "breakout", "volume_spike"
	Direction     string    `json:"direction"`      // "up", "down"
	ChangePercent float64   `json:"change_percent"` // процент изменения
	Period        int       `json:"period"`         // период в минутах
	Confidence    float64   `json:"confidence"`     // уверенность 0-100
	DataPoints    int       `json:"data_points"`    // количество точек данных
	StartPrice    float64   `json:"start_price"`
	EndPrice      float64   `json:"end_price"`
	Volume        float64   `json:"volume"`
	Timestamp     time.Time `json:"timestamp"`
	Metadata      Metadata  `json:"metadata"`
}

// Metadata - метаданные сигнала
type Metadata struct {
	Strategy     string             `json:"strategy"`
	Tags         []string           `json:"tags"`
	Indicators   map[string]float64 `json:"indicators"`
	IsContinuous bool               `json:"is_continuous"`
	Patterns     []string           `json:"patterns"`
}

// AnalysisRequest - запрос на анализ
type AnalysisRequest struct {
	Symbol         string        `json:"symbol"`
	Period         time.Duration `json:"period"`
	Strategies     []string      `json:"strategies"`
	IncludeHistory bool          `json:"include_history"`
}

// AnalysisResult - результат анализа
type AnalysisResult struct {
	Symbol    string        `json:"symbol"`
	Signals   []Signal      `json:"signals"`
	Timestamp time.Time     `json:"timestamp"`
	Duration  time.Duration `json:"duration"`
}
