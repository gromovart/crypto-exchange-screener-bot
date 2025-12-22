// internal/types/events/event.go
package events

import (
	"crypto_exchange_screener_bot/internal/types/analysis"
	"crypto_exchange_screener_bot/internal/types/common"

	"log"
	"time"
)

// EventType - тип события
type EventType string

const (
	EventTypeSignal        EventType = "signal"
	EventTypePriceUpdate   EventType = "price_update"
	EventTypeAnalysisStart EventType = "analysis_start"
	EventTypeAnalysisEnd   EventType = "analysis_end"
	EventTypeError         EventType = "error"
	EventTypeCounterAlert  EventType = "counter_alert"

	// События сервисов
	EventServiceStarted EventType = "service_started"
	EventServiceStopped EventType = "service_stopped"
	EventServiceError   EventType = "service_error"

	// События данных
	EventPriceUpdated   EventType = "price_updated"
	EventSignalDetected EventType = "signal_detected"
	EventSignalFiltered EventType = "signal_filtered"
	EventSignalRated    EventType = "signal_rated"
	EventSymbolAdded    EventType = "symbol_added"
	EventSymbolRemoved  EventType = "symbol_removed"

	// События системы
	EventSystemStarted EventType = "system_started"
	EventSystemStopped EventType = "system_stopped"
	EventHealthCheck   EventType = "health_check"
	EventError         EventType = "error"

	// События интеграций
	EventTelegramSent  EventType = "telegram_sent"
	EventConfigChanged EventType = "config_changed"

	// События анализа
	EventAnalysisStarted EventType = "analysis_started"
	EventAnalysisEnded   EventType = "analysis_ended"
)

// Event - базовое событие
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Source    string                 `json:"source"`
	Timestamp time.Time              `json:"timestamp"`
	Payload   interface{}            `json:"payload"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// SignalEvent - событие сигнала
type SignalEvent struct {
	Signal    analysis.Signal `json:"signal"`
	Processed bool            `json:"processed"`
}

// PriceEvent - событие обновления цены
type PriceEvent struct {
	Data      common.PriceData `json:"data"`
	Processed bool             `json:"processed"`
}

// ErrorEvent - событие ошибки
type ErrorEvent struct {
	Error     error  `json:"error"`
	Component string `json:"component"`
	Context   string `json:"context,omitempty"`
}

// EventHandler - обработчик событий
type EventHandler func(Event) error

// Metadata - метаданные события
type Metadata struct {
	CorrelationID string            `json:"correlation_id"`
	Priority      int               `json:"priority"`
	Tags          []string          `json:"tags"`
	Properties    map[string]string `json:"properties"`
}

// Subscriber - интерфейс подписчика
type Subscriber interface {
	HandleEvent(event Event) error
	GetName() string
	GetSubscribedEvents() []EventType
}

// Middleware - промежуточное ПО для обработки событий
type Middleware interface {
	Process(event Event, next HandlerFunc) error
}

// HandlerFunc - функция обработки события
type HandlerFunc func(event Event) error

// Subscription - подписка на события
type Subscription struct {
	ID           string    `json:"id"`
	SubscriberID string    `json:"subscriber_id"`
	EventType    EventType `json:"event_type"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
}

// BaseSubscriber - локальная реализация базового подписчика
type BaseSubscriber struct {
	name             string
	subscribedEvents []EventType
	handler          func(Event) error
}

// NewBaseSubscriber создает нового подписчика
func NewBaseSubscriber(name string, eventTypes []EventType, handler func(Event) error) *BaseSubscriber {
	return &BaseSubscriber{
		name:             name,
		subscribedEvents: eventTypes,
		handler:          handler,
	}
}

// HandleEvent обрабатывает событие
func (s *BaseSubscriber) HandleEvent(event Event) error {
	if s.handler != nil {
		return s.handler(event)
	}
	return nil
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
					data, ok := event.Data["data"].(common.PriceData)
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
