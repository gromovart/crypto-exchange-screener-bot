// internal/delivery/telegram/command_handlers.go
package telegram

import (
	"fmt"
	"log"
	"time"
)

// StartCommandHandler обрабатывает команду /start
func (mh *MenuHandlers) StartCommandHandler(chatID string) error {
	log.Printf("🔍 StartCommandHandler ВЫЗВАН: chatID=%s", chatID)

	message := "🚀 *Crypto Exchange Screener Bot*\n\n" +
		"*Основные команды:*\n" +
		"• /start - Начало работы\n" +
		"• /status - Статус системы\n" +
		"• /notify_on - Включить уведомления\n" +
		"• /notify_off - Выключить уведомления\n" +
		"• /help - Справка\n\n" +
		"Используйте меню ниже для управления ботом:"

	// 1. Сначала отправляем приветственное сообщение с inline клавиатурой
	err := mh.messageSender.SendMessageToChat(chatID, message, nil)
	if err != nil {
		log.Printf("❌ Ошибка отправки приветственного сообщения: %v", err)
		return err
	}

	// 2. Затем устанавливаем reply клавиатуру (меню) для этого чата
	mainMenu := mh.keyboardSystem.GetMainMenu()

	// Добавляем небольшую задержку перед установкой меню
	time.Sleep(300 * time.Millisecond)

	setupErr := mh.messageSender.SetReplyKeyboard(chatID, mainMenu)
	if setupErr != nil {
		log.Printf("⚠️ Ошибка установки меню: %v", setupErr)
		return nil
	}

	log.Printf("✅ Меню установлено для чата %s после команды /start", chatID)
	return nil
}

// HandleCommand обрабатывает текстовые команды
func (mh *MenuHandlers) HandleCommand(cmd, chatID string) error {
	// Получаем userID для команд, которые работают с настройками
	userID := mh.getUserIDFromChatID(chatID)

	switch cmd {
	case "/start":
		return mh.StartCommandHandler(chatID)
	case "/help":
		return mh.SendHelp(chatID)
	case "/status":
		return mh.SendStatus(chatID, userID)
	case "/notify_on":
		return mh.HandleNotifyOn(chatID, userID)
	case "/notify_off":
		return mh.HandleNotifyOff(chatID, userID)
	case "/settings":
		mh.messageSender.SetReplyKeyboard(chatID, mh.keyboardSystem.GetSettingsMenu())
		return mh.SendSettingsInfo(chatID, userID)
	case "/test":
		return mh.messageSender.SendTestMessage()
	default:
		// Проверяем, является ли команда командой авторизации
		if mh.isAuthCommand(cmd) {
			return mh.handleAuthMessage(cmd, chatID)
		}
		return mh.messageSender.SendMessageToChat(chatID,
			fmt.Sprintf("❓ Неизвестная команда: %s. Используйте /help", cmd), nil)
	}
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

// HandleResetAllCounters сбрасывает все счетчики
func (mh *MenuHandlers) HandleResetAllCounters(chatID string) error {
	message := "🔄 Все счетчики сигналов сброшены"
	return mh.messageSender.SendMessageToChat(chatID, message, nil)
}

// SendSymbolSelectionInline отправляет inline меню выбора символа
func (mh *MenuHandlers) SendSymbolSelectionInline(chatID string) error {
	message := "Выберите символ для сброса счетчика:"

	// Используем KeyboardSystem для создания клавиатуры
	keyboard := mh.keyboardSystem.CreateSymbolSelectionKeyboard()

	return mh.messageSender.SendMessageToChat(chatID, message, keyboard)
}
