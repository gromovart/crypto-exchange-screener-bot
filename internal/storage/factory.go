// cmd/internal/storage/factory.go
package storage

import (
	"container/list"
	"crypto_exchange_screener_bot/internal/types/common"
	"crypto_exchange_screener_bot/internal/types/events" // ДОБАВЬТЕ этот импорт
	typesStorage "crypto_exchange_screener_bot/internal/types/storage"
	"sync"
	"time"
)
// NewInMemoryPriceStorage создает новое in-memory хранилище
func NewInMemoryPriceStorage(config *storage.StorageConfig) *InMemoryPriceStorage {
	if config == nil {
		config = &storage.StorageConfig{
			MaxHistoryPerSymbol: 10000,
			MaxSymbols:          1000,
			CleanupInterval:     5 * time.Minute,
			RetentionPeriod:     24 * time.Hour,
		}
	}

	storage := &InMemoryPriceStorage{
		current:       make(map[string]*storage.PriceSnapshot),
		history:       make(map[string]*list.List),
		subscriptions: NewSubscriptionManager(),
		config:        config,
		lastCleanup:   time.Now(),
	}

	// Запускаем очистку старых данных
	go storage.startCleanupRoutine()

	return storage
}
