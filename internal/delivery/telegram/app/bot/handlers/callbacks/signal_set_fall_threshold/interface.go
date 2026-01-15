package signal_set_fall_threshold

import "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"

// SignalSetFallThresholdHandler интерфейс обработчика установки порога падения
type SignalSetFallThresholdHandler interface {
	handlers.Handler
}
