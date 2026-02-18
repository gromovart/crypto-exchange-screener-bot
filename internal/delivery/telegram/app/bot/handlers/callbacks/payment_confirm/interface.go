// internal/delivery/telegram/app/bot/handlers/callbacks/payment_confirm/interface.go
package payment_confirm

import (
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	telegram_http "crypto-exchange-screener-bot/internal/delivery/telegram/app/http_client"
)

// Handler интерфейс обработчика подтверждения платежа
type Handler interface {
	handlers.Handler
}

// Dependencies зависимости для создания обработчика
type Dependencies struct {
	Config       *config.Config
	StarsClient  *telegram_http.StarsClient
}
