// internal/delivery/telegram/app/bot/factory_handlers.go
package bot

import (
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
	payment_confirm_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/payment_confirm"
	payment_history_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/payment_history"
	payment_plan_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/payment_plan"
	payment_tbank_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/payment_sbp"
	tbank_service "crypto-exchange-screener-bot/internal/delivery/telegram/services/tbank"
	period_manage_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/period_manage"
	period_select_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/period_select"
	periods_menu "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/periods_menu"
	profile_main "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/profile_main"
	profile_stats_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/profile_stats"
	profile_subscription_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/profile_subscription"
	reset_menu_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/reset_menu"
	reset_settings_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/reset_settings"
	session_duration_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/session_duration"
	session_start_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/session_start"
	session_stop_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/session_stop"
	settings_main "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/settings_main"
	signal_set_fall_threshold_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/signal_set_fall_threshold"
	signal_set_growth_threshold_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/signal_set_growth_threshold"
	signal_toggle_fall_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/signal_toggle_fall"
	signal_toggle_growth_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/signal_toggle_growth"
	signals_menu_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/signals_menu"
	stats_callback "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/stats"
	thresholds_menu_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/thresholds_menu"
	with_params_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/callbacks/with_params"
	buy_command "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/commands/buy"
	commands_command "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/commands/commands"
	help_command "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/commands/help"
	link_command "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/commands/link"
	notifications_command "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/commands/notifications"
	paysupport_command "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/commands/paysupport"
	periods_command "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/commands/periods"
	profile_command "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/commands/profile"
	settings_command "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/commands/settings"
	terms_command "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/commands/terms"
	thresholds_command "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/commands/thresholds"
	precheckout_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/events/payment/pre_checkout"
	successful_payment_handler "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/events/payment/successful_payment"
	start_command "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/start"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/middlewares"
	telegram_http "crypto-exchange-screener-bot/internal/delivery/telegram/app/http_client"
	notifications_toggle_service "crypto-exchange-screener-bot/internal/delivery/telegram/services/notifications_toggle"
	payment_service "crypto-exchange-screener-bot/internal/delivery/telegram/services/payment"
	profile_service "crypto-exchange-screener-bot/internal/delivery/telegram/services/profile"
	signal_settings_service "crypto-exchange-screener-bot/internal/delivery/telegram/services/signal_settings"
	trading_session_service "crypto-exchange-screener-bot/internal/delivery/telegram/services/trading_session"
	"crypto-exchange-screener-bot/internal/core/domain/payment"
	"crypto-exchange-screener-bot/internal/core/domain/users"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	currency_client "crypto-exchange-screener-bot/internal/infrastructure/http/currency"
	"crypto-exchange-screener-bot/pkg/logger"
)

type Services struct {
	paymentService             payment_service.Service
	notificationsToggleService notifications_toggle_service.Service
	signalSettingsService      signal_settings_service.Service
	profileService             profile_service.Service
	tradingSessionService      trading_session_service.Service
	starsClient                *telegram_http.StarsClient
	userService                *users.Service
	tbankService               tbank_service.Service
	currencyClient             *currency_client.Client
	paymentCoreService         *payment.PaymentService
}

// InitHandlerFactory инициализирует фабрику хэндлеров
func InitHandlerFactory(
	factory *handlers.HandlerFactory,
	cfg *config.Config,
	services *Services,
	subscriptionMiddleware *middlewares.SubscriptionMiddleware,
) {
	logger.Info("🔧 Инициализация создателей хэндлеров...")

	// Регистрируем создателей КОМАНД (без подписки)
	factory.RegisterHandlerCreator("start", func() handlers.Handler {
		return start_command.NewHandler(subscriptionMiddleware, services.tradingSessionService)
	})
	factory.RegisterHandlerCreator("help", func() handlers.Handler {
		return help_command.NewHandler()
	})
	factory.RegisterHandlerCreator("link", func() handlers.Handler {
		return link_command.NewHandler(services.userService)
	})
	factory.RegisterHandlerCreator("terms", func() handlers.Handler {
		return terms_command.NewHandler()
	})

	factory.RegisterHandlerCreator("buy", func() handlers.Handler {
		return buy_command.NewHandler(buy_command.Dependencies{
			IsDev:          cfg.IsDev(),
			CurrencyClient: services.currencyClient,
		})
	})

	// Команды, требующие подписки (будут обернуты middleware)
	factory.RegisterHandlerCreator("commands", func() handlers.Handler {
		handler := commands_command.NewHandler()
		if subscriptionMiddleware != nil {
			return subscriptionMiddleware.RequireSubscription(handler)
		}
		return handler
	})

	factory.RegisterHandlerCreator("settings", func() handlers.Handler {
		handler := settings_command.NewHandler()
		if subscriptionMiddleware != nil {
			return subscriptionMiddleware.RequireSubscription(handler)
		}
		return handler
	})

	factory.RegisterHandlerCreator("notifications", func() handlers.Handler {
		handler := notifications_command.NewHandler()
		if subscriptionMiddleware != nil {
			return subscriptionMiddleware.RequireSubscription(handler)
		}
		return handler
	})

	factory.RegisterHandlerCreator("profile", func() handlers.Handler {
		handler := profile_command.NewHandler()
		if subscriptionMiddleware != nil {
			return subscriptionMiddleware.RequireSubscription(handler)
		}
		return handler
	})

	factory.RegisterHandlerCreator("thresholds", func() handlers.Handler {
		handler := thresholds_command.NewHandler()
		if subscriptionMiddleware != nil {
			return subscriptionMiddleware.RequireSubscription(handler)
		}
		return handler
	})

	factory.RegisterHandlerCreator("periods", func() handlers.Handler {
		handler := periods_command.NewHandler()
		if subscriptionMiddleware != nil {
			return subscriptionMiddleware.RequireSubscription(handler)
		}
		return handler
	})

	// Регистрируем создателей CALLBACKS (без подписки)
	factory.RegisterHandlerCreator(constants.CallbackHelp, func() handlers.Handler {
		return help_callback.NewHandler()
	})

	factory.RegisterHandlerCreator(constants.CallbackMenuMain, func() handlers.Handler {
		return menu_main.NewHandler()
	})

	factory.RegisterHandlerCreator(constants.CallbackAuthLogin, func() handlers.Handler {
		return auth_login_handler.NewHandler()
	})

	factory.RegisterHandlerCreator(constants.CallbackAuthLogout, func() handlers.Handler {
		return auth_logout_handler.NewHandler()
	})

	factory.RegisterHandlerCreator("paysupport", func() handlers.Handler {
		return paysupport_command.NewHandler()
	})

	// Callback-и, требующие подписки
	factory.RegisterHandlerCreator(constants.CallbackProfileMain, func() handlers.Handler {
		// ⭐ ИЗМЕНЕНО: передаем profileService
		handler := profile_main.NewHandler(services.profileService)
		if subscriptionMiddleware != nil {
			return subscriptionMiddleware.RequireSubscription(handler)
		}
		return handler
	})

	factory.RegisterHandlerCreator(constants.CallbackSettingsMain, func() handlers.Handler {
		handler := settings_main.NewHandler()
		if subscriptionMiddleware != nil {
			return subscriptionMiddleware.RequireSubscription(handler)
		}
		return handler
	})

	factory.RegisterHandlerCreator(constants.CallbackNotificationsMenu, func() handlers.Handler {
		handler := notifications_menu.NewHandler()
		if subscriptionMiddleware != nil {
			return subscriptionMiddleware.RequireSubscription(handler)
		}
		return handler
	})

	factory.RegisterHandlerCreator(constants.CallbackPeriodsMenu, func() handlers.Handler {
		handler := periods_menu.NewHandler()
		if subscriptionMiddleware != nil {
			return subscriptionMiddleware.RequireSubscription(handler)
		}
		return handler
	})

	factory.RegisterHandlerCreator(constants.CallbackStats, func() handlers.Handler {
		handler := stats_callback.NewHandler()
		if subscriptionMiddleware != nil {
			return subscriptionMiddleware.RequireSubscription(handler)
		}
		return handler
	})

	factory.RegisterHandlerCreator(constants.CallbackSignalsMenu, func() handlers.Handler {
		handler := signals_menu_handler.NewHandler()
		if subscriptionMiddleware != nil {
			return subscriptionMiddleware.RequireSubscription(handler)
		}
		return handler
	})

	factory.RegisterHandlerCreator(constants.CallbackResetMenu, func() handlers.Handler {
		handler := reset_menu_handler.NewHandler()
		if subscriptionMiddleware != nil {
			return subscriptionMiddleware.RequireSubscription(handler)
		}
		return handler
	})

	factory.RegisterHandlerCreator(constants.CallbackThresholdsMenu, func() handlers.Handler {
		handler := thresholds_menu_handler.NewHandler()
		if subscriptionMiddleware != nil {
			return subscriptionMiddleware.RequireSubscription(handler)
		}
		return handler
	})

	factory.RegisterHandlerCreator(constants.CallbackResetSettings, func() handlers.Handler {
		handler := reset_settings_handler.NewHandler()
		if subscriptionMiddleware != nil {
			return subscriptionMiddleware.RequireSubscription(handler)
		}
		return handler
	})

	factory.RegisterHandlerCreator(constants.CallbackPeriodManage, func() handlers.Handler {
		handler := period_manage_handler.NewHandler()
		if subscriptionMiddleware != nil {
			return subscriptionMiddleware.RequireSubscription(handler)
		}
		return handler
	})

	factory.RegisterHandlerCreator(constants.CallbackProfileStats, func() handlers.Handler {
		handler := profile_stats_handler.NewHandler()
		if subscriptionMiddleware != nil {
			return subscriptionMiddleware.RequireSubscription(handler)
		}
		return handler
	})

	factory.RegisterHandlerCreator(constants.CallbackProfileSubscription, func() handlers.Handler {
		handler := profile_subscription_handler.NewHandler()
		if subscriptionMiddleware != nil {
			return subscriptionMiddleware.RequireSubscription(handler)
		}
		return handler
	})

	// CALLBACK ОБРАБОТЧИКИ ДЛЯ УВЕДОМЛЕНИЙ (требуют подписки)
	factory.RegisterHandlerCreator(constants.CallbackNotifyGrowthOnly, func() handlers.Handler {
		handler := notify_growth_only_handler.NewHandler()
		if subscriptionMiddleware != nil {
			return subscriptionMiddleware.RequireSubscription(handler)
		}
		return handler
	})

	factory.RegisterHandlerCreator(constants.CallbackNotifyFallOnly, func() handlers.Handler {
		handler := notify_fall_only_handler.NewHandler()
		if subscriptionMiddleware != nil {
			return subscriptionMiddleware.RequireSubscription(handler)
		}
		return handler
	})

	factory.RegisterHandlerCreator(constants.CallbackNotifyBoth, func() handlers.Handler {
		handler := notify_both_handler.NewHandler()
		if subscriptionMiddleware != nil {
			return subscriptionMiddleware.RequireSubscription(handler)
		}
		return handler
	})

	factory.RegisterHandlerCreator(constants.CallbackNotifyToggleAll, func() handlers.Handler {
		handler := notifications_toggle_handler.NewHandler(services.notificationsToggleService)
		if subscriptionMiddleware != nil {
			return subscriptionMiddleware.RequireSubscription(handler)
		}
		return handler
	})

	// CALLBACK ОБРАБОТЧИКИ ДЛЯ СИГНАЛОВ (требуют подписки)
	factory.RegisterHandlerCreator(constants.CallbackSignalToggleGrowth, func() handlers.Handler {
		handler := signal_toggle_growth_handler.NewHandler(services.signalSettingsService)
		if subscriptionMiddleware != nil {
			return subscriptionMiddleware.RequireSubscription(handler)
		}
		return handler
	})

	factory.RegisterHandlerCreator(constants.CallbackSignalToggleFall, func() handlers.Handler {
		handler := signal_toggle_fall_handler.NewHandler(services.signalSettingsService)
		if subscriptionMiddleware != nil {
			return subscriptionMiddleware.RequireSubscription(handler)
		}
		return handler
	})

	factory.RegisterHandlerCreator(constants.CallbackSignalSetGrowthThreshold, func() handlers.Handler {
		handler := signal_set_growth_threshold_handler.NewHandler(services.signalSettingsService)
		if subscriptionMiddleware != nil {
			return subscriptionMiddleware.RequireSubscription(handler)
		}
		return handler
	})

	factory.RegisterHandlerCreator(constants.CallbackSignalSetFallThreshold, func() handlers.Handler {
		handler := signal_set_fall_threshold_handler.NewHandler(services.signalSettingsService)
		if subscriptionMiddleware != nil {
			return subscriptionMiddleware.RequireSubscription(handler)
		}
		return handler
	})

	// Регистрируем универсальный обработчик для параметризованных callback-ов (требует подписки)
	factory.RegisterHandlerCreator("with_params", func() handlers.Handler {
		handler := with_params_handler.NewHandler(services.signalSettingsService)
		if subscriptionMiddleware != nil {
			return subscriptionMiddleware.RequireSubscription(handler)
		}
		return handler
	})

	// Обработчик для выбора периода (требует подписки)
	factory.RegisterHandlerCreator("period_select", func() handlers.Handler {
		handler := period_select_handler.NewHandler(services.signalSettingsService)
		if subscriptionMiddleware != nil {
			return subscriptionMiddleware.RequireSubscription(handler)
		}
		return handler
	})

	// ИСТОРИЯ ПЛАТЕЖЕЙ
	factory.RegisterHandlerCreator(constants.PaymentConstants.CallbackPaymentHistory, func() handlers.Handler {
		return payment_history_handler.NewHandler(payment_history_handler.Dependencies{
			PaymentCoreService: services.paymentCoreService,
		})
	})

	// ПЛАТЕЖНЫЕ CALLBACK ОБРАБОТЧИКИ (без подписки - доступны всем)
	factory.RegisterHandlerCreator(constants.PaymentConstants.CallbackPaymentPlan, func() handlers.Handler {
		return payment_plan_handler.NewHandler(payment_plan_handler.Dependencies{
			IsDev:          cfg.IsDev(),
			TBankEnabled:   cfg.TBank.Enabled,
			CurrencyClient: services.currencyClient,
		})
	})

	factory.RegisterHandlerCreator(constants.PaymentConstants.CallbackPaymentConfirm, func() handlers.Handler {
		return payment_confirm_handler.NewHandler(payment_confirm_handler.Dependencies{
			Config:      cfg,
			StarsClient: services.starsClient,
		})
	})

	// Оплата через Т-Банк (СБП, карта)
	factory.RegisterHandlerCreator(constants.PaymentConstants.CallbackPaymentTBank, func() handlers.Handler {
		return payment_tbank_handler.NewHandler(payment_tbank_handler.Dependencies{
			TBankService: services.tbankService,
		})
	})

	// ПЛАТЕЖНЫЕ СОБЫТИЯ (без подписки)
	if services.paymentService != nil {
		factory.RegisterHandlerCreator("pre_checkout_query", func() handlers.Handler {
			return precheckout_handler.NewHandler(services.paymentService)
		})

		factory.RegisterHandlerCreator("successful_payment", func() handlers.Handler {
			return successful_payment_handler.NewHandler(services.paymentService)
		})

		logger.Info("✅ Платежные обработчики событий зарегистрированы")
	} else {
		logger.Warn("⚠️ PaymentService не предоставлен, платежные события не будут обрабатываться")
	}

	// ТОРГОВЫЕ СЕССИИ (без подписки — доступны всем авторизованным)
	if services.tradingSessionService != nil {
		factory.RegisterHandlerCreator("session_start", func() handlers.Handler {
			return session_start_handler.NewHandler(services.tradingSessionService)
		})

		factory.RegisterHandlerCreator("session_stop", func() handlers.Handler {
			return session_stop_handler.NewHandler(services.tradingSessionService)
		})

		factory.RegisterHandlerCreator(constants.CallbackSessionDuration, func() handlers.Handler {
			return session_duration_handler.NewHandler(services.tradingSessionService)
		})

		logger.Info("✅ Обработчики торговых сессий зарегистрированы")
	} else {
		logger.Warn("⚠️ TradingSessionService не предоставлен, торговые сессии недоступны")
	}

	logger.Info("✅ Инициализация создателей хэндлеров завершена")
}

// UpdateSubscriptionMiddleware обновляет middleware подписки после создания бота
func UpdateSubscriptionMiddleware(factory *handlers.HandlerFactory, subscriptionMiddleware *middlewares.SubscriptionMiddleware) {
	// TODO: Обновить все хэндлеры с подпиской
	logger.Info("🔄 Subscription middleware обновлен")
}
