// internal/delivery/telegram/controllers/counter/controller.go
package counter

import (
	counterService "crypto-exchange-screener-bot/internal/delivery/telegram/services/counter"
	"crypto-exchange-screener-bot/internal/types"
	"log"
)

// controllerImpl реализация CounterController
type controllerImpl struct {
	service counterService.Service
}

// NewController создает новый контроллер счетчика
func NewController(service counterService.Service) Controller {
	return &controllerImpl{service: service}
}

// HandleEvent обрабатывает событие от EventBus
func (c *controllerImpl) HandleEvent(event types.Event) error {
	log.Printf("🤖 CounterController: Событие %s от %s", event.Type, event.Source)

	// Создаем параметры для сервиса
	params := counterService.CounterParams{
		Event: event,
	}

	// Вызываем Exec сервиса
	result, err := c.service.Exec(params)
	if err != nil {
		log.Printf("❌ CounterController: Ошибка обработки: %v", err)
		return err
	}

	// Логируем результат
	log.Printf("✅ CounterController: Результат: %+v", result)
	return nil
}

// GetName возвращает имя контроллера
func (c *controllerImpl) GetName() string {
	return "counter_controller"
}

// GetSubscribedEvents возвращает типы событий для подписки
func (c *controllerImpl) GetSubscribedEvents() []types.EventType {
	return []types.EventType{
		types.EventCounterSignalDetected,
	}
}
