// internal/delivery/telegram/constants.go
package telegram

// ButtonTexts содержит тексты для кнопок
var ButtonTexts = struct {
	Chart       string
	Trade       string
	CoinGecko   string
	TradingView string
	Coinglass   string
	Settings    string
	Status      string
	Help        string
	Back        string
}{
	Chart:       "📊 График", // Общий текст, меняется в buttonBuilder
	Trade:       "💱 Торговать",
	CoinGecko:   "📰 CoinGecko",
	TradingView: "📈 TradingView",
	Coinglass:   "🧊 Coinglass",
	Settings:    "⚙️ Настройки",
	Status:      "📊 Статус",
	Help:        "📋 Помощь",
	Back:        "🔙 Назад",
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
