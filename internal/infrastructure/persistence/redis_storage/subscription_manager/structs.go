// internal/infrastructure/persistence/redis_storage/subscription_manager/structs.go
package subscription_manager

import (
	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	"sync"
)

// SubscriptionManager реализует интерфейс SubscriptionManagerInterface
type SubscriptionManager struct {
	mu             sync.RWMutex
	subscribers    map[string]map[storage.SubscriberInterface]struct{} // symbol -> subscribers
	allSubscribers []storage.SubscriberInterface                       // Подписчики на все символы
}
