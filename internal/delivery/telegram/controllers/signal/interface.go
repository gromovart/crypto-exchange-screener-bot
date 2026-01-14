// internal/delivery/telegram/controllers/signal/interface.go
package signal

import "crypto-exchange-screener-bot/internal/types"

type Controller interface {
	HandleEvent(event types.Event) error
	GetName() string
	GetSubscribedEvents() []types.EventType
}
