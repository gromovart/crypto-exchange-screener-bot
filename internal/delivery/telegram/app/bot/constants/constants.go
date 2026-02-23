// internal/delivery/telegram/app/bot/constants/constants.go
package constants

// ButtonTexts содержит тексты для кнопок
var ButtonTexts = struct {
	Chart         string
	Trade         string
	CoinGecko     string
	TradingView   string
	Coinglass     string
	Settings      string
	Status        string
	Help          string
	Back          string
	Documentation string // новое
	Support       string // новое
}{
	Chart:         "📊 График",
	Trade:         "💱 Торговать",
	CoinGecko:     "📰 CoinGecko",
	TradingView:   "📈 TradingView",
	Coinglass:     "🧊 Coinglass",
	Settings:      "⚙️ Настройки",
	Status:        "📊 Статус",
	Help:          "📋 Помощь",
	Back:          "🔙 Назад",
	Documentation: "📚 Полная документация",
	Support:       "📧 Поддержка",
}

// AuthButtonTexts содержит тексты для кнопок авторизации
var AuthButtonTexts = struct {
	Profile             string
	Settings            string
	Notifications       string
	Stats               string
	Thresholds          string
	Periods             string
	Language            string
	Timezone            string
	DisplayMode         string
	Login               string
	Logout              string
	Premium             string
	Advanced            string
	Admin               string
	Users               string
	System              string
	Logs                string
	Analytics           string
	Priority            string
	CustomNotifications string
	ResetSettings       string
	Toggle              string
}{
	Profile:             "🔑 Профиль",
	Settings:            "⚙️ Настройки",
	Notifications:       "🔔 Уведомления",
	Stats:               "📊 Статистика",
	Thresholds:          "🎯 Пороги",
	Periods:             "⏱️ Периоды",
	Language:            "🌐 Язык",
	Timezone:            "🕐 Часовой пояс",
	DisplayMode:         "👁️ Отображение",
	Login:               "🔑 Войти",
	Logout:              "🚪 Выйти",
	Premium:             "🌟 Премиум",
	Advanced:            "🚀 Расширенная",
	Admin:               "👑 Админ",
	Users:               "👥 Пользователи",
	System:              "⚙️ Система",
	Logs:                "🔄 Логи",
	Analytics:           "📈 Аналитика",
	Priority:            "⏱️ Приоритет",
	CustomNotifications: "🔔 Кастомные",
	ResetSettings:       "🔄 Сбросить",
	Toggle:              "🔄",
}

// ButtonStyles содержит стили для кнопок
var ButtonStyles = struct {
	Primary   string
	Secondary string
	Success   string
	Danger    string
	Warning   string
	Info      string
	Light     string
	Dark      string
	Link      string
}{
	Primary:   "primary",
	Secondary: "secondary",
	Success:   "success",
	Danger:    "danger",
	Warning:   "warning",
	Info:      "info",
	Light:     "light",
	Dark:      "dark",
	Link:      "link",
}

// SignalIcons содержит иконки для разных типов сигналов
var SignalIcons = struct {
	Growth     string
	Fall       string
	Extreme    string
	Divergence string
	Counter    string
	Test       string
}{
	Growth:     "🚀",
	Fall:       "📉",
	Extreme:    "⚡",
	Divergence: "🔀",
	Counter:    "📊",
	Test:       "🧪",
}

// SignalTypes содержит типы сигналов
var SignalTypes = struct {
	Growth        string
	Fall          string
	ExtremeOI     string
	Divergence    string
	CounterGrowth string
	CounterFall   string
}{
	Growth:        "growth",
	Fall:          "fall",
	ExtremeOI:     "extreme_oi",
	Divergence:    "divergence",
	CounterGrowth: "counter_growth",
	CounterFall:   "counter_fall",
}

// DirectionIcons содержит иконки направлений
var DirectionIcons = struct {
	Up      string
	Down    string
	Neutral string
	Bullish string
	Bearish string
}{
	Up:      "⬆️",
	Down:    "⬇️",
	Neutral: "➡️",
	Bullish: "🐂",
	Bearish: "🐻",
}

// MenuButtonTexts содержит тексты для кнопок меню
var MenuButtonTexts = struct {
	Reset         string
	ResetAll      string
	ResetCounters string
	ResetBySymbol string
	Signals       string
	MainMenu      string
	Profile       string
	Notifications string
	Periods       string
}{
	Reset:         "🔄 Сбросить",
	ResetAll:      "🗑️ Сбросить все",
	ResetCounters: "📊 Сбросить счетчики",
	ResetBySymbol: "🔤 Сбросить по символу",
	Signals:       "📈 Сигналы",
	MainMenu:      "🏠 Главное меню",
	Profile:       "👤 Профиль",
	Notifications: "🔔 Уведомления",
	Periods:       "⏱️ Периоды",
}

// NotificationButtonTexts содержит тексты для кнопок уведомлений
var NotificationButtonTexts = struct {
	ToggleAll  string
	GrowthOnly string
	FallOnly   string
	Both       string
	NotifyOn   string
	NotifyOff  string
}{
	ToggleAll:  "✅/❌ Включить/Выключить",
	GrowthOnly: "📈 Только рост",
	FallOnly:   "📉 Только падение",
	Both:       "📊 Все сигналы",
	NotifyOn:   "✅ Включить",
	NotifyOff:  "❌ Выключить",
}

// PeriodButtonTexts содержит тексты для кнопок периодов
var PeriodButtonTexts = struct {
	Period1m     string
	Period5m     string
	Period15m    string
	Period30m    string
	Period1h     string
	Period4h     string
	Period1d     string
	ManageAdd    string
	ManageRemove string
	ManageReset  string
}{
	Period1m:     "⏱️ 1 минута",
	Period5m:     "⏱️ 5 минут",
	Period15m:    "⏱️ 15 минут",
	Period30m:    "⏱️ 30 минут",
	Period1h:     "⏱️ 1 час",
	Period4h:     "⏱️ 4 часа",
	Period1d:     "⏱️ 1 день",
	ManageAdd:    "➕ Добавить период",
	ManageRemove: "➖ Удалить период",
	ManageReset:  "🔄 Сбросить выбор",
}

// ThresholdButtonTexts содержит тексты для кнопок порогов
var ThresholdButtonTexts = struct {
	Growth string
	Fall   string
}{
	Growth: "📈 Мин. рост: X%",
	Fall:   "📉 Мин. падение: X%",
}

// SymbolButtonTexts содержит тексты для кнопок символов
var SymbolButtonTexts = struct {
	BTCUSDT string
	ETHUSDT string
	BNBUSDT string
	SOLUSDT string
	XRPUSDT string
	Back    string
}{
	BTCUSDT: "BTC/USDT",
	ETHUSDT: "ETH/USDT",
	BNBUSDT: "BNB/USDT",
	SOLUSDT: "SOL/USDT",
	XRPUSDT: "XRP/USDT",
	Back:    "🔙 Назад к сбросу",
}

// SessionButtonTexts содержит тексты кнопок торговой сессии
var SessionButtonTexts = struct {
	Start       string
	Stop        string
	Duration2h  string
	Duration4h  string
	Duration8h  string
	DurationDay string
}{
	Start:       "🟢 Начать торговую сессию",
	Stop:        "🔴 Завершить торговую сессию",
	Duration2h:  "⏱ 2 часа",
	Duration4h:  "⏱ 4 часа",
	Duration8h:  "⏱ 8 часов",
	DurationDay: "🕐 Весь день",
}

// TestButtonTexts содержит тексты для тестовых кнопок
var TestButtonTexts = struct {
	Test       string
	TestOK     string
	TestCancel string
	ToggleTest string
	Chart      string
}{
	Test:       "🧪 Тестовое сообщение",
	TestOK:     "✅ Тест OK",
	TestCancel: "❌ Тест отмена",
	ToggleTest: "🧪 Переключить тестовый режим",
	Chart:      "📈 Графики",
}

// SignalButtonTexts содержит тексты для кнопок меню сигналов
var SignalButtonTexts = struct {
	ToggleGrowth    string
	ToggleFall      string
	GrowthThreshold string
	FallThreshold   string
	Sensitivity     string
	History         string
	TestSignal      string
	ThresholdFormat string
}{
	ToggleGrowth:    "📈 Рост",
	ToggleFall:      "📉 Падение",
	GrowthThreshold: "📈 Порог роста",
	FallThreshold:   "📉 Порог падения",
	Sensitivity:     "🎯 Чувствительность",
	History:         "📊 История сигналов",
	TestSignal:      "⚡ Тестовый сигнал",
	ThresholdFormat: "%s Порог: %.1f%%",
}

// CommandButtonTexts содержит тексты для кнопок команд
var CommandButtonTexts = struct {
	Start         string
	Help          string
	Profile       string
	Settings      string
	Notifications string
	Periods       string
	Thresholds    string
	Commands      string
	Stats         string
	Back          string
}{
	Start:         "🚀 /start",
	Help:          "📋 /help",
	Profile:       "👤 /profile",
	Settings:      "⚙️ /settings",
	Notifications: "🔔 /notifications",
	Periods:       "⏱️ /periods",
	Thresholds:    "🎯 /thresholds",
	Commands:      "📜 /commands",
	Stats:         "📊 /stats",
	Back:          "🔙 Назад",
}

// CommandDescriptions содержит описания для команд меню
var CommandDescriptions = struct {
	Start         string
	Help          string
	Buy           string
	Profile       string
	Settings      string
	Notifications string
	Periods       string
	Thresholds    string
	Commands      string
	Stats         string
	PaySupport    string
	Terms         string
}{
	Start:         "Запустить бота",
	Help:          "Помощь и инструкции",
	Buy:           "Купить подписку",
	Profile:       "Мой профиль",
	Settings:      "Настройки",
	Notifications: "Управление уведомлениями",
	Periods:       "Периоды анализа",
	Thresholds:    "Пороги сигналов",
	Commands:      "Список всех команд",
	Stats:         "Статистика системы",
	PaySupport:    "Поддержка по платежам",
	Terms:         "Условия использования",
}

// PaymentButtonTexts содержит тексты для кнопок платежей
var PaymentButtonTexts = struct {
	Buy         string
	Plans       string
	Confirm     string
	Cancel      string
	History     string
	BackToPlans string
	SelectPlan  string
	PayNow      string
	CheckStatus string
}{
	Buy:         "💎 Купить подписку",
	Plans:       "📋 Тарифные планы",
	Confirm:     "✅ Подтвердить оплату",
	Cancel:      "❌ Отмена",
	History:     "📊 История платежей",
	BackToPlans: "← К планам",
	SelectPlan:  "📋 Выбрать план",
	PayNow:      "💳 Оплатить сейчас",
	CheckStatus: "🔄 Проверить статус",
}

// PaymentConstants содержит callback'и и команды для платежей
var PaymentConstants = struct {
	CommandBuy             string
	CallbackPaymentPlan    string
	CallbackPaymentConfirm string
	CallbackPaymentSuccess string
	CallbackPaymentFailed  string
	CallbackPaymentCancel  string
	CallbackPaymentHistory string
	CallbackPaymentCheck   string
	CallbackPaymentSupport string
}{
	CommandBuy:             "buy",
	CallbackPaymentPlan:    "payment_plan:",
	CallbackPaymentConfirm: "payment_confirm:",
	CallbackPaymentSuccess: "payment_success:",
	CallbackPaymentFailed:  "payment_failed:",
	CallbackPaymentCancel:  "payment_cancel",
	CallbackPaymentHistory: "payment_history",
	CallbackPaymentCheck:   "payment_check",
	CallbackPaymentSupport: "payment_support",
}

// PaymentDescriptions содержит описания для платежных команд
var PaymentDescriptions = struct {
	Buy string
}{
	Buy: "Покупка подписки через Telegram Stars",
}
