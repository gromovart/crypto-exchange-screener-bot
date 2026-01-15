package signals_menu

import "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"

// SignalsMenuHandler интерфейс обработчика меню сигналов
type SignalsMenuHandler interface {
	handlers.Handler
}
