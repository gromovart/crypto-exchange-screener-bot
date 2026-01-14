// internal/delivery/telegram/app/bot/handlers/commands/profile/interface.go
package profile

import "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"

// ProfileCommandHandler интерфейс обработчика команды /profile
type ProfileCommandHandler interface {
	handlers.Handler
}
