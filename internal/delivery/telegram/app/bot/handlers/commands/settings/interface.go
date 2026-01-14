// internal/delivery/telegram/app/bot/handlers/commands/settings/interface.go
package settings

import "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"

// SettingsCommandHandler интерфейс обработчика команды /settings
type SettingsCommandHandler interface {
	handlers.Handler
}
