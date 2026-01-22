// internal/infrastructure/persistence/redis_storage/price_storage/interface.go
package price_storage

import (
	"time"
)

// SymbolMetrics интерфейс для метрик символа
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

// SymbolStats интерфейс для статистики символа
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

// StorageStats интерфейс для статистики хранилища
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

// SymbolVolume интерфейс для данных объема
type SymbolVolumeInterface interface {
	GetSymbol() string
	GetVolume() float64
	GetVolumeUSD() float64
}

// StorageConfig интерфейс конфигурации хранилища
type StorageConfigInterface interface {
	GetMaxHistoryPerSymbol() int
	GetMaxSymbols() int
	GetCleanupInterval() time.Duration
	GetRetentionPeriod() time.Duration
	GetEnableCompression() bool
	GetEnablePersistence() bool
	GetPersistencePath() string
}
