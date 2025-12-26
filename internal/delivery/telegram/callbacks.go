// internal/telegram/callbacks.go
package telegram

// Callback constants
const (
	// Main menu
	CallbackStats     = "stats"
	CallbackSettings  = "settings"
	CallbackNotifyOn  = "notify_on"
	CallbackNotifyOff = "notify_off"

	// Settings menu
	CallbackSettingsNotifyToggle = "settings_notify_toggle"
	CallbackSettingsSignalType   = "settings_signal_type"
	CallbackSettingsChangePeriod = "settings_change_period"
	CallbackSettingsResetCounter = "settings_reset_counter"
	CallbackSettingsBack         = "settings_back"
	CallbackSettingsBackToMain   = "settings_back_to_main"

	// Signal type menu
	CallbackTrackGrowthOnly = "settings_track_growth_only"
	CallbackTrackFallOnly   = "settings_track_fall_only"
	CallbackTrackBoth       = "settings_track_both"

	// Period menu
	CallbackPeriod5m  = "settings_period_5m"
	CallbackPeriod15m = "settings_period_15m"
	CallbackPeriod30m = "settings_period_30m"
	CallbackPeriod1h  = "settings_period_1h"
	CallbackPeriod4h  = "settings_period_4h"
	CallbackPeriod1d  = "settings_period_1d"

	// Reset menu
	CallbackResetAll      = "settings_reset_all"
	CallbackResetBySymbol = "settings_reset_by_symbol"

	// Counter callbacks (for existing functionality)
	CallbackCounterSettings  = "counter_settings"
	CallbackCounterNotifyOn  = "counter_notify_on"
	CallbackCounterNotifyOff = "counter_notify_off"
)
