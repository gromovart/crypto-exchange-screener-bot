// internal/types/storage/storage.go
package storage

import (
	"crypto_exchange_screener_bot/internal/types/common"
	"crypto_exchange_screener_bot/internal/types/events"
	"crypto_exchange_screener_bot/internal/types/fetcher"
	"time"
)

// StorageConfig - конфигурация хранилища
type StorageConfig struct {
	Type                string `json:"type"` // "memory", "redis", "postgres"
	MaxItems            int    `json:"max_items"`
	MaxHistoryPerSymbol int
	MaxSymbols          int
	CleanupInterval     time.Duration `json:"cleanup_interval"`
	RetentionPeriod     time.Duration `json:"retention_period"`
	EnablePersistence   bool          `json:"enable_persistence"`
	EnableCompression   bool          `json:"enable_compression"`
	PersistencePath     string        `json:"persistence_path"`
}

// StoredData - хранимые данные
type StoredData struct {
	Key       string                 `json:"key"`
	Value     interface{}            `json:"value"`
	Timestamp time.Time              `json:"timestamp"`
	ExpiresAt time.Time              `json:"expires_at,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// QueryOptions - опции запроса к хранилищу
type QueryOptions struct {
	FromTime time.Time `json:"from_time,omitempty"`
	ToTime   time.Time `json:"to_time,omitempty"`
	Limit    int       `json:"limit,omitempty"`
	Offset   int       `json:"offset,omitempty"`
	OrderBy  string    `json:"order_by,omitempty"` // "asc" or "desc"
}

// PriceSnapshot текущий снапшот цены
type PriceSnapshot struct {
	Symbol    common.Symbol `json:"symbol"`
	Price     float64       `json:"price"`
	Volume24h float64       `json:"volume_24h"`
	Timestamp time.Time     `json:"timestamp"`
}

// StorageStats статистика хранилища
type StorageStats struct {
	TotalSymbols        int           `json:"total_symbols"`
	TotalDataPoints     int64         `json:"total_data_points"`
	MemoryUsageBytes    int64         `json:"memory_usage_bytes"`
	OldestTimestamp     time.Time     `json:"oldest_timestamp"`
	NewestTimestamp     time.Time     `json:"newest_timestamp"`
	UpdateRatePerSecond float64       `json:"update_rate_per_second"`
	StorageType         string        `json:"storage_type"`
	MaxHistoryPerSymbol int           `json:"max_history_per_symbol"`
	RetentionPeriod     time.Duration `json:"retention_period"`
}

// PriceHistory запрос истории цен
type PriceHistoryRequest struct {
	Symbol    common.Symbol `json:"symbol"`
	StartTime time.Time     `json:"start_time,omitempty"`
	EndTime   time.Time     `json:"end_time,omitempty"`
	Limit     int           `json:"limit,omitempty"`
}

// Ошибки хранилища
var (
	ErrSymbolNotFound  = StorageError{"symbol not found"}
	ErrStorageFull     = StorageError{"storage is full"}
	ErrInvalidLimit    = StorageError{"invalid limit"}
	ErrAlreadyExists   = StorageError{"symbol already exists"}
	ErrSubscriberError = StorageError{"subscriber error"}
)

// StorageError ошибка хранилища
type StorageError struct {
	Message string
}

func (e StorageError) Error() string {
	return e.Message
}

type SymbolStats struct {
	Symbol         common.Symbol `json:"symbol"`
	DataPoints     int           `json:"data_points"`
	FirstTimestamp time.Time     `json:"first_timestamp"`
	LastTimestamp  time.Time     `json:"last_timestamp"`
	CurrentPrice   float64       `json:"current_price"`
	AvgVolume24h   float64       `json:"avg_volume_24h"`
	PriceChange24h float64       `json:"price_change_24h"`
}

// SymbolVolume символ с объемом
type SymbolVolume struct {
	Symbol string  `json:"symbol"`
	Volume float64 `json:"volume"`
}

// PriceStorage интерфейс хранилища цен
type PriceStorage interface {
	// Основные операции
	StorePrice(symbol string, price, volume24h float64, timestamp time.Time) error
	GetCurrentPrice(symbol string) (float64, bool)
	GetCurrentSnapshot(symbol string) (*PriceSnapshot, bool)
	GetAllCurrentPrices() map[string]PriceSnapshot
	GetSymbols() []string
	SymbolExists(symbol string) bool

	// История цен
	GetPriceHistory(symbol string, limit int) ([]common.PriceData, error)
	GetPriceHistoryRange(symbol string, start, end time.Time) ([]common.PriceData, error)
	GetLatestPrice(symbol string) (*common.PriceData, bool)
	// Расчеты
	CalculatePriceChange(symbol string, interval time.Duration) (*fetcher.PriceChange, error)
	GetAveragePrice(symbol string, period time.Duration) (float64, error)
	GetMinMaxPrice(symbol string, period time.Duration) (min, max float64, err error)

	// Подписки
	Subscribe(symbol string, subscriber events.Subscriber) error
	Unsubscribe(symbol string, subscriber events.Subscriber) error
	GetSubscriberCount(symbol string) int

	// Управление
	CleanOldData(maxAge time.Duration) (int, error)
	TruncateHistory(symbol string, maxPoints int) error
	RemoveSymbol(symbol string) error
	Clear() error

	// Статистика
	GetStats() StorageStats
	GetSymbolStats(symbol string) (SymbolStats, error)

	// Поиск
	FindSymbolsByPattern(pattern string) ([]string, error)
	GetTopSymbolsByVolume(limit int) ([]SymbolVolume, error)
}

// SymbolStats статистика по символу
