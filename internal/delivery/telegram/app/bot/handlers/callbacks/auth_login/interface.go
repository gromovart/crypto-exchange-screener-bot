package auth_login

import "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"

// AuthLoginHandler интерфейс обработчика авторизации
type AuthLoginHandler interface {
	handlers.Handler
}
