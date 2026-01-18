// internal/delivery/telegram/services/counter/filters.go
package counter

import (
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"math"
)

// getUsersToNotify возвращает пользователей, которым нужно отправить уведомление
func (s *serviceImpl) getUsersToNotify(data RawCounterData) ([]*models.User, error) {
	if s.userService == nil {
		return nil, fmt.Errorf("userService not initialized")
	}

	// Получаем всех пользователей
	allUsers, err := s.userService.GetAllUsers(1000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	// Фильтруем пользователей
	var filteredUsers []*models.User
	for _, user := range allUsers {
		if s.shouldSendToUser(user, data) {
			filteredUsers = append(filteredUsers, user)
		}
	}

	return filteredUsers, nil
}

// shouldSendToUser проверяет, нужно ли отправлять пользователю
func (s *serviceImpl) shouldSendToUser(user *models.User, data RawCounterData) bool {
	// БАЗОВЫЕ ПРОВЕРКИ
	if user == nil {
		return false
	}

	// Проверяем ChatID
	if user.ChatID == "" {
		logger.Debug("⚠️ User %d (%s) skipped: empty chat_id", user.ID, user.Username)
		return false
	}

	// Проверяем активность
	if !user.IsActive {
		logger.Debug("⚠️ User %d (%s) skipped: not active", user.ID, user.Username)
		return false
	}

	// Определяем тип сигнала для проверки
	var signalType string
	switch data.Direction {
	case "growth":
		signalType = "growth"
	case "fall":
		signalType = "fall"
	default:
		logger.Debug("⚠️ User %d (%s) skipped: unknown direction %s",
			user.ID, user.Username, data.Direction)
		return false
	}

	// Используем метод ShouldReceiveSignal из модели User
	var changePercentForCheck float64
	if signalType == "fall" {
		changePercentForCheck = -data.ChangePercent
	} else {
		changePercentForCheck = data.ChangePercent
	}

	shouldReceive := user.ShouldReceiveSignal(signalType, changePercentForCheck)

	if !shouldReceive {
		s.logUserSkipReason(user, signalType, changePercentForCheck, data)
		return false
	}

	// ПРИМЕНЯЕМ ДОПОЛНИТЕЛЬНЫЕ ФИЛЬТРЫ ПОЛЬЗОВАТЕЛЯ
	if !s.applyUserFilters(user, data) {
		return false
	}

	logger.Debug("✅ User %d (%s) passed all checks for %s signal (%.2f%%)",
		user.ID, user.Username, signalType, changePercentForCheck)
	return true
}

// logUserSkipReason логирует причину пропуска пользователя
func (s *serviceImpl) logUserSkipReason(user *models.User, signalType string, changePercent float64, data RawCounterData) {
	// Проверяем базовые условия в том же порядке, что и ShouldReceiveSignal
	if !user.IsActive {
		logger.Debug("⚠️ User %d (%s) skipped: not active", user.ID, user.Username)
		return
	}

	if !user.CanReceiveNotifications() {
		logger.Debug("⚠️ User %d (%s) skipped: notifications disabled", user.ID, user.Username)
		return
	}

	if signalType == "growth" && !user.CanReceiveGrowthSignals() {
		logger.Debug("⚠️ User %d (%s) skipped: growth signals disabled", user.ID, user.Username)
		return
	}

	if signalType == "fall" && !user.CanReceiveFallSignals() {
		logger.Debug("⚠️ User %d (%s) skipped: fall signals disabled", user.ID, user.Username)
		return
	}

	// Проверка порогов с учетом знака changePercent
	if signalType == "growth" && changePercent < user.MinGrowthThreshold {
		logger.Debug("⚠️ User %d (%s) skipped: growth threshold not met (%.2f%% < %.1f%%)",
			user.ID, user.Username, changePercent, user.MinGrowthThreshold)
		return
	}

	if signalType == "fall" && math.Abs(changePercent) < user.MinFallThreshold {
		logger.Debug("⚠️ User %d (%s) skipped: fall threshold not met (%.2f%% < %.1f%%)",
			user.ID, user.Username, math.Abs(changePercent), user.MinFallThreshold)
		return
	}

	if user.IsInQuietHours() {
		logger.Debug("⚠️ User %d (%s) skipped: in quiet hours (%d-%d)",
			user.ID, user.Username, user.QuietHoursStart, user.QuietHoursEnd)
		return
	}

	if user.HasReachedDailyLimit() {
		logger.Debug("⚠️ User %d (%s) skipped: daily limit reached (%d/%d)",
			user.ID, user.Username, user.SignalsToday, user.MaxSignalsPerDay)
		return
	}

	// Если все проверки прошли, но ShouldReceiveSignal вернул false, логируем общую причину
	logger.Debug("⚠️ User %d (%s) skipped by ShouldReceiveSignal (type: %s, change: %.2f%%)",
		user.ID, user.Username, signalType, changePercent)
}

// applyUserFilters применяет фильтры пользователя к данным счетчика
func (s *serviceImpl) applyUserFilters(user *models.User, data RawCounterData) bool {
	if user == nil {
		return false
	}

	// Проверяем минимальный объем (если настроено) - используем Volume24h
	if user.MinVolumeFilter > 0 && data.Volume24h < user.MinVolumeFilter {
		logger.Debug("⚠️ User %d (%s) skipped: volume filter (%.0f < %.0f)",
			user.ID, user.Username, data.Volume24h, user.MinVolumeFilter)
		return false
	}

	// Проверяем исключенные паттерны (если настроены)
	if len(user.ExcludePatterns) > 0 {
		for _, pattern := range user.ExcludePatterns {
			// Простая проверка на вхождение подстроки в символ
			// TODO: Реализовать более сложную логику сопоставления
			if pattern != "" && containsString(data.Symbol, pattern) {
				logger.Debug("⚠️ User %d (%s) skipped: excluded pattern '%s' in symbol '%s'",
					user.ID, user.Username, pattern, data.Symbol)
				return false
			}
		}
	}

	// Проверяем предпочтительные периоды (если настроены)
	if len(user.PreferredPeriods) > 0 {
		// Преобразуем Period из string в int для сравнения
		periodInt, err := convertPeriodToInt(data.Period)
		if err != nil {
			logger.Debug("⚠️ User %d (%s) skipped: invalid period format '%s'",
				user.ID, user.Username, data.Period)
			return false
		}

		periodMatch := false
		for _, period := range user.PreferredPeriods {
			if periodInt == period {
				periodMatch = true
				break
			}
		}
		if !periodMatch {
			logger.Debug("⚠️ User %d (%s) skipped: period %s (%d) not in preferred periods",
				user.ID, user.Username, data.Period, periodInt)
			return false
		}
	}

	return true
}

// convertPeriodToInt преобразует период из string в int
func convertPeriodToInt(periodStr string) (int, error) {
	switch periodStr {
	case "5m":
		return 5, nil
	case "15m":
		return 15, nil
	case "30m":
		return 30, nil
	case "1h":
		return 60, nil // 1 час = 60 минут
	case "4h":
		return 240, nil // 4 часа = 240 минут
	case "1d":
		return 1440, nil // 1 день = 1440 минут
	default:
		// Пробуем распарсить как число
		var minutes int
		_, err := fmt.Sscanf(periodStr, "%dm", &minutes)
		if err == nil {
			return minutes, nil
		}
		return 0, fmt.Errorf("неизвестный формат периода: %s", periodStr)
	}
}

// containsString проверяет наличие подстроки в строке (вспомогательная функция)
func containsString(str, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(str) == 0 {
		return false
	}
	// Простая проверка на вхождение (можно заменить на regexp при необходимости)
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// filterByUserSettings применяет все настройки пользователя к данным
func (s *serviceImpl) filterByUserSettings(user *models.User, data RawCounterData) bool {
	// Применяем базовые фильтры
	if !s.shouldSendToUser(user, data) {
		return false
	}

	// Применяем дополнительные фильтры
	if !s.applyUserFilters(user, data) {
		return false
	}

	return true
}
