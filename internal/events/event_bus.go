// internal/events/event_bus.go
package events

import (
	"fmt"
	"log"
	"runtime/debug"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// EventBus - центральная шина событий
type EventBus struct {
	mu          sync.RWMutex
	subscribers map[EventType][]Subscriber
	middlewares []Middleware
	eventBuffer chan Event
	metrics     *EventMetrics
	config      EventBusConfig
	running     bool
	stopChan    chan struct{}
	wg          sync.WaitGroup
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

// EventMetrics - метрики EventBus
type EventMetrics struct {
	mu               sync.RWMutex
	EventsPublished  int64             `json:"events_published"`
	EventsProcessed  int64             `json:"events_processed"`
	EventsFailed     int64             `json:"events_failed"`
	SubscribersCount map[EventType]int `json:"subscribers_count"`
	ProcessingTime   time.Duration     `json:"processing_time"`
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
		subscribers: make(map[EventType][]Subscriber),
		middlewares: make([]Middleware, 0),
		eventBuffer: make(chan Event, cfg.BufferSize),
		metrics: &EventMetrics{
			SubscribersCount: make(map[EventType]int),
		},
		config:   cfg,
		stopChan: make(chan struct{}),
		running:  false,
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

	// Запускаем обработчиков событий
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
func (b *EventBus) Subscribe(eventType EventType, subscriber Subscriber) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Проверяем, что подписчик подписан на этот тип события
	subscribedEvents := subscriber.GetSubscribedEvents()
	found := false
	for _, et := range subscribedEvents {
		if et == eventType {
			found = true
			break
		}
	}

	if !found {
		log.Printf("⚠️ Подписчик %s не подписан на событие %s",
			subscriber.GetName(), eventType)
		return
	}

	// Добавляем подписчика
	b.subscribers[eventType] = append(b.subscribers[eventType], subscriber)

	// Обновляем метрики
	b.metrics.SubscribersCount[eventType] = len(b.subscribers[eventType])

	if b.config.EnableLogging {
		log.Printf("✅ %s подписался на %s",
			subscriber.GetName(), eventType)
	}
}

// Unsubscribe отписывает обработчик от типа события
func (b *EventBus) Unsubscribe(eventType EventType, subscriber Subscriber) {
	b.mu.Lock()
	defer b.mu.Unlock()

	subscribers, exists := b.subscribers[eventType]
	if !exists {
		return
	}

	for i, sub := range subscribers {
		if sub == subscriber {
			b.subscribers[eventType] = append(subscribers[:i], subscribers[i+1:]...)

			// Обновляем метрики
			b.metrics.SubscribersCount[eventType] = len(b.subscribers[eventType])

			if b.config.EnableLogging {
				log.Printf("❌ %s отписался от %s",
					subscriber.GetName(), eventType)
			}
			return
		}
	}
}

// Publish публикует событие
func (b *EventBus) Publish(event Event) error {
	if !b.running {
		return fmt.Errorf("event bus is not running")
	}

	// Устанавливаем ID и временную метку если они не установлены
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	select {
	case b.eventBuffer <- event:
		// Обновляем метрики
		b.metrics.mu.Lock()
		b.metrics.EventsPublished++
		b.metrics.mu.Unlock()

		if b.config.EnableLogging && event.Type != EventPriceUpdated {
			log.Printf("📤 Опубликовано событие: %s от %s",
				event.Type, event.Source)
		}
		return nil
	default:
		// Буфер полон
		if b.config.EnableLogging {
			log.Printf("⚠️ Буфер событий полон, событие отброшено: %s", event.Type)
		}
		return fmt.Errorf("event buffer is full")
	}
}

// PublishSync публикует событие синхронно
func (b *EventBus) PublishSync(event Event) error {
	return b.processEvent(event)
}

// AddMiddleware добавляет middleware
func (b *EventBus) AddMiddleware(middleware Middleware) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.middlewares = append(b.middlewares, middleware)

	if b.config.EnableLogging {
		log.Printf("➕ Добавлен middleware: %T", middleware)
	}
}

// eventWorker - обработчик событий
func (b *EventBus) eventWorker(id int) {
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
func (b *EventBus) processEvent(event Event) error {
	startTime := time.Now()
	defer func() {
		// Обновляем метрики времени обработки
		b.metrics.mu.Lock()
		b.metrics.ProcessingTime += time.Since(startTime)
		b.metrics.EventsProcessed++
		b.metrics.mu.Unlock()
	}()

	// Получаем подписчиков для этого типа события
	b.mu.RLock()
	subscribers, exists := b.subscribers[event.Type]
	b.mu.RUnlock()

	if !exists || len(subscribers) == 0 {
		if b.config.EnableLogging {
			log.Printf("⚠️ Нет подписчиков для события: %s", event.Type)
		}
		return nil
	}

	// Создаем цепочку middleware
	handler := b.createHandlerChain(subscribers)

	// Запускаем обработку через middleware
	return b.executeWithMiddleware(event, handler)
}

// createHandlerChain создает цепочку обработчиков
func (b *EventBus) createHandlerChain(subscribers []Subscriber) HandlerFunc {
	return func(event Event) error {
		var lastError error

		for _, subscriber := range subscribers {
			// Обрабатываем событие в отдельной горутине для каждого подписчика
			go func(s Subscriber) {
				if err := b.handleEventWithRetry(event, s); err != nil {
					lastError = err
					log.Printf("❌ Ошибка обработки события %s подписчиком %s: %v",
						event.Type, s.GetName(), err)
				}
			}(subscriber)
		}

		return lastError
	}
}

// handleEventWithRetry обрабатывает событие с повторными попытками
func (b *EventBus) handleEventWithRetry(event Event, subscriber Subscriber) error {
	var lastError error

	for attempt := 1; attempt <= b.config.MaxRetries; attempt++ {
		err := subscriber.HandleEvent(event)
		if err == nil {
			return nil
		}

		lastError = err

		if attempt < b.config.MaxRetries {
			time.Sleep(b.config.RetryDelay * time.Duration(attempt))
		}
	}

	// Увеличиваем счетчик ошибок
	b.metrics.mu.Lock()
	b.metrics.EventsFailed++
	b.metrics.mu.Unlock()

	return lastError
}

// executeWithMiddleware выполняет обработку через цепочку middleware
func (b *EventBus) executeWithMiddleware(event Event, handler HandlerFunc) error {
	// Создаем цепочку middleware
	chain := handler
	for i := len(b.middlewares) - 1; i >= 0; i-- {
		mw := b.middlewares[i]
		next := chain
		chain = func(event Event) error {
			return mw.Process(event, next)
		}
	}

	// Запускаем цепочку
	return chain(event)
}

// GetMetrics возвращает метрики
func (b *EventBus) GetMetrics() EventMetrics {
	b.metrics.mu.RLock()
	defer b.metrics.mu.RUnlock()

	return *b.metrics
}

// GetSubscriberCount возвращает количество подписчиков
func (b *EventBus) GetSubscriberCount(eventType EventType) int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return len(b.subscribers[eventType])
}

// GetEventTypes возвращает все типы событий с подписчиками
func (b *EventBus) GetEventTypes() []EventType {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var types []EventType
	for eventType := range b.subscribers {
		types = append(types, eventType)
	}

	sort.Slice(types, func(i, j int) bool {
		return types[i] < types[j]
	})

	return types
}

// startMetricsCollection запускает сбор метрик
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

// logMetrics логирует метрики
func (b *EventBus) logMetrics() {
	metrics := b.GetMetrics()

	log.Printf("📊 EventBus метрики:")
	log.Printf("   Опубликовано: %d событий", metrics.EventsPublished)
	log.Printf("   Обработано: %d событий", metrics.EventsProcessed)
	log.Printf("   Ошибок: %d событий", metrics.EventsFailed)
	log.Printf("   Среднее время обработки: %v",
		metrics.ProcessingTime/time.Duration(metrics.EventsProcessed))

	for eventType, count := range metrics.SubscribersCount {
		log.Printf("   %s: %d подписчиков", eventType, count)
	}
}

// safeExecute безопасно выполняет функцию с обработкой паники
func (b *EventBus) safeExecute(fn func()) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("⚠️ Паника восстановлена: %v\n%s", r, debug.Stack())

			// Публикуем событие об ошибке
			b.Publish(Event{
				Type:   EventError,
				Source: "event_bus",
				Data:   fmt.Sprintf("Panic recovered: %v", r),
			})
		}
	}()

	fn()
}
