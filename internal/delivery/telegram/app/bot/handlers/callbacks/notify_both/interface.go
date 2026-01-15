package notify_both

import "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"

// NotifyBothHandler интерфейс обработчика всех уведомлений
type NotifyBothHandler interface {
	handlers.Handler
}
