// internal/delivery/telegram/button_url_builder.go
package telegram

import (
	"fmt"
	"strings"
)

// ButtonURLBuilder - строитель URL для кнопок
type ButtonURLBuilder struct {
	exchange      string
	chartProvider string // coinglass или tradingview
}

// NewButtonURLBuilder создает новый строитель URL
func NewButtonURLBuilder(exchange string) *ButtonURLBuilder {
	return &ButtonURLBuilder{
		exchange:      strings.ToLower(exchange),
		chartProvider: "coinglass", // значение по умолчанию
	}
}

// NewButtonURLBuilderWithProvider создает строитель URL с указанным провайдером графиков
func NewButtonURLBuilderWithProvider(exchange, chartProvider string) *ButtonURLBuilder {
	provider := strings.ToLower(chartProvider)
	if provider != "coinglass" && provider != "tradingview" {
		provider = "coinglass" // fallback на coinglass
	}

	return &ButtonURLBuilder{
		exchange:      strings.ToLower(exchange),
		chartProvider: provider,
	}
}

// SetChartProvider устанавливает провайдера графиков
func (b *ButtonURLBuilder) SetChartProvider(provider string) {
	provider = strings.ToLower(provider)
	if provider == "coinglass" || provider == "tradingview" {
		b.chartProvider = provider
	}
}

// GetChartURL возвращает URL графика (ЗАМЕНЕН - теперь зависит от chartProvider)
func (b *ButtonURLBuilder) GetChartURL(symbol string) string {
	cleanSymbol := strings.ToUpper(symbol)

	switch b.chartProvider {
	case "tradingview":
		return b.getTradingViewURL(cleanSymbol)
	case "coinglass":
		fallthrough
	default:
		return b.GetCoinglassURL(cleanSymbol)
	}
}

// getTradingViewURL возвращает URL TradingView
func (b *ButtonURLBuilder) getTradingViewURL(symbol string) string {
	// TradingView использует разные коды для бирж
	var exchangeCode string
	switch b.exchange {
	case "binance":
		exchangeCode = "BINANCE"
	case "kucoin":
		exchangeCode = "KUCOIN"
	case "okx":
		exchangeCode = "OKX"
	case "bybit":
		exchangeCode = "BYBIT"
	default:
		exchangeCode = "BYBIT"
	}

	return fmt.Sprintf("https://www.tradingview.com/chart/?symbol=%s:%s",
		exchangeCode, symbol)
}

// GetCoinglassURL возвращает URL Coinglass
func (b *ButtonURLBuilder) GetCoinglassURL(symbol string) string {
	cleanSymbol := strings.ToUpper(symbol)

	// Coinglass использует форматы вроде BTC (без USDT)
	baseSymbol := strings.ReplaceAll(cleanSymbol, "USDT", "")

	// Для некоторых пар может потребоваться преобразование
	symbolMap := map[string]string{
		"BTCUSDT":  "BTC",
		"ETHUSDT":  "ETH",
		"BNBUSDT":  "BNB",
		"SOLUSDT":  "SOL",
		"XRPUSDT":  "XRP",
		"ADAUSDT":  "ADA",
		"DOGEUSDT": "DOGE",
		"DOTUSDT":  "DOT",
		"LTCUSDT":  "LTC",
	}

	if mapped, ok := symbolMap[cleanSymbol]; ok {
		baseSymbol = mapped
	}

	return fmt.Sprintf("https://www.coinglass.com/pro/%s", baseSymbol)
}

// GetTradeURL возвращает URL для торговли
func (b *ButtonURLBuilder) GetTradeURL(symbol string, periodMinutes int) string {
	cleanSymbol := strings.ToUpper(symbol)
	interval := b.getIntervalString(periodMinutes)

	switch b.exchange {
	case "binance":
		return fmt.Sprintf("https://www.binance.com/en/trade/%s?layout=pro&interval=%s", cleanSymbol, interval)
	case "kucoin":
		return fmt.Sprintf("https://www.kucoin.com/trade/%s", cleanSymbol)
	case "okx":
		return fmt.Sprintf("https://www.okx.com/trade-spot/%s", strings.ToLower(symbol))
	default: // bybit
		return fmt.Sprintf("https://www.bybit.com/trade/usdt/%s?interval=%s", cleanSymbol, interval)
	}
}

// GetCoinGeckoURL возвращает URL CoinGecko
func (b *ButtonURLBuilder) GetCoinGeckoURL(symbol string) string {
	// Преобразуем символ биржи в название монеты для CoinGecko
	baseSymbol := strings.ToLower(strings.ReplaceAll(symbol, "USDT", ""))
	return fmt.Sprintf("https://www.coingecko.com/en/coins/%s", baseSymbol)
}

// GetChartButton создает кнопку "График" (ЗАМЕНЕН - теперь зависит от chartProvider)
func (b *ButtonURLBuilder) GetChartButton(symbol string) InlineKeyboardButton {
	// Меняем текст кнопки в зависимости от провайдера
	buttonText := ButtonTexts.Chart
	if b.chartProvider == "coinglass" {
		buttonText = "🧊 Coinglass"
	} else if b.chartProvider == "tradingview" {
		buttonText = "📈 TradingView"
	}

	return InlineKeyboardButton{
		Text: buttonText,
		URL:  b.GetChartURL(symbol),
	}
}

// GetTradeButton создает кнопку "Торговать"
func (b *ButtonURLBuilder) GetTradeButton(symbol string, periodMinutes int) InlineKeyboardButton {
	return InlineKeyboardButton{
		Text: ButtonTexts.Trade,
		URL:  b.GetTradeURL(symbol, periodMinutes),
	}
}

// GetCoinGeckoButton создает кнопку "CoinGecko"
func (b *ButtonURLBuilder) GetCoinGeckoButton(symbol string) InlineKeyboardButton {
	return InlineKeyboardButton{
		Text: ButtonTexts.CoinGecko,
		URL:  b.GetCoinGeckoURL(symbol),
	}
}

// GetCoinglassButton создает кнопку "Coinglass"
func (b *ButtonURLBuilder) GetCoinglassButton(symbol string) InlineKeyboardButton {
	return InlineKeyboardButton{
		Text: "🧊 Coinglass",
		URL:  b.GetCoinglassURL(symbol),
	}
}

// GetTradingViewButton создает кнопку "TradingView"
func (b *ButtonURLBuilder) GetTradingViewButton(symbol string) InlineKeyboardButton {
	return InlineKeyboardButton{
		Text: "📈 TradingView",
		URL:  b.getTradingViewURL(symbol),
	}
}

// StandardNotificationKeyboard создает стандартную клавиатуру для уведомлений
func (b *ButtonURLBuilder) StandardNotificationKeyboard(symbol string, periodMinutes int) *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				b.GetChartButton(symbol),
				b.GetTradeButton(symbol, periodMinutes),
			},
		},
	}
}

// EnhancedNotificationKeyboard создает расширенную клавиатуру для уведомлений
func (b *ButtonURLBuilder) EnhancedNotificationKeyboard(symbol string, periodMinutes int) *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				b.GetChartButton(symbol),
				b.GetTradeButton(symbol, periodMinutes),
			},
			{
				b.GetCoinGeckoButton(symbol),
				// В зависимости от провайдера используем другую кнопку
				b.getAdditionalChartButton(symbol),
			},
		},
	}
}

// getAdditionalChartButton возвращает дополнительную кнопку графика
func (b *ButtonURLBuilder) getAdditionalChartButton(symbol string) InlineKeyboardButton {
	// Если основной провайдер coinglass, то показываем tradingview и наоборот
	if b.chartProvider == "coinglass" {
		return b.GetTradingViewButton(symbol)
	} else {
		return b.GetCoinglassButton(symbol)
	}
}

// CounterNotificationKeyboard создает клавиатуру для счетчика
func (b *ButtonURLBuilder) CounterNotificationKeyboard(symbol string, periodMinutes int) *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: ButtonTexts.Status, CallbackData: CallbackStats},
				b.GetTradeButton(symbol, periodMinutes),
			},
		},
	}
}

// UpdateSettingsKeyboard создает клавиатуру настроек с актуальными статусами
func (b *ButtonURLBuilder) UpdateSettingsKeyboard(bot *TelegramBot) *InlineKeyboardMarkup {
	notificationsEnabled := bot.IsNotifyEnabled()
	testMode := bot.IsTestMode()

	notifyText := "🔔 Включить уведомления"
	if notificationsEnabled {
		notifyText = "🔕 Выключить уведомления"
	}

	testModeText := "🧪 Включить тестовый режим"
	if testMode {
		testModeText = "🚫 Выключить тестовый режим"
	}

	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: notifyText, CallbackData: CallbackSettingsNotifyToggle},
				{Text: "⚙️ Изменить пороги", CallbackData: "change_thresholds"},
			},
			{
				{Text: "📊 Изменить период", CallbackData: CallbackSettingsChangePeriod},
				{Text: testModeText, CallbackData: "toggle_test_mode"},
			},
			{
				{Text: ButtonTexts.Back, CallbackData: CallbackSettingsBack},
			},
		},
	}
}

// getIntervalString преобразует минуты в строку интервала
func (b *ButtonURLBuilder) getIntervalString(minutes int) string {
	switch minutes {
	case 5:
		return "5"
	case 15:
		return "15"
	case 30:
		return "30"
	case 60:
		return "60"
	case 240:
		return "240"
	case 1440:
		return "1D"
	default:
		return "15"
	}
}

// GetExchange возвращает биржу
func (b *ButtonURLBuilder) GetExchange() string {
	return b.exchange
}

// GetChartProvider возвращает провайдера графиков
func (b *ButtonURLBuilder) GetChartProvider() string {
	return b.chartProvider
}
