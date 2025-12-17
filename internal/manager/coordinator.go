package manager

import (
	"crypto-exchange-screener-bot/internal/storage"
	"log"
	"sync"
	"time"
)

// EventCoordinator координатор событий
type EventCoordinator struct {
	mu            sync.RWMutex
	subscribers   []DataSubscriber
	eventBuffer   []Event
	bufferSize    int
	enableLogging bool
	eventChan     chan Event
	stopChan      chan struct{}
	wg            sync.WaitGroup // Добавляем WaitGroup для graceful shutdown
}

// NewEventCoordinator создает нового координатора событий
func NewEventCoordinator(config CoordinatorConfig) *EventCoordinator {
	coordinator := &EventCoordinator{
		subscribers:   make([]DataSubscriber, 0),
		eventBuffer:   make([]Event, 0),
		bufferSize:    config.EventBufferSize,
		enableLogging: config.EnableEventLogging,
		eventChan:     make(chan Event, 1000), // Увеличиваем буфер
		stopChan:      make(chan struct{}),
	}

	if coordinator.bufferSize <= 0 {
		coordinator.bufferSize = 1000
	}

	// Запускаем обработчик событий
	coordinator.wg.Add(1)
	go coordinator.eventHandler()

	return coordinator
}

// Subscribe подписывает на события
func (ec *EventCoordinator) Subscribe(subscriber DataSubscriber) {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	ec.subscribers = append(ec.subscribers, subscriber)

	// Логируем подписку
	ec.logEvent(Event{
		Type:      EventServiceStarted,
		Service:   "EventCoordinator",
		Message:   "New subscriber registered",
		Timestamp: time.Now(),
		Severity:  "info",
	})
}

// Unsubscribe отписывает от событий
func (ec *EventCoordinator) Unsubscribe(subscriber DataSubscriber) {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	for i, sub := range ec.subscribers {
		if sub == subscriber {
			ec.subscribers = append(ec.subscribers[:i], ec.subscribers[i+1:]...)
			break
		}
	}
}

// PublishEvent публикует событие
func (ec *EventCoordinator) PublishEvent(event Event) {
	// Если событие не имеет временной метки, добавляем
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	select {
	case ec.eventChan <- event:
		// Событие отправлено
	default:
		// Буфер полон, логируем потерю события
		ec.logEvent(Event{
			Type:      EventServiceError,
			Service:   "EventCoordinator",
			Message:   "Event buffer overflow, event dropped",
			Timestamp: time.Now(),
			Severity:  "warning",
		})
	}
}

// PublishPriceUpdate публикует обновление цены
func (ec *EventCoordinator) PublishPriceUpdate(symbol string, price, volume float64, timestamp time.Time) {
	// Отправляем событие в канал вместо прямого уведомления
	event := Event{
		Type:      EventPriceUpdated,
		Service:   "PriceMonitor",
		Message:   "Price updated for " + symbol,
		Data:      map[string]interface{}{"symbol": symbol, "price": price, "volume": volume},
		Timestamp: timestamp,
		Severity:  "info",
	}

	select {
	case ec.eventChan <- event:
		// Событие отправлено
	default:
		// Буфер полон, пропускаем событие
	}
}

// PublishSignalDetected публикует обнаружение сигнала
func (ec *EventCoordinator) PublishSignalDetected(symbol string, direction string, changePercent float64, confidence float64) {
	event := Event{
		Type:      EventSignalDetected,
		Service:   "GrowthMonitor",
		Message:   direction + " signal detected for " + symbol,
		Data:      map[string]interface{}{"symbol": symbol, "direction": direction, "change": changePercent, "confidence": confidence},
		Timestamp: time.Now(),
		Severity:  "info",
	}

	select {
	case ec.eventChan <- event:
		// Событие отправлено
	default:
		// Буфер полон, пропускаем событие
	}
}

// GetEvents возвращает события из буфера
func (ec *EventCoordinator) GetEvents(limit int) []Event {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	if limit <= 0 || limit > len(ec.eventBuffer) {
		limit = len(ec.eventBuffer)
	}

	// Возвращаем последние limit событий
	start := len(ec.eventBuffer) - limit
	if start < 0 {
		start = 0
	}

	result := make([]Event, limit)
	copy(result, ec.eventBuffer[start:])

	return result
}

// ClearBuffer очищает буфер событий
func (ec *EventCoordinator) ClearBuffer() {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	ec.eventBuffer = make([]Event, 0)
}

// Stop останавливает координатор
func (ec *EventCoordinator) Stop() {
	close(ec.stopChan)
	ec.wg.Wait()
}

// eventHandler обрабатывает события
func (ec *EventCoordinator) eventHandler() {
	defer ec.wg.Done()

	for {
		select {
		case event, ok := <-ec.eventChan:
			if !ok {
				return // Канал закрыт
			}
			ec.processEvent(event)
		case <-ec.stopChan:
			return
		}
	}
}

// processEvent обрабатывает событие
func (ec *EventCoordinator) processEvent(event Event) {
	// 1. Добавляем в буфер (под блокировкой)
	ec.mu.Lock()
	ec.eventBuffer = append(ec.eventBuffer, event)

	// Ограничиваем размер буфера
	if len(ec.eventBuffer) > ec.bufferSize {
		ec.eventBuffer = ec.eventBuffer[len(ec.eventBuffer)-ec.bufferSize:]
	}
	ec.mu.Unlock()

	// 2. Логируем если включено
	if ec.enableLogging {
		ec.logEvent(event)
	}

	// 3. Уведомляем подписчиков
	ec.notifySubscribers(event)
}

// notifySubscribers уведомляет подписчиков о событии
func (ec *EventCoordinator) notifySubscribers(event Event) {
	// Создаем копию подписчиков под блокировкой
	ec.mu.RLock()
	subscribers := make([]DataSubscriber, len(ec.subscribers))
	copy(subscribers, ec.subscribers)
	ec.mu.RUnlock()

	// Уведомляем каждого подписчика в отдельной горутине
	for _, subscriber := range subscribers {
		go func(sub DataSubscriber) {
			defer func() {
				if r := recover(); r != nil {
					// Логируем ошибку но не паникуем
					log.Printf("[ERROR] Subscriber panic recovered: %v", r)
				}
			}()

			sub.OnEvent(event)
		}(subscriber)
	}
}

// logEvent логирует событие
func (ec *EventCoordinator) logEvent(event Event) {
	if !ec.enableLogging {
		return
	}

	// Структурированное логирование
	switch event.Severity {
	case "error":
		log.Printf("[ERROR] [%s] %s (service: %s)",
			event.Timestamp.Format("15:04:05"),
			event.Message,
			event.Service)
	case "warning":
		log.Printf("[WARN] [%s] %s (service: %s)",
			event.Timestamp.Format("15:04:05"),
			event.Message,
			event.Service)
	default:
		// Для событий об обновлении цены не логируем, чтобы не засорять логи
		if event.Type != EventPriceUpdated {
			log.Printf("[INFO] [%s] %s (service: %s)",
				event.Timestamp.Format("15:04:05"),
				event.Message,
				event.Service)
		}
	}
}

// StorageCoordinator координатор хранилища
type StorageCoordinator struct {
	storage     storage.PriceStorage
	coordinator *EventCoordinator
}

// NewStorageCoordinator создает координатор хранилища
func NewStorageCoordinator(storage storage.PriceStorage, coordinator *EventCoordinator) *StorageCoordinator {
	sc := &StorageCoordinator{
		storage:     storage,
		coordinator: coordinator,
	}

	// Создаем подписчика и подписываемся
	subscriber := &storageCoordinatorSubscriber{sc: sc}
	storage.Subscribe("all", subscriber)

	return sc
}

// storageCoordinatorSubscriber подписчик для StorageCoordinator
type storageCoordinatorSubscriber struct {
	sc *StorageCoordinator
}

func (s *storageCoordinatorSubscriber) OnPriceUpdate(symbol string, price, volume float64, timestamp time.Time) {
	// Публикуем событие об обновлении цены
	s.sc.coordinator.PublishPriceUpdate(symbol, price, volume, timestamp)
}

func (s *storageCoordinatorSubscriber) OnSymbolAdded(symbol string) {
	s.sc.coordinator.PublishEvent(Event{
		Type:      EventSymbolAdded,
		Service:   "StorageCoordinator",
		Message:   "Symbol added: " + symbol,
		Data:      map[string]interface{}{"symbol": symbol, "action": "added"},
		Timestamp: time.Now(),
		Severity:  "info",
	})
}

func (s *storageCoordinatorSubscriber) OnSymbolRemoved(symbol string) {
	s.sc.coordinator.PublishEvent(Event{
		Type:      EventSymbolRemoved,
		Service:   "StorageCoordinator",
		Message:   "Symbol removed: " + symbol,
		Data:      map[string]interface{}{"symbol": symbol, "action": "removed"},
		Timestamp: time.Now(),
		Severity:  "info",
	})
}

// GetStorage возвращает хранилище
func (sc *StorageCoordinator) GetStorage() storage.PriceStorage {
	return sc.storage
}

// Cleanup выполняет очистку старых данных
func (sc *StorageCoordinator) Cleanup(maxAge time.Duration) (int, error) {
	removed, err := sc.storage.CleanOldData(maxAge)

	if err == nil && removed > 0 {
		sc.coordinator.PublishEvent(Event{
			Type:      EventHealthCheck,
			Service:   "StorageCoordinator",
			Message:   "Cleaned up old data",
			Data:      map[string]interface{}{"removed": removed, "maxAge": maxAge.String()},
			Timestamp: time.Now(),
			Severity:  "info",
		})
	}

	return removed, err
}
