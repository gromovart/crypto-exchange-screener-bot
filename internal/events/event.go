// internal/events/event.go
package events

import (
	"time"
)

// EventType - тип события
type EventType string

const (
	// События сервисов
	EventServiceStarted EventType = "service_started"
	EventServiceStopped EventType = "service_stopped"
	EventServiceError   EventType = "service_error"

	// События данных
	EventPriceUpdated   EventType = "price_updated"
	EventSignalDetected EventType = "signal_detected"
	EventSignalFiltered EventType = "signal_filtered"
	EventSignalRated    EventType = "signal_rated"

	// События системы
	EventSystemStarted EventType = "system_started"
	EventSystemStopped EventType = "system_stopped"
	EventHealthCheck   EventType = "health_check"
	EventError         EventType = "error"

	// События интеграций
	EventTelegramSent  EventType = "telegram_sent"
	EventConfigChanged EventType = "config_changed"
)

// Event - структура события
type Event struct {
	ID        string      `json:"id"`
	Type      EventType   `json:"type"`
	Source    string      `json:"source"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
	Metadata  Metadata    `json:"metadata"`
}

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
