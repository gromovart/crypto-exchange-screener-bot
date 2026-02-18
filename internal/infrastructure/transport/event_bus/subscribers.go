// internal/infrastructure/transport/event_bus/subscribers.go
package events

import (
	"crypto-exchange-screener-bot/internal/delivery/telegram/controllers/counter"
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
)

// BaseSubscriber - базовая реализация подписчика
type BaseSubscriber struct {
	name             string
	subscribedEvents []types.EventType
	handler          func(types.Event) error
}

// NewBaseSubscriber создает нового подписчика
func NewBaseSubscriber(name string, events []types.EventType, handler func(types.Event) error) *BaseSubscriber {
	return &BaseSubscriber{
		name:             name,
		subscribedEvents: events,
		handler:          handler,
	}
}

// HandleEvent обрабатывает событие
func (s *BaseSubscriber) HandleEvent(event types.Event) error {
	return s.handler(event)
}

// GetName возвращает имя подписчика
func (s *BaseSubscriber) GetName() string {
	return s.name
}

// GetSubscribedEvents возвращает типы событий
func (s *BaseSubscriber) GetSubscribedEvents() []types.EventType {
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
			[]types.EventType{
				types.EventPriceUpdated,
				types.EventSignalDetected,
				types.EventError,
			},
			func(event types.Event) error {
				switch event.Type {
				case types.EventPriceUpdated:
					data, ok := event.Data.(map[string]interface{})
					if ok {
						logger.Info("💰 Цена обновлена: %v", data)
					}
				case types.EventSignalDetected:
					logger.Info("📈 Обнаружен сигнал: %v", event.Data)
				case types.EventError:
					logger.Info("❌ Ошибка: %v", event.Data)
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
			[]types.EventType{types.EventSignalDetected},
			func(event types.Event) error {
				// Логика отправки в Telegram
				logger.Info("🤖 Отправка в Telegram: %v", event.Data)
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
			[]types.EventType{types.EventPriceUpdated},
			func(event types.Event) error {
				// Логика сохранения в хранилище
				logger.Info("💾 Сохранение в хранилище: %v", event.Data)
				return nil
			},
		),
		storage: storage,
	}
}

// CounterControllerSubscriber - подписчик для контроллера счетчика
type CounterControllerSubscriber struct {
	BaseSubscriber
	counterController counter.Controller
}

// NewCounterControllerSubscriber создает нового подписчика для контроллера счетчика
func NewCounterControllerSubscriber(controller counter.Controller) *CounterControllerSubscriber {
	return &CounterControllerSubscriber{
		BaseSubscriber: *NewBaseSubscriber(
			"counter_controller",
			[]types.EventType{types.EventCounterSignalDetected},
			func(event types.Event) error {
				// Делегируем обработку контроллеру счетчика
				return controller.HandleEvent(event)
			},
		),
		counterController: controller,
	}
}

// CounterControllerWrapper - обертка для контроллера счетчика как подписчика
// (альтернативная реализация без BaseSubscriber)
type CounterControllerWrapper struct {
	controller counter.Controller
}

// NewCounterControllerWrapper создает обертку для контроллера счетчика
func NewCounterControllerWrapper(controller counter.Controller) *CounterControllerWrapper {
	return &CounterControllerWrapper{
		controller: controller,
	}
}

// HandleEvent обрабатывает событие
func (w *CounterControllerWrapper) HandleEvent(event types.Event) error {
	return w.controller.HandleEvent(event)
}

// GetName возвращает имя подписчика
func (w *CounterControllerWrapper) GetName() string {
	return w.controller.GetName()
}

// GetSubscribedEvents возвращает типы событий
func (w *CounterControllerWrapper) GetSubscribedEvents() []types.EventType {
	return w.controller.GetSubscribedEvents()
}

// ==================== НОВЫЕ ПОДПИСЧИКИ ДЛЯ ПЛАТЕЖЕЙ ====================

// PaymentControllerSubscriber - подписчик для контроллера платежей
type PaymentControllerSubscriber struct {
	BaseSubscriber
	paymentController types.EventSubscriber
}

// NewPaymentControllerSubscriber создает нового подписчика для контроллера платежей
func NewPaymentControllerSubscriber(controller types.EventSubscriber) *PaymentControllerSubscriber {
	return &PaymentControllerSubscriber{
		BaseSubscriber: *NewBaseSubscriber(
			"payment_controller",
			[]types.EventType{
				types.EventPaymentComplete,
				types.EventPaymentCreated,
				types.EventPaymentFailed,
				types.EventPaymentRefunded,
			},
			func(event types.Event) error {
				logger.Warn("💰 [PAYMENT SUBSCRIBER] Получено событие: %s", event.Type)
				return controller.HandleEvent(event)
			},
		),
		paymentController: controller,
	}
}

// PaymentControllerWrapper - обертка для контроллера платежей как подписчика
type PaymentControllerWrapper struct {
	controller types.EventSubscriber
}

// NewPaymentControllerWrapper создает обертку для контроллера платежей
func NewPaymentControllerWrapper(controller types.EventSubscriber) *PaymentControllerWrapper {
	return &PaymentControllerWrapper{
		controller: controller,
	}
}

// HandleEvent обрабатывает событие
func (w *PaymentControllerWrapper) HandleEvent(event types.Event) error {
	return w.controller.HandleEvent(event)
}

// GetName возвращает имя подписчика
func (w *PaymentControllerWrapper) GetName() string {
	return w.controller.GetName()
}

// GetSubscribedEvents возвращает типы событий
func (w *PaymentControllerWrapper) GetSubscribedEvents() []types.EventType {
	return w.controller.GetSubscribedEvents()
}
