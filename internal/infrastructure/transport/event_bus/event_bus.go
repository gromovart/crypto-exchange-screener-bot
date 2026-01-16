// internal/infrastructure/transport/event_bus/event_bus.go
package events

import (
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
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
	subscribers map[types.EventType][]types.EventSubscriber
	middlewares []Middleware
	eventBuffer chan types.Event
	metrics     *types.EventBusMetrics
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
		metrics: &types.EventBusMetrics{
			SubscribersCount: make(map[types.EventType]int),
		},
		config:   cfg,
		stopChan: make(chan struct{}),
		running:  false,
	}

	if cfg.EnableMetrics {
		bus.startMetricsCollection()
	}

	// 🔴 ДОБАВЬТЕ ОТЛАДОЧНЫЙ ВЫВОД:
	logger.Info("🔍 EventBus config: MaxRetries=%d, RetryDelay=%v\n",
		cfg.MaxRetries, cfg.RetryDelay)

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
func (b *EventBus) Subscribe(eventType types.EventType, subscriber types.EventSubscriber) {
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
func (b *EventBus) Unsubscribe(eventType types.EventType, subscriber types.EventSubscriber) {
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
func (b *EventBus) Publish(event types.Event) error {
	if !b.running {
		return fmt.Errorf("event bus is not running")
	}

	logger.Debug("[EventBus.Publish] Публикую %s от %s", event.Type, event.Source)
	logger.Debug("📤 Опубликовано событие: %s от %s", event.Type, event.Source)

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
		b.metrics.Mu.Lock()
		b.metrics.EventsPublished++
		b.metrics.Mu.Unlock()

		if b.config.EnableLogging && event.Type != types.EventPriceUpdated {
			logger.Debug("📤 Опубликовано событие: %s от %s",
				event.Type, event.Source)
		}
		logger.Debug("✅ [EventBus.Publish] Событие %s добавлено в буфер\n", event.Type)
		return nil
	default:
		// Буфер полен
		if b.config.EnableLogging {
			logger.Warn("⚠️ Буфер событий полен, событие отброшено: %s", event.Type)
		}
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

	if b.config.EnableLogging {
		log.Printf("➕ Добавлен middleware: %T", middleware)
	}
}

// eventWorker - обработчик событий
func (b *EventBus) eventWorker(id int) {
	defer b.wg.Done()

	logger.Info("🔍 [EventWorker %d] Запущен\n", id)

	for {
		select {
		case event := <-b.eventBuffer:
			logger.Debug("🔍 [EventWorker %d] Получил событие %s из буфера\n", id, event.Type)
			b.processEvent(event)
		case <-b.stopChan:
			logger.Info("🔍 [EventWorker %d] Остановлен\n", id)
			return
		}
	}
}

// processEvent обрабатывает одно событие
func (b *EventBus) processEvent(event types.Event) error {
	startTime := time.Now()

	// 🔴 ДОБАВЬТЕ ОТЛАДОЧНЫЙ ВЫВОД:
	logger.Debug("🔍 EventBus.processEvent: обработка %s от %s\n", event.Type, event.Source)

	defer func() {
		// Обновляем метрики времени обработки
		b.metrics.Mu.Lock()
		b.metrics.ProcessingTime += time.Since(startTime)
		b.metrics.EventsProcessed++
		b.metrics.Mu.Unlock()

		// 🔴 ДОБАВЬТЕ:
		logger.Debug("✅ EventBus.processEvent: %s обработано за %v\n",
			event.Type, time.Since(startTime))
	}()

	// Получаем подписчиков для этого типа события
	b.mu.RLock()
	subscribers, exists := b.subscribers[event.Type]
	b.mu.RUnlock()

	logger.Debug("🔍 EventBus.processEvent: найдено %d подписчиков для %s\n",
		len(subscribers), event.Type) // 🔴 ДОБАВЬТЕ

	if !exists || len(subscribers) == 0 {
		if b.config.EnableLogging {
			logger.Warn("⚠️ Нет подписчиков для события: %s", event.Type)
		}
		return nil
	}
	// Создаем цепочку middleware
	handler := b.createHandlerChain(subscribers)

	// Запускаем обработку через middleware
	return b.executeWithMiddleware(event, handler)
}

// createHandlerChain создает цепочку обработчиков
func (b *EventBus) createHandlerChain(subscribers []types.EventSubscriber) HandlerFunc {
	return func(event types.Event) error {
		logger.Debug("🔍 [createHandlerChain] Начало обработки %s для %d подписчиков\n",
			event.Type, len(subscribers))

		var lastError error

		for i, subscriber := range subscribers {
			logger.Debug("🔍 [createHandlerChain] Обработка подписчика [%d] %s\n",
				i, subscriber.GetName())

			if err := b.handleEventWithRetry(event, subscriber); err != nil {
				logger.Debug("❌ [createHandlerChain] Ошибка от %s: %v\n",
					subscriber.GetName(), err)
				lastError = err
				log.Printf("❌ Ошибка обработки события %s подписчиком %s: %v",
					event.Type, subscriber.GetName(), err)
			} else {
				logger.Debug("✅ [createHandlerChain] %s успешно обработал %s\n",
					subscriber.GetName(), event.Type)
			}
		}

		logger.Debug("🔍 [createHandlerChain] Завершение обработки %s, ошибка: %v\n",
			event.Type, lastError)
		return lastError
	}
}

// handleEventWithRetry обрабатывает событие с повторными попытками
func (b *EventBus) handleEventWithRetry(event types.Event, subscriber types.EventSubscriber) error {
	logger.Debug("🔍 [handleEventWithRetry] Вызов %s для события %s\n",
		subscriber.GetName(), event.Type)

	// Просто вызываем обработчик
	err := subscriber.HandleEvent(event)

	if err != nil {
		logger.Info("❌ [handleEventWithRetry] Ошибка от %s: %v\n",
			subscriber.GetName(), err)
		b.metrics.Mu.Lock()
		b.metrics.EventsFailed++
		b.metrics.Mu.Unlock()
		return err
	}

	logger.Debug("✅ [handleEventWithRetry] %s успешно обработал %s\n",
		subscriber.GetName(), event.Type)
	return nil
}

// executeWithMiddleware выполняет обработку через цепочку middleware
func (b *EventBus) executeWithMiddleware(event types.Event, handler HandlerFunc) error {
	// Создаем цепочку middleware
	chain := handler
	for i := len(b.middlewares) - 1; i >= 0; i-- {
		mw := b.middlewares[i]
		next := chain
		chain = func(event types.Event) error {
			logger.Debug("🔍 [executeWithMiddleware] Вызов middleware %T\n", mw)
			return mw.Process(event, next)
		}
	}

	logger.Debug("🔍 [executeWithMiddleware] Запуск цепочки для %s\n", event.Type)

	// Запускаем цепочку
	return chain(event)
}

// GetMetrics возвращает метрики
func (b *EventBus) GetMetrics() types.EventBusMetrics {
	b.metrics.Mu.RLock()
	defer b.metrics.Mu.RUnlock()

	return *b.metrics
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

	var types []types.EventType
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

	logger.Info("📊 EventBus метрики:")
	logger.Info("   Опубликовано: %d событий", metrics.EventsPublished)
	logger.Info("   Обработано: %d событий", metrics.EventsProcessed)
	logger.Info("   Ошибок: %d событий", metrics.EventsFailed)

	// ИСПРАВЛЕНИЕ: проверка деления на ноль
	var avgProcessingTime time.Duration
	if metrics.EventsProcessed > 0 {
		avgProcessingTime = metrics.ProcessingTime / time.Duration(metrics.EventsProcessed)
		logger.Info("   Среднее время обработки: %v", avgProcessingTime)
	} else {
		logger.Info("   Среднее время обработки: нет данных (0 событий)")
	}

	for eventType, count := range metrics.SubscribersCount {
		logger.Info("   %s: %d подписчиков", eventType, count)
	}
}

// safeExecute безопасно выполняет функцию с обработкой паники
func (b *EventBus) safeExecute(fn func()) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("⚠️ Паника восстановлена: %v\n%s", r, debug.Stack())

			// Публикуем событие об ошибке
			b.Publish(types.Event{
				Type:   types.EventError,
				Source: "event_bus",
				Data:   fmt.Sprintf("Panic recovered: %v", r),
			})
		}
	}()

	fn()
}

// GetMiddlewares возвращает список middleware (для отладки)
func (b *EventBus) GetMiddlewares() []Middleware {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Создаем копию
	result := make([]Middleware, len(b.middlewares))
	copy(result, b.middlewares)
	return result
}

// ClearMiddlewares очищает все middleware
func (b *EventBus) ClearMiddlewares() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.middlewares = []Middleware{}

	if b.config.EnableLogging {
		log.Println("✅ Все middleware удалены из EventBus")
	}
}

// IsRunning возвращает true если EventBus запущен
func (b *EventBus) IsRunning() bool {
	return b.running
}

// Name возвращает имя сервиса
func (b *EventBus) Name() string {
	return "EventBus"
}

// HealthCheck проверяет здоровье сервиса
func (b *EventBus) HealthCheck() bool {
	// Базовые проверки
	if !b.running {
		return false
	}
	if b.eventBuffer == nil {
		return false
	}

	// Проверяем что канал остановки не закрыт
	select {
	case <-b.stopChan:
		return false
	default:
		return true
	}
}

// GetMetricsMap возвращает метрики в виде map (для совместимости)
func (b *EventBus) GetMetricsMap() map[string]interface{} {
	metrics := b.GetMetrics()
	return map[string]interface{}{
		"events_published": metrics.EventsPublished,
		"events_processed": metrics.EventsProcessed,
		"events_failed":    metrics.EventsFailed,
		"processing_time":  metrics.ProcessingTime.String(),
		"subscribers":      metrics.SubscribersCount,
	}
}
