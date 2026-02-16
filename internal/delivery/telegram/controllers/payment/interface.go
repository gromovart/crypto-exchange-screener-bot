// internal/delivery/telegram/controllers/payment/interface.go
package payment

import "crypto-exchange-screener-bot/internal/types"

// Controller интерфейс для обработки платежных событий
type Controller interface {
	// HandleEvent обрабатывает событие от EventBus
	HandleEvent(event types.Event) error

	// GetName возвращает имя контроллера
	GetName() string

	// GetSubscribedEvents возвращает типы событий для подписки
	GetSubscribedEvents() []types.EventType
}
