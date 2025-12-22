// шт internal/pipeline/types.go
package pipeline

import (
	"crypto_exchange_screener_bot/internal/types/analysis"
	"time"
)

// PipelineStage этап обработки сигнала
type PipelineStage interface {
	Name() string
	Process(signal analysis.Signal) (analysis.Signal, error)
}

// PipelineStats статистика пайплайна
type PipelineStats struct {
	SignalsReceived  int64         `json:"signals_received"`
	SignalsProcessed int64         `json:"signals_processed"`
	SignalsFiltered  int64         `json:"signals_filtered"`
	AverageTime      time.Duration `json:"average_time"`
	LastProcessed    time.Time     `json:"last_processed"`
}
