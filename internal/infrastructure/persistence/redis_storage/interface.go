// internal/infrastructure/persistence/redis_storage/interface.go
package redis_storage

import (
	"time"

	"github.com/go-redis/redis/v8"
)

// SubscriberInterface интерфейс подписчика
type SubscriberInterface interface {
	OnPriceUpdate(symbol string, price, volume24h, volumeUSD float64, timestamp time.Time)
	OnSymbolAdded(symbol string)
	OnSymbolRemoved(symbol string)
}

// PriceDataInterface интерфейс для данных цены
type PriceDataInterface interface {
	GetSymbol() string
	GetPrice() float64
	GetVolume24h() float64
	GetVolumeUSD() float64
	GetTimestamp() time.Time
	GetOpenInterest() float64
	GetFundingRate() float64
	GetChange24h() float64
	GetHigh24h() float64
	GetLow24h() float64
}

// PriceSnapshotInterface интерфейс для снапшота цены
type PriceSnapshotInterface interface {
	GetSymbol() string
	GetPrice() float64
	GetVolume24h() float64
	GetVolumeUSD() float64
	GetTimestamp() time.Time
	GetOpenInterest() float64
	GetFundingRate() float64
	GetChange24h() float64
	GetHigh24h() float64
	GetLow24h() float64
}

// PriceChangeInterface интерфейс для изменения цены
type PriceChangeInterface interface {
	GetSymbol() string
	GetCurrentPrice() float64
	GetPreviousPrice() float64
	GetChange() float64
	GetChangePercent() float64
	GetInterval() string
	GetTimestamp() time.Time
	GetVolumeUSD() float64
}

// SymbolMetricsInterface интерфейс для метрик символа
type SymbolMetricsInterface interface {
	GetSymbol() string
	GetPrice() float64
	GetVolume24h() float64
	GetVolumeUSD() float64
	GetOpenInterest() float64
	GetFundingRate() float64
	GetChange24h() float64
	GetHigh24h() float64
	GetLow24h() float64
	GetOIChange24h() float64
	GetFundingChange() float64
	GetTimestamp() time.Time
}

// SymbolStatsInterface интерфейс для статистики символа
type SymbolStatsInterface interface {
	GetSymbol() string
	GetDataPoints() int
	GetFirstTimestamp() time.Time
	GetLastTimestamp() time.Time
	GetCurrentPrice() float64
	GetAvgVolume24h() float64
	GetAvgVolumeUSD() float64
	GetPriceChange24h() float64
	GetOpenInterest() float64
	GetOIChange24h() float64
	GetFundingRate() float64
	GetFundingChange() float64
	GetHigh24h() float64
	GetLow24h() float64
}

// StorageStatsInterface интерфейс для статистики хранилища
type StorageStatsInterface interface {
	GetTotalSymbols() int
	GetTotalDataPoints() int64
	GetMemoryUsageBytes() int64
	GetOldestTimestamp() time.Time
	GetNewestTimestamp() time.Time
	GetUpdateRatePerSecond() float64
	GetStorageType() string
	GetMaxHistoryPerSymbol() int
	GetRetentionPeriod() time.Duration
	GetSymbolsWithOI() int
	GetSymbolsWithFunding() int
}

// SymbolVolumeInterface интерфейс для данных объема
type SymbolVolumeInterface interface {
	GetSymbol() string
	GetVolume() float64
	GetVolumeUSD() float64
}

// StorageConfigInterface интерфейс конфигурации хранилища
type StorageConfigInterface interface {
	GetMaxHistoryPerSymbol() int
	GetMaxSymbols() int
	GetCleanupInterval() time.Duration
	GetRetentionPeriod() time.Duration
	GetEnableCompression() bool
	GetEnablePersistence() bool
	GetPersistencePath() string
}

// PriceStorageInterface интерфейс хранилища цен
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
	StorePriceData(priceData PriceDataInterface) error
	GetCurrentPrice(symbol string) (float64, bool)
	GetCurrentSnapshot(symbol string) (PriceSnapshotInterface, bool)
	GetAllCurrentPrices() map[string]PriceSnapshotInterface
	GetSymbols() []string
	SymbolExists(symbol string) bool
	GetPriceHistory(symbol string, limit int) ([]PriceDataInterface, error)
	GetPriceHistoryRange(symbol string, start, end time.Time) ([]PriceDataInterface, error)
	GetLatestPrice(symbol string) (PriceDataInterface, bool)
	CalculatePriceChange(symbol string, interval time.Duration) (PriceChangeInterface, error)
	GetAveragePrice(symbol string, period time.Duration) (float64, error)
	GetMinMaxPrice(symbol string, period time.Duration) (min, max float64, err error)
	GetOpenInterest(symbol string) (float64, bool)
	GetFundingRate(symbol string) (float64, bool)
	GetSymbolMetrics(symbol string) (SymbolMetricsInterface, bool)
	Subscribe(symbol string, subscriber SubscriberInterface) error
	Unsubscribe(symbol string, subscriber SubscriberInterface) error
	GetSubscriberCount(symbol string) int
	CleanOldData(maxAge time.Duration) (int, error)
	TruncateHistory(symbol string, maxPoints int) error
	RemoveSymbol(symbol string) error
	Clear() error
	GetStats() StorageStatsInterface
	GetSymbolStats(symbol string) (SymbolStatsInterface, error)
	GetTopSymbolsByVolumeUSD(limit int) ([]SymbolVolumeInterface, error)
	GetTopSymbolsByVolume(limit int) ([]SymbolVolumeInterface, error)
	FindSymbolsByPattern(pattern string) ([]string, error)
	StorePriceLegacy(symbol string, price, volume24h float64, timestamp time.Time) error
}

// CacheManagerInterface интерфейс менеджера кэша
type CacheManagerInterface interface {
	Initialize(client *redis.Client)
	SaveSnapshot(pipe redis.Pipeliner, symbol string, snapshot PriceSnapshotInterface)
	GetSnapshot(symbol string) (PriceSnapshotInterface, bool)
	GetAllSnapshots() map[string]PriceSnapshotInterface
	GetSymbols() []string
	ClearCache()
	RemoveFromCache(symbol string)
}

// HistoryManagerInterface интерфейс менеджера истории
type HistoryManagerInterface interface {
	Initialize(client *redis.Client, config *StorageConfig)
	AddToHistory(pipe redis.Pipeliner, symbol string, snapshot PriceSnapshotInterface) error
	GetHistory(symbol string, limit int) ([]PriceDataInterface, error)
	GetHistoryRange(symbol string, start, end time.Time) ([]PriceDataInterface, error)
	CleanupOldHistory(maxAge time.Duration) (int, error)
	TruncateHistory(symbol string, maxPoints int) error
	GetSymbolsWithHistory() ([]string, error)
}

// SubscriptionManagerInterface интерфейс менеджера подписок
type SubscriptionManagerInterface interface {
	Subscribe(symbol string, subscriber SubscriberInterface) error
	Unsubscribe(symbol string, subscriber SubscriberInterface) error
	GetSubscriberCount(symbol string) int
	NotifyAll(symbol string, price, volume24h, volumeUSD float64, timestamp time.Time)
	NotifySymbolRemoved(symbol string)
}
