// internal/delivery/telegram/app/bot/buttons/builder.go
package buttons

import (
	"crypto-exchange-screener-bot/internal/delivery/telegram"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"fmt"
	"strings"
)

// ButtonBuilder - построитель кнопок
type ButtonBuilder struct{}

// NewButtonBuilder создает новый построитель кнопок
func NewButtonBuilder() *ButtonBuilder {
	return &ButtonBuilder{}
}

// CreateSignalKeyboard создает клавиатуру для сигнала
func (b *ButtonBuilder) CreateSignalKeyboard(symbol string) telegram.InlineKeyboardMarkup {
	return telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{
				{
					Text: constants.ButtonTexts.Trade,
					URL:  b.getTradeURL(symbol),
				},
				{
					Text: constants.ButtonTexts.Chart,
					URL:  b.getTradingViewURL(symbol),
				},
			},
		},
	}
}

// CreateNotificationKeyboard создает клавиатуру для уведомлений
func (b *ButtonBuilder) CreateNotificationKeyboard() telegram.InlineKeyboardMarkup {
	return telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{
				{Text: constants.NotificationButtonTexts.ToggleAll, CallbackData: constants.CallbackNotifyToggleAll},
			},
			{
				{Text: constants.NotificationButtonTexts.GrowthOnly, CallbackData: constants.CallbackNotifyGrowthOnly},
				{Text: constants.NotificationButtonTexts.FallOnly, CallbackData: constants.CallbackNotifyFallOnly},
			},
			{
				{Text: constants.NotificationButtonTexts.Both, CallbackData: constants.CallbackNotifyBoth},
			},
		},
	}
}

// CreatePeriodsKeyboard создает клавиатуру для периодов
func (b *ButtonBuilder) CreatePeriodsKeyboard() telegram.InlineKeyboardMarkup {
	return telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{
				{Text: constants.PeriodButtonTexts.Period5m, CallbackData: constants.CallbackPeriod5m},
				{Text: constants.PeriodButtonTexts.Period15m, CallbackData: constants.CallbackPeriod15m},
				{Text: constants.PeriodButtonTexts.Period30m, CallbackData: constants.CallbackPeriod30m},
			},
			{
				{Text: constants.PeriodButtonTexts.Period1h, CallbackData: constants.CallbackPeriod1h},
				{Text: constants.PeriodButtonTexts.Period4h, CallbackData: constants.CallbackPeriod4h},
				{Text: constants.PeriodButtonTexts.Period1d, CallbackData: constants.CallbackPeriod1d},
			},
			{
				{Text: constants.ButtonTexts.Back, CallbackData: constants.CallbackMenuMain},
			},
		},
	}
}

// CreateSettingsKeyboard создает клавиатуру для настроек
func (b *ButtonBuilder) CreateSettingsKeyboard(isAuth bool) telegram.InlineKeyboardMarkup {
	if isAuth {
		// Для авторизованных пользователей
		return telegram.InlineKeyboardMarkup{
			InlineKeyboard: [][]telegram.InlineKeyboardButton{
				{
					{Text: constants.MenuButtonTexts.Profile, CallbackData: constants.CallbackProfileMain},
					{Text: constants.ButtonTexts.Settings, CallbackData: constants.CallbackSettingsMain},
				},
				{
					{Text: constants.MenuButtonTexts.Notifications, CallbackData: constants.CallbackNotificationsMenu},
					{Text: constants.MenuButtonTexts.Signals, CallbackData: constants.CallbackSignalsMenu},
				},
				{
					{Text: constants.MenuButtonTexts.Periods, CallbackData: constants.CallbackPeriodsMenu},
					{Text: constants.ButtonTexts.Status, CallbackData: constants.CallbackStats},
				},
				{
					{Text: constants.MenuButtonTexts.Reset, CallbackData: constants.CallbackResetMenu},
					{Text: constants.ButtonTexts.Help, CallbackData: constants.CallbackHelp},
				},
			},
		}
	}

	// Для неавторизованных пользователей
	return telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{
				{Text: constants.ButtonTexts.Settings, CallbackData: constants.CallbackSettingsMain},
				{Text: constants.MenuButtonTexts.Notifications, CallbackData: constants.CallbackNotificationsMenu},
			},
			{
				{Text: constants.MenuButtonTexts.Periods, CallbackData: constants.CallbackPeriodsMenu},
				{Text: constants.ButtonTexts.Status, CallbackData: constants.CallbackStats},
			},
			{
				{Text: constants.AuthButtonTexts.Login, CallbackData: constants.CallbackAuthLogin},
				{Text: constants.ButtonTexts.Help, CallbackData: constants.CallbackHelp},
			},
		},
	}
}

// CreateThresholdKeyboard создает клавиатуру для порогов
func (b *ButtonBuilder) CreateThresholdKeyboard(growthThreshold, fallThreshold float64) telegram.InlineKeyboardMarkup {
	return telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{
				{Text: fmt.Sprintf("📈 Рост: %.2f%%", growthThreshold), CallbackData: constants.CallbackThresholdGrowth},
			},
			{
				{Text: fmt.Sprintf("📉 Падение: %.2f%%", fallThreshold), CallbackData: constants.CallbackThresholdFall},
			},
			{
				{Text: "2.0% (по умолчанию)", CallbackData: "threshold_2"},
				{Text: "3.0% (средний)", CallbackData: "threshold_3"},
			},
			{
				{Text: "5.0% (строгий)", CallbackData: "threshold_5"},
				{Text: constants.ButtonTexts.Back, CallbackData: constants.CallbackMenuMain},
			},
		},
	}
}

// CreateSymbolKeyboard создает клавиатуру для выбора символа
func (b *ButtonBuilder) CreateSymbolKeyboard() telegram.InlineKeyboardMarkup {
	return telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{
				{Text: constants.SymbolButtonTexts.BTCUSDT, CallbackData: constants.CallbackSymbolBTCUSDT},
				{Text: constants.SymbolButtonTexts.ETHUSDT, CallbackData: constants.CallbackSymbolETHUSDT},
			},
			{
				{Text: constants.SymbolButtonTexts.BNBUSDT, CallbackData: constants.CallbackSymbolBNBUSDT},
				{Text: constants.SymbolButtonTexts.SOLUSDT, CallbackData: constants.CallbackSymbolSOLUSDT},
			},
			{
				{Text: constants.SymbolButtonTexts.XRPUSDT, CallbackData: constants.CallbackSymbolXRPUSDT},
				{Text: constants.SymbolButtonTexts.Back, CallbackData: constants.CallbackSymbolBack},
			},
		},
	}
}

// CreateTestKeyboard создает тестовую клавиатуру
func (b *ButtonBuilder) CreateTestKeyboard() telegram.InlineKeyboardMarkup {
	return telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{
				{Text: constants.TestButtonTexts.TestOK, CallbackData: constants.CallbackTestOK},
				{Text: constants.TestButtonTexts.TestCancel, CallbackData: constants.CallbackTestCancel},
			},
			{
				{Text: constants.TestButtonTexts.ToggleTest, CallbackData: constants.CallbackToggleTestMode},
				{Text: constants.TestButtonTexts.Chart, CallbackData: constants.CallbackChart},
			},
		},
	}
}

// Вспомогательные методы для URL

func (b *ButtonBuilder) getChartURL(symbol string) string {
	cleanSymbol := strings.ReplaceAll(symbol, "/", "")
	return fmt.Sprintf("https://www.tradingview.com/chart/?symbol=BYBIT:%s", cleanSymbol)
}

func (b *ButtonBuilder) getTradeURL(symbol string) string {
	cleanSymbol := strings.ToUpper(strings.ReplaceAll(symbol, "/", ""))
	return fmt.Sprintf("https://www.bybit.com/trade/usdt/%s", cleanSymbol)
}

func (b *ButtonBuilder) getCoinGeckoURL(symbol string) string {
	baseSymbol := strings.ToLower(strings.Split(symbol, "/")[0])
	return fmt.Sprintf("https://www.coingecko.com/en/coins/%s", baseSymbol)
}

func (b *ButtonBuilder) getTradingViewURL(symbol string) string {
	cleanSymbol := strings.ReplaceAll(symbol, "/", "")
	return fmt.Sprintf("https://www.tradingview.com/symbols/%s/?exchange=BYBIT", cleanSymbol)
}

func (b *ButtonBuilder) getCoinglassURL(symbol string) string {
	cleanSymbol := strings.ReplaceAll(symbol, "/", "")
	return fmt.Sprintf("https://www.coinglass.com/t/%s", cleanSymbol)
}
