// internal/infrastructure/persistence/redis_storage/subscription_manager/interface.go
package subscription_manager

import "time"

// Subscriber интерфейс подписчика
type SubscriberInterface interface {
	OnPriceUpdate(symbol string, price, volume24h, volumeUSD float64, timestamp time.Time)
	OnSymbolAdded(symbol string)
	OnSymbolRemoved(symbol string)
}
