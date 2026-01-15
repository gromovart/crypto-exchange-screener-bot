package profile_subscription

import "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"

// ProfileSubscriptionHandler интерфейс обработчика подписки профиля
type ProfileSubscriptionHandler interface {
	handlers.Handler
}
