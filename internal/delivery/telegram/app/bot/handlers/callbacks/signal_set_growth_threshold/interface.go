package signal_set_growth_threshold

import "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"

// SignalSetGrowthThresholdHandler интерфейс обработчика установки порога роста
type SignalSetGrowthThresholdHandler interface {
	handlers.Handler
}
