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
	Chart:       "📊 График",
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
