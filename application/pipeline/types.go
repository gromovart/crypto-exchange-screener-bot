// шт internal/pipeline/types.go
package pipeline

import (
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"sync"
	"time"
)

// PipelineStage этап обработки сигнала
type PipelineStage interface {
	Name() string
	Process(signal analysis.Signal) (analysis.Signal, error)
}

// RateLimiter ограничитель частоты
type RateLimiter struct {
	lastSent map[string]time.Time
	minDelay time.Duration
	mu       sync.RWMutex
}

// PipelineStats статистика пайплайна
type PipelineStats struct {
	SignalsReceived  int64         `json:"signals_received"`
	SignalsProcessed int64         `json:"signals_processed"`
	SignalsFiltered  int64         `json:"signals_filtered"`
	AverageTime      time.Duration `json:"average_time"`
	LastProcessed    time.Time     `json:"last_processed"`
}
