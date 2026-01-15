package signal_toggle_fall

import "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"

// SignalToggleFallHandler интерфейс обработчика переключения уведомлений о падении
type SignalToggleFallHandler interface {
	handlers.Handler
}
