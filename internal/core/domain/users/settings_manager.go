// internal/core/domain/users/settings_manager.go
package users

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"

	"github.com/go-redis/redis/v8"
)

// SettingsManager управляет настройками пользователей через Telegram
type SettingsManager struct {
	userService *Service
	redisClient *redis.Client
	cachePrefix string
}

// NewSettingsManager создает новый менеджер настроек
func NewSettingsManager(userService *Service, redisClient *redis.Client) *SettingsManager {
	return &SettingsManager{
		userService: userService,
		redisClient: redisClient,
		cachePrefix: "user_settings:",
	}
}

// GetUserSettingsTelegram получает настройки для отображения в Telegram
func (sm *SettingsManager) GetUserSettingsTelegram(userID int) (string, error) {
	user, err := sm.userService.GetUserByID(userID)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return "", fmt.Errorf("user not found")
	}

	return sm.formatSettingsForTelegram(user), nil
}

// ToggleNotification переключает тип уведомлений
func (sm *SettingsManager) ToggleNotification(userID int, notificationType string) (string, error) {
	user, err := sm.userService.GetUserByID(userID)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return "", fmt.Errorf("user not found")
	}

	// Убрана объявление неиспользуемой переменной field
	var oldValue, newValue bool

	switch notificationType {
	case "all":
		oldValue = user.NotificationsEnabled
		user.NotificationsEnabled = !oldValue
		newValue = user.NotificationsEnabled
	case "growth":
		oldValue = user.NotifyGrowth
		user.NotifyGrowth = !oldValue
		newValue = user.NotifyGrowth
	case "fall":
		oldValue = user.NotifyFall
		user.NotifyFall = !oldValue
		newValue = user.NotifyFall
	case "continuous":
		oldValue = user.NotifyContinuous
		user.NotifyContinuous = !oldValue
		newValue = user.NotifyContinuous
	default:
		return "", fmt.Errorf("unknown notification type: %s", notificationType)
	}

	// Сохраняем изменения
	if err := sm.userService.UpdateUser(user); err != nil {
		return "", fmt.Errorf("failed to update user: %w", err)
	}

	// Инвалидируем кэш
	sm.invalidateUserCache(userID)

	return sm.formatToggleResult(notificationType, oldValue, newValue), nil
}

// UpdateThreshold обновляет порог сигнала
func (sm *SettingsManager) UpdateThreshold(userID int, thresholdType string, value float64) (string, error) {
	user, err := sm.userService.GetUserByID(userID)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return "", fmt.Errorf("user not found")
	}

	// Валидация
	if value < 0.1 || value > 50.0 {
		return "", fmt.Errorf("threshold must be between 0.1%% and 50%%")
	}

	var oldValue float64
	var field string

	switch thresholdType {
	case "growth":
		field = "min_growth_threshold"
		oldValue = user.MinGrowthThreshold
		user.MinGrowthThreshold = value
	case "fall":
		field = "min_fall_threshold"
		oldValue = user.MinFallThreshold
		user.MinFallThreshold = value
	default:
		return "", fmt.Errorf("unknown threshold type: %s", thresholdType)
	}

	// Сохраняем изменения
	if err := sm.userService.UpdateUser(user); err != nil {
		return "", fmt.Errorf("failed to update user: %w", err)
	}

	// Инвалидируем кэш
	sm.invalidateUserCache(userID)

	log.Printf("User %d updated %s from %.2f%% to %.2f%%",
		userID, field, oldValue, value)

	return fmt.Sprintf("✅ Порог %s изменен: %.2f%% → %.2f%%",
		thresholdType, oldValue, value), nil
}

// SetQuietHours устанавливает тихие часы
func (sm *SettingsManager) SetQuietHours(userID, startHour, endHour int) (string, error) {
	// Валидация
	if startHour < 0 || startHour > 23 || endHour < 0 || endHour > 23 {
		return "", fmt.Errorf("часы должны быть от 0 до 23")
	}

	user, err := sm.userService.GetUserByID(userID)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return "", fmt.Errorf("user not found")
	}

	oldStart := user.QuietHoursStart
	oldEnd := user.QuietHoursEnd

	user.QuietHoursStart = startHour
	user.QuietHoursEnd = endHour

	// Сохраняем изменения
	if err := sm.userService.UpdateUser(user); err != nil {
		return "", fmt.Errorf("failed to update user: %w", err)
	}

	// Инвалидируем кэш
	sm.invalidateUserCache(userID)

	log.Printf("User %d updated quiet hours: %d-%d → %d-%d",
		userID, oldStart, oldEnd, startHour, endHour)

	return fmt.Sprintf("✅ Тихие часы установлены: %02d:00 - %02d:00",
		startHour, endHour), nil
}

// ResetToDefault сбрасывает настройки к значениям по умолчанию
func (sm *SettingsManager) ResetToDefault(userID int) (string, error) {
	user, err := sm.userService.GetUserByID(userID)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return "", fmt.Errorf("user not found")
	}

	// Создаем пользователя с настройками по умолчанию
	defaultUser := models.NewUser(user.TelegramID, user.Username,
		user.FirstName, user.LastName, user.ChatID)

	// Копируем только настройки, сохраняем остальные данные
	user.MinGrowthThreshold = defaultUser.MinGrowthThreshold
	user.MinFallThreshold = defaultUser.MinFallThreshold
	user.NotificationsEnabled = defaultUser.NotificationsEnabled
	user.NotifyGrowth = defaultUser.NotifyGrowth
	user.NotifyFall = defaultUser.NotifyFall
	user.NotifyContinuous = defaultUser.NotifyContinuous
	user.QuietHoursStart = defaultUser.QuietHoursStart
	user.QuietHoursEnd = defaultUser.QuietHoursEnd
	user.PreferredPeriods = defaultUser.PreferredPeriods
	user.MinVolumeFilter = defaultUser.MinVolumeFilter
	user.ExcludePatterns = defaultUser.ExcludePatterns
	user.Language = defaultUser.Language
	user.Timezone = defaultUser.Timezone
	user.DisplayMode = defaultUser.DisplayMode

	// Сохраняем изменения
	if err := sm.userService.UpdateUser(user); err != nil {
		return "", fmt.Errorf("failed to update user: %w", err)
	}

	// Инвалидируем кэш
	sm.invalidateUserCache(userID)

	log.Printf("User %d settings reset to default", userID)

	return "✅ Настройки сброшены к значениям по умолчанию", nil
}

// Вспомогательные методы

// formatSettingsForTelegram форматирует настройки для Telegram
func (sm *SettingsManager) formatSettingsForTelegram(user *models.User) string {
	notificationsStatus := "❌"
	if user.NotificationsEnabled {
		notificationsStatus = "✅"
	}

	growthStatus := "❌"
	if user.NotifyGrowth {
		growthStatus = "✅"
	}

	fallStatus := "❌"
	if user.NotifyFall {
		fallStatus = "✅"
	}

	quietHours := "Не настроены"
	if user.QuietHoursStart != 0 || user.QuietHoursEnd != 0 {
		quietHours = fmt.Sprintf("%02d:00 - %02d:00",
			user.QuietHoursStart, user.QuietHoursEnd)
	}

	return fmt.Sprintf(
		"⚙️ *Настройки пользователя* @%s\n\n"+
			"📊 *Анализ:*\n"+
			"   Рост: ≥ %.2f%%\n"+
			"   Падение: ≥ %.2f%%\n\n"+
			"🔔 *Уведомления:*\n"+
			"   Все: %s\n"+
			"   Рост: %s\n"+
			"   Падение: %s\n"+
			"   Тихие часы: %s\n\n"+
			"📈 *Статистика:*\n"+
			"   Сигналов сегодня: %d/%d\n"+
			"   Подписка: %s",
		user.Username,
		user.MinGrowthThreshold,
		user.MinFallThreshold,
		notificationsStatus,
		growthStatus,
		fallStatus,
		quietHours,
		user.SignalsToday,
		user.MaxSignalsPerDay,
		user.SubscriptionTier,
	)
}

// formatToggleResult форматирует результат переключения
func (sm *SettingsManager) formatToggleResult(notificationType string, oldValue, newValue bool) string {
	status := "❌ Выключено"
	if newValue {
		status = "✅ Включено"
	}

	typeNames := map[string]string{
		"all":        "все уведомления",
		"growth":     "уведомления о росте",
		"fall":       "уведомления о падении",
		"continuous": "уведомления о непрерывных сигналах",
	}

	typeName := typeNames[notificationType]
	if typeName == "" {
		typeName = notificationType
	}

	return fmt.Sprintf("%s: %s", typeName, status)
}

// invalidateUserCache инвалидирует кэш пользователя
func (sm *SettingsManager) invalidateUserCache(userID int) {
	keys := []string{
		sm.cachePrefix + strconv.Itoa(userID),
		"user:" + strconv.Itoa(userID),
		"user_settings:" + strconv.Itoa(userID),
	}

	ctx := context.Background()
	for _, key := range keys {
		sm.redisClient.Del(ctx, key)
	}
}
