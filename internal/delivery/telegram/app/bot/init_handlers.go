// internal/delivery/telegram/app/bot/init_handlers.go
package bot

import (
	"log"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	help_callback "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/help"
	notifications_menu "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/notifications_menu"
	periods_menu "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/periods_menu"
	profile_main "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/profile_main"
	settings_main "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/settings_main"
	stats_callback "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/stats"
	help_command "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/commands/help"
	notifications_command "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/commands/notifications"
	periods_command "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/commands/periods"
	profile_command "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/commands/profile"
	settings_command "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/commands/settings"
	thresholds_command "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/commands/thresholds"
	start_command "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/start"
)

// InitHandlerFactory инициализирует фабрику хэндлеров
func InitHandlerFactory(factory *handlers.HandlerFactory) {
	log.Println("🔧 Инициализация создателей хэндлеров...")

	// Регистрируем создателей КОМАНД
	factory.RegisterHandlerCreator("start_command", func() handlers.Handler {
		return start_command.NewHandler()
	})

	factory.RegisterHandlerCreator("help_command", func() handlers.Handler {
		return help_command.NewHandler()
	})

	factory.RegisterHandlerCreator("settings_command", func() handlers.Handler {
		return settings_command.NewHandler()
	})

	factory.RegisterHandlerCreator("notifications_command", func() handlers.Handler {
		return notifications_command.NewHandler()
	})

	factory.RegisterHandlerCreator("profile_command", func() handlers.Handler {
		return profile_command.NewHandler()
	})

	factory.RegisterHandlerCreator("thresholds_command", func() handlers.Handler {
		return thresholds_command.NewHandler()
	})

	factory.RegisterHandlerCreator("periods_command", func() handlers.Handler {
		return periods_command.NewHandler()
	})

	// Регистрируем создателей CALLBACKS
	factory.RegisterHandlerCreator("help_callback", func() handlers.Handler {
		return help_callback.NewHandler()
	})

	factory.RegisterHandlerCreator("profile_main", func() handlers.Handler {
		return profile_main.NewHandler()
	})

	factory.RegisterHandlerCreator("settings_main", func() handlers.Handler {
		return settings_main.NewHandler()
	})

	factory.RegisterHandlerCreator("notifications_menu", func() handlers.Handler {
		return notifications_menu.NewHandler()
	})

	factory.RegisterHandlerCreator("periods_menu", func() handlers.Handler {
		return periods_menu.NewHandler()
	})

	factory.RegisterHandlerCreator("stats_callback", func() handlers.Handler {
		return stats_callback.NewHandler()
	})

	log.Println("✅ Инициализация создателей хэндлеров завершена")
}
