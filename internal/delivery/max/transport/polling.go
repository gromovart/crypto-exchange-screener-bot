// internal/delivery/max/transport/polling.go
package transport

import (
	"context"
	"time"

	"crypto-exchange-screener-bot/internal/delivery/max"
	"crypto-exchange-screener-bot/pkg/logger"
)

const (
	pollingTimeout  = 30 // секунд — long-polling timeout
	pollingInterval = 1 * time.Second
)

// UpdateHandler функция-обработчик входящих обновлений
type UpdateHandler func(update max.Update)

// Poller реализует long-polling для MAX Bot API
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
	var offset int64
	logger.Info("🔄 MAX Polling запущен (timeout=%ds)", pollingTimeout)

	for {
		select {
		case <-ctx.Done():
			logger.Info("🛑 MAX Polling остановлен")
			return
		default:
		}

		updates, err := p.client.GetUpdates(offset, pollingTimeout)
		if err != nil {
			logger.Warn("⚠️ MAX getUpdates error: %v — повтор через %s", err, pollingInterval)
			select {
			case <-ctx.Done():
				return
			case <-time.After(pollingInterval):
				continue
			}
		}

		for _, upd := range updates {
			if upd.UpdateID >= offset {
				offset = upd.UpdateID + 1
			}
			// Обрабатываем каждое обновление в отдельной горутине
			go p.handler(upd)
		}
	}
}
