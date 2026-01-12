// internal/delivery/telegram/message_handlers.go
package telegram

import (
	"log"
	"strings"
)

// handleSettings обрабатывает кнопку "Настройки" (версия с userID)
func (mh *MenuHandlers) handleSettings(chatID string, userID int) error {
	log.Printf("🎯 Обработка кнопки 'Настройки' для чата %s, user %d", chatID, userID)

	// 1. Получаем меню настроек
	log.Printf("📋 Получаем меню настроек...")
	settingsMenu := mh.keyboardSystem.GetSettingsMenu()
	log.Printf("✅ Меню настроек получено: %+v", settingsMenu)

	// 2. Устанавливаем клавиатуру для этого чата
	log.Printf("⌨️ Устанавливаем клавиатуру для чата %s...", chatID)
	err := mh.messageSender.SetReplyKeyboard(chatID, settingsMenu)
	if err != nil {
		log.Printf("❌ Ошибка установки клавиатуры: %v", err)
		return err
	}
	log.Printf("✅ Клавиатура установлена")

	// 3. Отправляем сообщение с информацией
	log.Printf("📤 Отправляем сообщение с информацией...")
	return mh.SendSettingsInfo(chatID, userID)
}

// handleNotifications обрабатывает кнопку "Уведомления" (версия с userID)
func (mh *MenuHandlers) handleNotifications(chatID string, userID int) error {
	log.Printf("🎯 Обработка кнопки 'Уведомления' для чата %s, user %d", chatID, userID)
	// Получаем меню уведомлений
	notificationsMenu := mh.keyboardSystem.GetNotificationsMenu()
	// Устанавливаем клавиатуру
	mh.messageSender.SetReplyKeyboard(chatID, notificationsMenu)
	// Отправляем информацию
	return mh.SendNotificationsInfo(chatID, userID)
}

// handleSignals обрабатывает кнопку "Сигналы" (версия с userID)
func (mh *MenuHandlers) handleSignals(chatID string, userID int) error {
	log.Printf("🎯 Обработка кнопки 'Сигналы' для чата %s, user %d", chatID, userID)
	// Получаем меню типов сигналов
	signalTypesMenu := mh.keyboardSystem.GetSignalTypesMenu()
	// Устанавливаем клавиатуру
	mh.messageSender.SetReplyKeyboard(chatID, signalTypesMenu)
	// Отправляем информацию
	return mh.SendSignalTypesInfo(chatID, userID)
}

// handleGrowthOnly обрабатывает кнопку "Только рост" (версия с userID)
func (mh *MenuHandlers) handleGrowthOnly(chatID string, userID int) error {
	return mh.handleTrackGrowthOnly(chatID, userID)
}

// handleFallOnly обрабатывает кнопку "Только падение" (версия с userID)
func (mh *MenuHandlers) handleFallOnly(chatID string, userID int) error {
	return mh.handleTrackFallOnly(chatID, userID)
}

// handleAllSignals обрабатывает кнопку "Все сигналы" (версия с userID)
func (mh *MenuHandlers) handleAllSignals(chatID string, userID int) error {
	return mh.handleTrackBoth(chatID, userID)
}

// handlePeriods обрабатывает кнопку "Периоды" (версия с userID)
func (mh *MenuHandlers) handlePeriods(chatID string, userID int) error {
	log.Printf("🎯 Обработка кнопки 'Периоды' для чата %s, user %d", chatID, userID)
	// Получаем меню периодов
	periodsMenu := mh.keyboardSystem.GetPeriodsMenu()
	// Устанавливаем клавиатуру
	mh.messageSender.SetReplyKeyboard(chatID, periodsMenu)
	// Отправляем информацию
	return mh.SendPeriodsInfo(chatID, userID)
}

// handleReset обрабатывает кнопку "Сбросить" (версия с userID)
func (mh *MenuHandlers) handleReset(chatID string, userID int) error {
	log.Printf("🎯 Обработка кнопки 'Сбросить' для чата %s, user %d", chatID, userID)
	// Получаем меню сброса
	resetMenu := mh.keyboardSystem.GetResetMenu()
	// Устанавливаем клавиатуру
	mh.messageSender.SetReplyKeyboard(chatID, resetMenu)
	// Отправляем информацию
	return mh.SendResetInfo(chatID, userID)
}

// handleBack обрабатывает кнопку "Назад"
func (mh *MenuHandlers) handleBack(chatID string) error {
	log.Printf("🎯 Обработка кнопки 'Назад' для чата %s", chatID)
	// Получаем главное меню
	mainMenu := mh.keyboardSystem.GetMainMenu()
	// Устанавливаем клавиатуру
	mh.messageSender.SetReplyKeyboard(chatID, mainMenu)
	return mh.messageSender.SendMessageToChat(chatID, "🔙 Возврат в главное меню", nil)
}

// handleDefault обрабатывает неизвестные команды
func (mh *MenuHandlers) handleDefault(text, chatID string) error {
	log.Printf("❓ Неизвестная команда: '%s' для чата %s", text, chatID)
	if strings.HasPrefix(text, "/") {
		// Проверяем, является ли команда командой авторизации
		if mh.isAuthCommand(text) {
			log.Printf("🔐 Команда авторизации: '%s'", text)
			return mh.handleAuthMessage(text, chatID)
		}
		log.Printf("⚡ Обработка команды: '%s'", text)
		return mh.HandleCommand(text, chatID)
	}
	log.Printf("📝 Отправка сообщения об ошибке для неизвестной команды")
	return mh.messageSender.SendMessageToChat(chatID,
		"❓ Неизвестная команда. Используйте меню ниже или /help", nil)
}
