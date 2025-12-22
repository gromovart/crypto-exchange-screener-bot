// cmd/internal/storage/price_storage.go
package storage

import (
	typesStorage "crypto_exchange_screener_bot/internal/types/storage"
	"time"
)

// StorageFactory фабрика хранилищ
type StorageFactory struct{}

// NewInMemoryStorage создает in-memory хранилище
// NewInMemoryStorage создает in-memory хранилище
func (sf *StorageFactory) NewInMemoryStorage(options ...StorageOption) typesStorage.PriceStorage {
	config := &typesStorage.StorageConfig{
		MaxHistoryPerSymbol: 10000,
		MaxSymbols:          1000,
		CleanupInterval:     5 * time.Minute,
		RetentionPeriod:     24 * time.Hour,
	}

	for _, option := range options {
		option(config)
	}

	// Используйте локальную функцию, не импортируйте другой пакет storage
	return NewInMemoryPriceStorage(config)
}

// StorageOption функция настройки хранилища
type StorageOption func(*typesStorage.StorageConfig)

func WithMaxHistoryPerSymbol(max int) StorageOption {
	return func(c *typesStorage.StorageConfig) {
		c.MaxHistoryPerSymbol = max
	}
}

func WithMaxSymbols(max int) StorageOption {
	return func(c *typesStorage.StorageConfig) {
		c.MaxSymbols = max
	}
}

func WithRetentionPeriod(period time.Duration) StorageOption {
	return func(c *typesStorage.StorageConfig) {
		c.RetentionPeriod = period
	}
}
