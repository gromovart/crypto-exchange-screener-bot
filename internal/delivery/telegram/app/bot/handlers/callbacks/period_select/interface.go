package period_select

import "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"

// PeriodSelectHandler интерфейс обработчика выбора периода
type PeriodSelectHandler interface {
	handlers.Handler
}
