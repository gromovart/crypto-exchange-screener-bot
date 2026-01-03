// internal/infrastructure/persistence/in_memory_storage/price_storage.go
package storage

import "time"

// PriceStorage интерфейс хранилища цен
type PriceStorage interface {
	// Основные операции
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

	StorePriceData(priceData PriceData) error // Альтернативный метод для удобства

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
	GetTopSymbolsByVolumeUSD(limit int) ([]SymbolVolume, error)

	// Дополнительные методы для новых данных
	GetOpenInterest(symbol string) (float64, bool)
	GetFundingRate(symbol string) (float64, bool)
	GetSymbolMetrics(symbol string) (*SymbolMetrics, bool)
}

// SymbolMetrics содержит все метрики символа
type SymbolMetrics struct {
	Symbol        string    `json:"symbol"`
	Price         float64   `json:"price"`
	Volume24h     float64   `json:"volume_24h"`
	VolumeUSD     float64   `json:"volume_usd"`
	OpenInterest  float64   `json:"open_interest"`
	FundingRate   float64   `json:"funding_rate"`
	Change24h     float64   `json:"change_24h"`
	High24h       float64   `json:"high_24h"`
	Low24h        float64   `json:"low_24h"`
	OIChange24h   float64   `json:"oi_change_24h"`
	FundingChange float64   `json:"funding_change"`
	Timestamp     time.Time `json:"timestamp"`
}

// SymbolStats статистика по символу
type SymbolStats struct {
	Symbol         string    `json:"symbol"`
	DataPoints     int       `json:"data_points"`
	FirstTimestamp time.Time `json:"first_timestamp"`
	LastTimestamp  time.Time `json:"last_timestamp"`
	CurrentPrice   float64   `json:"current_price"`
	AvgVolume24h   float64   `json:"avg_volume_24h"`
	AvgVolumeUSD   float64   `json:"avg_volume_usd"`
	PriceChange24h float64   `json:"price_change_24h"`
	OpenInterest   float64   `json:"open_interest"`
	OIChange24h    float64   `json:"oi_change_24h"`
	FundingRate    float64   `json:"funding_rate"`
	FundingChange  float64   `json:"funding_change"`
	High24h        float64   `json:"high_24h"`
	Low24h         float64   `json:"low_24h"`
}

// SymbolVolume символ с объемом
type SymbolVolume struct {
	Symbol    string  `json:"symbol"`
	Volume    float64 `json:"volume"`
	VolumeUSD float64 `json:"volume_usd,omitempty"`
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
