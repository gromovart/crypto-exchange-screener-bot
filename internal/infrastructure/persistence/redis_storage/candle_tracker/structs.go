// internal/infrastructure/persistence/redis_storage/candle_tracker/structs.go
package candletracker

import (
	"context"
	redis_service "crypto-exchange-screener-bot/internal/infrastructure/cache/redis"
	"time"

	"github.com/go-redis/redis/v8"
)

// CandleTracker отслеживает обработанные свечи в Redis
type CandleTracker struct {
	redisService *redis_service.RedisService
	client       *redis.Client
	ctx          context.Context
	prefix       string
	ttl          time.Duration
}

// NewCandleTracker создает новый трекер свечей
func NewCandleTracker(redisService *redis_service.RedisService, ttl time.Duration) *CandleTracker {
	return &CandleTracker{
		redisService: redisService,
		ctx:          context.Background(),
		prefix:       "processed_candle:",
		ttl:          ttl,
	}
}

// CandleKey представляет уникальный ключ свечи
type CandleKey struct {
	Symbol    string
	Period    string
	StartTime int64
}
