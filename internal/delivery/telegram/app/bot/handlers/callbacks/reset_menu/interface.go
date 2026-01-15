package reset_menu

import "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"

// ResetMenuHandler интерфейс обработчика меню сброса
type ResetMenuHandler interface {
	handlers.Handler
}
