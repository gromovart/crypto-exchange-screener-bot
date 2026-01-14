// internal/delivery/telegram/controllers/signal/controller.go
package signal

import (
	"crypto-exchange-screener-bot/internal/delivery/telegram/services/signal"
	"crypto-exchange-screener-bot/internal/types"
	"log"
)

// controllerImpl реализация SignalController
type controllerImpl struct {
	service signal.Service
}

// NewController создает новый контроллер сигналов
func NewController(service signal.Service) Controller {
	return &controllerImpl{service: service}
}

// HandleEvent обрабатывает событие от EventBus
func (c *controllerImpl) HandleEvent(event types.Event) error {
	log.Printf("🤖 SignalController: Событие %s от %s", event.Type, event.Source)

	// Создаем параметры для сервиса
	params := struct {
		Event types.Event `json:"event"`
	}{
		Event: event,
	}

	// Вызываем Exec сервиса
	result, err := c.service.Exec(params)
	if err != nil {
		log.Printf("❌ SignalController: Ошибка обработки: %v", err)
		return err
	}

	// Логируем результат
	log.Printf("✅ SignalController: Результат: %+v", result)
	return nil
}

// GetName возвращает имя контроллера
func (c *controllerImpl) GetName() string {
	return "signal_controller"
}

// GetSubscribedEvents возвращает типы событий для подписки
func (c *controllerImpl) GetSubscribedEvents() []types.EventType {
	return []types.EventType{
		types.EventSignalDetected,
	}
}
