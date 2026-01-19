// internal/delivery/telegram/app/bot/handlers/commands/commands/interface.go
package commands

import "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"

// CommandsCommandHandler интерфейс обработчика команды /commands
type CommandsCommandHandler interface {
	handlers.Handler
}
