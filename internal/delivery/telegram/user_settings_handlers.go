// internal/delivery/telegram/user_settings_handlers.go
package telegram

import (
	"fmt"
	"log"
	"time"
)

// HandleNotifyOn включает уведомления (версия с userID)
func (mh *MenuHandlers) HandleNotifyOn(chatID string, userID int) error {
	// Если есть сервисы пользователей, используем персональные настройки
	if mh.hasUserServices() && userID > 0 {
		result, err := mh.settingsManager.SetNotification(userID, "all", true)
		if err != nil {
			log.Printf("❌ Ошибка включения уведомлений для user %d: %v", userID, err)
			// Fallback к глобальным настройкам
			mh.config.TelegramEnabled = true
			return mh.messageSender.SendMessageToChat(chatID, "✅ Уведомления включены", nil)
		}
		return mh.messageSender.SendMessageToChat(chatID, result, nil)
	}

	// Fallback: глобальные настройки
	mh.config.TelegramEnabled = true
	return mh.messageSender.SendMessageToChat(chatID, "✅ Уведомления включены", nil)
}

// HandleNotifyOff выключает уведомления (версия с userID)
func (mh *MenuHandlers) HandleNotifyOff(chatID string, userID int) error {
	// Если есть сервисы пользователей, используем персональные настройки
	if mh.hasUserServices() && userID > 0 {
		result, err := mh.settingsManager.SetNotification(userID, "all", false)
		if err != nil {
			log.Printf("❌ Ошибка выключения уведомлений для user %d: %v", userID, err)
			// Fallback к глобальным настройки
			mh.config.TelegramEnabled = false
			return mh.messageSender.SendMessageToChat(chatID, "❌ Уведомления выключены", nil)
		}
		return mh.messageSender.SendMessageToChat(chatID, result, nil)
	}

	// Fallback: глобальные настройки
	mh.config.TelegramEnabled = false
	return mh.messageSender.SendMessageToChat(chatID, "❌ Уведомления выключены", nil)
}

// handleNotifyToggle обрабатывает переключение уведомлений для callback
func (mh *MenuHandlers) handleNotifyToggle(chatID string, userID int) error {
	if mh.hasUserServices() && userID > 0 {
		// Получаем текущие настройки
		settings, err := mh.settingsManager.GetUserNotificationSettings(userID)
		if err != nil {
			log.Printf("❌ Ошибка получения настроек для user %d: %v", userID, err)
			return mh.messageSender.SendMessageToChat(chatID,
				"⚠️ Ошибка получения настроек", nil)
		}

		// Переключаем
		newValue := !settings.NotificationsEnabled
		result, err := mh.settingsManager.SetNotification(userID, "all", newValue)
		if err != nil {
			log.Printf("❌ Ошибка переключения уведомлений для user %d: %v", userID, err)
			return mh.messageSender.SendMessageToChat(chatID,
				"⚠️ Ошибка переключения уведомлений", nil)
		}
		return mh.messageSender.SendMessageToChat(chatID, result, nil)
	}

	// Fallback: глобальные настройки
	if mh.config.TelegramEnabled {
		return mh.HandleNotifyOff(chatID, userID)
	} else {
		return mh.HandleNotifyOn(chatID, userID)
	}
}

// handleTrackGrowthOnly обрабатывает отслеживание только роста
func (mh *MenuHandlers) handleTrackGrowthOnly(chatID string, userID int) error {
	if mh.hasUserServices() && userID > 0 {
		// Включаем рост, выключаем падение
		_, err1 := mh.settingsManager.SetNotification(userID, "growth", true)
		_, err2 := mh.settingsManager.SetNotification(userID, "fall", false)

		if err1 != nil || err2 != nil {
			log.Printf("⚠️ Ошибка настройки сигналов для user %d: %v, %v", userID, err1, err2)
			// Fallback к глобальным настройкам
			mh.config.TelegramNotifyGrowth = true
			mh.config.TelegramNotifyFall = false
		}

		return mh.messageSender.SendMessageToChat(chatID,
			"✅ Теперь отслеживается только рост", nil)
	}

	// Fallback: глобальные настройки
	mh.config.TelegramNotifyGrowth = true
	mh.config.TelegramNotifyFall = false
	return mh.messageSender.SendMessageToChat(chatID,
		"✅ Теперь отслеживается только рост", nil)
}

// handleTrackFallOnly обрабатывает отслеживание только падения
func (mh *MenuHandlers) handleTrackFallOnly(chatID string, userID int) error {
	if mh.hasUserServices() && userID > 0 {
		// Включаем падение, выключаем рост
		_, err1 := mh.settingsManager.SetNotification(userID, "growth", false)
		_, err2 := mh.settingsManager.SetNotification(userID, "fall", true)

		if err1 != nil || err2 != nil {
			log.Printf("⚠️ Ошибка настройки сигналов для user %d: %v, %v", userID, err1, err2)
			// Fallback к глобальным настройкам
			mh.config.TelegramNotifyGrowth = false
			mh.config.TelegramNotifyFall = true
		}

		return mh.messageSender.SendMessageToChat(chatID,
			"✅ Теперь отслеживается только падение", nil)
	}

	// Fallback: глобальные настройки
	mh.config.TelegramNotifyGrowth = false
	mh.config.TelegramNotifyFall = true
	return mh.messageSender.SendMessageToChat(chatID,
		"✅ Теперь отслеживается только падение", nil)
}

// handleTrackBoth обрабатывает отслеживание всех сигналов
func (mh *MenuHandlers) handleTrackBoth(chatID string, userID int) error {
	if mh.hasUserServices() && userID > 0 {
		// Включаем и рост, и падение
		_, err1 := mh.settingsManager.SetNotification(userID, "growth", true)
		_, err2 := mh.settingsManager.SetNotification(userID, "fall", true)

		if err1 != nil || err2 != nil {
			log.Printf("⚠️ Ошибка настройки сигналов для user %d: %v, %v", userID, err1, err2)
			// Fallback к глобальным настройкам
			mh.config.TelegramNotifyGrowth = true
			mh.config.TelegramNotifyFall = true
		}

		return mh.messageSender.SendMessageToChat(chatID,
			"✅ Теперь отслеживаются все сигналы", nil)
	}

	// Fallback: глобальные настройки
	mh.config.TelegramNotifyGrowth = true
	mh.config.TelegramNotifyFall = true
	return mh.messageSender.SendMessageToChat(chatID,
		"✅ Теперь отслеживаются все сигналы", nil)
}

// HandlePeriodChange обрабатывает изменение периода (обновленная версия)
func (mh *MenuHandlers) HandlePeriodChange(chatID string, userID int, period string) error {
	// Если есть сервисы пользователей, используем персональные настройки
	if mh.hasUserServices() && userID > 0 {
		result, err := mh.settingsManager.SetPreferredPeriod(userID, period)
		if err != nil {
			log.Printf("❌ Ошибка установки периода для user %d: %v", userID, err)
			// Fallback к глобальным настройкам
			return mh.handleGlobalPeriodChange(chatID, period)
		}
		return mh.messageSender.SendMessageToChat(chatID, result, nil)
	}

	// Fallback: глобальные настройки
	return mh.handleGlobalPeriodChange(chatID, period)
}

// handleGlobalPeriodChange устанавливает период в глобальных настройках
func (mh *MenuHandlers) handleGlobalPeriodChange(chatID string, period string) error {
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

// SendStatus отправляет статус системы (версия с userID)
func (mh *MenuHandlers) SendStatus(chatID string, userID int) error {
	// Если есть сервисы пользователей, получаем персональные настройки
	if mh.hasUserServices() && userID > 0 {
		settings, err := mh.settingsManager.GetUserSettingsTelegram(userID)
		if err == nil {
			// Получаем предпочтительный период
			period, err := mh.settingsManager.GetPreferredPeriod(userID)
			periodName := "15 минут"
			if err == nil && mh.menuUtils != nil {
				periodName = mh.menuUtils.GetPeriodName(period)
			}

			message := fmt.Sprintf(
				"📊 *Статус системы*\n\n"+
					"✅ Бот работает\n"+
					"👤 *Персональные настройки:*\n%s\n\n"+
					"⏱️ Период анализа: %s\n"+
					"🕐 Время сервера: %s",
				settings,
				periodName,
				time.Now().Format("15:04:05"),
			)
			return mh.messageSender.SendMessageToChat(chatID, message, nil)
		}
		log.Printf("⚠️ Не удалось получить настройки пользователя %d: %v", userID, err)
	}

	// Fallback: глобальные настройки
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
