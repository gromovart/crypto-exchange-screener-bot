package telegram

import (
	"fmt"
	"regexp"
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

// GetChartURL возвращает URL графика с проверкой поддерживаемых символов
func (b *ButtonURLBuilder) GetChartURL(symbol string) string {
	cleanSymbol := strings.ToUpper(symbol)
	baseSymbol := b.extractBaseSymbol(cleanSymbol)

	// Определяем, использовать ли Coinglass
	useCoinglass := b.chartProvider == "coinglass" && b.supportsCoinglass(baseSymbol)

	if useCoinglass {
		return b.GetCoinglassURL(cleanSymbol)
	} else {
		// Всегда TradingView для неподдерживаемых символов
		return b.getTradingViewURL(cleanSymbol)
	}
}

// supportsCoinglass проверяет, поддерживает ли Coinglass этот символ
func (b *ButtonURLBuilder) supportsCoinglass(baseSymbol string) bool {
	// Список символов, которые поддерживает Coinglass
	supportedSymbols := map[string]bool{
		// Основные криптовалюты (Top 100 по market cap)
		"BTC": true, "ETH": true, "BNB": true, "SOL": true, "XRP": true,
		"ADA": true, "DOGE": true, "DOT": true, "LTC": true, "AVAX": true,
		"MATIC": true, "TRX": true, "LINK": true, "UNI": true, "ATOM": true,
		"FIL": true, "ETC": true, "ALGO": true, "VET": true, "AXS": true,
		"SAND": true, "MANA": true, "SHIB": true, "PEPE": true, "FLOKI": true,
		"ARB": true, "OP": true, "IMX": true, "RNDR": true, "TAO": true,
		"FET": true, "ONDO": true, "WIF": true, "BONK": true, "JUP": true,
		"APT": true, "NEAR": true, "AAVE": true, "MKR": true, "SNX": true,
		"CRV": true, "COMP": true, "YFI": true, "SUSHI": true, "CAKE": true,
		"1INCH": true, "RUNE": true, "KAVA": true, "INJ": true, "SEI": true,
		"SUI": true, "TIA": true, "DYM": true, "STRK": true, "ENA": true,
		"BCH": true, "XLM": true, "ICP": true, "HBAR": true, "FTM": true,
		"QNT": true, "EGLD": true, "THETA": true, "XTZ": true,
		"EOS": true, "BSV": true, "OKB": true, "KLAY": true, "NEO": true,

		// Stablecoins
		"USDT": true, "USDC": true, "DAI": true, "TUSD": true, "BUSD": true,
		"USDD": true, "FDUSD": true,

		// Layer 1
		"ONE": true, "FLOW": true, "MINA": true,

		// Мемкоины
		"MEME": true, "FARTCOIN": false, // пример неподдерживаемого

		// AI
		"AGIX": true, "OCEAN": true, "NMR": true,
		"GRT": true,

		// RWA
		"CFG": true, "RIO": true, "TRU": true,

		// Gaming
		"GALA": true, "ENJ": true, "ILV": true, "YGG": true,

		// NFT
		"BLUR": true, "LOOKS": true,

		// Oracles
		"BAND": true, "API3": true, "UMA": true,
	}

	// Проверяем, есть ли символ в списке поддерживаемых
	supported, exists := supportedSymbols[baseSymbol]
	if !exists {
		// Если символа нет в списке, считаем что не поддерживается
		return false
	}

	return supported
}

// extractBaseSymbol извлекает базовый символ (без USDT и т.д.)
func (b *ButtonURLBuilder) extractBaseSymbol(symbol string) string {
	cleanSymbol := strings.ToUpper(symbol)

	// Удаляем суффиксы в правильном порядке (самые длинные сначала)
	suffixes := []string{
		"USDT", "USDC", "BUSD", "FDUSD", "TUSD",
		"BTC", "ETH", "BNB", "EUR", "GBP", "JPY",
		"DAI", "USDD", "USTC",
	}

	for _, suffix := range suffixes {
		if strings.HasSuffix(cleanSymbol, suffix) {
			return strings.TrimSuffix(cleanSymbol, suffix)
		}
	}

	return cleanSymbol
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
	baseSymbol := b.extractBaseSymbol(cleanSymbol)

	// Очищаем символ от специальных символов
	re := regexp.MustCompile(`[^A-Z0-9-]`)
	cleanBaseSymbol := re.ReplaceAllString(baseSymbol, "")

	if cleanBaseSymbol == "" {
		cleanBaseSymbol = "BTC" // fallback
	}

	return fmt.Sprintf("https://www.coinglass.com/pro/%s", cleanBaseSymbol)
}

// GetChartButton создает кнопку "График" с умным выбором провайдера
func (b *ButtonURLBuilder) GetChartButton(symbol string) InlineKeyboardButton {
	cleanSymbol := strings.ToUpper(symbol)
	baseSymbol := b.extractBaseSymbol(cleanSymbol)

	// Определяем, использовать ли Coinglass
	useCoinglass := b.chartProvider == "coinglass" && b.supportsCoinglass(baseSymbol)

	var buttonText string
	if useCoinglass {
		buttonText = "🧊 Coinglass"
	} else {
		buttonText = "📈 TradingView"
	}

	return InlineKeyboardButton{
		Text: buttonText,
		URL:  b.GetChartURL(symbol),
	}
}

// GetTradeURL возвращает URL для торговли
func (b *ButtonURLBuilder) GetTradeURL(symbol string, periodMinutes int) string {
	cleanSymbol := strings.ToUpper(symbol)
	intervalParam := b.getIntervalString(periodMinutes)

	switch b.exchange {
	case "binance":
		return fmt.Sprintf("https://www.binance.com/en/trade/%s?layout=pro&%s",
			strings.Replace(cleanSymbol, "USDT", "_USDT", 1), intervalParam)

	case "kucoin":
		return fmt.Sprintf("https://www.kucoin.com/trade/%s?%s", cleanSymbol, intervalParam)

	case "okx":
		return fmt.Sprintf("https://www.okx.com/trade-spot/%s?%s", strings.ToLower(cleanSymbol), intervalParam)

	default: // bybit и другие
		return fmt.Sprintf("https://www.bybit.com/trade/usdt/%s?%s", cleanSymbol, intervalParam)
	}
}

// GetCoinGeckoURL возвращает URL CoinGecko
func (b *ButtonURLBuilder) GetCoinGeckoURL(symbol string) string {
	// Преобразуем символ биржи в название монеты для CoinGecko
	baseSymbol := strings.ToLower(b.extractBaseSymbol(symbol))
	return fmt.Sprintf("https://www.coingecko.com/en/coins/%s", baseSymbol)
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
	cleanSymbol := strings.ToUpper(symbol)
	baseSymbol := b.extractBaseSymbol(cleanSymbol)

	// Если основной провайдер coinglass и символ поддерживается, показываем tradingview
	if b.chartProvider == "coinglass" && b.supportsCoinglass(baseSymbol) {
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
				b.GetChartButton(symbol),
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
				{Text: notifyText, CallbackData: CallbackNotifyToggle},
				{Text: "⚙️ Изменить пороги", CallbackData: CallbackThresholdsMenu},
			},
			{
				{Text: "📊 Изменить период", CallbackData: CallbackPeriodManage},
				{Text: testModeText, CallbackData: "toggle_test_mode"},
			},
			{
				{Text: ButtonTexts.Back, CallbackData: CallbackMenuBack},
			},
		},
	}
}

// getIntervalString преобразует минуты в строку интервала
func (b *ButtonURLBuilder) getIntervalString(minutes int) string {
	switch b.exchange {
	case "bybit":
		// Bybit использует параметр defaultChartInterval с числовым значением в минутах
		// Например: defaultChartInterval=60 (1 час), defaultChartInterval=240 (4 часа)
		return fmt.Sprintf("defaultChartInterval=%d", minutes)

	case "binance":
		// Binance: 1m, 3m, 5m, 15m, 30m, 1h, 2h, 4h, 6h, 8h, 12h, 1d
		switch minutes {
		case 1:
			return "interval=1m"
		case 3:
			return "interval=3m"
		case 5:
			return "interval=5m"
		case 15:
			return "interval=15m"
		case 30:
			return "interval=30m"
		case 60:
			return "interval=1h"
		case 120:
			return "interval=2h"
		case 240:
			return "interval=4h"
		case 360:
			return "interval=6h"
		case 480:
			return "interval=8h"
		case 720:
			return "interval=12h"
		case 1440:
			return "interval=1d"
		default:
			return "interval=15m"
		}

	case "kucoin":
		// KuCoin использует простой параметр interval с числовым значением
		return fmt.Sprintf("interval=%d", minutes)

	case "okx":
		// OKX использует granularity параметр
		switch minutes {
		case 1:
			return "granularity=60"
		case 5:
			return "granularity=300"
		case 15:
			return "granularity=900"
		case 30:
			return "granularity=1800"
		case 60:
			return "granularity=3600"
		case 240:
			return "granularity=14400"
		case 1440:
			return "granularity=86400"
		default:
			return "granularity=900" // 15 минут по умолчанию
		}

	default:
		// Для других бирж используем Bybit формат как default
		return fmt.Sprintf("defaultChartInterval=%d", minutes)
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

// GetBaseSymbol возвращает базовый символ (без суффикса)
func (b *ButtonURLBuilder) GetBaseSymbol(symbol string) string {
	return b.extractBaseSymbol(symbol)
}

// IsSymbolSupported проверяет, поддерживается ли символ текущим провайдером
func (b *ButtonURLBuilder) IsSymbolSupported(symbol string) bool {
	baseSymbol := b.extractBaseSymbol(strings.ToUpper(symbol))

	if b.chartProvider == "coinglass" {
		return b.supportsCoinglass(baseSymbol)
	}

	// TradingView поддерживает все символы
	return true
}
