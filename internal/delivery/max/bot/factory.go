// internal/delivery/max/bot/factory.go
package bot

import (
	"crypto-exchange-screener-bot/internal/core/domain/subscription"
	"crypto-exchange-screener-bot/internal/core/domain/users"
	cbAuthLogin "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/callbacks/auth_login"
	cbAuthLogout "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/callbacks/auth_logout"
	cbHelp "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/callbacks/help"
	cbMenuMain "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/callbacks/menu_main"
	cbNotificationsMenu "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/callbacks/notifications_menu"
	cbNotifyBoth "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/callbacks/notify_both"
	cbNotifyFallOnly "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/callbacks/notify_fall_only"
	cbNotifyGrowthOnly "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/callbacks/notify_growth_only"
	cbNotifyToggle "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/callbacks/notify_toggle"
	cbPeriodSelect "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/callbacks/period_select"
	cbPeriodsMenu "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/callbacks/periods_menu"
	cbProfileMain "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/callbacks/profile_main"
	cbProfileStats "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/callbacks/profile_stats"
	cbProfileSubscription "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/callbacks/profile_subscription"
	cbResetAll "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/callbacks/reset_all"
	cbResetMenu "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/callbacks/reset_menu"
	cbResetSettings "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/callbacks/reset_settings"
	cbSettingsMain "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/callbacks/settings_main"
	cbSignalSetFall "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/callbacks/signal_set_fall_threshold"
	cbSignalSetGrowth "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/callbacks/signal_set_growth_threshold"
	cbSignalToggleFall "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/callbacks/signal_toggle_fall"
	cbSignalToggleGrowth "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/callbacks/signal_toggle_growth"
	cbSignalsMenu "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/callbacks/signals_menu"
	cbStats "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/callbacks/stats"
	cbThresholdsMenu "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/callbacks/thresholds_menu"
	cbLinkTelegram "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/callbacks/link_telegram"
	cbSessionDuration "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/callbacks/session_duration"
	cbSessionStart "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/callbacks/session_start"
	cbSessionStop "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/callbacks/session_stop"
	cbBuy          "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/callbacks/buy"
	cbPaymentTBank "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/callbacks/payment_tbank"
	cbWithParams   "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/callbacks/with_params"
	cmdHelp        "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/commands/help"
	cmdLink       "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/commands/link"
	cmdPaysupport "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/commands/paysupport"
	cmdStart      "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/commands/start"
	cmdTerms      "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/commands/terms"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/router"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/middleware"
	notifySvc  "crypto-exchange-screener-bot/internal/delivery/telegram/services/notifications_toggle"
	signalSvc  "crypto-exchange-screener-bot/internal/delivery/telegram/services/signal_settings"
	tbankSvc   "crypto-exchange-screener-bot/internal/delivery/telegram/services/tbank"
	sessionSvc "crypto-exchange-screener-bot/internal/delivery/telegram/services/trading_session"
)

// Dependencies — зависимости для регистрации хэндлеров
type Dependencies struct {
	UserService         *users.Service
	NotifyService       notifySvc.Service
	SignalService       signalSvc.Service
	SessionService      sessionSvc.Service
	TBankService        tbankSvc.Service        // nil — если Т-Банк не настроен
	SubscriptionService *subscription.Service   // nil — если проверка подписки отключена
	MaxTBankSuccessURL  string                  // URL редиректа после успешной оплаты (MAX)
	MaxTBankFailURL     string                  // URL редиректа после неудачной оплаты (MAX)
}

// RegisterAll регистрирует все команды и callback-хэндлеры в роутере
func RegisterAll(r router.Router, deps Dependencies) {
	// Создаём middleware подписки (если сервис передан)
	var subMW *middleware.SubscriptionMiddleware
	if deps.SubscriptionService != nil {
		subMW = middleware.NewSubscriptionMiddleware(deps.SubscriptionService)
	}

	// protect оборачивает хэндлер проверкой подписки (если middleware создан)
	protect := func(h handlers.Handler) handlers.Handler {
		if subMW != nil {
			return subMW.RequireSubscription(h)
		}
		return h
	}

	// ── Команды (свободные) ──────────────────────────────────
	r.RegisterCommand("start", cmdStart.NewHandler(deps.SessionService))
	r.RegisterCommand("help", cmdHelp.NewHandler())
	r.RegisterCommand("menu", cbMenuMain.New(deps.SessionService))
	r.RegisterCommand("link", cmdLink.New(deps.UserService))
	r.RegisterCommand("paysupport", cmdPaysupport.NewHandler())
	r.RegisterCommand("terms", cmdTerms.NewHandler())
	r.RegisterCommand("buy", cbBuy.New())

	// ── Команды (защищённые подпиской) ──────────────────────
	r.RegisterCommand("settings", protect(cbSettingsMain.New()))
	r.RegisterCommand("notifications", protect(cbNotificationsMenu.New()))
	r.RegisterCommand("signals", protect(cbSignalsMenu.New()))
	r.RegisterCommand("periods", protect(cbPeriodsMenu.New()))
	r.RegisterCommand("thresholds", protect(cbThresholdsMenu.New()))
	r.RegisterCommand("profile", cbProfileMain.New()) // профиль всегда доступен
	r.RegisterCommand("stats", protect(cbStats.New()))

	// ── Callback: навигация (свободные) ─────────────────────
	r.RegisterCallback(kb.CbMenuMain, cbMenuMain.New(deps.SessionService))
	r.RegisterCallback(kb.CbHelp, cbHelp.New())

	// ── Callback: навигация (защищённые) ────────────────────
	r.RegisterCallback(kb.CbSettingsMain, protect(cbSettingsMain.New()))
	r.RegisterCallback(kb.CbStats, protect(cbStats.New()))

	// ── Callback: уведомления (защищённые) ──────────────────
	r.RegisterCallback(kb.CbNotificationsMenu, protect(cbNotificationsMenu.New()))
	r.RegisterCallback(kb.CbNotifyToggleAll, protect(cbNotifyToggle.New(deps.NotifyService)))
	r.RegisterCallback(kb.CbNotifyGrowthOnly, protect(cbNotifyGrowthOnly.New(deps.SignalService)))
	r.RegisterCallback(kb.CbNotifyFallOnly, protect(cbNotifyFallOnly.New(deps.SignalService)))
	r.RegisterCallback(kb.CbNotifyBoth, protect(cbNotifyBoth.New(deps.SignalService)))

	// ── Callback: сигналы (защищённые) ──────────────────────
	r.RegisterCallback(kb.CbSignalsMenu, protect(cbSignalsMenu.New()))
	r.RegisterCallback(kb.CbSignalToggleGrowth, protect(cbSignalToggleGrowth.New(deps.SignalService)))
	r.RegisterCallback(kb.CbSignalToggleFall, protect(cbSignalToggleFall.New(deps.SignalService)))
	r.RegisterCallback(kb.CbSignalSetGrowthThreshold, protect(cbSignalSetGrowth.New(deps.SignalService)))
	r.RegisterCallback(kb.CbSignalSetFallThreshold, protect(cbSignalSetFall.New(deps.SignalService)))

	// ── Callback: периоды (защищённые) ──────────────────────
	r.RegisterCallback(kb.CbPeriodsMenu, protect(cbPeriodsMenu.New()))
	// Wildcard: период → period_select handler
	r.RegisterCallback("period_*", protect(cbPeriodSelect.New(deps.SignalService)))
	r.RegisterCallback(kb.CbPeriod1m, protect(cbPeriodSelect.New(deps.SignalService)))
	r.RegisterCallback(kb.CbPeriod5m, protect(cbPeriodSelect.New(deps.SignalService)))
	r.RegisterCallback(kb.CbPeriod15m, protect(cbPeriodSelect.New(deps.SignalService)))
	r.RegisterCallback(kb.CbPeriod30m, protect(cbPeriodSelect.New(deps.SignalService)))
	r.RegisterCallback(kb.CbPeriod1h, protect(cbPeriodSelect.New(deps.SignalService)))
	r.RegisterCallback(kb.CbPeriod4h, protect(cbPeriodSelect.New(deps.SignalService)))
	r.RegisterCallback(kb.CbPeriod1d, protect(cbPeriodSelect.New(deps.SignalService)))

	// ── Callback: пороги (защищённые) ───────────────────────
	r.RegisterCallback(kb.CbThresholdsMenu, protect(cbThresholdsMenu.New()))

	// ── Callback: профиль (профиль открыт, статистика/подписка — нет) ──
	r.RegisterCallback(kb.CbProfileMain, cbProfileMain.New())
	r.RegisterCallback(kb.CbProfileStats, protect(cbProfileStats.New()))
	r.RegisterCallback(kb.CbProfileSubscription, cbProfileSubscription.New()) // покажет статус подписки

	// ── Callback: auth (свободные) ──────────────────────────
	r.RegisterCallback(kb.CbAuthLogin, cbAuthLogin.New())
	r.RegisterCallback(kb.CbAuthLogout, cbAuthLogout.New())

	// ── Callback: сброс (защищённые) ────────────────────────
	r.RegisterCallback(kb.CbResetMenu, protect(cbResetMenu.New()))
	r.RegisterCallback(kb.CbResetSettings, protect(cbResetSettings.New(deps.UserService)))
	r.RegisterCallback(kb.CbResetAll, protect(cbResetAll.New(deps.UserService)))

	// ── Callback: привязка Telegram (свободная) ──────────────
	r.RegisterCallback(kb.CbLinkTelegram, cbLinkTelegram.New())

	// ── Callback: торговая сессия (защищённая) ───────────────
	r.RegisterCallback(kb.CbSessionStart, protect(cbSessionStart.New(deps.SessionService)))
	r.RegisterCallback(kb.CbSessionStop, protect(cbSessionStop.New(deps.SessionService)))
	r.RegisterCallback(kb.CbSessionDuration, protect(cbSessionDuration.New(deps.SessionService)))

	// ── Callback: with_params (fallback) ────────────────────
	r.RegisterCallback(kb.CbWithParams, cbWithParams.New())

	// ── Callback: платежи Т-Банк (свободные) ────────────────
	r.RegisterCallback(kb.CbBuy, cbBuy.New())
	r.RegisterCallback(kb.CbPaymentTBankWildcard, cbPaymentTBank.New(deps.TBankService, deps.MaxTBankSuccessURL, deps.MaxTBankFailURL))
}
