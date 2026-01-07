// internal/events/event.go
package events

import "crypto-exchange-screener-bot/internal/types"

// Middleware - промежуточное ПО для обработки событий
type Middleware interface {
	Process(event types.Event, next HandlerFunc) error
}

// HandlerFunc - функция обработки события
type HandlerFunc func(event types.Event) error
