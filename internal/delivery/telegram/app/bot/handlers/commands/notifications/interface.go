// internal/delivery/telegram/app/bot/handlers/commands/notifications/interface.go
package notifications

import "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"

// NotificationsCommandHandler интерфейс обработчика команды /notifications
type NotificationsCommandHandler interface {
	handlers.Handler
}
