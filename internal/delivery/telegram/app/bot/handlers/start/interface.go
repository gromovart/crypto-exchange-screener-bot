// internal/delivery/telegram/app/bot/handlers/start/interface.go
package start

import "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"

// StartHandler интерфейс хэндлера команды /start
type StartHandler interface {
	handlers.Handler
}
