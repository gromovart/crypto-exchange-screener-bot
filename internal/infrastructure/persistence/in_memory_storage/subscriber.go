// internal/infrastructure/persistence/in_memory_storage/subscriber.go
package storage

import (
	"sync"
	"time"
)

// Subscriber интерфейс подписчика
type Subscriber interface {
	OnPriceUpdate(symbol string, price, volume24h, volumeUSD float64, timestamp time.Time)
	OnSymbolAdded(symbol string)
	OnSymbolRemoved(symbol string)
}

// SubscriberFunc функциональный тип подписчика
type SubscriberFunc func(symbol string, price, volume24h, volumeUSD float64, timestamp time.Time)

func (f SubscriberFunc) OnPriceUpdate(symbol string, price, volume24h, volumeUSD float64, timestamp time.Time) {
	f(symbol, price, volume24h, volumeUSD, timestamp)
}
func (f SubscriberFunc) OnSymbolAdded(symbol string) {
	// По умолчанию ничего не делаем
}

func (f SubscriberFunc) OnSymbolRemoved(symbol string) {
	// По умолчанию ничего не делаем
}

// SubscriptionManager управляет подписками
type SubscriptionManager struct {
	mu             sync.RWMutex
	subscribers    map[string]map[Subscriber]struct{} // symbol -> subscribers
	allSubscribers []Subscriber                       // Подписчики на все символы
}

// NewSubscriptionManager создает нового менеджера подписок
func NewSubscriptionManager() *SubscriptionManager {
	return &SubscriptionManager{
		subscribers:    make(map[string]map[Subscriber]struct{}),
		allSubscribers: make([]Subscriber, 0),
	}
}

// StorePriceLegacy поддерживает старый интерфейс (для обратной совместимости)
// Используется другими частями системы, которые еще не обновлены
func (s *InMemoryPriceStorage) StorePriceLegacy(symbol string, price, volume24h float64, timestamp time.Time) error {
	// Рассчитываем VolumeUSD на основе цены и объема
	volumeUSD := price * volume24h
	return s.StorePrice(
		symbol,
		price,
		volume24h,
		volumeUSD,
		timestamp,
		0, // OpenInterest - значение по умолчанию
		0, // FundingRate - значение по умолчанию
		0, // Change24h - значение по умолчанию
		0, // High24h - значение по умолчанию
		0, // Low24h - значение по умолчанию
	)
}

// Subscribe подписывает на обновления символа
func (sm *SubscriptionManager) Subscribe(symbol string, subscriber Subscriber) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if symbol == "all" {
		sm.allSubscribers = append(sm.allSubscribers, subscriber)
		return
	}

	if _, exists := sm.subscribers[symbol]; !exists {
		sm.subscribers[symbol] = make(map[Subscriber]struct{})
	}
	sm.subscribers[symbol][subscriber] = struct{}{}
}

// Unsubscribe отписывает от обновлений символа
func (sm *SubscriptionManager) Unsubscribe(symbol string, subscriber Subscriber) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if symbol == "all" {
		for i, sub := range sm.allSubscribers {
			if sub == subscriber {
				sm.allSubscribers = append(sm.allSubscribers[:i], sm.allSubscribers[i+1:]...)
				break
			}
		}
		return
	}

	if subs, exists := sm.subscribers[symbol]; exists {
		delete(subs, subscriber)
		if len(subs) == 0 {
			delete(sm.subscribers, symbol)
		}
	}
}

// NotifyAll уведомляет всех подписчиков на символ
func (sm *SubscriptionManager) NotifyAll(symbol string, price, volume24h, volumeUSD float64, timestamp time.Time) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Уведомляем подписчиков на конкретный символ
	if subs, exists := sm.subscribers[symbol]; exists {
		for subscriber := range subs {
			go subscriber.OnPriceUpdate(symbol, price, volume24h, volumeUSD, timestamp)
		}
	}

	// Уведомляем подписчиков на все символы
	for _, subscriber := range sm.allSubscribers {
		go subscriber.OnPriceUpdate(symbol, price, volume24h, volumeUSD, timestamp)
	}
}

// NotifySymbolAdded уведомляет о добавлении символа
func (sm *SubscriptionManager) NotifySymbolAdded(symbol string) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for _, subscriber := range sm.allSubscribers {
		go subscriber.OnSymbolAdded(symbol)
	}
}

// NotifySymbolRemoved уведомляет об удалении символа
func (sm *SubscriptionManager) NotifySymbolRemoved(symbol string) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for _, subscriber := range sm.allSubscribers {
		go subscriber.OnSymbolRemoved(symbol)
	}
}

// GetSubscriberCount возвращает количество подписчиков
func (sm *SubscriptionManager) GetSubscriberCount(symbol string) int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if symbol == "all" {
		return len(sm.allSubscribers)
	}

	if subs, exists := sm.subscribers[symbol]; exists {
		return len(subs)
	}
	return 0
}
