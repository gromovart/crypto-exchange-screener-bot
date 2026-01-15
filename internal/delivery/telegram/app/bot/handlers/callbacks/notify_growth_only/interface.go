package notify_growth_only

import "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"

// NotifyGrowthOnlyHandler интерфейс обработчика уведомлений о росте
type NotifyGrowthOnlyHandler interface {
	handlers.Handler
}
