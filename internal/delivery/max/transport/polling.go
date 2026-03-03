// internal/delivery/max/transport/polling.go
package transport

import (
	"context"
	"time"

	"crypto-exchange-screener-bot/internal/delivery/max"
	"crypto-exchange-screener-bot/pkg/logger"
)

const (
	pollingTimeout  = 30              // секунд — long-polling timeout
	pollingInterval = 3 * time.Second // пауза при ошибке
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
			logger.Warn("⚠️ MAX getUpdates error: %v — повтор через %s", err, pollingInterval)
			select {
			case <-ctx.Done():
				return
			case <-time.After(pollingInterval):
				continue
			}
		}

		// Обновляем маркер только если он изменился
		if newMarker > marker {
			marker = newMarker
		}

		for _, upd := range updates {
			go p.handler(upd)
		}
	}
}
