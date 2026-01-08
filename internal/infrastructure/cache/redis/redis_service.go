// internal/infrastructure/cache/redis/redis_service.go
package redis

import (
	"context"
	"fmt"
	"time"

	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"crypto-exchange-screener-bot/pkg/logger"

	"github.com/go-redis/redis/v8"
)

// RedisService сервис для работы с Redis
type RedisService struct {
	config *config.Config
	client *redis.Client
	state  ServiceState
}

// ServiceState состояние сервиса
type ServiceState string

const (
	StateStopped  ServiceState = "stopped"
	StateStarting ServiceState = "starting"
	StateRunning  ServiceState = "running"
	StateStopping ServiceState = "stopping"
	StateError    ServiceState = "error"
)

// NewRedisService создает новый Redis сервис
func NewRedisService(cfg *config.Config) *RedisService {
	return &RedisService{
		config: cfg,
		state:  StateStopped,
	}
}

// Start запускает Redis сервис
func (rs *RedisService) Start() error {
	if rs.state == StateRunning {
		return fmt.Errorf("Redis service already running")
	}

	logger.Info("🔄 Starting Redis service...")
	rs.state = StateStarting

	// Получаем конфигурацию Redis
	redisConfig := rs.config.Redis

	// Создаем клиент Redis
	options := &redis.Options{
		Addr:     fmt.Sprintf("%s:%d", redisConfig.Host, redisConfig.Port),
		Password: redisConfig.Password,
		DB:       redisConfig.DB,

		// Настройки пула соединений
		PoolSize:     redisConfig.PoolSize,
		MinIdleConns: redisConfig.MinIdleConns,

		// Таймауты
		DialTimeout:  redisConfig.DialTimeout,
		ReadTimeout:  redisConfig.ReadTimeout,
		WriteTimeout: redisConfig.WriteTimeout,
		PoolTimeout:  redisConfig.PoolTimeout,

		// Повторные попытки
		MaxRetries:      redisConfig.MaxRetries,
		MinRetryBackoff: redisConfig.MinRetryBackoff,
		MaxRetryBackoff: redisConfig.MaxRetryBackoff,

		// Для go-redis/v8 используем MaxConnAge вместо ConnMaxLifetime
		// ConnMaxLifetime и ConnMaxIdleTime не поддерживаются в v8
	}

	rs.client = redis.NewClient(options)

	// Проверяем подключение
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logger.Info("📡 Connecting to Redis: %s:%d (DB: %d)",
		redisConfig.Host, redisConfig.Port, redisConfig.DB)

	if _, err := rs.client.Ping(ctx).Result(); err != nil {
		rs.client.Close()
		rs.state = StateError
		logger.Error("❌ Failed to connect to Redis: %v (address: %s)", err, options.Addr)
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	rs.state = StateRunning

	// Логируем успешное подключение
	logger.Info("✅ Successfully connected to Redis")
	logger.Info("   • Host: %s:%d", redisConfig.Host, redisConfig.Port)
	logger.Info("   • Database: %d", redisConfig.DB)
	logger.Info("   • Pool size: %d", redisConfig.PoolSize)
	logger.Info("   • Min idle connections: %d", redisConfig.MinIdleConns)

	return nil
}

// Stop останавливает Redis сервис
func (rs *RedisService) Stop() error {
	if rs.state != StateRunning {
		return fmt.Errorf("Redis service is not running")
	}

	logger.Info("🛑 Stopping Redis service...")
	rs.state = StateStopping

	if rs.client != nil {
		if err := rs.client.Close(); err != nil {
			rs.state = StateError
			logger.Error("❌ Failed to close Redis client: %v", err)
			return fmt.Errorf("failed to close Redis client: %w", err)
		}
	}

	rs.client = nil
	rs.state = StateStopped
	logger.Info("✅ Redis service stopped")

	return nil
}

// GetClient возвращает клиент Redis
func (rs *RedisService) GetClient() *redis.Client {
	return rs.client
}

// State возвращает состояние сервиса
func (rs *RedisService) State() ServiceState {
	return rs.state
}

// HealthCheck проверяет здоровье Redis
func (rs *RedisService) HealthCheck() bool {
	if rs.state != StateRunning || rs.client == nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if _, err := rs.client.Ping(ctx).Result(); err != nil {
		logger.Info("⚠️ Redis health check failed: %v", err)
		return false
	}

	return true
}

// GetStats возвращает статистику Redis
func (rs *RedisService) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"state":     rs.state,
		"connected": rs.client != nil,
	}

	if rs.client != nil {
		poolStats := rs.client.PoolStats()

		stats["pool_hits"] = poolStats.Hits
		stats["pool_misses"] = poolStats.Misses
		stats["pool_timeouts"] = poolStats.Timeouts
		stats["pool_total_conns"] = poolStats.TotalConns
		stats["pool_idle_conns"] = poolStats.IdleConns
		stats["pool_stale_conns"] = poolStats.StaleConns

		// Информация о конфигурации
		redisConfig := rs.config.Redis
		stats["host"] = redisConfig.Host
		stats["port"] = redisConfig.Port
		stats["db"] = redisConfig.DB
		stats["pool_size"] = redisConfig.PoolSize
		stats["min_idle_conns"] = redisConfig.MinIdleConns
	}

	return stats
}

// TestConnection тестирует подключение к Redis
func (rs *RedisService) TestConnection() error {
	if rs.state != StateRunning || rs.client == nil {
		return fmt.Errorf("Redis service is not running")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if _, err := rs.client.Ping(ctx).Result(); err != nil {
		return fmt.Errorf("Redis connection test failed: %w", err)
	}

	return nil
}

// Set устанавливает значение в Redis
func (rs *RedisService) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if rs.client == nil {
		return fmt.Errorf("Redis client is not initialized")
	}

	if ttl == 0 {
		ttl = rs.config.Redis.DefaultTTL
	}

	return rs.client.Set(ctx, key, value, ttl).Err()
}

// Get получает значение из Redis
func (rs *RedisService) Get(ctx context.Context, key string) (string, error) {
	if rs.client == nil {
		return "", fmt.Errorf("Redis client is not initialized")
	}

	return rs.client.Get(ctx, key).Result()
}

// Delete удаляет ключ из Redis
func (rs *RedisService) Delete(ctx context.Context, key string) error {
	if rs.client == nil {
		return fmt.Errorf("Redis client is not initialized")
	}

	return rs.client.Del(ctx, key).Err()
}

// Exists проверяет существование ключа
func (rs *RedisService) Exists(ctx context.Context, key string) (bool, error) {
	if rs.client == nil {
		return false, fmt.Errorf("Redis client is not initialized")
	}

	result, err := rs.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}

	return result > 0, nil
}
