// internal/infrastructure/transport/event_bus/middleware.go
package events

import (
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"log"
	"sync"
	"time"
)

// LoggingMiddleware - middleware для логирования (только ошибки)
type LoggingMiddleware struct{}

func (m *LoggingMiddleware) Process(event types.Event, next HandlerFunc) error {
	err := next(event)
	if err != nil {
		log.Printf("❌ [LoggingMiddleware] Ошибка обработки %s: %v", event.Type, err)
	}
	return err
}

// MetricsMiddleware - middleware для сбора метрик (no-op: метрики теперь в EventBus)
type MetricsMiddleware struct {
	metrics *types.EventBusMetrics
}

func (m *MetricsMiddleware) Process(event types.Event, next HandlerFunc) error {
	return next(event)
}

// RateLimitingMiddleware - middleware для ограничения частоты
type RateLimitingMiddleware struct {
	limits   map[types.EventType]time.Duration
	lastCall map[types.EventType]time.Time
	mu       sync.RWMutex
}

func NewRateLimitingMiddleware(limits map[types.EventType]time.Duration) *RateLimitingMiddleware {
	return &RateLimitingMiddleware{
		limits:   limits,
		lastCall: make(map[types.EventType]time.Time),
	}
}

func (m *RateLimitingMiddleware) Process(event types.Event, next HandlerFunc) error {
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

func (m *ValidationMiddleware) Process(event types.Event, next HandlerFunc) error {
	logger.Info("🔍 [ValidationMiddleware] Проверка %s от %s\n",
		event.Type, event.Source)

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

	logger.Info("✅ [ValidationMiddleware] Все проверки пройдены, вызываю next\n")

	// 🔴 ВЫЗЫВАЕМ next В ЛЮБОМ СЛУЧАЕ!
	return next(event)
}
