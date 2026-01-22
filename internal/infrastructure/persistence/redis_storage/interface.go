// internal/infrastructure/persistence/redis_storage/interface.go
package redis_storage

import (
	"time"
)

// PriceStorageInterface интерфейс хранилища цен (общее имя)
type PriceStorageInterface interface {
	Initialize() error
	StorePrice(
		symbol string,
		price, volume24h, volumeUSD float64,
		timestamp time.Time,
		openInterest float64,
		fundingRate float64,
		change24h float64,
		high24h float64,
		low24h float64,
	) error
	StorePriceData(priceData PriceData) error
	GetCurrentPrice(symbol string) (float64, bool)
	GetCurrentSnapshot(symbol string) (*PriceSnapshot, bool)
	GetAllCurrentPrices() map[string]PriceSnapshot
	GetSymbols() []string
	SymbolExists(symbol string) bool
	GetPriceHistory(symbol string, limit int) ([]PriceData, error)
	GetPriceHistoryRange(symbol string, start, end time.Time) ([]PriceData, error)
	GetLatestPrice(symbol string) (*PriceData, bool)
	CalculatePriceChange(symbol string, interval time.Duration) (*PriceChange, error)
	GetAveragePrice(symbol string, period time.Duration) (float64, error)
	GetMinMaxPrice(symbol string, period time.Duration) (min, max float64, err error)
	GetOpenInterest(symbol string) (float64, bool)
	GetFundingRate(symbol string) (float64, bool)
	GetSymbolMetrics(symbol string) (*SymbolMetrics, bool)
	Subscribe(symbol string, subscriber Subscriber) error
	Unsubscribe(symbol string, subscriber Subscriber) error
	GetSubscriberCount(symbol string) int
	CleanOldData(maxAge time.Duration) (int, error)
	TruncateHistory(symbol string, maxPoints int) error
	RemoveSymbol(symbol string) error
	Clear() error
	GetStats() StorageStats
	GetSymbolStats(symbol string) (SymbolStats, error)
	GetTopSymbolsByVolumeUSD(limit int) ([]SymbolVolume, error)
	GetTopSymbolsByVolume(limit int) ([]SymbolVolume, error)
	FindSymbolsByPattern(pattern string) ([]string, error)
	StorePriceLegacy(symbol string, price, volume24h float64, timestamp time.Time) error
}
