package signal_test

import "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"

// SignalTestHandler интерфейс обработчика тестового сигнала
type SignalTestHandler interface {
	handlers.Handler
}
