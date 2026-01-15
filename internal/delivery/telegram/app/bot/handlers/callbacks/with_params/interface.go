package with_params

import "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"

// WithParamsHandler интерфейс обработчика callback-ов с параметрами
type WithParamsHandler interface {
	handlers.Handler
}
