// internal/delivery/telegram/services/counter/filters.go
package counter

import (
	"context"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
	"crypto-exchange-screener-bot/pkg/logger"
	"crypto-exchange-screener-bot/pkg/period"
	"fmt"
	"math"
)

// Константы для типов сигналов
const (
	SignalTypeGrowth = "growth"
	SignalTypeFall   = "fall"
)

// Константы для фильтрации пользователей
const (
	DefaultUserFetchLimit = 1000
	DefaultUserOffset     = 0
)

// getUsersToNotify возвращает пользователей, которым нужно отправить уведомление
func (s *serviceImpl) getUsersToNotify(data RawCounterData) ([]*models.User, error) {
	if s.userService == nil {
		return nil, fmt.Errorf("сервис пользователей не инициализирован")
	}

	// Получаем всех пользователей
	allUsers, err := s.userService.GetAllUsers(DefaultUserFetchLimit, DefaultUserOffset)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения пользователей: %w", err)
	}

	// Фильтруем пользователей
	filteredUsers := s.filterUsers(allUsers, data)

	logger.Debug("🔍 getUsersToNotify результат: символ=%s, отфильтровано: %d, всего пользователей: %d",
		data.Symbol, len(filteredUsers), len(allUsers))

	return filteredUsers, nil
}

// filterUsers применяет все фильтры к списку пользователей
func (s *serviceImpl) filterUsers(users []*models.User, data RawCounterData) []*models.User {
	var filteredUsers []*models.User

	for _, user := range users {
		if s.shouldSendToUser(user, data) {
			filteredUsers = append(filteredUsers, user)
		}
	}

	return filteredUsers
}

// shouldSendToUser проверяет, нужно ли отправлять пользователю
func (s *serviceImpl) shouldSendToUser(user *models.User, data RawCounterData) bool {
	// Базовые проверки
	if !s.checkBasicConditions(user, data) {
		return false
	}

	// ⭐ ПРОВЕРКА ПОДПИСКИ
	if !s.hasActiveSubscription(user.ID) {
		logger.Debug("🔍 Пропуск user=%d: нет активной подписки", user.ID)
		return false
	}

	// Проверка типа сигнала
	signalType, valid := s.determineSignalType(data)
	if !valid {
		return false
	}

	// Проверка настроек пользователя для этого типа сигнала
	if !s.checkSignalTypeSettings(user, signalType) {
		return false
	}

	// Проверка порогов и лимитов пользователя
	changePercentForCheck := s.calculateChangePercentForCheck(signalType, data.ChangePercent)
	if !s.checkUserThresholds(user, signalType, changePercentForCheck, data) {
		return false
	}

	// Применяем дополнительные фильтры пользователя
	if !s.applyUserFilters(user, data) {
		return false
	}

	logger.Debug("✅ shouldSendToUser ПРОШЕЛ: user=%d (%s) для %s signal (%.2f%%)",
		user.ID, user.Username, signalType, changePercentForCheck)
	return true
}

// hasActiveSubscription проверяет наличие активной подписки
func (s *serviceImpl) hasActiveSubscription(userID int) bool {
	if s.subscriptionService == nil {
		logger.Warn("⚠️ subscriptionService не инициализирован в counter service")
		return true // Если сервис не инициализирован, пропускаем (для обратной совместимости)
	}

	ctx := context.Background()
	sub, err := s.subscriptionService.GetActiveSubscription(ctx, userID)
	if err != nil {
		logger.Warn("⚠️ Ошибка проверки подписки для user %d: %v", userID, err)
		return false
	}

	return sub != nil
}

// checkBasicConditions проверяет базовые условия пользователя
func (s *serviceImpl) checkBasicConditions(user *models.User, data RawCounterData) bool {
	if user == nil {
		return false
	}

	// Пропускаем MAX-only пользователей (они обрабатываются MAX контроллером)
	if user.IsMaxOnlyUser() {
		logger.Debug("🔍 Пропуск user=%d: MAX-only пользователь (обрабатывается MAX контроллером)", user.ID)
		return false
	}

	// Проверяем ChatID
	if user.ChatID == "" {
		logger.Debug("🔍 Пропуск user=%d: пустой chat_id", user.ID)
		return false
	}

	// Проверяем активность
	if !user.IsActive {
		logger.Debug("🔍 Пропуск user=%d: не активен", user.ID)
		return false
	}

	// Проверяем включены ли уведомления
	if !user.NotificationsEnabled {
		logger.Debug("🔍 Пропуск user=%d: уведомления отключены", user.ID)
		return false
	}

	return true
}

// determineSignalType определяет тип сигнала
func (s *serviceImpl) determineSignalType(data RawCounterData) (string, bool) {
	switch data.Direction {
	case SignalTypeGrowth:
		return SignalTypeGrowth, true
	case SignalTypeFall:
		return SignalTypeFall, true
	default:
		logger.Debug("🔍 Неизвестный direction=%s", data.Direction)
		return "", false
	}
}

// checkSignalTypeSettings проверяет настройки пользователя для типа сигнала
func (s *serviceImpl) checkSignalTypeSettings(user *models.User, signalType string) bool {
	switch signalType {
	case SignalTypeGrowth:
		if !user.NotifyGrowth {
			logger.Debug("🔍 Пропуск user=%d: рост отключен", user.ID)
			return false
		}
	case SignalTypeFall:
		if !user.NotifyFall {
			logger.Debug("🔍 Пропуск user=%d: падение отключено", user.ID)
			return false
		}
	}
	return true
}

// calculateChangePercentForCheck рассчитывает процент изменения для проверки
func (s *serviceImpl) calculateChangePercentForCheck(signalType string, changePercent float64) float64 {
	if signalType == SignalTypeFall {
		return -changePercent
	}
	return changePercent
}

// checkUserThresholds проверяет пороги и лимиты пользователя
func (s *serviceImpl) checkUserThresholds(user *models.User, signalType string, changePercent float64, data RawCounterData) bool {
	shouldReceive := user.ShouldReceiveSignal(signalType, changePercent)

	if !shouldReceive {
		s.logUserSkipReason(user, signalType, changePercent, data)
		return false
	}

	return true
}

// logUserSkipReason логирует причину пропуска пользователя (только для отладки, но без спама)
func (s *serviceImpl) logUserSkipReason(user *models.User, signalType string, changePercent float64, data RawCounterData) {
	// ⭐ Логируем только для важных случаев (большие изменения, но не достигшие порога)
	absChange := math.Abs(changePercent)

	// Логируем если изменение больше 0.5% но не достигло порога (это интересно)
	if absChange > 0.5 {
		if signalType == "growth" && changePercent < user.MinGrowthThreshold {
			logger.Debug("🔍 Пропуск user=%d: рост %.2f%% < порога %.2f%%",
				user.ID, changePercent, user.MinGrowthThreshold)
		} else if signalType == "fall" && absChange < user.MinFallThreshold {
			logger.Debug("🔍 Пропуск user=%d: падение |%.2f|%% < порога %.2f%%",
				user.ID, absChange, user.MinFallThreshold)
		}
	}
}

// applyUserFilters применяет фильтры пользователя к данным счетчика
func (s *serviceImpl) applyUserFilters(user *models.User, data RawCounterData) bool {
	if user == nil {
		return false
	}

	// Проверяем минимальный объем
	if user.MinVolumeFilter > 0 && data.Volume24h < user.MinVolumeFilter {
		logger.Debug("⚠️ User %d (%s) пропущен: фильтр объема (%.0f < %.0f)",
			user.ID, user.Username, data.Volume24h, user.MinVolumeFilter)
		return false
	}

	// Проверяем исключенные паттерны
	if len(user.ExcludePatterns) > 0 {
		for _, pattern := range user.ExcludePatterns {
			if pattern != "" && ContainsString(data.Symbol, pattern) {
				logger.Debug("⚠️ User %d (%s) пропущен: исключенный паттерн '%s' в символе '%s'",
					user.ID, user.Username, pattern, data.Symbol)
				return false
			}
		}
	}

	// Проверяем вотчлист (если задан — отправляем только символы из списка)
	if user.HasWatchlist() && !user.ShouldTrackSymbol(data.Symbol) {
		logger.Debug("⚠️ User %d (%s) пропущен: символ '%s' не в вотчлисте",
			user.ID, user.Username, data.Symbol)
		return false
	}

	// Проверяем предпочтительные периоды
	if len(user.PreferredPeriods) > 0 {
		periodInt, err := period.StringToMinutes(data.Period)
		if err != nil {
			logger.Debug("⚠️ User %d (%s) пропущен: неверный формат периода '%s'",
				user.ID, user.Username, data.Period)
			return false
		}

		if !s.isPeriodPreferred(periodInt, user.PreferredPeriods) {
			return false
		}
	} else {
		// Если у пользователя нет предпочтительных периодов - используем дефолтный 15 минут
		defaultPeriod := 15
		periodInt, err := period.StringToMinutes(data.Period)
		if err != nil {
			return false
		}
		if periodInt != defaultPeriod {
			return false
		}
	}

	return true
}

// isPeriodPreferred проверяет, находится ли период в предпочтительных
func (s *serviceImpl) isPeriodPreferred(periodInt int, preferredPeriods []int) bool {
	for _, period := range preferredPeriods {
		if periodInt == period {
			return true
		}
	}
	return false
}

// filterByUserSettings применяет все настройки пользователя к данным
func (s *serviceImpl) filterByUserSettings(user *models.User, data RawCounterData) bool {
	return s.shouldSendToUser(user, data)
}
