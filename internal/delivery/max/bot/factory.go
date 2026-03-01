// internal/delivery/max/bot/factory.go
package bot

import (
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
	cbWithParams "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/callbacks/with_params"
	cmdHelp "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/commands/help"
	cmdStart "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/commands/start"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/router"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
	notifySvc "crypto-exchange-screener-bot/internal/delivery/telegram/services/notifications_toggle"
	signalSvc "crypto-exchange-screener-bot/internal/delivery/telegram/services/signal_settings"
)

// Dependencies — зависимости для регистрации хэндлеров
type Dependencies struct {
	UserService   *users.Service
	NotifyService notifySvc.Service
	SignalService signalSvc.Service
}

// RegisterAll регистрирует все команды и callback-хэндлеры в роутере
func RegisterAll(r router.Router, deps Dependencies) {
	// ── Команды ──────────────────────────────────────────────
	r.RegisterCommand("start", cmdStart.NewHandler())
	r.RegisterCommand("help", cmdHelp.NewHandler())
	r.RegisterCommand("menu", cbMenuMain.New())

	// ── Callback: навигация ──────────────────────────────────
	r.RegisterCallback(kb.CbMenuMain, cbMenuMain.New())
	r.RegisterCallback(kb.CbSettingsMain, cbSettingsMain.New())
	r.RegisterCallback(kb.CbHelp, cbHelp.New())
	r.RegisterCallback(kb.CbStats, cbStats.New())

	// ── Callback: уведомления ────────────────────────────────
	r.RegisterCallback(kb.CbNotificationsMenu, cbNotificationsMenu.New())
	r.RegisterCallback(kb.CbNotifyToggleAll, cbNotifyToggle.New(deps.NotifyService))
	r.RegisterCallback(kb.CbNotifyGrowthOnly, cbNotifyGrowthOnly.New(deps.SignalService))
	r.RegisterCallback(kb.CbNotifyFallOnly, cbNotifyFallOnly.New(deps.SignalService))
	r.RegisterCallback(kb.CbNotifyBoth, cbNotifyBoth.New(deps.SignalService))

	// ── Callback: сигналы ────────────────────────────────────
	r.RegisterCallback(kb.CbSignalsMenu, cbSignalsMenu.New())
	r.RegisterCallback(kb.CbSignalToggleGrowth, cbSignalToggleGrowth.New(deps.SignalService))
	r.RegisterCallback(kb.CbSignalToggleFall, cbSignalToggleFall.New(deps.SignalService))
	r.RegisterCallback(kb.CbSignalSetGrowthThreshold, cbSignalSetGrowth.New(deps.SignalService))
	r.RegisterCallback(kb.CbSignalSetFallThreshold, cbSignalSetFall.New(deps.SignalService))

	// ── Callback: периоды ────────────────────────────────────
	r.RegisterCallback(kb.CbPeriodsMenu, cbPeriodsMenu.New())
	// Wildcard: период → period_select handler
	r.RegisterCallback("period_*", cbPeriodSelect.New(deps.SignalService))
	r.RegisterCallback(kb.CbPeriod1m, cbPeriodSelect.New(deps.SignalService))
	r.RegisterCallback(kb.CbPeriod5m, cbPeriodSelect.New(deps.SignalService))
	r.RegisterCallback(kb.CbPeriod15m, cbPeriodSelect.New(deps.SignalService))
	r.RegisterCallback(kb.CbPeriod30m, cbPeriodSelect.New(deps.SignalService))
	r.RegisterCallback(kb.CbPeriod1h, cbPeriodSelect.New(deps.SignalService))
	r.RegisterCallback(kb.CbPeriod4h, cbPeriodSelect.New(deps.SignalService))
	r.RegisterCallback(kb.CbPeriod1d, cbPeriodSelect.New(deps.SignalService))

	// ── Callback: пороги ────────────────────────────────────
	r.RegisterCallback(kb.CbThresholdsMenu, cbThresholdsMenu.New())

	// ── Callback: профиль ────────────────────────────────────
	r.RegisterCallback(kb.CbProfileMain, cbProfileMain.New())
	r.RegisterCallback(kb.CbProfileStats, cbProfileStats.New())
	r.RegisterCallback(kb.CbProfileSubscription, cbProfileSubscription.New())

	// ── Callback: auth ──────────────────────────────────────
	r.RegisterCallback(kb.CbAuthLogin, cbAuthLogin.New())
	r.RegisterCallback(kb.CbAuthLogout, cbAuthLogout.New())

	// ── Callback: сброс ─────────────────────────────────────
	r.RegisterCallback(kb.CbResetMenu, cbResetMenu.New())
	r.RegisterCallback(kb.CbResetSettings, cbResetSettings.New(deps.UserService))
	r.RegisterCallback(kb.CbResetAll, cbResetAll.New(deps.UserService))

	// ── Callback: with_params (fallback для параметризованных callback) ───
	r.RegisterCallback(kb.CbWithParams, cbWithParams.New())
}
