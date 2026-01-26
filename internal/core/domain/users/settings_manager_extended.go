// internal/core/domain/users/settings_manager_extended.go
package users

import (
	"fmt"
	"log"

	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
)

// SetNotification устанавливает конкретное значение для типа уведомлений
func (sm *SettingsManager) SetNotification(userID int, notificationType string, enabled bool) (string, error) {
	user, err := sm.userService.GetUserByID(userID)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return "", fmt.Errorf("user not found")
	}

	var oldValue bool
	var fieldName string

	switch notificationType {
	case "all":
		oldValue = user.NotificationsEnabled
		user.NotificationsEnabled = enabled
		fieldName = "все уведомления"
	case "growth":
		oldValue = user.NotifyGrowth
		user.NotifyGrowth = enabled
		fieldName = "уведомления о росте"
	case "fall":
		oldValue = user.NotifyFall
		user.NotifyFall = enabled
		fieldName = "уведомления о падении"
	case "continuous":
		oldValue = user.NotifyContinuous
		user.NotifyContinuous = enabled
		fieldName = "уведомления о непрерывных сигналах"
	default:
		return "", fmt.Errorf("unknown notification type: %s", notificationType)
	}

	// Сохраняем изменения
	if err := sm.userService.UpdateUser(user); err != nil {
		return "", fmt.Errorf("failed to update user: %w", err)
	}

	// Инвалидируем кэш
	sm.invalidateUserCache(userID)

	status := "❌ Выключено"
	if enabled {
		status = "✅ Включено"
	}

	log.Printf("User %d set %s: %v -> %v", userID, notificationType, oldValue, enabled)

	return fmt.Sprintf("%s: %s", fieldName, status), nil
}

// SetPreferredPeriod устанавливает предпочтительный период анализа
func (sm *SettingsManager) SetPreferredPeriod(userID int, period string) (string, error) {
	// Конвертируем строковый период в минуты
	periodMinutes, err := sm.periodingToMinutes(period)
	if err != nil {
		return "", err
	}

	user, err := sm.userService.GetUserByID(userID)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return "", fmt.Errorf("user not found")
	}

	// Проверяем, есть ли уже этот период в списке
	found := false
	for i, p := range user.PreferredPeriods {
		if p == periodMinutes {
			found = true
			// Если период уже есть, удаляем его (toggle)
			user.PreferredPeriods = append(user.PreferredPeriods[:i], user.PreferredPeriods[i+1:]...)
			break
		}
	}

	if !found {
		// Добавляем новый период
		user.PreferredPeriods = append(user.PreferredPeriods, periodMinutes)
	}

	// Сохраняем изменения
	if err := sm.userService.UpdateUser(user); err != nil {
		return "", fmt.Errorf("failed to update user: %w", err)
	}

	// Инвалидируем кэш
	sm.invalidateUserCache(userID)

	// Получаем читаемое имя периода
	periodName := sm.getPeriodName(period)

	log.Printf("User %d set preferred period: %s (%d minutes)", userID, period, periodMinutes)

	action := "добавлен"
	if found {
		action = "удален"
	}

	return fmt.Sprintf("✅ Период %s %s в предпочтительные", periodName, action), nil
}

// GetPreferredPeriod получает предпочтительный период пользователя
func (sm *SettingsManager) GetPreferredPeriod(userID int) (string, error) {
	user, err := sm.userService.GetUserByID(userID)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return "", fmt.Errorf("user not found")
	}

	if len(user.PreferredPeriods) == 0 {
		return "15m", nil // Значение по умолчанию
	}

	// Возвращаем первый период из списка в строковом формате
	return sm.minutesToPeriodString(user.PreferredPeriods[0]), nil
}

// GetUserNotificationSettings получает текущие настройки уведомлений пользователя
func (sm *SettingsManager) GetUserNotificationSettings(userID int) (*models.UserNotificationSettings, error) {
	user, err := sm.userService.GetUserByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	return &models.UserNotificationSettings{
		NotificationsEnabled: user.NotificationsEnabled,
		NotifyGrowth:         user.NotifyGrowth,
		NotifyFall:           user.NotifyFall,
		NotifyContinuous:     user.NotifyContinuous,
		MinGrowthThreshold:   user.MinGrowthThreshold,
		MinFallThreshold:     user.MinFallThreshold,
		QuietHoursStart:      user.QuietHoursStart,
		QuietHoursEnd:        user.QuietHoursEnd,
		PreferredPeriods:     user.PreferredPeriods,
	}, nil
}

// periodingToMinutes конвертирует строковый период в минуты
func (sm *SettingsManager) periodingToMinutes(period string) (int, error) {
	switch period {
	case "5m":
		return 5, nil
	case "15m":
		return 15, nil
	case "30m":
		return 30, nil
	case "1h":
		return 60, nil
	case "4h":
		return 240, nil
	case "1d":
		return 1440, nil
	default:
		return 0, fmt.Errorf("недопустимый период: %s. Допустимые: 5m, 15m, 30m, 1h, 4h, 1d", period)
	}
}

// minutesToPeriodString конвертирует минуты в строковый период
func (sm *SettingsManager) minutesToPeriodString(minutes int) string {
	switch minutes {
	case 5:
		return "5m"
	case 15:
		return "15m"
	case 30:
		return "30m"
	case 60:
		return "1h"
	case 240:
		return "4h"
	case 1440:
		return "1d"
	default:
		return fmt.Sprintf("%dm", minutes)
	}
}

// getPeriodName получает читаемое имя периода
func (sm *SettingsManager) getPeriodName(period string) string {
	periodNames := map[string]string{
		"5m":  "5 минут",
		"15m": "15 минут",
		"30m": "30 минут",
		"1h":  "1 час",
		"4h":  "4 часа",
		"1d":  "1 день",
	}

	periodName := periodNames[period]
	if periodName == "" {
		periodName = period
	}
	return periodName
}
