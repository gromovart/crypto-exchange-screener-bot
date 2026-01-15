package notify_fall_only

import "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"

// NotifyFallOnlyHandler интерфейс обработчика уведомлений о падении
type NotifyFallOnlyHandler interface {
	handlers.Handler
}
