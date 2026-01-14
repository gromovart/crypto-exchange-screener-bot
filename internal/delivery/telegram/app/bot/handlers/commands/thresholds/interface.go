// internal/delivery/telegram/app/bot/handlers/commands/thresholds/interface.go
package thresholds

import "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"

// ThresholdsCommandHandler интерфейс обработчика команды /thresholds
type ThresholdsCommandHandler interface {
	handlers.Handler
}
