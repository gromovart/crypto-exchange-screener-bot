// internal/infrastructure/persistence/redis_storage/cache_manager/structs.go
package cache_manager

import (
	"context"
	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	"sync"

	"github.com/go-redis/redis/v8"
)

// CacheManager реализует интерфейс CacheManagerInterface
type CacheManager struct {
	client *redis.Client
	ctx    context.Context
	prefix string

	// Локальный кэш для быстрого доступа
	localCache   map[string]storage.PriceSnapshotInterface
	localCacheMu sync.RWMutex
}
