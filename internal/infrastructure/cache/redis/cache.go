// internal/infrastructure/cache/redis/cache.go
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type Cache struct {
	client *redis.Client
	prefix string
}

func NewCache(addr, password string, db int) *Cache {
	return &Cache{
		client: redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       db,
		}),
		prefix: "cryptobot:",
	}
}

// Set устанавливает значение в Redis с TTL
func (c *Cache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	fullKey := c.prefix + key

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, fullKey, data, ttl).Err()
}

// Get получает значение из Redis
func (c *Cache) Get(ctx context.Context, key string, dest interface{}) error {
	fullKey := c.prefix + key

	data, err := c.client.Get(ctx, fullKey).Result()
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(data), dest)
}

// Delete удаляет ключ из Redis
func (c *Cache) Delete(ctx context.Context, key string) error {
	fullKey := c.prefix + key
	return c.client.Del(ctx, fullKey).Err()
}

// DeleteMulti удаляет несколько ключей из Redis
func (c *Cache) DeleteMulti(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = c.prefix + key
	}

	return c.client.Del(ctx, fullKeys...).Err()
}

// SetUser устанавливает пользователя в кэш
func (c *Cache) SetUser(ctx context.Context, user interface{}, userID int, ttl time.Duration) error {
	key := fmt.Sprintf("user:%d", userID)
	return c.Set(ctx, key, user, ttl)
}

// GetUser получает пользователя из кэша
func (c *Cache) GetUser(ctx context.Context, userID int, dest interface{}) error {
	key := fmt.Sprintf("user:%d", userID)
	return c.Get(ctx, key, dest)
}

// DeleteUser удаляет пользователя из кэша
func (c *Cache) DeleteUser(ctx context.Context, userID int) error {
	key := fmt.Sprintf("user:%d", userID)
	return c.Delete(ctx, key)
}

// SetActiveUsers устанавливает список активных пользователей в кэш
func (c *Cache) SetActiveUsers(ctx context.Context, users interface{}, ttl time.Duration) error {
	return c.Set(ctx, "active_users", users, ttl)
}

// GetActiveUsers получает список активных пользователей из кэша
func (c *Cache) GetActiveUsers(ctx context.Context, dest interface{}) error {
	return c.Get(ctx, "active_users", dest)
}

// SetUserByTelegramID устанавливает пользователя по Telegram ID
func (c *Cache) SetUserByTelegramID(ctx context.Context, user interface{}, telegramID int64, ttl time.Duration) error {
	key := fmt.Sprintf("user:telegram:%d", telegramID)
	return c.Set(ctx, key, user, ttl)
}

// GetUserByTelegramID получает пользователя по Telegram ID
func (c *Cache) GetUserByTelegramID(ctx context.Context, telegramID int64, dest interface{}) error {
	key := fmt.Sprintf("user:telegram:%d", telegramID)
	return c.Get(ctx, key, dest)
}

// SetUserByChatID устанавливает пользователя по Chat ID
func (c *Cache) SetUserByChatID(ctx context.Context, user interface{}, chatID string, ttl time.Duration) error {
	key := fmt.Sprintf("user:chat:%s", chatID)
	return c.Set(ctx, key, user, ttl)
}

// GetUserByChatID получает пользователя по Chat ID
func (c *Cache) GetUserByChatID(ctx context.Context, chatID string, dest interface{}) error {
	key := fmt.Sprintf("user:chat:%s", chatID)
	return c.Get(ctx, key, dest)
}

// Rate limiting
func (c *Cache) CheckRateLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, int, error) {
	fullKey := c.prefix + "ratelimit:" + key

	// Используем Redis pipeline для атомарности
	pipe := c.client.Pipeline()

	incr := pipe.Incr(ctx, fullKey)
	pipe.Expire(ctx, fullKey, window)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, 0, err
	}

	count := int(incr.Val())
	return count <= limit, count, nil
}
