package events

import (
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"log"
	"runtime/debug"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

const maxProcessingDepth = 15

// EventBus - центральная шина событий
type EventBus struct {
	mu          sync.RWMutex
	subscribers map[types.EventType][]types.EventSubscriber
	middlewares []Middleware
	eventBuffer chan types.Event
	config      EventBusConfig
	running     bool
	stopChan    chan struct{}
	wg          sync.WaitGroup

	// Атомарные метрики — без мьютекса
	eventsPublished   atomic.Int64
	eventsProcessed   atomic.Int64
	eventsFailed      atomic.Int64
	processingNsTotal atomic.Int64

	// Защита от рекурсии — только атомарный счётчик
	processingDepth atomic.Int32
}

// EventBusConfig - конфигурация EventBus
type EventBusConfig struct {
	BufferSize      int           `json:"buffer_size"`
	WorkerCount     int           `json:"worker_count"`
	MaxRetries      int           `json:"max_retries"`
	RetryDelay      time.Duration `json:"retry_delay"`
	EnableMetrics   bool          `json:"enable_metrics"`
	EnableLogging   bool          `json:"enable_logging"`
	DeadLetterQueue bool          `json:"dead_letter_queue"`
}

// DefaultConfig - конфигурация по умолчанию
var DefaultConfig = EventBusConfig{
	BufferSize:      1000,
	WorkerCount:     10,
	MaxRetries:      3,
	RetryDelay:      100 * time.Millisecond,
	EnableMetrics:   true,
	EnableLogging:   true,
	DeadLetterQueue: true,
}

// NewEventBus создает новую шину событий
func NewEventBus(config ...EventBusConfig) *EventBus {
	cfg := DefaultConfig
	if len(config) > 0 {
		cfg = config[0]
	}
	bus := &EventBus{
		subscribers: make(map[types.EventType][]types.EventSubscriber),
		middlewares: make([]Middleware, 0),
		eventBuffer: make(chan types.Event, cfg.BufferSize),
		config:      cfg,
		stopChan:    make(chan struct{}),
	}
	if cfg.EnableMetrics {
		bus.startMetricsCollection()
	}
	return bus
}

// Start запускает EventBus
func (b *EventBus) Start() {
	if b.running {
		return
	}
	b.running = true
	for i := 0; i < b.config.WorkerCount; i++ {
		b.wg.Add(1)
		go b.eventWorker(i)
	}
	if b.config.EnableLogging {
		log.Printf("🚀 EventBus запущен с %d обработчиками", b.config.WorkerCount)
	}
}

// Stop останавливает EventBus
func (b *EventBus) Stop() {
	if !b.running {
		return
	}
	b.running = false
	close(b.stopChan)
	b.wg.Wait()
	close(b.eventBuffer)
	if b.config.EnableLogging {
		log.Println("🛑 EventBus остановлен")
	}
}

// Subscribe подписывает обработчик на тип события
func (b *EventBus) Subscribe(eventType types.EventType, subscriber types.EventSubscriber) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, et := range subscriber.GetSubscribedEvents() {
		if et == eventType {
			b.subscribers[eventType] = append(b.subscribers[eventType], subscriber)
			if b.config.EnableLogging {
				log.Printf("✅ %s подписался на %s", subscriber.GetName(), eventType)
			}
			return
		}
	}
	log.Printf("⚠️ Подписчик %s не подписан на событие %s", subscriber.GetName(), eventType)
}

// Unsubscribe отписывает обработчик от типа события
func (b *EventBus) Unsubscribe(eventType types.EventType, subscriber types.EventSubscriber) {
	b.mu.Lock()
	defer b.mu.Unlock()
	subscribers := b.subscribers[eventType]
	for i, sub := range subscribers {
		if sub == subscriber {
			b.subscribers[eventType] = append(subscribers[:i], subscribers[i+1:]...)
			if b.config.EnableLogging {
				log.Printf("❌ %s отписался от %s", subscriber.GetName(), eventType)
			}
			return
		}
	}
}

// Publish публикует событие асинхронно
func (b *EventBus) Publish(event types.Event) error {
	if !b.running {
		return fmt.Errorf("event bus is not running")
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	select {
	case b.eventBuffer <- event:
		b.eventsPublished.Add(1)
		return nil
	default:
		logger.Warn("⚠️ Буфер событий полен, событие отброшено: %s", event.Type)
		return fmt.Errorf("event buffer is full")
	}
}

// PublishSync публикует событие синхронно
func (b *EventBus) PublishSync(event types.Event) error {
	return b.processEvent(event)
}

// AddMiddleware добавляет middleware
func (b *EventBus) AddMiddleware(middleware Middleware) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.middlewares = append(b.middlewares, middleware)
}

// eventWorker - обработчик событий
func (b *EventBus) eventWorker(_ int) {
	defer b.wg.Done()
	for {
		select {
		case event := <-b.eventBuffer:
			b.processEvent(event)
		case <-b.stopChan:
			return
		}
	}
}

// processEvent обрабатывает одно событие
func (b *EventBus) processEvent(event types.Event) error {
	depth := b.processingDepth.Add(1)
	defer b.processingDepth.Add(-1)

	if depth > maxProcessingDepth {
		logger.Warn("⚠️ Достигнута максимальная глубина обработки: %d", depth)
		return fmt.Errorf("max processing depth reached")
	}

	start := time.Now()

	b.mu.RLock()
	subscribers := b.subscribers[event.Type]
	b.mu.RUnlock()

	if len(subscribers) == 0 {
		return nil
	}

	handler := b.createHandlerChain(subscribers)
	err := b.executeWithMiddleware(event, handler)

	b.eventsProcessed.Add(1)
	b.processingNsTotal.Add(time.Since(start).Nanoseconds())
	if err != nil {
		b.eventsFailed.Add(1)
	}
	return err
}

// createHandlerChain создает цепочку обработчиков
func (b *EventBus) createHandlerChain(subscribers []types.EventSubscriber) HandlerFunc {
	return func(event types.Event) error {
		var lastErr error
		for _, sub := range subscribers {
			if err := sub.HandleEvent(event); err != nil {
				lastErr = err
				log.Printf("❌ Ошибка обработки %s подписчиком %s: %v",
					event.Type, sub.GetName(), err)
			}
		}
		return lastErr
	}
}

// executeWithMiddleware выполняет обработку через цепочку middleware
func (b *EventBus) executeWithMiddleware(event types.Event, handler HandlerFunc) error {
	chain := handler
	for i := len(b.middlewares) - 1; i >= 0; i-- {
		mw := b.middlewares[i]
		next := chain
		chain = func(e types.Event) error { return mw.Process(e, next) }
	}
	return chain(event)
}

// GetMetrics возвращает метрики
func (b *EventBus) GetMetrics() *types.EventBusMetrics {
	b.mu.RLock()
	subscribersCount := make(map[types.EventType]int, len(b.subscribers))
	for k, v := range b.subscribers {
		subscribersCount[k] = len(v)
	}
	b.mu.RUnlock()

	processed := b.eventsProcessed.Load()
	var avgProcessing time.Duration
	if processed > 0 {
		avgProcessing = time.Duration(b.processingNsTotal.Load() / processed)
	}
	return &types.EventBusMetrics{
		EventsPublished:  b.eventsPublished.Load(),
		EventsProcessed:  processed,
		EventsFailed:     b.eventsFailed.Load(),
		SubscribersCount: subscribersCount,
		ProcessingTime:   avgProcessing,
	}
}

// GetSubscriberCount возвращает количество подписчиков
func (b *EventBus) GetSubscriberCount(eventType types.EventType) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers[eventType])
}

// GetEventTypes возвращает все типы событий с подписчиками
func (b *EventBus) GetEventTypes() []types.EventType {
	b.mu.RLock()
	defer b.mu.RUnlock()
	var result []types.EventType
	for et := range b.subscribers {
		result = append(result, et)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

// startMetricsCollection запускает периодический лог метрик
func (b *EventBus) startMetricsCollection() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				b.logMetrics()
			case <-b.stopChan:
				return
			}
		}
	}()
}

func (b *EventBus) logMetrics() {
	m := b.GetMetrics()
	logger.Info("📊 EventBus: опубликовано=%d обработано=%d ошибок=%d avg=%v",
		m.EventsPublished, m.EventsProcessed, m.EventsFailed, m.ProcessingTime)
}

// safeExecute безопасно выполняет функцию с обработкой паники
func (b *EventBus) safeExecute(fn func()) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("⚠️ Паника восстановлена: %v\n%s", r, debug.Stack())
			_ = b.Publish(types.Event{
				Type:   types.EventError,
				Source: "event_bus",
				Data:   fmt.Sprintf("Panic recovered: %v", r),
			})
		}
	}()
	fn()
}

// IsRunning возвращает true если EventBus запущен
func (b *EventBus) IsRunning() bool { return b.running }

// Name возвращает имя сервиса
func (b *EventBus) Name() string { return "EventBus" }

// HealthCheck проверяет здоровье сервиса
func (b *EventBus) HealthCheck() bool {
	if !b.running || b.eventBuffer == nil {
		return false
	}
	select {
	case <-b.stopChan:
		return false
	default:
		return true
	}
}

// GetMetricsMap возвращает метрики в виде map
func (b *EventBus) GetMetricsMap() map[string]interface{} {
	m := b.GetMetrics()
	return map[string]interface{}{
		"events_published": m.EventsPublished,
		"events_processed": m.EventsProcessed,
		"events_failed":    m.EventsFailed,
		"processing_time":  m.ProcessingTime.String(),
		"subscribers":      m.SubscribersCount,
	}
}

// GetMiddlewares возвращает список middleware
func (b *EventBus) GetMiddlewares() []Middleware {
	b.mu.RLock()
	defer b.mu.RUnlock()
	result := make([]Middleware, len(b.middlewares))
	copy(result, b.middlewares)
	return result
}

// ClearMiddlewares очищает все middleware
func (b *EventBus) ClearMiddlewares() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.middlewares = []Middleware{}
}

// ClearEventChain — оставлен для совместимости (no-op)
func (b *EventBus) ClearEventChain() {}

// unused — подавляем предупреждение компилятора
var _ = atomic.Int32{}
