// internal/delivery/telegram/services/counter/filters.go
package counter

import (
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
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

	// Базовые проверки из модели User
	if !user.CanReceiveNotifications() {
		logger.Debug("⚠️ User %d (%s) skipped: notifications disabled", user.ID, user.Username)
		return false
	}

	// Проверяем тип сигнала
	if data.Direction == "growth" && !user.CanReceiveGrowthSignals() {
		logger.Debug("⚠️ User %d (%s) skipped: growth signals disabled", user.ID, user.Username)
		return false
	}

	if data.Direction == "fall" && !user.CanReceiveFallSignals() {
		logger.Debug("⚠️ User %d (%s) skipped: fall signals disabled", user.ID, user.Username)
		return false
	}

	// Проверяем тихие часы
	if user.IsInQuietHours() {
		logger.Debug("⚠️ User %d (%s) skipped: in quiet hours (%d-%d)",
			user.ID, user.Username, user.QuietHoursStart, user.QuietHoursEnd)
		return false
	}

	// Проверяем лимиты
	if user.HasReachedDailyLimit() {
		logger.Debug("⚠️ User %d (%s) skipped: daily limit reached (%d/%d)",
			user.ID, user.Username, user.SignalsToday, user.MaxSignalsPerDay)
		return false
	}

	// ПРАВИЛЬНАЯ ПРОВЕРКА ПОРОГОВ: используем data.ChangePercent
	// Используем абсолютное значение изменения
	changePercent := data.ChangePercent
	if changePercent < 0 {
		changePercent = -changePercent // берем модуль для сравнения
	}

	// Проверяем пороги роста
	if data.Direction == "growth" {
		if changePercent < user.MinGrowthThreshold {
			logger.Debug("⚠️ User %d (%s) skipped: growth threshold not met (%.2f%% < %.1f%%)",
				user.ID, user.Username, changePercent, user.MinGrowthThreshold)
			return false
		}
	}

	// Проверяем пороги падения
	if data.Direction == "fall" {
		if changePercent < user.MinFallThreshold {
			logger.Debug("⚠️ User %d (%s) skipped: fall threshold not met (%.2f%% < %.1f%%)",
				user.ID, user.Username, changePercent, user.MinFallThreshold)
			return false
		}
	}

	logger.Debug("✅ User %d (%s) passed all checks", user.ID, user.Username)
	return true
}
