// internal/delivery/max/transport/polling.go
package transport

import (
	"context"
	"time"

	"crypto-exchange-screener-bot/internal/delivery/max"
	"crypto-exchange-screener-bot/pkg/logger"
)

const (
	pollingTimeout      = 30               // секунд — long-polling timeout
	pollingRetryMin     = 3 * time.Second  // начальная пауза при ошибке
	pollingRetryMax     = 5 * time.Minute  // максимальная пауза при ошибке
)

// UpdateHandler — функция-обработчик входящих обновлений
type UpdateHandler func(update max.Update)

// Poller — long-polling цикл для MAX API (использует marker вместо offset)
type Poller struct {
	client  *max.Client
	handler UpdateHandler
}

// NewPoller создаёт новый поллер
func NewPoller(client *max.Client, handler UpdateHandler) *Poller {
	return &Poller{
		client:  client,
		handler: handler,
	}
}

// Run запускает цикл опроса. Блокирует до отмены ctx.
func (p *Poller) Run(ctx context.Context) {
	var marker int64
	retryDelay := pollingRetryMin
	logger.Info("🔄 MAX Polling запущен (timeout=%ds)", pollingTimeout)

	for {
		select {
		case <-ctx.Done():
			logger.Info("🛑 MAX Polling остановлен")
			return
		default:
		}

		updates, newMarker, err := p.client.GetUpdates(marker, pollingTimeout)
		if err != nil {
			logger.Warn("⚠️ MAX getUpdates error: %v — повтор через %s", err, retryDelay)
			select {
			case <-ctx.Done():
				return
			case <-time.After(retryDelay):
			}
			// Exponential backoff: 3s → 6s → 12s → ... → 5min
			retryDelay *= 2
			if retryDelay > pollingRetryMax {
				retryDelay = pollingRetryMax
			}
			continue
		}

		// Успешный запрос — сбрасываем задержку
		retryDelay = pollingRetryMin

		// Обновляем маркер только если он изменился
		if newMarker > marker {
			marker = newMarker
		}

		for _, upd := range updates {
			go p.handler(upd)
		}
	}
}
