package monitor

import (
	"crypto-exchange-screener-bot/internal/types"
	"time"
)

// PriceFetcher интерфейс получения данных
type PriceFetcher interface {
	FetchPrices() ([]types.PriceData, error)
	StartFetching(interval time.Duration) error
	StopFetching() error
	IsRunning() bool
	GetLastFetchTime() time.Time
	GetStats() map[string]interface{} // Добавляем метод
}

// TrendAnalyzer интерфейс анализатора трендов
type TrendAnalyzer interface {
	Analyze(symbol string, history []types.PriceData) (types.TrendSignal, error)
	GetSupportedPeriods() []int
	SetThresholds(growth, fall float64)
	GetThresholds() (float64, float64)
	GetStats() map[string]interface{} // Добавляем метод
}
