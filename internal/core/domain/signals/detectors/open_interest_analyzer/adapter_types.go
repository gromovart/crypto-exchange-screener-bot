// internal/core/domain/signals/detectors/open_interest_analyzer/adapter_types.go
package oianalyzer

import (
	"time"

	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/types"
)

// AnalyzerConfigCopy - копия AnalyzerConfig без импорта пакета analyzers
type AnalyzerConfigCopy struct {
	Enabled        bool
	Weight         float64
	MinConfidence  float64
	MinDataPoints  int
	CustomSettings map[string]interface{}
}

// AnalyzerStatsCopy - копия AnalyzerStats без импорта пакета analyzers
type AnalyzerStatsCopy struct {
	TotalCalls   int
	SuccessCount int
	ErrorCount   int
	TotalTime    time.Duration
	AverageTime  time.Duration
	LastCallTime time.Time
}

// AnalyzerCopy - копия интерфейса Analyzer без импорта пакета analyzers
type AnalyzerCopy interface {
	Name() string
	Version() string
	Supports(symbol string) bool
	Analyze(data []types.PriceData, config AnalyzerConfigCopy) ([]analysis.Signal, error)
	GetConfig() AnalyzerConfigCopy
	GetStats() AnalyzerStatsCopy
}

// ConvertToAnalyzerConfigCopy конвертирует оригинальный AnalyzerConfig в копию
func ConvertToAnalyzerConfigCopy(config interface{}) AnalyzerConfigCopy {
	// Эта функция будет реализована в factory.go
	// Пока возвращаем пустую конфигурацию
	return AnalyzerConfigCopy{
		Enabled:        true,
		Weight:         0.6,
		MinConfidence:  50.0,
		MinDataPoints:  3,
		CustomSettings: make(map[string]interface{}),
	}
}
