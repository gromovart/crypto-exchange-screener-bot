// internal/types/analysis/analyzer.go
package analysis

import (
	"crypto_exchange_screener_bot/internal/types/common"
	"time"
)

// Analyzer - интерфейс анализатора
type Analyzer interface {
	Name() string
	Version() string
	Supports(symbol common.Symbol) bool
	Analyze(data []common.PriceData, config AnalyzerConfig) ([]Signal, error)
	GetConfig() AnalyzerConfig
	GetStats() AnalyzerStats
}

// AnalyzerConfig - конфигурация анализатора
type AnalyzerConfig struct {
	Enabled        bool                   `json:"enabled"`
	Weight         float64                `json:"weight"`          // вес в общем результате
	MinConfidence  float64                `json:"min_confidence"`  // минимальная уверенность
	MinDataPoints  int                    `json:"min_data_points"` // минимальное количество точек
	CustomSettings map[string]interface{} `json:"custom_settings"`
}

// AnalyzerStats - статистика анализатора
type AnalyzerStats struct {
	TotalCalls   int           `json:"total_calls"`
	SuccessCount int           `json:"success_count"`
	ErrorCount   int           `json:"error_count"`
	TotalTime    time.Duration `json:"total_time"`
	AverageTime  time.Duration `json:"average_time"`
	LastCallTime time.Time     `json:"last_call_time"`
}

// AnalysisRequest - запрос на анализ
type AnalysisRequest struct {
	Symbol         common.Symbol    `json:"symbol"`
	Exchange       common.Exchange  `json:"exchange"`
	Timeframe      common.Timeframe `json:"timeframe"`
	Period         time.Duration    `json:"period"`
	Strategies     []string         `json:"strategies"`
	IncludeHistory bool             `json:"include_history"`
}

// AnalysisResult - результат анализа
type AnalysisResult struct {
	Symbol    common.Symbol   `json:"symbol"`
	Exchange  common.Exchange `json:"exchange"`
	Signals   []Signal        `json:"signals"`
	Timestamp time.Time       `json:"timestamp"`
	Duration  time.Duration   `json:"duration"`
}
