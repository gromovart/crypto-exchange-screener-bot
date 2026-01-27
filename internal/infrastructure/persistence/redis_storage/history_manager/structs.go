// internal/infrastructure/persistence/redis_storage/history_manager/structs.go
package history_manager

import (
	"context"
	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"

	"github.com/go-redis/redis/v8"
)

// HistoryManager реализует интерфейс HistoryManagerInterface
type HistoryManager struct {
	client *redis.Client
	ctx    context.Context
	prefix string
	config *storage.StorageConfig
}
