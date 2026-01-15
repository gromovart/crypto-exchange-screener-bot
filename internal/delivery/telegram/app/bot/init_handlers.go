// internal/delivery/telegram/app/bot/init_handlers.go
package bot

import (
	"log"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	help_callback "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/help"
	notifications_menu "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/notifications_menu"
	notifications_toggle_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/notifications_toggle"
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
	notifications_toggle_service "crypto-exchange-screener-bot/internal/delivery/telegram/services/notifications_toggle"
)

// InitHandlerFactory инициализирует фабрику хэндлеров
func InitHandlerFactory(
	factory *handlers.HandlerFactory,
	notificationsToggleService notifications_toggle_service.Service,
) {
	log.Println("🔧 Инициализация создателей хэндлеров...")

	// Регистрируем создателей КОМАНД
	factory.RegisterHandlerCreator("start", func() handlers.Handler {
		return start_command.NewHandler()
	})

	factory.RegisterHandlerCreator("help", func() handlers.Handler {
		return help_command.NewHandler()
	})

	factory.RegisterHandlerCreator("settings", func() handlers.Handler {
		return settings_command.NewHandler()
	})

	factory.RegisterHandlerCreator("notifications", func() handlers.Handler {
		return notifications_command.NewHandler()
	})

	factory.RegisterHandlerCreator("profile", func() handlers.Handler {
		return profile_command.NewHandler()
	})

	factory.RegisterHandlerCreator("thresholds", func() handlers.Handler {
		return thresholds_command.NewHandler()
	})

	factory.RegisterHandlerCreator("periods", func() handlers.Handler {
		return periods_command.NewHandler()
	})

	// Регистрируем создателей CALLBACKS
	factory.RegisterHandlerCreator(constants.CallbackHelp, func() handlers.Handler {
		return help_callback.NewHandler()
	})

	factory.RegisterHandlerCreator(constants.CallbackProfileMain, func() handlers.Handler {
		return profile_main.NewHandler()
	})

	factory.RegisterHandlerCreator(constants.CallbackSettingsMain, func() handlers.Handler {
		return settings_main.NewHandler()
	})

	factory.RegisterHandlerCreator(constants.CallbackNotificationsMenu, func() handlers.Handler {
		return notifications_menu.NewHandler()
	})

	factory.RegisterHandlerCreator(constants.CallbackPeriodsMenu, func() handlers.Handler {
		return periods_menu.NewHandler()
	})

	factory.RegisterHandlerCreator(constants.CallbackStats, func() handlers.Handler {
		return stats_callback.NewHandler()
	})

	factory.RegisterHandlerCreator(constants.CallbackNotifyToggleAll, func() handlers.Handler {
		return notifications_toggle_handler.NewHandler(notificationsToggleService)
	})

	log.Println("✅ Инициализация создателей хэндлеров завершена")
}
