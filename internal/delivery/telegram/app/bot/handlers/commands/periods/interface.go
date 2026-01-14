// internal/delivery/telegram/app/bot/handlers/commands/periods/interface.go
package periods

import "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"

// PeriodsCommandHandler интерфейс обработчика команды /periods
type PeriodsCommandHandler interface {
	handlers.Handler
}
