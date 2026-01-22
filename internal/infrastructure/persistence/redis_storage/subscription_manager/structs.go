// internal/infrastructure/persistence/redis_storage/subscription_manager/structs.go
package subscription_manager

import (
	redis_storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	"sync"
)

// SubscriptionManager реализует интерфейс SubscriptionManagerInterface
type SubscriptionManager struct {
	mu             sync.RWMutex
	subscribers    map[string]map[redis_storage.SubscriberInterface]struct{} // symbol -> subscribers
	allSubscribers []redis_storage.SubscriberInterface                       // Подписчики на все символы
}
