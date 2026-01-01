// internal/delivery/telegram/buttons.go
package telegram

import (
	"fmt"
	"strings"
)

// ButtonURLBuilder - строитель URL для кнопок
type ButtonURLBuilder struct {
	exchange  string
	baseURLs  map[string]string
	intervals map[int]string
}

// NewButtonURLBuilder создает новый строитель URL для кнопок
func NewButtonURLBuilder(exchange string) *ButtonURLBuilder {
	baseURLs := map[string]string{
		"bybit":   "https://www.bybit.com",
		"binance": "https://www.binance.com",
		"kucoin":  "https://www.kucoin.com",
		"okx":     "https://www.okx.com",
	}

	intervals := map[int]string{
		1:     "1",
		3:     "3",
		5:     "5",
		15:    "15",
		30:    "30",
		60:    "60",
		240:   "240",
		1440:  "1D",
		10080: "1W",
	}

	return &ButtonURLBuilder{
		exchange:  strings.ToLower(exchange),
		baseURLs:  baseURLs,
		intervals: intervals,
	}
}

// GetChartButton создает кнопку "График" с использованием константы
func (b *ButtonURLBuilder) GetChartButton(symbol string) InlineKeyboardButton {
	return InlineKeyboardButton{
		Text: ButtonTexts.Chart,
		URL:  b.GetChartURL(symbol),
	}
}

// GetTradeButton создает кнопку "Торговать" с использованием константы
func (b *ButtonURLBuilder) GetTradeButton(symbol string, periodMinutes int) InlineKeyboardButton {
	return InlineKeyboardButton{
		Text: ButtonTexts.Trade,
		URL:  b.GetTradeURL(symbol, periodMinutes),
	}
}

// GetCoinGeckoButton создает кнопку "CoinGecko" с использованием константы
func (b *ButtonURLBuilder) GetCoinGeckoButton(symbol string) InlineKeyboardButton {
	return InlineKeyboardButton{
		Text: ButtonTexts.CoinGecko,
		URL:  b.GetCoinGeckoURL(symbol),
	}
}

// GetCoinglassButton создает кнопку "Coinglass" с использованием константы
func (b *ButtonURLBuilder) GetCoinglassButton(symbol string) InlineKeyboardButton {
	return InlineKeyboardButton{
		Text: ButtonTexts.Coinglass,
		URL:  b.GetCoinglassURL(symbol),
	}
}

// GetTradingViewButton создает кнопку "TradingView" с использованием константы
func (b *ButtonURLBuilder) GetTradingViewButton(symbol string) InlineKeyboardButton {
	return InlineKeyboardButton{
		Text: ButtonTexts.TradingView,
		URL:  b.GetChartURL(symbol), // TradingView URL тот же, что и для графика
	}
}

// URL методы
func (b *ButtonURLBuilder) GetChartURL(symbol string) string {
	return fmt.Sprintf("https://www.tradingview.com/chart/?symbol=%s:%s", strings.ToUpper(b.exchange), symbol)
}

func (b *ButtonURLBuilder) GetTradeURL(symbol string, periodMinutes int) string {
	interval := b.getTradingInterval(periodMinutes)

	switch b.exchange {
	case "bybit":
		return fmt.Sprintf("%s/trade/usdt/%s?interval=%s", b.baseURLs["bybit"], symbol, interval)
	case "binance":
		return fmt.Sprintf("%s/en/trade/%s?layout=pro&interval=%s", b.baseURLs["binance"], symbol, interval)
	case "kucoin":
		return fmt.Sprintf("%s/trade/%s?interval=%s", b.baseURLs["kucoin"], symbol, interval)
	case "okx":
		return fmt.Sprintf("%s/trade/spot/%s", b.baseURLs["okx"], symbol)
	default:
		return fmt.Sprintf("%s/trade/usdt/%s?interval=%s", b.baseURLs["bybit"], symbol, interval)
	}
}

func (b *ButtonURLBuilder) GetCoinGeckoURL(symbol string) string {
	cleanSymbol := strings.TrimSuffix(strings.TrimSuffix(symbol, "USDT"), "USD")
	return fmt.Sprintf("https://www.coingecko.com/en/coins/%s", strings.ToLower(cleanSymbol))
}

func (b *ButtonURLBuilder) GetCoinglassURL(symbol string) string {
	return fmt.Sprintf("https://www.coinglass.com/tv/%s", symbol)
}

// Создание различных типов клавиатур
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

func (b *ButtonURLBuilder) EnhancedNotificationKeyboard(symbol string, periodMinutes int) *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				b.GetChartButton(symbol),
				b.GetTradeButton(symbol, periodMinutes),
			},
			{
				b.GetCoinGeckoButton(symbol),
				b.GetCoinglassButton(symbol),
			},
		},
	}
}

func (b *ButtonURLBuilder) CounterNotificationKeyboard(symbol string, periodMinutes int) *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: ButtonTexts.Status, CallbackData: fmt.Sprintf("counter_%s", symbol)},
				b.GetTradeButton(symbol, periodMinutes),
			},
		},
	}
}

// Вспомогательный метод
func (b *ButtonURLBuilder) getTradingInterval(periodMinutes int) string {
	if interval, exists := b.intervals[periodMinutes]; exists {
		return interval
	}

	// Находим ближайший интервал
	availableIntervals := []int{1, 3, 5, 15, 30, 60, 240, 1440, 10080}
	for _, interval := range availableIntervals {
		if periodMinutes <= interval {
			return b.intervals[interval]
		}
	}

	return "15" // По умолчанию 15 минут
}
