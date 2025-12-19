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
