// internal/delivery/telegram/queue/producer.go
package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis/v8"
)

// Producer помещает сообщения в Redis очередь
type Producer struct {
	client *redis.Client
}

// NewProducer создает Producer
func NewProducer(client *redis.Client) *Producer {
	return &Producer{client: client}
}

// Enqueue помещает сообщение в очередь по приоритету (LPUSH)
func (p *Producer) Enqueue(ctx context.Context, msg QueuedMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("queue: marshal message: %w", err)
	}
	return p.client.LPush(ctx, string(msg.Priority), data).Err()
}
