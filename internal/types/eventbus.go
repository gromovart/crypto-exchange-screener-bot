// /internal/types/eventbus.go
package types

import (
	"sync"
	"time"
)

// EventBus - интерфейс шины событий
type EventBus interface {
	// Publish публикует событие
	Publish(event Event) error

	// PublishSync публикует событие синхронно
	PublishSync(event Event) error

	// Subscribe подписывает обработчик на тип события
	Subscribe(eventType EventType, subscriber EventSubscriber)

	// Unsubscribe отписывает обработчика от типа события
	Unsubscribe(eventType EventType, subscriber EventSubscriber)

	// Start запускает EventBus
	Start()

	// Stop останавливает EventBus
	Stop()

	// GetMetrics возвращает метрики
	GetMetrics() *EventBusMetrics
}

// Event - структура события
type Event struct {
	ID        string      `json:"id"`
	Type      EventType   `json:"type"`
	Source    string      `json:"source"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
	Metadata  Metadata    `json:"metadata"`
}

// EventType - тип события
type EventType string

// Metadata - метаданные события
type Metadata struct {
	CorrelationID string            `json:"correlation_id"`
	Priority      int               `json:"priority"`
	Tags          []string          `json:"tags"`
	Properties    map[string]string `json:"properties"`
}

// EventSubscriber - интерфейс подписчика
type EventSubscriber interface {
	HandleEvent(event Event) error
	GetName() string
	GetSubscribedEvents() []EventType
}

// EventBusMetrics - метрики EventBus
type EventBusMetrics struct {
	Mu               sync.RWMutex
	EventsPublished  int64             `json:"events_published"`
	EventsProcessed  int64             `json:"events_processed"`
	EventsFailed     int64             `json:"events_failed"`
	SubscribersCount map[EventType]int `json:"subscribers_count"`
	ProcessingTime   time.Duration     `json:"processing_time"`
}

// Константы типов событий
const (
	EventServiceStarted             EventType = "service_started"
	EventServiceStopped             EventType = "service_stopped"
	EventServiceError               EventType = "service_error"
	EventPriceUpdated               EventType = "price_updated"
	EventSignalDetected             EventType = "signal_detected"
	EventSignalFiltered             EventType = "signal_filtered"
	EventSignalRated                EventType = "signal_rated"
	EventSystemStarted              EventType = "system_started"
	EventSystemStopped              EventType = "system_stopped"
	EventHealthCheck                EventType = "health_check"
	EventError                      EventType = "error"
	EventTelegramSent               EventType = "telegram_sent"
	EventConfigChanged              EventType = "config_changed"
	EventCounterSignalDetected      EventType = "counter_signal_detected"
	EventCounterNotificationRequest EventType = "counter_notification_request"
	EventUserNotification           EventType = "user_notification"
)
