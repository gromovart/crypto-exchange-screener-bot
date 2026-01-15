package signal_set_sensitivity

import "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"

// SignalSetSensitivityHandler интерфейс обработчика настройки чувствительности
type SignalSetSensitivityHandler interface {
	handlers.Handler
}
