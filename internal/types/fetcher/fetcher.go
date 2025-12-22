// internal/types/fetcher/fetcher.go
package fetcher

import (
	"crypto_exchange_screener_bot/internal/types/analysis"
	"crypto_exchange_screener_bot/internal/types/common"
	"time"
)

// FetcherConfig - конфигурация фетчера
type FetcherConfig struct {
	Exchange        common.Exchange    `json:"exchange"`
	Symbols         []common.Symbol    `json:"symbols"`
	Timeframes      []common.Timeframe `json:"timeframes"`
	UpdateInterval  time.Duration      `json:"update_interval"`
	MaxRetries      int                `json:"max_retries"`
	RetryDelay      time.Duration      `json:"retry_delay"`
	EnableWebsocket bool               `json:"enable_websocket"`
}

// FetcherStats - статистика фетчера
type FetcherStats struct {
	TotalRequests      int           `json:"total_requests"`
	SuccessfulRequests int           `json:"successful_requests"`
	FailedRequests     int           `json:"failed_requests"`
	LastUpdateTime     time.Time     `json:"last_update_time"`
	AverageLatency     time.Duration `json:"average_latency"`
	ActiveConnections  int           `json:"active_connections"`
}

// FetcherResult - результат фетчера
type FetcherResult struct {
	Data      []common.PriceData `json:"data"`
	Timestamp time.Time          `json:"timestamp"`
	Success   bool               `json:"success"`
	Error     error              `json:"error,omitempty"`
	Stats     FetcherStats       `json:"stats,omitempty"`
}

// PriceFetcher интерфейс
type PriceFetcher interface {
	Start(interval time.Duration) error
	Stop() error
	IsRunning() bool
	GetStats() map[string]interface{}
}

// PriceFetcherConfig - конфигурация PriceFetcher
type PriceFetcherConfig struct {
	UpdateInterval      time.Duration
	MaxConcurrent       int
	RequestTimeout      time.Duration
	SymbolFilter        string
	ExcludeSymbols      string
	MaxSymbolsToMonitor int
	MinVolumeFilter     float64
	InitialDataFetch    bool
	DataFetchLimit      int
	FuturesCategory     string
}

// NotificationService интерфейс сервиса уведомлений
type NotificationService interface {
	Send(signal analysis.TrendSignal) error
	SendBatch(signals []analysis.TrendSignal) error
	SetEnabled(enabled bool)
	IsEnabled() bool
	GetStats() map[string]interface{}
}

// PriceChange изменение цены
type PriceChange struct {
	Symbol        common.Symbol `json:"symbol"`
	CurrentPrice  float64       `json:"current_price"`
	PreviousPrice float64       `json:"previous_price"`
	Change        float64       `json:"change"`
	ChangePercent float64       `json:"change_percent"`
	Interval      string        `json:"interval"`
	Timestamp     time.Time     `json:"timestamp"`
}
