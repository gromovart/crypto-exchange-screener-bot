package storage

import "time"

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
	GetPriceHistory(symbol string, limit int) ([]PriceData, error)
	GetPriceHistoryRange(symbol string, start, end time.Time) ([]PriceData, error)
	GetLatestPrice(symbol string) (*PriceData, bool)

	// Расчеты
	CalculatePriceChange(symbol string, interval time.Duration) (*PriceChange, error)
	GetAveragePrice(symbol string, period time.Duration) (float64, error)
	GetMinMaxPrice(symbol string, period time.Duration) (min, max float64, err error)

	// Подписки
	Subscribe(symbol string, subscriber Subscriber) error
	Unsubscribe(symbol string, subscriber Subscriber) error
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
type SymbolStats struct {
	Symbol         string    `json:"symbol"`
	DataPoints     int       `json:"data_points"`
	FirstTimestamp time.Time `json:"first_timestamp"`
	LastTimestamp  time.Time `json:"last_timestamp"`
	CurrentPrice   float64   `json:"current_price"`
	AvgVolume24h   float64   `json:"avg_volume_24h"`
	PriceChange24h float64   `json:"price_change_24h"`
}

// SymbolVolume символ с объемом
type SymbolVolume struct {
	Symbol string  `json:"symbol"`
	Volume float64 `json:"volume"`
}

// StorageFactory фабрика хранилищ
type StorageFactory struct{}

// NewInMemoryStorage создает in-memory хранилище
func (sf *StorageFactory) NewInMemoryStorage(options ...StorageOption) PriceStorage {
	config := &StorageConfig{
		MaxHistoryPerSymbol: 10000,
		MaxSymbols:          1000,
		CleanupInterval:     5 * time.Minute,
		RetentionPeriod:     24 * time.Hour,
	}

	for _, option := range options {
		option(config)
	}

	return NewInMemoryPriceStorage(config)
}

// StorageConfig конфигурация хранилища
type StorageConfig struct {
	MaxHistoryPerSymbol int
	MaxSymbols          int
	CleanupInterval     time.Duration
	RetentionPeriod     time.Duration
	EnableCompression   bool
	EnablePersistence   bool
	PersistencePath     string
}

// StorageOption функция настройки хранилища
type StorageOption func(*StorageConfig)

func WithMaxHistoryPerSymbol(max int) StorageOption {
	return func(c *StorageConfig) {
		c.MaxHistoryPerSymbol = max
	}
}

func WithMaxSymbols(max int) StorageOption {
	return func(c *StorageConfig) {
		c.MaxSymbols = max
	}
}

func WithRetentionPeriod(period time.Duration) StorageOption {
	return func(c *StorageConfig) {
		c.RetentionPeriod = period
	}
}
