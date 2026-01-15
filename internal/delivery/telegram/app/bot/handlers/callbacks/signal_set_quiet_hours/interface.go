package signal_set_quiet_hours

import "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"

// SignalSetQuietHoursHandler интерфейс обработчика настройки тихих часов
type SignalSetQuietHoursHandler interface {
	handlers.Handler
}
