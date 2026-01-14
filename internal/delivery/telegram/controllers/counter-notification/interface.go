// internal/delivery/telegram/controllers/counter-notification/interface.go
package counternotification

import "crypto-exchange-screener-bot/internal/types"

type Controller interface {
	HandleEvent(event types.Event) error
	GetName() string
	GetSubscribedEvents() []types.EventType
}
