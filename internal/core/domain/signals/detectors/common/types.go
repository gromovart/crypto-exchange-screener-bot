// internal/core/domain/signals/detectors/common/types.go
package common

import (
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	"time"
)

// AnalyzerConfig - конфигурация анализатора (общий тип)
type AnalyzerConfig struct {
	Enabled        bool                   `json:"enabled"`
	Weight         float64                `json:"weight"`
	MinConfidence  float64                `json:"min_confidence"`
	MinDataPoints  int                    `json:"min_data_points"`
	CustomSettings map[string]interface{} `json:"custom_settings"`
}

// AnalyzerStats - статистика анализатора (общий тип)
type AnalyzerStats struct {
	TotalCalls   int           `json:"total_calls"`
	SuccessCount int           `json:"success_count"`
	ErrorCount   int           `json:"error_count"`
	TotalTime    time.Duration `json:"total_time"`
	AverageTime  time.Duration `json:"average_time"`
	LastCallTime time.Time     `json:"last_call_time"`
}

// Analyzer - интерфейс анализатора (общий интерфейс)
type Analyzer interface {
	Name() string
	Version() string
	Supports(symbol string) bool
	Analyze(data []redis_storage.PriceData, config AnalyzerConfig) ([]analysis.Signal, error)
	GetConfig() AnalyzerConfig
	GetStats() AnalyzerStats
}
