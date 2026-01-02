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

// Статические методы для создания кнопок (не требуют экземпляра ButtonURLBuilder)

// CreateStatusButton создает кнопку "Статус"
func CreateStatusButton() InlineKeyboardButton {
	return InlineKeyboardButton{
		Text:         ButtonTexts.Status,
		CallbackData: "status",
	}
}

// CreateSettingsButton создает кнопку "Настройки"
func CreateSettingsButton() InlineKeyboardButton {
	return InlineKeyboardButton{
		Text:         ButtonTexts.Settings,
		CallbackData: "settings",
	}
}

// CreateHelpButton создает кнопку "Помощь"
func CreateHelpButton() InlineKeyboardButton {
	return InlineKeyboardButton{
		Text:         ButtonTexts.Help,
		CallbackData: "help",
	}
}

// CreateBackButton создает кнопку "Назад"
func CreateBackButton() InlineKeyboardButton {
	return InlineKeyboardButton{
		Text:         ButtonTexts.Back,
		CallbackData: "back",
	}
}

// CreateChartButtonWithCallback создает кнопку "График" с callback
func CreateChartButtonWithCallback() InlineKeyboardButton {
	return InlineKeyboardButton{
		Text:         ButtonTexts.Chart,
		CallbackData: "chart",
	}
}

// CreateTestButton создает кнопку "Тест"
func CreateTestButton() InlineKeyboardButton {
	return InlineKeyboardButton{
		Text:         "✅ Тест",
		CallbackData: "test_ok",
	}
}

// CreateCancelButton создает кнопку "Отмена"
func CreateCancelButton() InlineKeyboardButton {
	return InlineKeyboardButton{
		Text:         "❌ Отмена",
		CallbackData: "test_cancel",
	}
}

// Статические методы для создания клавиатур

// CreateWelcomeKeyboard создает клавиатуру для приветственного сообщения
func CreateWelcomeKeyboard() *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				CreateStatusButton(),
				CreateSettingsButton(),
			},
			{
				CreateHelpButton(),
				CreateChartButtonWithCallback(),
			},
		},
	}
}

// CreateTestKeyboard создает тестовую клавиатуру
func CreateTestKeyboard() *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				CreateTestButton(),
				CreateCancelButton(),
			},
			{
				CreateStatusButton(),
				CreateSettingsButton(),
			},
		},
	}
}

// Методы экземпляра ButtonURLBuilder (требуют настройки биржи)

// GetChartButton создает кнопку "График" (с URL)
func (b *ButtonURLBuilder) GetChartButton(symbol string) InlineKeyboardButton {
	return InlineKeyboardButton{
		Text: ButtonTexts.Chart,
		URL:  b.GetChartURL(symbol),
	}
}

// GetTradeButton создает кнопку "Торговать" (с URL)
func (b *ButtonURLBuilder) GetTradeButton(symbol string, periodMinutes int) InlineKeyboardButton {
	return InlineKeyboardButton{
		Text: ButtonTexts.Trade,
		URL:  b.GetTradeURL(symbol, periodMinutes),
	}
}

// GetCoinGeckoButton создает кнопку "CoinGecko" (с URL)
func (b *ButtonURLBuilder) GetCoinGeckoButton(symbol string) InlineKeyboardButton {
	return InlineKeyboardButton{
		Text: ButtonTexts.CoinGecko,
		URL:  b.GetCoinGeckoURL(symbol),
	}
}

// GetCoinglassButton создает кнопку "Coinglass" (с URL)
func (b *ButtonURLBuilder) GetCoinglassButton(symbol string) InlineKeyboardButton {
	return InlineKeyboardButton{
		Text: ButtonTexts.Coinglass,
		URL:  b.GetCoinglassURL(symbol),
	}
}

// GetTradingViewButton создает кнопку "TradingView" (с URL)
func (b *ButtonURLBuilder) GetTradingViewButton(symbol string) InlineKeyboardButton {
	return InlineKeyboardButton{
		Text: ButtonTexts.TradingView,
		URL:  b.GetChartURL(symbol),
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

// Методы создания клавиатур (требуют настройки биржи)

// StandardNotificationKeyboard создает стандартную клавиатуру уведомления
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

// EnhancedNotificationKeyboard создает расширенную клавиатуру уведомления
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

// CounterNotificationKeyboard создает клавиатуру для счетчика
func (b *ButtonURLBuilder) CounterNotificationKeyboard(symbol string, periodMinutes int) *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				CreateStatusButton(),
				b.GetTradeButton(symbol, periodMinutes),
			},
		},
	}
}

// SettingsKeyboard создает клавиатуру настроек
// SettingsKeyboard создает клавиатуру настроек (метод экземпляра)
func (b *ButtonURLBuilder) SettingsKeyboard(isNotificationsEnabled, isTestMode bool) *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				CreateToggleNotificationsButton(isNotificationsEnabled),
				CreateChangeThresholdsButton(),
			},
			{
				CreateChangePeriodButton(),
				CreateToggleTestModeButton(isTestMode),
			},
			{
				CreateBackButton(),
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

// CreateSettingsKeyboard создает клавиатуру настроек (статический метод)
func CreateSettingsKeyboard() *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "🔔 Вкл/Выкл уведомления", CallbackData: "toggle_notifications"},
				{Text: "⚙️ Изменить пороги", CallbackData: "change_thresholds"},
			},
			{
				{Text: "📊 Изменить период", CallbackData: "change_period"},
				{Text: "🧪 Тестовый режим", CallbackData: "toggle_test_mode"},
			},
			{
				CreateBackButton(),
			},
		},
	}
}

// CreateNotificationSettingsKeyboard создает клавиатуру для настроек уведомлений
func CreateNotificationSettingsKeyboard(isEnabled bool) *InlineKeyboardMarkup {
	statusText := "🔔 Включить уведомления"
	if isEnabled {
		statusText = "🔕 Выключить уведомления"
	}

	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: statusText, CallbackData: "toggle_notifications"},
			},
			{
				{Text: "📈 Порог роста", CallbackData: "set_growth_threshold"},
				{Text: "📉 Порог падения", CallbackData: "set_fall_threshold"},
			},
			{
				{Text: "⏱️ Интервал", CallbackData: "set_interval"},
			},
			{
				CreateBackButton(),
			},
		},
	}
}

// CreateThresholdKeyboard создает клавиатуру для выбора порогов
func CreateThresholdKeyboard() *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "1.0%", CallbackData: "threshold_1.0"},
				{Text: "2.0%", CallbackData: "threshold_2.0"},
				{Text: "3.0%", CallbackData: "threshold_3.0"},
			},
			{
				{Text: "5.0%", CallbackData: "threshold_5.0"},
				{Text: "7.5%", CallbackData: "threshold_7.5"},
				{Text: "10.0%", CallbackData: "threshold_10.0"},
			},
			{
				CreateBackButton(),
			},
		},
	}
}

// CreateIntervalKeyboard создает клавиатуру для выбора интервала
func CreateIntervalKeyboard() *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "5 минут", CallbackData: "interval_5"},
				{Text: "15 минут", CallbackData: "interval_15"},
				{Text: "30 минут", CallbackData: "interval_30"},
			},
			{
				{Text: "1 час", CallbackData: "interval_60"},
				{Text: "4 часа", CallbackData: "interval_240"},
				{Text: "1 день", CallbackData: "interval_1440"},
			},
			{
				CreateBackButton(),
			},
		},
	}
}

// CreateTestModeKeyboard создает клавиатуру для тестового режима
func CreateTestModeKeyboard(isTestMode bool) *InlineKeyboardMarkup {
	modeText := "🧪 Включить тестовый режим"
	if isTestMode {
		modeText = "🚫 Выключить тестовый режим"
	}

	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: modeText, CallbackData: "toggle_test_mode"},
			},
			{
				{Text: "📤 Тестовое сообщение", CallbackData: "send_test_message"},
			},
			{
				CreateBackButton(),
			},
		},
	}
}

// CreateToggleNotificationsButton создает кнопку для включения/выключения уведомлений
func CreateToggleNotificationsButton(isEnabled bool) InlineKeyboardButton {
	text := "🔔 Включить уведомления"
	if isEnabled {
		text = "🔕 Выключить уведомления"
	}
	return InlineKeyboardButton{
		Text:         text,
		CallbackData: "toggle_notifications",
	}
}

// CreateChangeThresholdsButton создает кнопку изменения порогов
func CreateChangeThresholdsButton() InlineKeyboardButton {
	return InlineKeyboardButton{
		Text:         "⚙️ Изменить пороги",
		CallbackData: "change_thresholds",
	}
}

// CreateChangePeriodButton создает кнопку изменения периода
func CreateChangePeriodButton() InlineKeyboardButton {
	return InlineKeyboardButton{
		Text:         "📊 Изменить период",
		CallbackData: "change_period",
	}
}

// CreateToggleTestModeButton создает кнопку переключения тестового режима
func CreateToggleTestModeButton(isTestMode bool) InlineKeyboardButton {
	text := "🧪 Тестовый режим"
	if isTestMode {
		text = "✅ Тестовый режим (вкл)"
	}
	return InlineKeyboardButton{
		Text:         text,
		CallbackData: "toggle_test_mode",
	}
}

// CreateSendTestMessageButton создает кнопку отправки тестового сообщения
func CreateSendTestMessageButton() InlineKeyboardButton {
	return InlineKeyboardButton{
		Text:         "📤 Тестовое сообщение",
		CallbackData: "send_test_message",
	}
}

// UpdateSettingsKeyboard создает клавиатуру настроек с текущими статусами
func (b *ButtonURLBuilder) UpdateSettingsKeyboard(bot *TelegramBot) *InlineKeyboardMarkup {
	if bot == nil {
		return CreateSettingsKeyboard()
	}

	return b.SettingsKeyboard(
		bot.IsNotifyEnabled(),
		bot.IsTestMode(),
	)
}
