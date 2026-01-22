// internal/infrastructure/persistence/redis_storage/subscription_manager.go
package subscription_manager

import (
	redis_storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	"time"
)

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

// NewSubscriptionManager создает нового менеджера подписок
func NewSubscriptionManager() *SubscriptionManager {
	return &SubscriptionManager{
		subscribers:    make(map[string]map[redis_storage.SubscriberInterface]struct{}),
		allSubscribers: make([]redis_storage.SubscriberInterface, 0),
	}
}

// Subscribe подписывает на обновления символа
func (sm *SubscriptionManager) Subscribe(symbol string, subscriber redis_storage.SubscriberInterface) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if symbol == "all" {
		sm.allSubscribers = append(sm.allSubscribers, subscriber)
		return nil
	}

	if _, exists := sm.subscribers[symbol]; !exists {
		sm.subscribers[symbol] = make(map[redis_storage.SubscriberInterface]struct{})
	}
	sm.subscribers[symbol][subscriber] = struct{}{}
	return nil
}

// Unsubscribe отписывает от обновлений символа
func (sm *SubscriptionManager) Unsubscribe(symbol string, subscriber redis_storage.SubscriberInterface) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if symbol == "all" {
		for i, sub := range sm.allSubscribers {
			if sub == subscriber {
				sm.allSubscribers = append(sm.allSubscribers[:i], sm.allSubscribers[i+1:]...)
				break
			}
		}
		return nil
	}

	if subs, exists := sm.subscribers[symbol]; exists {
		delete(subs, subscriber)
		if len(subs) == 0 {
			delete(sm.subscribers, symbol)
		}
	}
	return nil
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
