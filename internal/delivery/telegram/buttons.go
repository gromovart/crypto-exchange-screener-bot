// internal/delivery/telegram/buttons.go
package telegram

import (
	"fmt"
	"strings"
)

// ButtonURLBuilder - строитель URL для кнопок
type ButtonURLBuilder struct {
	exchange  string
	baseURLs  map[string]string // Базовые URL для разных бирж
	intervals map[int]string    // Интервалы для торговли
}

// NewButtonURLBuilder создает новый строитель URL для кнопок
func NewButtonURLBuilder(exchange string) *ButtonURLBuilder {
	// Базовые URL для разных бирж
	baseURLs := map[string]string{
		"bybit":   "https://www.bybit.com",
		"binance": "https://www.binance.com",
		"kucoin":  "https://www.kucoin.com",
	}

	// Интервалы для торговли Bybit
	intervals := map[int]string{
		1:    "1",   // 1 минута
		5:    "5",   // 5 минут
		15:   "15",  // 15 минут
		30:   "30",  // 30 минут
		60:   "60",  // 1 час
		240:  "240", // 4 часа
		1440: "1D",  // 1 день
	}

	return &ButtonURLBuilder{
		exchange:  strings.ToLower(exchange),
		baseURLs:  baseURLs,
		intervals: intervals,
	}
}

// GetChartButton создает кнопку "График"
func (b *ButtonURLBuilder) GetChartButton(symbol string) InlineKeyboardButton {
	return InlineKeyboardButton{
		Text: "📊 График",
		URL:  b.GetChartURL(symbol),
	}
}

// GetTradeButton создает кнопку "Торговать"
func (b *ButtonURLBuilder) GetTradeButton(symbol string, periodMinutes int) InlineKeyboardButton {
	return InlineKeyboardButton{
		Text: "💱 Торговать",
		URL:  b.GetTradeURL(symbol, periodMinutes),
	}
}

// GetChartURL возвращает URL графика для символа
func (b *ButtonURLBuilder) GetChartURL(symbol string) string {
	switch b.exchange {
	case "bybit":
		return fmt.Sprintf("https://www.tradingview.com/chart/?symbol=BYBIT:%s", symbol)
	case "binance":
		return fmt.Sprintf("https://www.tradingview.com/chart/?symbol=BINANCE:%s", symbol)
	case "kucoin":
		return fmt.Sprintf("https://www.tradingview.com/chart/?symbol=KUCOIN:%s", symbol)
	default:
		return fmt.Sprintf("https://www.tradingview.com/chart/?symbol=BYBIT:%s", symbol)
	}
}

// GetTradeURL возвращает URL для торговли символом
func (b *ButtonURLBuilder) GetTradeURL(symbol string, periodMinutes int) string {
	interval := b.getTradingInterval(periodMinutes)

	switch b.exchange {
	case "bybit":
		return fmt.Sprintf("%s/trade/usdt/%s?interval=%s", b.baseURLs["bybit"], symbol, interval)
	case "binance":
		return fmt.Sprintf("%s/en/trade/%s?layout=pro&interval=%s", b.baseURLs["binance"], symbol, interval)
	case "kucoin":
		return fmt.Sprintf("%s/trade/%s", b.baseURLs["kucoin"], symbol)
	default:
		return fmt.Sprintf("%s/trade/usdt/%s?interval=%s", b.baseURLs["bybit"], symbol, interval)
	}
}

// GetCoinGeckoButton создает кнопку "CoinGecko"
func (b *ButtonURLBuilder) GetCoinGeckoButton(symbol string) InlineKeyboardButton {
	cleanSymbol := strings.TrimSuffix(symbol, "USDT")
	cleanSymbol = strings.TrimSuffix(cleanSymbol, "USD")

	return InlineKeyboardButton{
		Text: "📰 CoinGecko",
		URL:  fmt.Sprintf("https://www.coingecko.com/en/coins/%s", strings.ToLower(cleanSymbol)),
	}
}

// GetCoinglassButton создает кнопку "Coinglass"
func (b *ButtonURLBuilder) GetCoinglassButton(symbol string) InlineKeyboardButton {
	return InlineKeyboardButton{
		Text: "🧊 Coinglass",
		URL:  fmt.Sprintf("https://www.coinglass.com/tv/%s", symbol),
	}
}

// GetTradingViewButton создает кнопку "TradingView"
func (b *ButtonURLBuilder) GetTradingViewButton(symbol string) InlineKeyboardButton {
	return InlineKeyboardButton{
		Text: "📈 TradingView",
		URL:  b.GetChartURL(symbol),
	}
}

// getTradingInterval преобразует минуты в интервал торгового терминала
func (b *ButtonURLBuilder) getTradingInterval(periodMinutes int) string {
	if interval, exists := b.intervals[periodMinutes]; exists {
		return interval
	}

	// Определяем ближайший доступный интервал
	switch {
	case periodMinutes <= 1:
		return "1"
	case periodMinutes <= 5:
		return "5"
	case periodMinutes <= 15:
		return "15"
	case periodMinutes <= 30:
		return "30"
	case periodMinutes <= 60:
		return "60"
	case periodMinutes <= 240:
		return "240"
	default:
		return "1D"
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
	cleanSymbol := strings.TrimSuffix(symbol, "USDT")
	cleanSymbol = strings.TrimSuffix(cleanSymbol, "USD")

	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				b.GetTradeButton(symbol, periodMinutes),
				b.GetChartButton(symbol),
			},
			{
				b.GetCoinGeckoButton(symbol),
				b.GetCoinglassButton(symbol),
			},
		},
	}
}

// CounterNotificationKeyboard создает клавиатуру для уведомлений счетчика
func (b *ButtonURLBuilder) CounterNotificationKeyboard(symbol string, periodMinutes int) *InlineKeyboardMarkup {
	// Для счетчика используем компактный формат
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				b.GetChartButton(symbol),
				b.GetTradeButton(symbol, periodMinutes),
			},
		},
	}
}
