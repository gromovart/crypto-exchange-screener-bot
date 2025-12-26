// persistence/redis/cache.go
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

func (c *Cache) SetUser(user interface{}, userID int, ttl time.Duration) error {
	key := fmt.Sprintf("%suser:%d", c.prefix, userID)

	data, err := json.Marshal(user)
	if err != nil {
		return err
	}

	return c.client.Set(context.Background(), key, data, ttl).Err()
}

func (c *Cache) GetUser(userID int, dest interface{}) error {
	key := fmt.Sprintf("%suser:%d", c.prefix, userID)

	data, err := c.client.Get(context.Background(), key).Result()
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(data), dest)
}

func (c *Cache) DeleteUser(userID int) error {
	key := fmt.Sprintf("%suser:%d", c.prefix, userID)
	return c.client.Del(context.Background(), key).Err()
}

func (c *Cache) SetActiveUsers(users interface{}, ttl time.Duration) error {
	key := c.prefix + "active_users"

	data, err := json.Marshal(users)
	if err != nil {
		return err
	}

	return c.client.Set(context.Background(), key, data, ttl).Err()
}

func (c *Cache) GetActiveUsers(dest interface{}) error {
	key := c.prefix + "active_users"

	data, err := c.client.Get(context.Background(), key).Result()
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(data), dest)
}

// Rate limiting
func (c *Cache) CheckRateLimit(key string, limit int, window time.Duration) (bool, int, error) {
	fullKey := c.prefix + "ratelimit:" + key

	// Используем Redis pipeline для атомарности
	pipe := c.client.Pipeline()

	incr := pipe.Incr(context.Background(), fullKey)
	pipe.Expire(context.Background(), fullKey, window)

	_, err := pipe.Exec(context.Background())
	if err != nil {
		return false, 0, err
	}

	count := int(incr.Val())
	return count <= limit, count, nil
}
