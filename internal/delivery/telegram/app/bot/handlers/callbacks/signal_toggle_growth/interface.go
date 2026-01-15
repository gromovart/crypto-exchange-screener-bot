package signal_toggle_growth

import "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"

// SignalToggleGrowthHandler интерфейс обработчика переключения уведомлений о росте
type SignalToggleGrowthHandler interface {
	handlers.Handler
}
