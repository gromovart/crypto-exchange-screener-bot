// internal/telegram/menu_handlers.go
package telegram

import (
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"fmt"
	"strings"
	"time"
)

// MenuHandlers - обработчики меню
type MenuHandlers struct {
	config        *config.Config
	messageSender *MessageSender
	keyboards     *MenuKeyboards
}

// NewMenuHandlers создает новые обработчики меню
func NewMenuHandlers(cfg *config.Config, messageSender *MessageSender) *MenuHandlers {
	return &MenuHandlers{
		config:        cfg,
		messageSender: messageSender,
		keyboards:     NewMenuKeyboards(),
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
		"*Используйте меню ниже для быстрого управления* ⬇️"

	return mh.messageSender.SendMessageToChat(chatID, message, nil)
}

// HandleMessage обрабатывает текстовые сообщения из меню
func (mh *MenuHandlers) HandleMessage(text, chatID string) error {
	switch text {
	case "⚙️ Настройки":
		mh.messageSender.SetReplyKeyboard(mh.keyboards.GetSettingsMenu())
		return mh.SendSettingsInfo(chatID)

	case "📊 Статус":
		return mh.SendStatus(chatID)

	case "🔔 Уведомления":
		mh.messageSender.SetReplyKeyboard(mh.keyboards.GetNotificationsMenu())
		return mh.SendNotificationsInfo(chatID)

	case "✅ Включить":
		return mh.HandleNotifyOn(chatID)

	case "❌ Выключить":
		return mh.HandleNotifyOff(chatID)

	case "📈 Сигналы":
		mh.messageSender.SetReplyKeyboard(mh.keyboards.GetSignalTypesMenu())
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
		mh.messageSender.SetReplyKeyboard(mh.keyboards.GetPeriodsMenu())
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
		mh.messageSender.SetReplyKeyboard(mh.keyboards.GetResetMenu())
		return mh.SendResetInfo(chatID)

	case "🔄 Все счетчики":
		return mh.HandleResetAllCounters(chatID)

	case "📋 Помощь":
		return mh.SendHelp(chatID)

	case "🔙 Назад", "🔙 Главное меню":
		mh.messageSender.SetReplyKeyboard(mh.keyboards.GetMainMenu())
		return mh.messageSender.SendMessageToChat(chatID, "🔙 Возврат в главное меню", nil)

	default:
		if strings.HasPrefix(text, "/") {
			return mh.HandleCommand(text, chatID)
		}
		return mh.messageSender.SendMessageToChat(chatID,
			"❓ Неизвестная команда. Используйте меню ниже или /help", nil)
	}
}

// SendSettingsInfo отправляет информацию о настройках
func (mh *MenuHandlers) SendSettingsInfo(chatID string) error {
	message := "⚙️ *Настройки бота*\n\n" +
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
	status := "❌ Выключены"
	if mh.config.TelegramEnabled {
		status = "✅ Включены"
	}

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
	message := "⏱️ *Настройка периодов анализа*\n\n" +
		"Текущий период: " + mh.config.CounterAnalyzer.DefaultPeriod + "\n\n" +
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
		"• 📈 Счетчик роста - сбросить счетчик роста\n" +
		"• 📉 Счетчик падения - сбросить счетчик падения\n" +
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
		mh.messageSender.SetReplyKeyboard(mh.keyboards.GetSettingsMenu())
		return mh.SendSettingsInfo(chatID)
	case "/test":
		return mh.messageSender.SendTestMessage()
	default:
		return mh.messageSender.SendMessageToChat(chatID,
			fmt.Sprintf("❓ Неизвестная команда: %s. Используйте /help", cmd), nil)
	}
}

// HandleCallback обрабатывает callback от inline кнопок
func (mh *MenuHandlers) HandleCallback(callbackData string, chatID string) error {
	switch callbackData {
	case "menu_notify":
		mh.messageSender.SetReplyKeyboard(mh.keyboards.GetNotificationsMenu())
		return mh.SendNotificationsInfo(chatID)
	case "menu_signals":
		mh.messageSender.SetReplyKeyboard(mh.keyboards.GetSignalTypesMenu())
		return mh.SendSignalTypesInfo(chatID)
	case "menu_periods":
		mh.messageSender.SetReplyKeyboard(mh.keyboards.GetPeriodsMenu())
		return mh.SendPeriodsInfo(chatID)
	case "menu_reset":
		mh.messageSender.SetReplyKeyboard(mh.keyboards.GetResetMenu())
		return mh.SendResetInfo(chatID)
	case "menu_back":
		mh.messageSender.SetReplyKeyboard(mh.keyboards.GetMainMenu())
		return mh.messageSender.SendMessageToChat(chatID, "🔙 Возврат в главное меню", nil)
	case "period_5m":
		return mh.HandlePeriodChange(chatID, "5m")
	case "period_15m":
		return mh.HandlePeriodChange(chatID, "15m")
	case "period_30m":
		return mh.HandlePeriodChange(chatID, "30m")
	case "period_1h":
		return mh.HandlePeriodChange(chatID, "1h")
	case "period_4h":
		return mh.HandlePeriodChange(chatID, "4h")
	case "reset_all":
		return mh.HandleResetAllCounters(chatID)
	case "reset_growth":
		return mh.messageSender.SendMessageToChat(chatID, "📈 Счетчик роста сброшен", nil)
	case "reset_fall":
		return mh.messageSender.SendMessageToChat(chatID, "📉 Счетчик падения сброшен", nil)
	case "reset_symbol":
		return mh.SendSymbolSelectionInline(chatID)
	default:
		return fmt.Errorf("unknown callback data: %s", callbackData)
	}
}

// SendSymbolSelectionInline отправляет inline меню выбора символа
func (mh *MenuHandlers) SendSymbolSelectionInline(chatID string) error {
	message := "Выберите символ для сброса счетчика:"

	keyboard := &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "BTCUSDT", CallbackData: "reset_btc"},
				{Text: "ETHUSDT", CallbackData: "reset_eth"},
				{Text: "SOLUSDT", CallbackData: "reset_sol"},
			},
			{
				{Text: "XRPUSDT", CallbackData: "reset_xrp"},
				{Text: "BNBUSDT", CallbackData: "reset_bnb"},
				{Text: "🔙 Назад", CallbackData: "menu_reset"},
			},
		},
	}

	return mh.messageSender.SendMessageToChat(chatID, message, keyboard)
}

// SendStatus отправляет статус системы
func (mh *MenuHandlers) SendStatus(chatID string) error {
	notifyStatus := "✅ Включены"
	if !mh.config.TelegramEnabled {
		notifyStatus = "❌ Выключены"
	}

	growthStatus := "✅ Включен"
	if !mh.config.TelegramNotifyGrowth {
		growthStatus = "❌ Выключен"
	}

	fallStatus := "✅ Включен"
	if !mh.config.TelegramNotifyFall {
		fallStatus = "❌ Выключен"
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
		mh.config.CounterAnalyzer.DefaultPeriod,
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
	periodMap := map[string]string{
		"5m":  "5 минут",
		"15m": "15 минут",
		"30m": "30 минут",
		"1h":  "1 час",
		"4h":  "4 часа",
	}

	periodName, exists := periodMap[period]
	if !exists {
		periodName = "15 минут"
	}

	mh.config.CounterAnalyzer.DefaultPeriod = period
	mh.config.CounterAnalyzer.AnalysisPeriod = period

	message := fmt.Sprintf("✅ Период анализа установлен на: %s\n\n"+
		"Все счетчики будут перезапущены с новым периодом.", periodName)

	return mh.messageSender.SendMessageToChat(chatID, message, nil)
}

// HandleResetAllCounters сбрасывает все счетчики
func (mh *MenuHandlers) HandleResetAllCounters(chatID string) error {
	message := "🔄 Все счетчики сигналов сброшены"
	return mh.messageSender.SendMessageToChat(chatID, message, nil)
}
