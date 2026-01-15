package reset_settings

import "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"

// ResetSettingsHandler интерфейс обработчика сброса настроек
type ResetSettingsHandler interface {
	handlers.Handler
}
