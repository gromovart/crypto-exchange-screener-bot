package signal_history

import "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"

// SignalHistoryHandler интерфейс обработчика истории сигналов
type SignalHistoryHandler interface {
	handlers.Handler
}
