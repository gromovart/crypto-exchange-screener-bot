// internal/events/subscribers.go
package events

import (
	"log"
)

// BaseSubscriber - базовая реализация подписчика
type BaseSubscriber struct {
	name             string
	subscribedEvents []EventType
	handler          func(Event) error
}

// NewBaseSubscriber создает нового подписчика
func NewBaseSubscriber(name string, events []EventType, handler func(Event) error) *BaseSubscriber {
	return &BaseSubscriber{
		name:             name,
		subscribedEvents: events,
		handler:          handler,
	}
}

// HandleEvent обрабатывает событие
func (s *BaseSubscriber) HandleEvent(event Event) error {
	return s.handler(event)
}

// GetName возвращает имя подписчика
func (s *BaseSubscriber) GetName() string {
	return s.name
}

// GetSubscribedEvents возвращает типы событий
func (s *BaseSubscriber) GetSubscribedEvents() []EventType {
	return s.subscribedEvents
}

// ConsoleLoggerSubscriber - подписчик для логирования в консоль
type ConsoleLoggerSubscriber struct {
	BaseSubscriber
}

func NewConsoleLoggerSubscriber() *ConsoleLoggerSubscriber {
	return &ConsoleLoggerSubscriber{
		BaseSubscriber: *NewBaseSubscriber(
			"console_logger",
			[]EventType{
				EventPriceUpdated,
				EventSignalDetected,
				EventError,
			},
			func(event Event) error {
				switch event.Type {
				case EventPriceUpdated:
					data, ok := event.Data.(map[string]interface{})
					if ok {
						log.Printf("💰 Цена обновлена: %v", data)
					}
				case EventSignalDetected:
					log.Printf("📈 Обнаружен сигнал: %v", event.Data)
				case EventError:
					log.Printf("❌ Ошибка: %v", event.Data)
				}
				return nil
			},
		),
	}
}

// TelegramNotifierSubscriber - подписчик для отправки в Telegram
type TelegramNotifierSubscriber struct {
	BaseSubscriber
	telegramBot interface{} // замените на ваш TelegramBot
}

func NewTelegramNotifierSubscriber(bot interface{}) *TelegramNotifierSubscriber {
	return &TelegramNotifierSubscriber{
		BaseSubscriber: *NewBaseSubscriber(
			"telegram_notifier",
			[]EventType{EventSignalDetected},
			func(event Event) error {
				// Логика отправки в Telegram
				log.Printf("🤖 Отправка в Telegram: %v", event.Data)
				return nil
			},
		),
		telegramBot: bot,
	}
}

// StorageSubscriber - подписчик для сохранения в хранилище
type StorageSubscriber struct {
	BaseSubscriber
	storage interface{} // замените на ваше хранилище
}

func NewStorageSubscriber(storage interface{}) *StorageSubscriber {
	return &StorageSubscriber{
		BaseSubscriber: *NewBaseSubscriber(
			"storage_saver",
			[]EventType{EventPriceUpdated},
			func(event Event) error {
				// Логика сохранения в хранилище
				log.Printf("💾 Сохранение в хранилище: %v", event.Data)
				return nil
			},
		),
		storage: storage,
	}
}
