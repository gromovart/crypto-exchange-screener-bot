// internal/delivery/telegram/controllers/counter-notification/controller.go
package counternotification

import (
	counternotification "crypto-exchange-screener-bot/internal/delivery/telegram/services/counter-notification"
	"crypto-exchange-screener-bot/internal/types"
	"log"
)

// controllerImpl реализация CounterNotificationController
type controllerImpl struct {
	service counternotification.Service
}

// NewController создает новый контроллер уведомлений счетчика
func NewController(service counternotification.Service) Controller {
	return &controllerImpl{service: service}
}

// HandleEvent обрабатывает событие от EventBus
func (c *controllerImpl) HandleEvent(event types.Event) error {
	log.Printf("🤖 CounterNotificationController: Событие %s от %s", event.Type, event.Source)

	// Создаем параметры для сервиса
	params := struct {
		Event types.Event `json:"event"`
	}{
		Event: event,
	}

	// Вызываем Exec сервиса
	result, err := c.service.Exec(params)
	if err != nil {
		log.Printf("❌ CounterNotificationController: Ошибка обработки: %v", err)
		return err
	}

	// Логируем результат
	log.Printf("✅ CounterNotificationController: Результат: %+v", result)
	return nil
}

// GetName возвращает имя контроллера
func (c *controllerImpl) GetName() string {
	return "counter_notification_controller"
}

// GetSubscribedEvents возвращает типы событий для подписки
func (c *controllerImpl) GetSubscribedEvents() []types.EventType {
	return []types.EventType{
		types.EventCounterNotificationRequest,
	}
}
