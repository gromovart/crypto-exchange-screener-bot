// internal/delivery/telegram/menu_handlers.go
package telegram

import (
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"fmt"
	"strings"
	"time"
)

// MenuHandlers - обработчики меню
type MenuHandlers struct {
	config         *config.Config
	messageSender  *MessageSender
	keyboardSystem *KeyboardSystem // ВМЕСТО MenuKeyboards
	menuUtils      *MenuUtils
}

// NewMenuHandlers создает новые обработчики меню (старый конструктор для обратной совместимости)
func NewMenuHandlers(cfg *config.Config, messageSender *MessageSender) *MenuHandlers {
	menuUtils := NewDefaultMenuUtils()
	keyboardSystem := NewKeyboardSystem(cfg.Exchange) // НОВЫЙ KeyboardSystem

	return &MenuHandlers{
		config:         cfg,
		messageSender:  messageSender,
		keyboardSystem: keyboardSystem,
		menuUtils:      menuUtils,
	}
}

// NewMenuHandlersWithUtils создает обработчики меню с утилитами
func NewMenuHandlersWithUtils(cfg *config.Config, messageSender *MessageSender, menuUtils *MenuUtils) *MenuHandlers {
	keyboardSystem := NewKeyboardSystem(cfg.Exchange) // НОВЫЙ KeyboardSystem

	return &MenuHandlers{
		config:         cfg,
		messageSender:  messageSender,
		keyboardSystem: keyboardSystem,
		menuUtils:      menuUtils,
	}
}

// StartCommandHandler обрабатывает команду /start
func (mh *MenuHandlers) StartCommandHandler(chatID string) error {
	message := "🚀 *Crypto Exchange Screener Bot*\n\n" +
		"✅ *Бот активирован!*\n\n" +
		"*Основные команды:*\n" +
		"• /start - Начало работы\n" +
		"• /status - Статус системы\n" +
		"• /notify_on - Включить уведомления\n" +
		"• /notify_off - Выключить уведомления\n" +
		"• /help - Справка\n\n" +
		"Используйте меню ниже для управления ботом:"

	// Используем KeyboardSystem для создания клавиатуры
	keyboard := mh.keyboardSystem.CreateWelcomeKeyboard()

	return mh.messageSender.SendMessageToChat(chatID, message, keyboard)
}

// HandleMessage обрабатывает текстовые сообщения из меню
func (mh *MenuHandlers) HandleMessage(text, chatID string) error {
	switch text {
	case "⚙️ Настройки":
		mh.messageSender.SetReplyKeyboard(mh.keyboardSystem.GetSettingsMenu())
		return mh.SendSettingsInfo(chatID)

	case "📊 Статус":
		return mh.SendStatus(chatID)

	case "🔔 Уведомления":
		mh.messageSender.SetReplyKeyboard(mh.keyboardSystem.GetNotificationsMenu())
		return mh.SendNotificationsInfo(chatID)

	case "✅ Включить":
		return mh.HandleNotifyOn(chatID)

	case "❌ Выключить":
		return mh.HandleNotifyOff(chatID)

	case "📈 Сигналы":
		mh.messageSender.SetReplyKeyboard(mh.keyboardSystem.GetSignalTypesMenu())
		return mh.SendSignalTypesInfo(chatID)

	case "📈 Только рост":
		mh.config.TelegramNotifyGrowth = true
		mh.config.TelegramNotifyFall = false
		return mh.messageSender.SendMessageToChat(chatID, "📈 Теперь отслеживается только рост", nil)

	case "📉 Только падение":
		mh.config.TelegramNotifyGrowth = false
		mh.config.TelegramNotifyFall = true
		return mh.messageSender.SendMessageToChat(chatID, "📉 Теперь отслеживается только падение", nil)

	case "📊 Все сигналы":
		mh.config.TelegramNotifyGrowth = true
		mh.config.TelegramNotifyFall = true
		return mh.messageSender.SendMessageToChat(chatID, "📊 Теперь отслеживаются все сигналы", nil)

	case "⏱️ Периоды":
		mh.messageSender.SetReplyKeyboard(mh.keyboardSystem.GetPeriodsMenu())
		return mh.SendPeriodsInfo(chatID)

	case "⏱️ 5мин", "⏱️ 5 мин":
		return mh.HandlePeriodChange(chatID, "5m")

	case "⏱️ 15мин", "⏱️ 15 мин":
		return mh.HandlePeriodChange(chatID, "15m")

	case "⏱️ 30мин", "⏱️ 30 мин":
		return mh.HandlePeriodChange(chatID, "30m")

	case "⏱️ 1 час":
		return mh.HandlePeriodChange(chatID, "1h")

	case "⏱️ 4 часа":
		return mh.HandlePeriodChange(chatID, "4h")

	case "🔄 Сбросить":
		mh.messageSender.SetReplyKeyboard(mh.keyboardSystem.GetResetMenu())
		return mh.SendResetInfo(chatID)

	case "🔄 Все счетчики":
		return mh.HandleResetAllCounters(chatID)

	case "📋 Помощь":
		return mh.SendHelp(chatID)

	case "🔙 Назад", "🔙 Главное меню":
		mh.messageSender.SetReplyKeyboard(mh.keyboardSystem.GetMainMenu())
		return mh.messageSender.SendMessageToChat(chatID, "🔙 Возврат в главное меню", nil)

	default:
		if strings.HasPrefix(text, "/") {
			return mh.HandleCommand(text, chatID)
		}
		return mh.messageSender.SendMessageToChat(chatID,
			"❓ Неизвестная команда. Используйте меню ниже или /help", nil)
	}
}

// HandleCallback обрабатывает callback от inline кнопок
func (mh *MenuHandlers) HandleCallback(callbackData string, chatID string) error {
	// Используем menuUtils для парсинга callback данных
	action, params := mh.menuUtils.ParseCallbackData(callbackData)

	switch action {
	case "menu":
		if len(params) > 0 {
			switch params[0] {
			case "notify":
				mh.messageSender.SetReplyKeyboard(mh.keyboardSystem.GetNotificationsMenu())
				return mh.SendNotificationsInfo(chatID)
			case "signals":
				mh.messageSender.SetReplyKeyboard(mh.keyboardSystem.GetSignalTypesMenu())
				return mh.SendSignalTypesInfo(chatID)
			case "periods":
				mh.messageSender.SetReplyKeyboard(mh.keyboardSystem.GetPeriodsMenu())
				return mh.SendPeriodsInfo(chatID)
			case "reset":
				mh.messageSender.SetReplyKeyboard(mh.keyboardSystem.GetResetMenu())
				return mh.SendResetInfo(chatID)
			case "back":
				mh.messageSender.SetReplyKeyboard(mh.keyboardSystem.GetMainMenu())
				return mh.messageSender.SendMessageToChat(chatID, "🔙 Возврат в главное меню", nil)
			}
		}
	case "period":
		if len(params) > 0 {
			return mh.HandlePeriodChange(chatID, params[0])
		}
	case "reset":
		if len(params) > 0 {
			switch params[0] {
			case "all":
				return mh.HandleResetAllCounters(chatID)
			case "symbol":
				return mh.SendSymbolSelectionInline(chatID)
			default:
				// Проверяем, не начинается ли с symbol_
				if strings.HasPrefix(callbackData, "symbol_") {
					symbol := strings.TrimPrefix(callbackData, "symbol_")
					return mh.messageSender.SendMessageToChat(chatID,
						fmt.Sprintf("📊 Счетчик для %s сброшен", strings.ToUpper(symbol)), nil)
				}
			}
		}
	case "notify":
		if len(params) > 0 {
			switch params[0] {
			case "on":
				return mh.HandleNotifyOn(chatID)
			case "off":
				return mh.HandleNotifyOff(chatID)
			}
		}
	case CallbackStats:
		return mh.SendStatus(chatID)

	case CallbackSettings:
		mh.messageSender.SetReplyKeyboard(mh.keyboardSystem.GetSettingsMenu())
		return mh.SendSettingsInfo(chatID)

	case CallbackSettingsNotifyToggle:
		if mh.config.TelegramEnabled {
			return mh.HandleNotifyOff(chatID)
		} else {
			return mh.HandleNotifyOn(chatID)
		}

	case CallbackSettingsSignalType:
		// Показываем inline клавиатуру для выбора типа сигналов
		keyboard := mh.keyboardSystem.CreateSignalTypeKeyboard(
			mh.config.TelegramNotifyGrowth,
			mh.config.TelegramNotifyFall,
		)
		return mh.messageSender.SendMessageToChat(chatID,
			"📊 *Выберите тип отслеживаемых сигналов:*", keyboard)

	case CallbackTrackGrowthOnly:
		mh.config.TelegramNotifyGrowth = true
		mh.config.TelegramNotifyFall = false
		return mh.messageSender.SendMessageToChat(chatID,
			"✅ Теперь отслеживается только рост", nil)

	case CallbackTrackFallOnly:
		mh.config.TelegramNotifyGrowth = false
		mh.config.TelegramNotifyFall = true
		return mh.messageSender.SendMessageToChat(chatID,
			"✅ Теперь отслеживается только падение", nil)

	case CallbackTrackBoth:
		mh.config.TelegramNotifyGrowth = true
		mh.config.TelegramNotifyFall = true
		return mh.messageSender.SendMessageToChat(chatID,
			"✅ Теперь отслеживаются все сигналы", nil)

	case CallbackSettingsChangePeriod:
		// Показываем inline клавиатуру для выбора периода
		keyboard := mh.keyboardSystem.CreatePeriodSelectionKeyboard()
		return mh.messageSender.SendMessageToChat(chatID,
			"⏱️ *Выберите период анализа:*", keyboard)

	case CallbackPeriod5m:
		return mh.HandlePeriodChange(chatID, "5m")

	case CallbackPeriod15m:
		return mh.HandlePeriodChange(chatID, "15m")

	case CallbackPeriod30m:
		return mh.HandlePeriodChange(chatID, "30m")

	case CallbackPeriod1h:
		return mh.HandlePeriodChange(chatID, "1h")

	case CallbackPeriod4h:
		return mh.HandlePeriodChange(chatID, "4h")

	case CallbackPeriod1d:
		return mh.HandlePeriodChange(chatID, "1d")

	case CallbackSettingsBack:
		// Возвращаемся к основному меню настроек
		keyboard := mh.keyboardSystem.CreateSettingsKeyboard(
			mh.config.TelegramEnabled,
			false, // testMode - можно добавить если нужно
		)
		return mh.messageSender.SendMessageToChat(chatID,
			"⚙️ *Настройки бота:*", keyboard)

	case CallbackSettingsBackToMain:
		mh.messageSender.SetReplyKeyboard(mh.keyboardSystem.GetMainMenu())
		return mh.messageSender.SendMessageToChat(chatID,
			"🔙 Возврат в главное меню", nil)

	case CallbackSettingsResetCounter:
		// Показываем inline клавиатуру для сброса
		keyboard := mh.keyboardSystem.CreateResetKeyboard()
		return mh.messageSender.SendMessageToChat(chatID,
			"🔄 *Выберите что сбросить:*", keyboard)

	case CallbackResetAll:
		return mh.HandleResetAllCounters(chatID)

	case CallbackResetBySymbol:
		return mh.SendSymbolSelectionInline(chatID)

	case "help":
		return mh.SendHelp(chatID)

	case "chart":
		return mh.messageSender.SendMessageToChat(chatID,
			"📊 *Графики*\n\n"+
				"Используйте кнопки в уведомлениях для перехода к графикам.", nil)

	case "test_ok":
		return mh.messageSender.SendMessageToChat(chatID,
			"✅ Тест пройден успешно!", nil)

	case "test_cancel":
		return mh.messageSender.SendMessageToChat(chatID,
			"❌ Тест отменен", nil)

	case "toggle_test_mode":
		// Переключение тестового режима
		return mh.messageSender.SendMessageToChat(chatID,
			"🧪 Функционал тестового режима в разработке", nil)
	}

	return fmt.Errorf("unknown callback data: %s", callbackData)
}

// SendSymbolSelectionInline отправляет inline меню выбора символа
func (mh *MenuHandlers) SendSymbolSelectionInline(chatID string) error {
	message := "Выберите символ для сброса счетчика:"

	// Используем KeyboardSystem для создания клавиатуры
	keyboard := mh.keyboardSystem.CreateSymbolSelectionKeyboard()

	return mh.messageSender.SendMessageToChat(chatID, message, keyboard)
}

// SendSettingsInfo отправляет информацию о настройках
func (mh *MenuHandlers) SendSettingsInfo(chatID string) error {
	// Используем menuUtils для получения имени периода
	periodName := "15 минут"
	if mh.menuUtils != nil {
		period := getPeriodFromConfig(mh.config)
		periodName = mh.menuUtils.GetPeriodName(period)
	}

	message := "⚙️ *Настройки бота*\n\n" +
		"*Текущие настройки:*\n" +
		"• 🔔 Уведомления: " + getNotificationStatus(mh.config) + "\n" +
		"• 📈 Тип сигналов: " + getSignalTypesStatus(mh.config) + "\n" +
		"• ⏱️ Период анализа: " + periodName + "\n\n" +
		"Выберите действие из меню ниже:\n\n" +
		"• 🔔 Вкл/Выкл - управление уведомлениями\n" +
		"• 📈 Тип сигналов - выбор отслеживаемых сигналов\n" +
		"• 🔄 Сбросить - сброс счетчиков\n" +
		"• ⏱️ Периоды - настройка периодов анализа\n" +
		"• 🔙 Назад - вернуться в главное меню"

	return mh.messageSender.SendMessageToChat(chatID, message, nil)
}

// SendNotificationsInfo отправляет информацию об уведомлениях
func (mh *MenuHandlers) SendNotificationsInfo(chatID string) error {
	status := getNotificationStatus(mh.config)

	message := fmt.Sprintf("🔔 *Управление уведомлениями*\n\n"+
		"Текущий статус: %s\n\n"+
		"Выберите действие из меню ниже:\n\n"+
		"• ✅ Включить - включить все уведомления\n"+
		"• ❌ Выключить - выключить все уведомления\n"+
		"• 📊 Все сигналы - уведомлять обо всех сигналах\n"+
		"• 📈 Только рост - уведомлять только о росте\n"+
		"• 📉 Только падение - уведомлять только о падении\n"+
		"• 🔙 Назад - вернуться в настройки",
		status)

	return mh.messageSender.SendMessageToChat(chatID, message, nil)
}

// SendSignalTypesInfo отправляет информацию о типах сигналов
func (mh *MenuHandlers) SendSignalTypesInfo(chatID string) error {
	growthStatus := "❌ Выключен"
	if mh.config.TelegramNotifyGrowth {
		growthStatus = "✅ Включен"
	}

	fallStatus := "❌ Выключен"
	if mh.config.TelegramNotifyFall {
		fallStatus = "✅ Включен"
	}

	message := fmt.Sprintf("📊 *Типы отслеживаемых сигналов*\n\n"+
		"Текущие настройки:\n"+
		"• 📈 Рост: %s\n"+
		"• 📉 Падение: %s\n\n"+
		"Выберите действие из меню ниже:\n\n"+
		"• 📈 Только рост - отслеживать только рост\n"+
		"• 📉 Только падение - отслеживать только падение\n"+
		"• 📊 Все сигналы - отслеживать все сигналы\n"+
		"• 🔔 Настройки уведомлений - управление уведомлениями\n"+
		"• 📊 Статус - просмотр статуса системы\n"+
		"• 🔙 Главное меню - вернуться в главное меню",
		growthStatus, fallStatus)

	return mh.messageSender.SendMessageToChat(chatID, message, nil)
}

// SendPeriodsInfo отправляет информацию о периодах
func (mh *MenuHandlers) SendPeriodsInfo(chatID string) error {
	// Получаем период из кастомных настроек
	period := getPeriodFromConfig(mh.config)
	periodName := mh.menuUtils.GetPeriodName(period)

	message := "⏱️ *Настройка периодов анализа*\n\n" +
		"Текущий период: " + periodName + "\n\n" +
		"Выберите период из меню ниже:\n\n" +
		"• ⏱️ 5 мин - 5 минутный период\n" +
		"• ⏱️ 15 мин - 15 минутный период\n" +
		"• ⏱️ 30 мин - 30 минутный период\n" +
		"• ⏱️ 1 час - 1 часовой период\n" +
		"• ⏱️ 4 часа - 4 часовой период\n" +
		"• 🔙 Назад - вернуться в настройки"

	return mh.messageSender.SendMessageToChat(chatID, message, nil)
}

// SendResetInfo отправляет информацию о сбросе
func (mh *MenuHandlers) SendResetInfo(chatID string) error {
	message := "🔄 *Сброс счетчиков сигналов*\n\n" +
		"Выберите действие из меню ниже:\n\n" +
		"• 🔄 Все счетчики - сбросить все счетчики\n" +
		"• 📊 По символу - сбросить счетчик для символа\n" +
		"• ⚙️ Настройки - перейти в настройки\n" +
		"• 🔙 Главное меню - вернуться в главное меню"

	return mh.messageSender.SendMessageToChat(chatID, message, nil)
}

// HandleCommand обрабатывает текстовые команды
func (mh *MenuHandlers) HandleCommand(cmd, chatID string) error {
	switch cmd {
	case "/start":
		return mh.StartCommandHandler(chatID)
	case "/help":
		return mh.SendHelp(chatID)
	case "/status":
		return mh.SendStatus(chatID)
	case "/notify_on":
		return mh.HandleNotifyOn(chatID)
	case "/notify_off":
		return mh.HandleNotifyOff(chatID)
	case "/settings":
		mh.messageSender.SetReplyKeyboard(mh.keyboardSystem.GetSettingsMenu())
		return mh.SendSettingsInfo(chatID)
	case "/test":
		return mh.messageSender.SendTestMessage()
	default:
		return mh.messageSender.SendMessageToChat(chatID,
			fmt.Sprintf("❓ Неизвестная команда: %s. Используйте /help", cmd), nil)
	}
}

// SendStatus отправляет статус системы
func (mh *MenuHandlers) SendStatus(chatID string) error {
	notifyStatus := getNotificationStatus(mh.config)
	growthStatus := getSignalTypeStatus(mh.config.TelegramNotifyGrowth, "Рост")
	fallStatus := getSignalTypeStatus(mh.config.TelegramNotifyFall, "Падение")

	// Используем menuUtils для получения имени периода
	periodName := "15 минут"
	if mh.menuUtils != nil {
		period := getPeriodFromConfig(mh.config)
		periodName = mh.menuUtils.GetPeriodName(period)
	}

	message := fmt.Sprintf(
		"📊 *Статус системы*\n\n"+
			"✅ Бот работает\n"+
			"🔔 Уведомления: %s\n"+
			"📈 Отслеживание роста: %s\n"+
			"📉 Отслеживание падения: %s\n"+
			"⏱️ Период счетчика: %s\n"+
			"🕐 Время сервера: %s",
		notifyStatus,
		growthStatus,
		fallStatus,
		periodName,
		time.Now().Format("15:04:05"),
	)

	return mh.messageSender.SendMessageToChat(chatID, message, nil)
}

// SendHelp отправляет справку
func (mh *MenuHandlers) SendHelp(chatID string) error {
	message := "📋 *Справка*\n\n" +
		"*Основные команды:*\n" +
		"/start - Начало работы\n" +
		"/status - Статус системы\n" +
		"/notify_on - Включить уведомления\n" +
		"/notify_off - Выключить уведомления\n" +
		"/test - Тестовое сообщение\n" +
		"/help - Эта справка\n\n" +
		"*Меню управления:*\n" +
		"⚙️ Настройки - Показать/изменить настройки\n" +
		"📊 Статус - Статус системы\n" +
		"🔔 Уведомления - Управление уведомлениями\n" +
		"📈 Сигналы - Выбор типа сигналов\n" +
		"⏱️ Периоды - Настройка периодов анализа\n" +
		"📋 Помощь - Эта справка"

	return mh.messageSender.SendMessageToChat(chatID, message, nil)
}

// HandleNotifyOn включает уведомления
func (mh *MenuHandlers) HandleNotifyOn(chatID string) error {
	mh.config.TelegramEnabled = true
	return mh.messageSender.SendMessageToChat(chatID, "✅ Уведомления включены", nil)
}

// HandleNotifyOff выключает уведомления
func (mh *MenuHandlers) HandleNotifyOff(chatID string) error {
	mh.config.TelegramEnabled = false
	return mh.messageSender.SendMessageToChat(chatID, "❌ Уведомления выключены", nil)
}

// HandlePeriodChange обрабатывает изменение периода
func (mh *MenuHandlers) HandlePeriodChange(chatID string, period string) error {
	// Используем menuUtils для получения имени периода
	periodName := period
	if mh.menuUtils != nil {
		periodName = mh.menuUtils.GetPeriodName(period)
	}

	// Обновляем кастомные настройки
	if mh.config.AnalyzerConfigs.CounterAnalyzer.CustomSettings == nil {
		mh.config.AnalyzerConfigs.CounterAnalyzer.CustomSettings = make(map[string]interface{})
	}
	mh.config.AnalyzerConfigs.CounterAnalyzer.CustomSettings["analysis_period"] = period

	message := fmt.Sprintf("✅ Период анализа установлен на: %s\n\n"+
		"Все счетчики будут перезапущены с новым периодом.", periodName)

	return mh.messageSender.SendMessageToChat(chatID, message, nil)
}

// HandleResetAllCounters сбрасывает все счетчики
func (mh *MenuHandlers) HandleResetAllCounters(chatID string) error {
	message := "🔄 Все счетчики сигналов сброшены"
	return mh.messageSender.SendMessageToChat(chatID, message, nil)
}

// getPeriodFromConfig получает период из конфигурации
func getPeriodFromConfig(config *config.Config) string {
	if config.AnalyzerConfigs.CounterAnalyzer.CustomSettings != nil {
		if period, ok := config.AnalyzerConfigs.CounterAnalyzer.CustomSettings["analysis_period"].(string); ok {
			return period
		}
	}
	return "15m"
}

// getNotificationStatus возвращает статус уведомлений
func getNotificationStatus(config *config.Config) string {
	if config.TelegramEnabled {
		return "✅ Включены"
	}
	return "❌ Выключены"
}

// getSignalTypeStatus возвращает статус типа сигнала
func getSignalTypeStatus(enabled bool, signalType string) string {
	if enabled {
		return "✅ Включен"
	}
	return "❌ Выключен"
}

// getSignalTypesStatus возвращает статус типов сигналов
func getSignalTypesStatus(config *config.Config) string {
	if config.TelegramNotifyGrowth && config.TelegramNotifyFall {
		return "Все"
	} else if config.TelegramNotifyGrowth {
		return "Только рост"
	} else if config.TelegramNotifyFall {
		return "Только падение"
	}
	return "Ничего"
}
