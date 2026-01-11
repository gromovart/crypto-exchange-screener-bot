// internal/delivery/telegram/callbacks.go
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

	// ============== AUTH CALLBACKS ==============
	// Auth callbacks
	CallbackAuthProfile       = "auth_profile"
	CallbackAuthSettings      = "auth_settings"
	CallbackAuthNotifications = "auth_notifications"
	CallbackAuthStats         = "auth_stats"
	CallbackAuthThresholds    = "auth_thresholds"
	CallbackAuthLogin         = "auth_login"
	CallbackAuthLogout        = "auth_logout"

	// Admin callbacks
	CallbackAdminUsers  = "admin_users"
	CallbackAdminStats  = "admin_stats"
	CallbackAdminSystem = "admin_system"
	CallbackAdminLogs   = "admin_logs"
	CallbackAdminBack   = "admin_back"

	// Premium callbacks
	CallbackPremiumAnalytics = "premium_analytics"
	CallbackPremiumSignals   = "premium_signals"
	CallbackPremiumPriority  = "premium_priority"
	CallbackPremiumBack      = "premium_back"

	// Settings callbacks
	CallbackSettingsToggleNotifications = "settings_toggle_notifications"
	CallbackSettingsToggleGrowth        = "settings_toggle_growth"
	CallbackSettingsToggleFall          = "settings_toggle_fall"
	CallbackSettingsToggleContinuous    = "settings_toggle_continuous"
	CallbackSettingsSetQuietHours       = "settings_set_quiet_hours"
	CallbackSettingsSetGrowthThreshold  = "settings_set_growth_threshold"
	CallbackSettingsSetFallThreshold    = "settings_set_fall_threshold"
	CallbackSettingsThreshold2          = "settings_threshold_2"
	CallbackSettingsThreshold3          = "settings_threshold_3"
	CallbackSettingsThreshold5          = "settings_threshold_5"
	CallbackSettingsReset               = "settings_reset"
	CallbackSettingsPeriod1m            = "settings_period_1"
	CallbackSettingsPeriod5m            = "settings_period_5"
	CallbackSettingsPeriod15m           = "settings_period_15"
	CallbackSettingsPeriod30m           = "settings_period_30"
	CallbackSettingsPeriod60m           = "settings_period_60"
	CallbackSettingsPeriod240m          = "settings_period_240"
	CallbackSettingsPeriod1440m         = "settings_period_1440"
	CallbackSettingsLanguageRu          = "settings_language_ru"
	CallbackSettingsLanguageEn          = "settings_language_en"
	CallbackSettingsLanguageEs          = "settings_language_es"
	CallbackSettingsLanguageZh          = "settings_language_zh"

	// Advanced callbacks
	CallbackAdvancedCharts   = "advanced_charts"
	CallbackAdvancedStats    = "advanced_stats"
	CallbackAdvancedAnalysis = "advanced_analysis"
	CallbackAdvancedRisks    = "advanced_risks"
	CallbackAdvancedReports  = "advanced_reports"
	CallbackAdvancedBack     = "advanced_back"

	// Admin users callbacks
	CallbackAdminUsersSearch = "admin_users_search"
	CallbackAdminUsersList   = "admin_users_list"
	CallbackAdminUsersRoles  = "admin_users_roles"
	CallbackAdminUsersStatus = "admin_users_status"
)
