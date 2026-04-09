// internal/delivery/telegram/services/counter/publisher.go
package counter

import (
	"context"
	"encoding/json"
	"time"

	"crypto-exchange-screener-bot/pkg/logger"

	goredis "github.com/go-redis/redis/v8"
)

const ScreenerSignalChannel = "signals:screener"

// SignalPublisher публикует сигналы для внешних потребителей (например Analyzer)
type SignalPublisher interface {
	PublishSignal(ctx context.Context, data RawCounterData) error
}

// screenerSignalPayload структура сигнала публикуемого в Redis
type screenerSignalPayload struct {
	Symbol    string    `json:"symbol"`
	Direction string    `json:"direction"` // "up" | "down"
	Change    float64   `json:"change"`
	Period    string    `json:"period"`
	Price     float64   `json:"price"`
	Timestamp time.Time `json:"timestamp"`
}

// RedisSignalPublisher реализация через Redis Pub/Sub
type RedisSignalPublisher struct {
	client *goredis.Client
}

func NewRedisSignalPublisher(client *goredis.Client) *RedisSignalPublisher {
	return &RedisSignalPublisher{client: client}
}

func (p *RedisSignalPublisher) PublishSignal(ctx context.Context, data RawCounterData) error {
	payload := screenerSignalPayload{
		Symbol:    data.Symbol,
		Direction: data.Direction,
		Change:    data.ChangePercent,
		Period:    data.Period,
		Price:     data.CurrentPrice,
		Timestamp: data.Timestamp,
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	result := p.client.Publish(ctx, ScreenerSignalChannel, string(b))
	if result.Err() != nil {
		return result.Err()
	}

	logger.Debug("📡 Сигнал опубликован в %s: %s %s %.2f%% (%s)",
		ScreenerSignalChannel, data.Symbol, data.Direction, data.ChangePercent, data.Period)
	return nil
}
