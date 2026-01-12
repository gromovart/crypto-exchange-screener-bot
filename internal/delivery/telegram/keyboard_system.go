// internal/delivery/telegram/keyboard_system.go
package telegram

import (
	"sync"
)

// KeyboardSystem - централизованная система управления клавиатурами
type KeyboardSystem struct {
	exchange string
	builder  *ButtonURLBuilder
	cache    *KeyboardCache
	mu       sync.RWMutex
}

// KeyboardCache - кэш часто используемых клавиатур
type KeyboardCache struct {
	mainMenu          ReplyKeyboardMarkup
	settingsMenu      ReplyKeyboardMarkup
	notificationsMenu ReplyKeyboardMarkup
	signalTypesMenu   ReplyKeyboardMarkup
	periodsMenu       ReplyKeyboardMarkup
	resetMenu         ReplyKeyboardMarkup
	welcomeKeyboard   *InlineKeyboardMarkup
	settingsKeyboard  *InlineKeyboardMarkup
}

// NewKeyboardSystem создает новую систему клавиатур
func NewKeyboardSystem(exchange string) *KeyboardSystem {
	return &KeyboardSystem{
		exchange: exchange,
		builder:  NewButtonURLBuilder(exchange),
		cache:    &KeyboardCache{},
	}
}

// =============================================
// Public API - Основные методы системы
// =============================================

// GetMainMenu возвращает главное меню (кэшируется)
func (ks *KeyboardSystem) GetMainMenu() ReplyKeyboardMarkup {
	ks.mu.RLock()
	if ks.cache.mainMenu.Keyboard != nil {
		ks.mu.RUnlock()
		return ks.cache.mainMenu
	}
	ks.mu.RUnlock()

	ks.mu.Lock()
	defer ks.mu.Unlock()

	ks.cache.mainMenu = ks.buildMainMenu()
	return ks.cache.mainMenu
}

// GetSettingsMenu возвращает меню настроек
func (ks *KeyboardSystem) GetSettingsMenu() ReplyKeyboardMarkup {
	ks.mu.RLock()
	if ks.cache.settingsMenu.Keyboard != nil {
		ks.mu.RUnlock()
		return ks.cache.settingsMenu
	}
	ks.mu.RUnlock()

	ks.mu.Lock()
	defer ks.mu.Unlock()

	ks.cache.settingsMenu = ks.buildSettingsMenu()
	return ks.cache.settingsMenu
}

// GetNotificationsMenu возвращает меню уведомлений
func (ks *KeyboardSystem) GetNotificationsMenu() ReplyKeyboardMarkup {
	ks.mu.RLock()
	if ks.cache.notificationsMenu.Keyboard != nil {
		ks.mu.RUnlock()
		return ks.cache.notificationsMenu
	}
	ks.mu.RUnlock()

	ks.mu.Lock()
	defer ks.mu.Unlock()

	ks.cache.notificationsMenu = ks.buildNotificationsMenu()
	return ks.cache.notificationsMenu
}

// GetSignalTypesMenu возвращает меню типов сигналов
func (ks *KeyboardSystem) GetSignalTypesMenu() ReplyKeyboardMarkup {
	ks.mu.RLock()
	if ks.cache.signalTypesMenu.Keyboard != nil {
		ks.mu.RUnlock()
		return ks.cache.signalTypesMenu
	}
	ks.mu.RUnlock()

	ks.mu.Lock()
	defer ks.mu.Unlock()

	ks.cache.signalTypesMenu = ks.buildSignalTypesMenu()
	return ks.cache.signalTypesMenu
}

// GetPeriodsMenu возвращает меню периодов
func (ks *KeyboardSystem) GetPeriodsMenu() ReplyKeyboardMarkup {
	ks.mu.RLock()
	if ks.cache.periodsMenu.Keyboard != nil {
		ks.mu.RUnlock()
		return ks.cache.periodsMenu
	}
	ks.mu.RUnlock()

	ks.mu.Lock()
	defer ks.mu.Unlock()

	ks.cache.periodsMenu = ks.buildPeriodsMenu()
	return ks.cache.periodsMenu
}

// GetResetMenu возвращает меню сброса
func (ks *KeyboardSystem) GetResetMenu() ReplyKeyboardMarkup {
	ks.mu.RLock()
	if ks.cache.resetMenu.Keyboard != nil {
		ks.mu.RUnlock()
		return ks.cache.resetMenu
	}
	ks.mu.RUnlock()

	ks.mu.Lock()
	defer ks.mu.Unlock()

	ks.cache.resetMenu = ks.buildResetMenu()
	return ks.cache.resetMenu
}

// CreateWelcomeKeyboard создает инлайн клавиатуру для приветствия
func (ks *KeyboardSystem) CreateWelcomeKeyboard() *InlineKeyboardMarkup {
	ks.mu.RLock()
	if ks.cache.welcomeKeyboard != nil {
		ks.mu.RUnlock()
		return ks.cache.welcomeKeyboard
	}
	ks.mu.RUnlock()

	ks.mu.Lock()
	defer ks.mu.Unlock()

	ks.cache.welcomeKeyboard = ks.buildWelcomeKeyboard()
	return ks.cache.welcomeKeyboard
}

// CreateSettingsKeyboard создает инлайн клавиатуру настроек
func (ks *KeyboardSystem) CreateSettingsKeyboard(notificationsEnabled, testMode bool) *InlineKeyboardMarkup {
	// Для динамических клавиатур не используем кэш
	return ks.buildSettingsKeyboard(notificationsEnabled, testMode)
}

// CreateNotificationKeyboard создает клавиатуру для уведомлений
func (ks *KeyboardSystem) CreateNotificationKeyboard(symbol string, periodMinutes int) *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				ks.builder.GetChartButton(symbol),
				ks.builder.GetTradeButton(symbol, periodMinutes),
			},
		},
	}
}

// CreateEnhancedNotificationKeyboard создает расширенную клавиатуру для уведомлений
func (ks *KeyboardSystem) CreateEnhancedNotificationKeyboard(symbol string, periodMinutes int) *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				ks.builder.GetChartButton(symbol),
				ks.builder.GetTradeButton(symbol, periodMinutes),
			},
			{
				ks.builder.GetCoinGeckoButton(symbol),
				ks.builder.GetCoinglassButton(symbol),
			},
		},
	}
}

// CreateCounterNotificationKeyboard создает клавиатуру для счетчика
func (ks *KeyboardSystem) CreateCounterNotificationKeyboard(symbol string, periodMinutes int) *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: ButtonTexts.Status, CallbackData: CallbackStats},
				ks.builder.GetTradeButton(symbol, periodMinutes),
			},
		},
	}
}

// CreateInlineSettingsMenu создает инлайн меню настроек
func (ks *KeyboardSystem) CreateInlineSettingsMenu() *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "🔔 Уведомления", CallbackData: "menu_notify"},
				{Text: "📊 Тип сигналов", CallbackData: "menu_signals"},
			},
			{
				{Text: "⏱️ Периоды", CallbackData: "menu_periods"},
				{Text: "🔄 Сбросить", CallbackData: "menu_reset"},
			},
		},
	}
}

// CreateInlineMenuPeriods создает инлайн меню периодов
func (ks *KeyboardSystem) CreateInlineMenuPeriods() *InlineKeyboardMarkup {
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
				{Text: "🔙 Назад", CallbackData: CallbackMenuBack},
			},
		},
	}
}

// CreateInlineMenuReset создает инлайн меню сброса
func (ks *KeyboardSystem) CreateInlineMenuReset() *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "🔄 Все счетчики", CallbackData: CallbackResetAll},
				{Text: "📊 По символу", CallbackData: CallbackResetBySymbol},
			},
			{
				{Text: "📈 Счетчик роста", CallbackData: "reset_growth"},
				{Text: "📉 Счетчик падения", CallbackData: "reset_fall"},
			},
			{
				{Text: "🔙 Назад", CallbackData: CallbackMenuBack},
			},
		},
	}
}

// CreateSignalTypeKeyboard создает инлайн клавиатуру выбора типа сигналов
func (ks *KeyboardSystem) CreateSignalTypeKeyboard(growthEnabled, fallEnabled bool) *InlineKeyboardMarkup {
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
				{Text: growthText, CallbackData: CallbackNotifyGrowthOnly},
				{Text: fallText, CallbackData: CallbackNotifyFallOnly},
			},
			{
				{Text: bothText, CallbackData: CallbackNotifyBoth},
			},
			{
				{Text: ButtonTexts.Back, CallbackData: CallbackMenuBack},
			},
		},
	}
}

// CreatePeriodSelectionKeyboard создает инлайн клавиатуру выбора периода
func (ks *KeyboardSystem) CreatePeriodSelectionKeyboard() *InlineKeyboardMarkup {
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
				{Text: ButtonTexts.Back, CallbackData: CallbackMenuBack},
			},
		},
	}
}

// CreateResetKeyboard создает инлайн клавиатуру сброса
func (ks *KeyboardSystem) CreateResetKeyboard() *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "🔄 Все счетчики", CallbackData: CallbackResetAll},
				{Text: "📊 По символу", CallbackData: CallbackResetBySymbol},
			},
			{
				{Text: ButtonTexts.Back, CallbackData: CallbackMenuBack},
			},
		},
	}
}

// CreateSymbolSelectionKeyboard создает инлайн клавиатуру выбора символа
func (ks *KeyboardSystem) CreateSymbolSelectionKeyboard() *InlineKeyboardMarkup {
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
				{Text: ButtonTexts.Back, CallbackData: CallbackMenuBack},
			},
		},
	}
}

// CreateTestKeyboard создает тестовую клавиатуру
func (ks *KeyboardSystem) CreateTestKeyboard() *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "✅ Тест", CallbackData: "test_ok"},
				{Text: "❌ Отмена", CallbackData: "test_cancel"},
			},
			{
				{Text: ButtonTexts.Status, CallbackData: CallbackStats},
				{Text: ButtonTexts.Settings, CallbackData: CallbackSettingsMain},
			},
		},
	}
}

// =============================================
// Private methods - Внутренняя реализация
// =============================================

func (ks *KeyboardSystem) buildMainMenu() ReplyKeyboardMarkup {
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

func (ks *KeyboardSystem) buildSettingsMenu() ReplyKeyboardMarkup {
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

func (ks *KeyboardSystem) buildNotificationsMenu() ReplyKeyboardMarkup {
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

func (ks *KeyboardSystem) buildSignalTypesMenu() ReplyKeyboardMarkup {
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

func (ks *KeyboardSystem) buildPeriodsMenu() ReplyKeyboardMarkup {
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

func (ks *KeyboardSystem) buildResetMenu() ReplyKeyboardMarkup {
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

func (ks *KeyboardSystem) buildWelcomeKeyboard() *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: ButtonTexts.Status, CallbackData: CallbackStats},
				{Text: ButtonTexts.Settings, CallbackData: CallbackSettingsMain},
			},
			{
				{Text: ButtonTexts.Help, CallbackData: "help"},
				{Text: ButtonTexts.Chart, CallbackData: "chart"},
			},
		},
	}
}

func (ks *KeyboardSystem) buildSettingsKeyboard(notificationsEnabled, testMode bool) *InlineKeyboardMarkup {
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
				{Text: "⚙️ Изменить пороги", CallbackData: "change_thresholds"},
			},
			{
				{Text: "📊 Изменить период", CallbackData: CallbackPeriodSelect},
				{Text: testModeText, CallbackData: "toggle_test_mode"},
			},
			{
				{Text: ButtonTexts.Back, CallbackData: CallbackMenuBack},
			},
		},
	}
}

// =============================================
// Вспомогательные методы
// =============================================

// ClearCache очищает кэш клавиатур
func (ks *KeyboardSystem) ClearCache() {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	ks.cache = &KeyboardCache{}
}

// GetExchange возвращает биржу системы
func (ks *KeyboardSystem) GetExchange() string {
	return ks.exchange
}

// GetBuilder возвращает строитель кнопок
func (ks *KeyboardSystem) GetBuilder() *ButtonURLBuilder {
	return ks.builder
}

// GetChartURL возвращает URL графика
func (ks *KeyboardSystem) GetChartURL(symbol string) string {
	return ks.builder.GetChartURL(symbol)
}

// GetTradeURL возвращает URL для торговли
func (ks *KeyboardSystem) GetTradeURL(symbol string, periodMinutes int) string {
	return ks.builder.GetTradeURL(symbol, periodMinutes)
}

// GetCoinGeckoURL возвращает URL CoinGecko
func (ks *KeyboardSystem) GetCoinGeckoURL(symbol string) string {
	return ks.builder.GetCoinGeckoURL(symbol)
}

// GetCoinglassURL возвращает URL Coinglass
func (ks *KeyboardSystem) GetCoinglassURL(symbol string) string {
	return ks.builder.GetCoinglassURL(symbol)
}

// UpdateSettingsKeyboard создает клавиатуру настроек с текущими статусами
func (ks *KeyboardSystem) UpdateSettingsKeyboard(notificationsEnabled, testMode bool) *InlineKeyboardMarkup {
	return ks.CreateSettingsKeyboard(notificationsEnabled, testMode)
}

// =============================================
// Статические методы (не требуют экземпляра)
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
		CallbackData: CallbackSettingsMain,
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
		CallbackData: CallbackMenuBack,
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

// CreateToggleNotificationsButton создает кнопку для включения/выключения уведомлений
func CreateToggleNotificationsButton(isEnabled bool) InlineKeyboardButton {
	text := "🔔 Включить уведомления"
	if isEnabled {
		text = "🔕 Выключить уведомления"
	}
	return InlineKeyboardButton{
		Text:         text,
		CallbackData: CallbackNotifyToggle,
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
		CallbackData: CallbackPeriodSelect,
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
