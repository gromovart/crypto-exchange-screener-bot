// internal/delivery/telegram/controllers/counter/interface.go
package counter

import "crypto-exchange-screener-bot/internal/types"

type Controller interface {
	// HandleEvent обрабатывает событие от EventBus
	HandleEvent(event types.Event) error

	// GetName возвращает имя контроллера
	GetName() string

	// GetSubscribedEvents возвращает типы событий для подписки
	GetSubscribedEvents() []types.EventType
}
