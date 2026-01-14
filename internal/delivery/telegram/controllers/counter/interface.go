// internal/delivery/telegram/controllers/counter/interface.go
package counter

import "crypto-exchange-screener-bot/internal/types"

type Controller interface {
	HandleEvent(event types.Event) error
	GetName() string
	GetSubscribedEvents() []types.EventType
}
