package auth_logout

import "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"

// AuthLogoutHandler интерфейс обработчика выхода из системы
type AuthLogoutHandler interface {
	handlers.Handler
}
