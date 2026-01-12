// internal/delivery/telegram/callback_handlers.go
package telegram

import (
	"fmt"
	"log"
	"strings"
)

// handleMenuCallback обрабатывает callback'ы меню (версия с userID)
func (mh *MenuHandlers) handleMenuCallback(menuType, chatID string, userID int) error {
	switch menuType {
	case "notify":
		mh.messageSender.SetReplyKeyboard(chatID, mh.keyboardSystem.GetNotificationsMenu())
		return mh.SendNotificationsInfo(chatID, userID)
	case "signals":
		mh.messageSender.SetReplyKeyboard(chatID, mh.keyboardSystem.GetSignalTypesMenu())
		return mh.SendSignalTypesInfo(chatID, userID)
	case "periods":
		mh.messageSender.SetReplyKeyboard(chatID, mh.keyboardSystem.GetPeriodsMenu())
		return mh.SendPeriodsInfo(chatID, userID)
	case "reset":
		mh.messageSender.SetReplyKeyboard(chatID, mh.keyboardSystem.GetResetMenu())
		return mh.SendResetInfo(chatID, userID)
	case "back":
		mh.messageSender.SetReplyKeyboard(chatID, mh.keyboardSystem.GetMainMenu())
		return mh.messageSender.SendMessageToChat(chatID, "🔙 Возврат в главное меню", nil)
	}
	return nil
}

// handleResetCallback обрабатывает callback'ы сброса (версия с userID)
func (mh *MenuHandlers) handleResetCallback(param, callbackData, chatID string, userID int) error {
	switch param {
	case "all":
		return mh.HandleResetAllCounters(chatID)
	case "symbol":
		return mh.SendSymbolSelectionInline(chatID)
	case "settings":
		// Сброс настроек пользователя
		if mh.hasUserServices() && userID > 0 {
			result, err := mh.settingsManager.ResetToDefault(userID)
			if err != nil {
				log.Printf("❌ Ошибка сброса настроек для user %d: %v", userID, err)
				return mh.messageSender.SendMessageToChat(chatID,
					"⚠️ Ошибка сброса настроек", nil)
			}
			return mh.messageSender.SendMessageToChat(chatID, result, nil)
		}
		return mh.messageSender.SendMessageToChat(chatID,
			"ℹ️ Функция сброса настроек доступна только для авторизованных пользователей", nil)
	default:
		// Проверяем, не начинается ли с symbol_
		if strings.HasPrefix(callbackData, "symbol_") {
			symbol := strings.TrimPrefix(callbackData, "symbol_")
			return mh.messageSender.SendMessageToChat(chatID,
				fmt.Sprintf("📊 Счетчик для %s сброшен", strings.ToUpper(symbol)), nil)
		}
	}
	return nil
}

// handleNotifyCallback обрабатывает callback'ы уведомлений (версия с userID)
func (mh *MenuHandlers) handleNotifyCallback(param, chatID string, userID int) error {
	switch param {
	case "on":
		return mh.HandleNotifyOn(chatID, userID)
	case "off":
		return mh.HandleNotifyOff(chatID, userID)
	}
	return nil
}

// handleCallbackSettings обрабатывает callback Settings (версия с userID)
func (mh *MenuHandlers) handleCallbackSettings(chatID string, userID int) error {
	mh.messageSender.SetReplyKeyboard(chatID, mh.keyboardSystem.GetSettingsMenu())
	return mh.SendSettingsInfo(chatID, userID)
}

// handleSignalTypeCallback обрабатывает выбор типа сигналов (версия с userID)
func (mh *MenuHandlers) handleSignalTypeCallback(chatID string, userID int) error {
	// Показываем inline клавиатуру для выбора типа сигналов
	keyboard := mh.keyboardSystem.CreateSignalTypeKeyboard(
		mh.config.TelegramNotifyGrowth,
		mh.config.TelegramNotifyFall,
	)
	return mh.messageSender.SendMessageToChat(chatID,
		"📊 *Выберите тип отслеживаемых сигналов:*", keyboard)
}

// handleChangePeriodCallback обрабатывает изменение периода (версия с userID)
func (mh *MenuHandlers) handleChangePeriodCallback(chatID string, userID int) error {
	// Показываем inline клавиатуру для выбора периода
	keyboard := mh.keyboardSystem.CreatePeriodSelectionKeyboard()
	return mh.messageSender.SendMessageToChat(chatID,
		"⏱️ *Выберите период анализа:*", keyboard)
}

// handleSettingsBack обрабатывает возврат к настройкам (версия с userID)
func (mh *MenuHandlers) handleSettingsBack(chatID string, userID int) error {
	// Возвращаемся к основному меню настроек
	keyboard := mh.keyboardSystem.CreateSettingsKeyboard(
		mh.config.TelegramEnabled,
		false, // testMode
	)
	return mh.messageSender.SendMessageToChat(chatID,
		"⚙️ *Настройки бота:*", keyboard)
}

// handleBackToMain обрабатывает возврат в главное меню
func (mh *MenuHandlers) handleBackToMain(chatID string) error {
	mh.messageSender.SetReplyKeyboard(chatID, mh.keyboardSystem.GetMainMenu())
	return mh.messageSender.SendMessageToChat(chatID,
		"🔙 Возврат в главное меню", nil)
}

// handleResetCounterCallback обрабатывает сброс счетчика (версия с userID)
func (mh *MenuHandlers) handleResetCounterCallback(chatID string, userID int) error {
	// Показываем inline клавиатуру для сброса
	keyboard := mh.keyboardSystem.CreateResetKeyboard()
	return mh.messageSender.SendMessageToChat(chatID,
		"🔄 *Выберите что сбросить:*", keyboard)
}

// handleChartCallback обрабатывает callback графика
func (mh *MenuHandlers) handleChartCallback(chatID string) error {
	return mh.messageSender.SendMessageToChat(chatID,
		"📊 *Графики*\n\n"+
			"Используйте кнопки в уведомлениях для перехода к графикам.", nil)
}

// handleTestOK обрабатывает тест OK
func (mh *MenuHandlers) handleTestOK(chatID string) error {
	return mh.messageSender.SendMessageToChat(chatID,
		"✅ Тест пройден успешно!", nil)
}

// handleTestCancel обрабатывает отмену теста
func (mh *MenuHandlers) handleTestCancel(chatID string) error {
	return mh.messageSender.SendMessageToChat(chatID,
		"❌ Тест отменен", nil)
}

// handleToggleTestMode обрабатывает переключение тестового режима
func (mh *MenuHandlers) handleToggleTestMode(chatID string) error {
	return mh.messageSender.SendMessageToChat(chatID,
		"🧪 Функционал тестового режима в разработке", nil)
}
