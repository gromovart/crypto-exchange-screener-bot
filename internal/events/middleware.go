// internal/events/middleware.go
package events

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// LoggingMiddleware - middleware для логирования
type LoggingMiddleware struct{}

func (m *LoggingMiddleware) Process(event Event, next HandlerFunc) error {
	start := time.Now()

	log.Printf("➡️  Начало обработки события: %s от %s",
		event.Type, event.Source)

	err := next(event)

	duration := time.Since(start)

	if err != nil {
		log.Printf("❌ Ошибка обработки события %s за %v: %v",
			event.Type, duration, err)
	} else {
		log.Printf("✅ Событие %s обработано за %v",
			event.Type, duration)
	}

	return err
}

// MetricsMiddleware - middleware для сбора метрик
type MetricsMiddleware struct {
	metrics *EventMetrics
}

func (m *MetricsMiddleware) Process(event Event, next HandlerFunc) error {
	start := time.Now()

	err := next(event)

	duration := time.Since(start)

	m.metrics.mu.Lock()
	m.metrics.ProcessingTime += duration
	m.metrics.mu.Unlock()

	return err
}

// RateLimitingMiddleware - middleware для ограничения частоты
type RateLimitingMiddleware struct {
	limits   map[EventType]time.Duration
	lastCall map[EventType]time.Time
	mu       sync.RWMutex
}

func NewRateLimitingMiddleware(limits map[EventType]time.Duration) *RateLimitingMiddleware {
	return &RateLimitingMiddleware{
		limits:   limits,
		lastCall: make(map[EventType]time.Time),
	}
}

func (m *RateLimitingMiddleware) Process(event Event, next HandlerFunc) error {
	m.mu.RLock()
	limit, hasLimit := m.limits[event.Type]
	last, hasLast := m.lastCall[event.Type]
	m.mu.RUnlock()

	if hasLimit && hasLast {
		sinceLast := time.Since(last)
		if sinceLast < limit {
			// Пропускаем событие из-за ограничения частоты
			log.Printf("⏳ Пропуск события %s (лимит частоты)", event.Type)
			return nil
		}
	}

	m.mu.Lock()
	m.lastCall[event.Type] = time.Now()
	m.mu.Unlock()

	return next(event)
}

// ValidationMiddleware - middleware для валидации событий
type ValidationMiddleware struct{}

func (m *ValidationMiddleware) Process(event Event, next HandlerFunc) error {
	// Проверяем обязательные поля
	if event.Type == "" {
		return fmt.Errorf("event type is required")
	}

	if event.Source == "" {
		return fmt.Errorf("event source is required")
	}

	if event.Timestamp.IsZero() {
		return fmt.Errorf("event timestamp is required")
	}

	return next(event)
}
