package bot

import (
	"log"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"

	auth_login_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/auth_login"
	auth_logout_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/auth_logout"
	help_callback "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/help"
	menu_main "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/menu_main"
	notifications_menu "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/notifications_menu"
	notifications_toggle_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/notifications_toggle"
	notify_both_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/notify_both"
	notify_fall_only_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/notify_fall_only"
	notify_growth_only_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/notify_growth_only"
	period_manage_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/period_manage"
	period_select_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/period_select"
	periods_menu "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/periods_menu"
	profile_main "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/profile_main"
	profile_stats_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/profile_stats"
	profile_subscription_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/profile_subscription"
	reset_menu_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/reset_menu"
	reset_settings_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/reset_settings"
	settings_main "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/settings_main"
	signal_set_fall_threshold_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/signal_set_fall_threshold"
	signal_set_growth_threshold_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/signal_set_growth_threshold"
	signal_toggle_fall_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/signal_toggle_fall"
	signal_toggle_growth_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/signal_toggle_growth"
	signals_menu_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/signals_menu"
	stats_callback "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/stats"
	thresholds_menu_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/thresholds_menu"
	with_params_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/with_params"
	help_command "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/commands/help"
	notifications_command "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/commands/notifications"
	periods_command "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/commands/periods"
	profile_command "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/commands/profile"
	settings_command "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/commands/settings"
	thresholds_command "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/commands/thresholds"
	start_command "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/start"
	notifications_toggle_service "crypto-exchange-screener-bot/internal/delivery/telegram/services/notifications_toggle"
	signal_settings_service "crypto-exchange-screener-bot/internal/delivery/telegram/services/signal_settings"
)

// InitHandlerFactory инициализирует фабрику хэндлеров
func InitHandlerFactory(
	factory *handlers.HandlerFactory,
	notificationsToggleService notifications_toggle_service.Service,
	signalSettingsService signal_settings_service.Service, // Добавляем
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

	factory.RegisterHandlerCreator(constants.CallbackMenuMain, func() handlers.Handler {
		return menu_main.NewHandler()
	})

	// НОВЫЕ CALLBACK ОБРАБОТЧИКИ ДЛЯ МЕНЮ
	factory.RegisterHandlerCreator(constants.CallbackSignalsMenu, func() handlers.Handler {
		return signals_menu_handler.NewHandler()
	})

	factory.RegisterHandlerCreator(constants.CallbackResetMenu, func() handlers.Handler {
		return reset_menu_handler.NewHandler()
	})

	factory.RegisterHandlerCreator(constants.CallbackThresholdsMenu, func() handlers.Handler {
		return thresholds_menu_handler.NewHandler()
	})

	factory.RegisterHandlerCreator(constants.CallbackAuthLogin, func() handlers.Handler {
		return auth_login_handler.NewHandler()
	})

	factory.RegisterHandlerCreator(constants.CallbackAuthLogout, func() handlers.Handler {
		return auth_logout_handler.NewHandler()
	})

	factory.RegisterHandlerCreator(constants.CallbackResetSettings, func() handlers.Handler {
		return reset_settings_handler.NewHandler()
	})

	factory.RegisterHandlerCreator(constants.CallbackPeriodManage, func() handlers.Handler {
		return period_manage_handler.NewHandler()
	})

	// Обработчик для выбора периода (использует общий префикс)
	factory.RegisterHandlerCreator("period_select", func() handlers.Handler {
		return period_select_handler.NewHandler()
	})

	factory.RegisterHandlerCreator(constants.CallbackProfileStats, func() handlers.Handler {
		return profile_stats_handler.NewHandler()
	})

	factory.RegisterHandlerCreator(constants.CallbackProfileSubscription, func() handlers.Handler {
		return profile_subscription_handler.NewHandler()
	})

	// CALLBACK ОБРАБОТЧИКИ ДЛЯ УВЕДОМЛЕНИЙ
	factory.RegisterHandlerCreator(constants.CallbackNotifyGrowthOnly, func() handlers.Handler {
		return notify_growth_only_handler.NewHandler()
	})

	factory.RegisterHandlerCreator(constants.CallbackNotifyFallOnly, func() handlers.Handler {
		return notify_fall_only_handler.NewHandler()
	})

	factory.RegisterHandlerCreator(constants.CallbackNotifyBoth, func() handlers.Handler {
		return notify_both_handler.NewHandler()
	})

	// Регистрируем универсальный обработчик для параметризованных callback-ов
	factory.RegisterHandlerCreator("with_params", func() handlers.Handler {
		return with_params_handler.NewHandler(signalSettingsService)
	})

	// CALLBACK ОБРАБОТЧИКИ ДЛЯ СИГНАЛОВ (с сервисами)
	factory.RegisterHandlerCreator(constants.CallbackSignalToggleGrowth, func() handlers.Handler {
		return signal_toggle_growth_handler.NewHandler(signalSettingsService)
	})

	factory.RegisterHandlerCreator(constants.CallbackSignalToggleFall, func() handlers.Handler {
		return signal_toggle_fall_handler.NewHandler(signalSettingsService)
	})

	factory.RegisterHandlerCreator(constants.CallbackSignalSetGrowthThreshold, func() handlers.Handler {
		return signal_set_growth_threshold_handler.NewHandler(signalSettingsService)
	})

	factory.RegisterHandlerCreator(constants.CallbackSignalSetFallThreshold, func() handlers.Handler {
		return signal_set_fall_threshold_handler.NewHandler(signalSettingsService)
	})

	// РЕГИСТРАЦИЯ ОБРАБОТЧИКОВ С СЕРВИСАМИ
	factory.RegisterHandlerCreator(constants.CallbackNotifyToggleAll, func() handlers.Handler {
		return notifications_toggle_handler.NewHandler(notificationsToggleService)
	})

	log.Println("✅ Инициализация создателей хэндлеров завершена")
}
