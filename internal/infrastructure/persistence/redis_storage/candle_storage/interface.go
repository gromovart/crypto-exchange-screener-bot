// internal/infrastructure/persistence/redis_storage/candle_storage/interface.go
package candle_storage

import (
	"time"
)

// Candle интерфейс для свечи
type CandleInterface interface {
	GetSymbol() string
	GetPeriod() string
	GetOpen() float64
	GetHigh() float64
	GetLow() float64
	GetClose() float64
	GetVolume() float64
	GetVolumeUSD() float64
	GetTrades() int
	GetStartTime() time.Time
	GetEndTime() time.Time
	IsClosed() bool
	IsReal() bool
}

// CandleConfig интерфейс для конфигурации свечей
type CandleConfigInterface interface {
	GetSupportedPeriods() []string
	GetMaxHistory() int
	GetCleanupInterval() time.Duration
}

// CandleStats интерфейс для статистики свечей
type CandleStatsInterface interface {
	GetTotalCandles() int
	GetActiveCandles() int
	GetSymbolsCount() int
	GetOldestCandle() time.Time
	GetNewestCandle() time.Time
	GetPeriodsCount() map[string]int
}

// CandleStorage интерфейс хранилища свечей
type CandleStorageInterface interface {
	// Основные операции
	SaveActiveCandle(candle CandleInterface) error
	GetActiveCandle(symbol, period string) (CandleInterface, bool)
	CloseAndArchiveCandle(candle CandleInterface) error
	GetHistory(symbol, period string, limit int) ([]CandleInterface, error)
	GetLatestCandle(symbol, period string) (CandleInterface, bool)
	GetCandle(symbol, period string) (CandleInterface, error)
	CleanupOldCandles(maxAge time.Duration) int
	GetSymbols() []string
	GetPeriodsForSymbol(symbol string) []string
	GetStats() CandleStatsInterface
}
