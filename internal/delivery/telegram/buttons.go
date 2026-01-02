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

// =============================================
// Inline кнопки (с URL)
// =============================================

// GetChartButton создает inline кнопку "График"
func (b *ButtonURLBuilder) GetChartButton(symbol string) InlineKeyboardButton {
	return InlineKeyboardButton{
		Text: ButtonTexts.Chart,
		URL:  b.GetChartURL(symbol),
	}
}

// GetTradeButton создает inline кнопку "Торговать"
func (b *ButtonURLBuilder) GetTradeButton(symbol string, periodMinutes int) InlineKeyboardButton {
	return InlineKeyboardButton{
		Text: ButtonTexts.Trade,
		URL:  b.GetTradeURL(symbol, periodMinutes),
	}
}

// GetCoinGeckoButton создает inline кнопку "CoinGecko"
func (b *ButtonURLBuilder) GetCoinGeckoButton(symbol string) InlineKeyboardButton {
	return InlineKeyboardButton{
		Text: ButtonTexts.CoinGecko,
		URL:  b.GetCoinGeckoURL(symbol),
	}
}

// GetCoinglassButton создает inline кнопку "Coinglass"
func (b *ButtonURLBuilder) GetCoinglassButton(symbol string) InlineKeyboardButton {
	return InlineKeyboardButton{
		Text: ButtonTexts.Coinglass,
		URL:  b.GetCoinglassURL(symbol),
	}
}

// GetTradingViewButton создает inline кнопку "TradingView"
func (b *ButtonURLBuilder) GetTradingViewButton(symbol string) InlineKeyboardButton {
	return InlineKeyboardButton{
		Text: ButtonTexts.TradingView,
		URL:  b.GetChartURL(symbol),
	}
}

// =============================================
// Inline кнопки (с Callback)
// =============================================

// CreateStatusButton создает кнопку "Статус"
func CreateStatusButton() InlineKeyboardButton {
	return InlineKeyboardButton{
		Text:         ButtonTexts.Status,
		CallbackData: CallbackStats,
	}
}

// CreateSettingsButton создает кнопку "Настройки"
func CreateSettingsButton() InlineKeyboardButton {
	return InlineKeyboardButton{
		Text:         ButtonTexts.Settings,
		CallbackData: CallbackSettings,
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
		CallbackData: CallbackSettingsBack,
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

// =============================================
// URL методы
// =============================================

// GetChartURL возвращает URL графика
func (b *ButtonURLBuilder) GetChartURL(symbol string) string {
	return fmt.Sprintf("https://www.tradingview.com/chart/?symbol=%s:%s", strings.ToUpper(b.exchange), symbol)
}

// GetTradeURL возвращает URL для торговли
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

// GetCoinGeckoURL возвращает URL CoinGecko
func (b *ButtonURLBuilder) GetCoinGeckoURL(symbol string) string {
	cleanSymbol := strings.TrimSuffix(strings.TrimSuffix(symbol, "USDT"), "USD")
	return fmt.Sprintf("https://www.coingecko.com/en/coins/%s", strings.ToLower(cleanSymbol))
}

// GetCoinglassURL возвращает URL Coinglass
func (b *ButtonURLBuilder) GetCoinglassURL(symbol string) string {
	return fmt.Sprintf("https://www.coinglass.com/tv/%s", symbol)
}

// =============================================
// Inline клавиатуры для уведомлений
// =============================================

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

// =============================================
// Статические клавиатуры (Reply и Inline)
// =============================================

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
				{Text: ButtonTexts.Chart, CallbackData: "chart"},
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

// CreateSettingsKeyboard создает inline клавиатуру настроек
func CreateSettingsKeyboard() *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "🔔 Вкл/Выкл уведомления", CallbackData: CallbackSettingsNotifyToggle},
				{Text: "⚙️ Изменить пороги", CallbackData: CallbackSettingsChangePeriod},
			},
			{
				{Text: "📊 Изменить период", CallbackData: CallbackSettingsSignalType},
				{Text: "🧪 Тестовый режим", CallbackData: "toggle_test_mode"},
			},
			{
				CreateBackButton(),
			},
		},
	}
}

// CreateNotificationSettingsKeyboard создает клавиатуру настроек уведомлений
func CreateNotificationSettingsKeyboard(isEnabled bool) *InlineKeyboardMarkup {
	statusText := "🔔 Включить уведомления"
	if isEnabled {
		statusText = "🔕 Выключить уведомления"
	}

	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: statusText, CallbackData: CallbackSettingsNotifyToggle},
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

// CreateSignalTypeKeyboard создает клавиатуру выбора типа сигналов
func CreateSignalTypeKeyboard(growthEnabled, fallEnabled bool) *InlineKeyboardMarkup {
	growthText := "📈 Только рост"
	fallText := "📉 Только падение"
	bothText := "📊 Все сигналы"

	if growthEnabled && !fallEnabled {
		growthText = "✅ " + growthText
	} else if !growthEnabled && fallEnabled {
		fallText = "✅ " + fallText
	} else if growthEnabled && fallEnabled {
		bothText = "✅ " + bothText
	}

	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: growthText, CallbackData: CallbackTrackGrowthOnly},
				{Text: fallText, CallbackData: CallbackTrackFallOnly},
			},
			{
				{Text: bothText, CallbackData: CallbackTrackBoth},
			},
			{
				CreateBackButton(),
			},
		},
	}
}

// CreatePeriodSelectionKeyboard создает клавиатуру выбора периода
func CreatePeriodSelectionKeyboard() *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "5 мин", CallbackData: CallbackPeriod5m},
				{Text: "15 мин", CallbackData: CallbackPeriod15m},
				{Text: "30 мин", CallbackData: CallbackPeriod30m},
			},
			{
				{Text: "1 час", CallbackData: CallbackPeriod1h},
				{Text: "4 часа", CallbackData: CallbackPeriod4h},
				{Text: "1 день", CallbackData: CallbackPeriod1d},
			},
			{
				CreateBackButton(),
			},
		},
	}
}

// CreateResetKeyboard создает клавиатуру сброса
func CreateResetKeyboard() *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "🔄 Все счетчики", CallbackData: CallbackResetAll},
				{Text: "📊 По символу", CallbackData: CallbackResetBySymbol},
			},
			{
				CreateBackButton(),
			},
		},
	}
}

// CreateSymbolSelectionKeyboard создает клавиатуру выбора символа
func CreateSymbolSelectionKeyboard() *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "BTCUSDT", CallbackData: "symbol_btcusdt"},
				{Text: "ETHUSDT", CallbackData: "symbol_ethusdt"},
				{Text: "SOLUSDT", CallbackData: "symbol_solusdt"},
			},
			{
				{Text: "XRPUSDT", CallbackData: "symbol_xrpusdt"},
				{Text: "BNBUSDT", CallbackData: "symbol_bnbusdt"},
				{Text: "DOGEUSDT", CallbackData: "symbol_dogeusdt"},
			},
			{
				CreateBackButton(),
			},
		},
	}
}

// =============================================
// Вспомогательные методы
// =============================================

// getTradingInterval преобразует минуты в интервал торгового терминала
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

// =============================================
// Комбинации клавиатур для MenuKeyboards
// =============================================

// GetMainMenuKeyboard возвращает Reply клавиатуру главного меню
func GetMainMenuKeyboard() ReplyKeyboardMarkup {
	return ReplyKeyboardMarkup{
		Keyboard: [][]ReplyKeyboardButton{
			{
				{Text: "⚙️ Настройки"},
				{Text: "📊 Статус"},
				{Text: "🔔 Уведомления"},
			},
			{
				{Text: "📈 Сигналы"},
				{Text: "⏱️ Периоды"},
				{Text: "📋 Помощь"},
			},
		},
		ResizeKeyboard:  true,
		OneTimeKeyboard: false,
		Selective:       false,
		IsPersistent:    true,
	}
}

// GetSettingsMenuKeyboard возвращает Reply клавиатуру настроек
func GetSettingsMenuKeyboard() ReplyKeyboardMarkup {
	return ReplyKeyboardMarkup{
		Keyboard: [][]ReplyKeyboardButton{
			{
				{Text: "🔔 Вкл/Выкл"},
				{Text: "📈 Тип сигналов"},
				{Text: "🔄 Сбросить"},
			},
			{
				{Text: "⏱️ 5мин"},
				{Text: "⏱️ 15мин"},
				{Text: "🔙 Назад"},
			},
		},
		ResizeKeyboard:  true,
		OneTimeKeyboard: false,
		Selective:       false,
		IsPersistent:    true,
	}
}

// GetNotificationsMenuKeyboard возвращает Reply клавиатуру уведомлений
func GetNotificationsMenuKeyboard() ReplyKeyboardMarkup {
	return ReplyKeyboardMarkup{
		Keyboard: [][]ReplyKeyboardButton{
			{
				{Text: "✅ Включить"},
				{Text: "❌ Выключить"},
				{Text: "📊 Все сигналы"},
			},
			{
				{Text: "📈 Только рост"},
				{Text: "📉 Только падение"},
				{Text: "🔙 Назад"},
			},
		},
		ResizeKeyboard:  true,
		OneTimeKeyboard: false,
		Selective:       false,
		IsPersistent:    true,
	}
}

// GetSignalTypesMenuKeyboard возвращает Reply клавиатуру типов сигналов
func GetSignalTypesMenuKeyboard() ReplyKeyboardMarkup {
	return ReplyKeyboardMarkup{
		Keyboard: [][]ReplyKeyboardButton{
			{
				{Text: "📈 Только рост"},
				{Text: "📉 Только падение"},
				{Text: "📊 Все сигналы"},
			},
			{
				{Text: "🔔 Настройки уведомлений"},
				{Text: "📊 Статус"},
				{Text: "🔙 Главное меню"},
			},
		},
		ResizeKeyboard:  true,
		OneTimeKeyboard: false,
		Selective:       false,
		IsPersistent:    true,
	}
}

// GetPeriodsMenuKeyboard возвращает Reply клавиатуру периодов
func GetPeriodsMenuKeyboard() ReplyKeyboardMarkup {
	return ReplyKeyboardMarkup{
		Keyboard: [][]ReplyKeyboardButton{
			{
				{Text: "⏱️ 5 мин"},
				{Text: "⏱️ 15 мин"},
				{Text: "⏱️ 30 мин"},
			},
			{
				{Text: "⏱️ 1 час"},
				{Text: "⏱️ 4 часа"},
				{Text: "🔙 Назад"},
			},
		},
		ResizeKeyboard:  true,
		OneTimeKeyboard: false,
		Selective:       false,
		IsPersistent:    true,
	}
}

// GetResetMenuKeyboard возвращает Reply клавиатуру сброса
func GetResetMenuKeyboard() ReplyKeyboardMarkup {
	return ReplyKeyboardMarkup{
		Keyboard: [][]ReplyKeyboardButton{
			{
				{Text: "🔄 Все счетчики"},
				{Text: "📊 По символу"},
				{Text: "📈 Счетчик роста"},
			},
			{
				{Text: "📉 Счетчик падения"},
				{Text: "⚙️ Настройки"},
				{Text: "🔙 Главное меню"},
			},
		},
		ResizeKeyboard:  true,
		OneTimeKeyboard: false,
		Selective:       false,
		IsPersistent:    true,
	}
}

// UpdateSettingsKeyboard создает клавиатуру настроек с текущими статусами
func (b *ButtonURLBuilder) UpdateSettingsKeyboard(bot *TelegramBot) *InlineKeyboardMarkup {
	if bot == nil {
		return CreateSettingsKeyboard()
	}

	// Получаем статусы из бота
	notificationsEnabled := bot.IsNotifyEnabled()
	testMode := bot.IsTestMode()

	// Создаем кнопки с актуальными статусами
	return b.SettingsKeyboard(notificationsEnabled, testMode)
}

// SettingsKeyboard создает клавиатуру настроек (метод экземпляра)
func (b *ButtonURLBuilder) SettingsKeyboard(isNotificationsEnabled, isTestMode bool) *InlineKeyboardMarkup {
	// Кнопка уведомлений
	notifyText := "🔔 Включить уведомления"
	if isNotificationsEnabled {
		notifyText = "🔕 Выключить уведомления"
	}

	// Кнопка тестового режима
	testModeText := "🧪 Включить тестовый режим"
	if isTestMode {
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
		CallbackData: CallbackSettingsNotifyToggle,
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
		CallbackData: CallbackSettingsChangePeriod,
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
