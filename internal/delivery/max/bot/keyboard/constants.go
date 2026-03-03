// internal/delivery/max/bot/keyboard/constants.go
package keyboard

// ──────────────────────────────────────────────
// CALLBACK IDENTIFIERS
// ──────────────────────────────────────────────

const (
	// Main menu
	CbStats             = "stats"
	CbSettingsMain      = "settings_main"
	CbNotificationsMenu = "notifications_menu"
	CbSignalsMenu       = "signals_menu"
	CbPeriodsMenu       = "periods_menu"
	CbResetMenu         = "reset_menu"
	CbHelp              = "help"
	CbMenuMain          = "menu_main"
	CbMenuBack          = "menu_back"

	// Auth
	CbAuthLogin  = "auth_login"
	CbAuthLogout = "auth_logout"

	// Profile
	CbProfileMain         = "profile_main"
	CbProfileStats        = "profile_stats"
	CbProfileSubscription = "profile_subscription"

	// Notifications
	CbNotifyToggleAll  = "notify_toggle_all"
	CbNotifyGrowthOnly = "notify_growth_only"
	CbNotifyFallOnly   = "notify_fall_only"
	CbNotifyBoth       = "notify_both"

	// Signals
	CbSignalToggleGrowth       = "signal_toggle_growth"
	CbSignalToggleFall         = "signal_toggle_fall"
	CbSignalSetGrowthThreshold = "signal_set_growth_threshold"
	CbSignalSetFallThreshold   = "signal_set_fall_threshold"

	// Periods
	CbPeriod1m  = "period_1m"
	CbPeriod5m  = "period_5m"
	CbPeriod15m = "period_15m"
	CbPeriod30m = "period_30m"
	CbPeriod1h  = "period_1h"
	CbPeriod4h  = "period_4h"
	CbPeriod1d  = "period_1d"

	// Reset
	CbResetSettings = "reset_settings"
	CbResetAll      = "reset_all"

	// Thresholds
	CbThresholdsMenu = "thresholds_menu"

	// With params
	CbWithParams = "with_params"

	// Linking Telegram account
	CbLinkTelegram = "link_telegram"

	// Trading session
	CbSessionStart    = "session_start"
	CbSessionStop     = "session_stop"
	CbSessionDuration = "session_duration"
)

// ──────────────────────────────────────────────
// BUTTON TEXTS
// ──────────────────────────────────────────────

var Btn = struct {
	// Navigation
	Back     string
	MainMenu string
	Help     string
	Settings string
	Status   string

	// Menu sections
	Profile       string
	Notifications string
	Signals       string
	Periods       string
	Reset         string
	Thresholds    string

	// Auth
	Login  string
	Logout string

	// Notifications
	NotifyToggleAll  string
	NotifyGrowthOnly string
	NotifyFallOnly   string
	NotifyBoth       string

	// Signals
	SignalToggleGrowth string
	SignalToggleFall   string
	ThresholdFormat    string

	// Periods
	Period1m  string
	Period5m  string
	Period15m string
	Period30m string
	Period1h  string
	Period4h  string
	Period1d  string

	// Profile
	ProfileStats        string
	ProfileSubscription string

	// Reset
	ResetAll      string
	ResetSettings string

	// Linking
	LinkTelegram string

	// Trading session
	SessionStart  string
	SessionStop   string
	Duration2h    string
	Duration4h    string
	Duration8h    string
	DurationDay   string
}{
	Back:     "🔙 Назад",
	MainMenu: "🏠 Главное меню",
	Help:     "📋 Помощь",
	Settings: "⚙️ Настройки",
	Status:   "📊 Статус",

	Profile:       "👤 Профиль",
	Notifications: "🔔 Уведомления",
	Signals:       "📈 Сигналы",
	Periods:       "⏱️ Периоды",
	Reset:         "🔄 Сбросить",
	Thresholds:    "🎯 Пороги",

	Login:  "🔑 Войти",
	Logout: "🚪 Выйти",

	NotifyToggleAll:  "✅/❌ Вкл/Выкл уведомления",
	NotifyGrowthOnly: "📈 Только рост",
	NotifyFallOnly:   "📉 Только падение",
	NotifyBoth:       "📊 Все сигналы",

	SignalToggleGrowth: "📈 Рост",
	SignalToggleFall:   "📉 Падение",
	ThresholdFormat:    "%s Порог: %.1f%%",

	Period1m:  "1 минута",
	Period5m:  "5 минут",
	Period15m: "15 минут",
	Period30m: "30 минут",
	Period1h:  "1 час",
	Period4h:  "4 часа",
	Period1d:  "1 день",

	ProfileStats:        "📊 Статистика",
	ProfileSubscription: "💎 Подписка",

	ResetAll:      "🗑️ Сбросить всё",
	ResetSettings: "⚙️ Сбросить настройки",

	LinkTelegram: "🔗 Привязать Telegram",

	SessionStart: "🟢 Начать торговую сессию",
	SessionStop:  "🔴 Завершить сессию",
	Duration2h:   "⏱ 2 часа",
	Duration4h:   "⏱ 4 часа",
	Duration8h:   "⏱ 8 часов",
	DurationDay:  "🕐 Весь день",
}

// Btn1Row — одна кнопка в строке
func Btn1Row(text, cb string) [][]map[string]string {
	return [][]map[string]string{{B(text, cb)}}
}

// BtnRow — несколько кнопок в одной строке
func BtnRow(buttons ...map[string]string) []map[string]string {
	return buttons
}

// B — callback-кнопка (MAX API: type=callback, payload=data)
func B(text, cb string) map[string]string {
	return map[string]string{"type": "callback", "text": text, "payload": cb}
}

// BUrl — кнопка-ссылка (MAX API: type=link)
func BUrl(text, url string) map[string]string {
	return map[string]string{"type": "link", "text": text, "url": url}
}

// BackRow — строка с кнопкой «Назад»
func BackRow(target string) []map[string]string {
	return BtnRow(B(Btn.Back, target))
}

// Keyboard преобразует 2D массив кнопок в MAX attachments format:
// []interface{}{ {"type":"inline_keyboard","payload":{"buttons":[[...]]}} }
func Keyboard(rows [][]map[string]string) interface{} {
	if len(rows) == 0 {
		return nil
	}
	// Конвертируем [][]map[string]string → [][][]interface{} для правильного JSON
	buttons := make([][]map[string]interface{}, len(rows))
	for i, row := range rows {
		buttons[i] = make([]map[string]interface{}, len(row))
		for j, btn := range row {
			m := make(map[string]interface{}, len(btn))
			for k, v := range btn {
				m[k] = v
			}
			buttons[i][j] = m
		}
	}
	return []interface{}{
		map[string]interface{}{
			"type": "inline_keyboard",
			"payload": map[string]interface{}{
				"buttons": buttons,
			},
		},
	}
}
