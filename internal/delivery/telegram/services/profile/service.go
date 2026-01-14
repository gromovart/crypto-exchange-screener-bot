// internal/delivery/telegram/services/profile/service.go
package profile

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"crypto-exchange-screener-bot/internal/core/domain/subscription"
	"crypto-exchange-screener-bot/internal/core/domain/users"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
)

// Service интерфейс профиля
type Service interface {
	Exec(params interface{}) (interface{}, error)
}

// serviceImpl реализация ProfileService
type serviceImpl struct {
	userService         *users.Service
	subscriptionService *subscription.Service
}

// NewService создает новый сервис профиля
func NewService(userService *users.Service, subscriptionService *subscription.Service) Service {
	return &serviceImpl{
		userService:         userService,
		subscriptionService: subscriptionService,
	}
}

// ProfileParams параметры для Exec
type ProfileParams struct {
	UserID int64  `json:"user_id"`
	Action string `json:"action,omitempty"` // "get", "stats"
}

// ProfileResult результат Exec
type ProfileResult struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Exec выполняет обработку запроса профиля
func (s *serviceImpl) Exec(params interface{}) (interface{}, error) {
	// Приводим параметры к нужному типу
	parsedParams, ok := params.(ProfileParams)
	if !ok {
		return ProfileResult{Success: false},
			fmt.Errorf("неверный тип параметров: ожидается ProfileParams")
	}

	// Обрабатываем действие
	switch parsedParams.Action {
	case "":
		fallthrough
	case "get":
		return s.getProfile(parsedParams.UserID)
	case "stats":
		return s.getProfileStats(parsedParams.UserID)
	default:
		return ProfileResult{Success: false},
			fmt.Errorf("неподдерживаемое действие: %s", parsedParams.Action)
	}
}

// getProfile получает профиль пользователя
func (s *serviceImpl) getProfile(userID int64) (ProfileResult, error) {
	// 1. Получаем данные пользователя из ядра
	user, err := s.userService.GetUserByID(int(userID))
	if err != nil {
		return ProfileResult{Success: false},
			fmt.Errorf("ошибка получения пользователя: %w", err)
	}

	// 2. Получаем подписку пользователя (если есть)
	var userSubscription *models.UserSubscription
	userSubscription, err = s.subscriptionService.GetUserSubscription(int(userID))
	if err != nil {
		// Может быть у пользователя нет подписки
		userSubscription = nil
	}

	// 3. Получаем информацию о плане подписки
	var planName, planCode string
	var expiresAt time.Time
	isActive := false

	if userSubscription != nil {
		// Проверяем статус подписки
		isActive = s.isSubscriptionActive(userSubscription)

		// Получаем информацию о плане
		if plan, err := s.subscriptionService.GetPlan(strconv.Itoa(userSubscription.PlanID)); err == nil && plan != nil {
			planName = plan.Name
			planCode = plan.Code
		}

		// Получаем дату окончания
		if userSubscription.CurrentPeriodEnd != nil {
			expiresAt = *userSubscription.CurrentPeriodEnd
		}
	}

	// 4. Форматируем сообщение для Telegram
	message := s.formatProfileMessage(user, planName, planCode, isActive, expiresAt)

	return ProfileResult{
		Success: true,
		Message: message,
		Data: map[string]interface{}{
			"user": map[string]interface{}{
				"id":                    user.ID,
				"telegram_id":           user.TelegramID,
				"username":              user.Username,
				"first_name":            user.FirstName,
				"last_name":             user.LastName,
				"is_active":             user.IsActive,
				"role":                  user.Role,
				"subscription_tier":     user.SubscriptionTier,
				"notifications_enabled": user.NotificationsEnabled,
				"notify_growth":         user.NotifyGrowth,
				"notify_fall":           user.NotifyFall,
				"min_growth_threshold":  user.MinGrowthThreshold,
				"min_fall_threshold":    user.MinFallThreshold,
				"quiet_hours_start":     user.QuietHoursStart,
				"quiet_hours_end":       user.QuietHoursEnd,
				"signals_today":         user.SignalsToday,
				"max_signals_per_day":   user.MaxSignalsPerDay,
				"created_at":            user.CreatedAt,
				"last_login_at":         user.LastLoginAt,
			},
			"subscription": map[string]interface{}{
				"has_subscription": userSubscription != nil,
				"plan_name":        planName,
				"plan_code":        planCode,
				"is_active":        isActive,
				"status":           safeStatus(userSubscription),
				"expires_at":       expiresAt,
				"plan_id":          safePlanID(userSubscription),
			},
		},
	}, nil
}

// getProfileStats получает статистику профиля
func (s *serviceImpl) getProfileStats(userID int64) (ProfileResult, error) {
	// 1. Получаем статистику пользователя из ядра
	stats, err := s.userService.GetUserStats(int(userID))
	if err != nil {
		return ProfileResult{Success: false},
			fmt.Errorf("ошибка получения статистики: %w", err)
	}

	// 2. Получаем пользователя для дополнительной информации
	user, err := s.userService.GetUserByID(int(userID))
	if err != nil {
		return ProfileResult{Success: false},
			fmt.Errorf("ошибка получения пользователя: %w", err)
	}

	// 3. Форматируем статистику для Telegram
	message := s.formatStatsMessage(user, stats)

	return ProfileResult{
		Success: true,
		Message: message,
		Data:    stats,
	}, nil
}

// formatProfileMessage форматирует сообщение профиля для Telegram
func (s *serviceImpl) formatProfileMessage(
	user *models.User,
	planName, planCode string,
	isActive bool,
	expiresAt time.Time,
) string {
	var sb strings.Builder

	sb.WriteString("👤 *Ваш профиль*\n\n")
	sb.WriteString(fmt.Sprintf("🆔 ID: `%d`\n", user.ID))

	if user.Username != "" {
		sb.WriteString(fmt.Sprintf("📛 Имя: @%s\n", user.Username))
	}

	if user.FirstName != "" {
		sb.WriteString(fmt.Sprintf("👋 Имя: %s\n", user.FirstName))
	}

	if user.LastName != "" {
		sb.WriteString(fmt.Sprintf("👔 Фамилия: %s\n", user.LastName))
	}

	// Форматируем даты
	createdAtStr := user.CreatedAt.Format("02.01.2006")
	sb.WriteString(fmt.Sprintf("📅 Регистрация: %s\n", createdAtStr))

	if !user.LastLoginAt.IsZero() {
		lastLoginStr := user.LastLoginAt.Format("02.01.2006 15:04")
		sb.WriteString(fmt.Sprintf("🕐 Последний вход: %s\n", lastLoginStr))
	}

	sb.WriteString("\n")

	// Информация о подписке
	sb.WriteString("💎 *Подписка*\n")
	if isActive && planName != "" {
		sb.WriteString(fmt.Sprintf("План: *%s* (%s)\n", planName, planCode))

		if !expiresAt.IsZero() {
			expiresStr := expiresAt.Format("02.01.2006 15:04")
			sb.WriteString(fmt.Sprintf("Действует до: %s\n", expiresStr))
		}
	} else {
		sb.WriteString("Статус: ❌ *Неактивна*\n")
		sb.WriteString("Используется бесплатный план\n")
	}

	// Настройки уведомлений
	sb.WriteString("\n🔔 *Настройки уведомлений*\n")
	if user.NotificationsEnabled {
		sb.WriteString("Статус: ✅ Включены\n")

		notifications := []string{}
		if user.NotifyGrowth {
			notifications = append(notifications, "📈 Рост")
		}
		if user.NotifyFall {
			notifications = append(notifications, "📉 Падение")
		}
		if user.NotifyContinuous {
			notifications = append(notifications, "🔄 Непрерывные")
		}

		if len(notifications) > 0 {
			sb.WriteString("Типы: " + strings.Join(notifications, ", ") + "\n")
		}

		sb.WriteString(fmt.Sprintf("Порог роста: %.1f%%\n", user.MinGrowthThreshold))
		sb.WriteString(fmt.Sprintf("Порог падения: %.1f%%\n", user.MinFallThreshold))

		if user.QuietHoursStart > 0 || user.QuietHoursEnd > 0 {
			sb.WriteString(fmt.Sprintf("Тихие часы: %02d:00 - %02d:00\n",
				user.QuietHoursStart, user.QuietHoursEnd))
		}
	} else {
		sb.WriteString("Статус: ❌ Выключены\n")
	}

	// Проверяем ограничения длины Telegram
	message := sb.String()
	if len(message) > 4096 {
		// Обрезаем до максимальной длины Telegram
		message = message[:4090] + "..."
	}

	return message
}

// formatStatsMessage форматирует сообщение статистики для Telegram
func (s *serviceImpl) formatStatsMessage(user *models.User, stats map[string]interface{}) string {
	var sb strings.Builder

	sb.WriteString("📊 *Статистика профиля*\n\n")

	// Основная информация
	sb.WriteString(fmt.Sprintf("👤 Пользователь: @%s\n", user.Username))
	sb.WriteString(fmt.Sprintf("🆔 ID: `%d`\n", user.ID))

	// Статистика использования
	if signalsToday, ok := stats["signals_today"].(int); ok {
		sb.WriteString(fmt.Sprintf("📈 Сигналов сегодня: %d\n", signalsToday))
	}

	if maxSignals, ok := stats["max_signals_per_day"].(int); ok {
		sb.WriteString(fmt.Sprintf("🎯 Лимит в день: %d\n", maxSignals))
	}

	// Сессии
	if sessionsData, ok := stats["sessions"].(map[string]interface{}); ok {
		if active, ok := sessionsData["active_sessions"].(int); ok {
			sb.WriteString(fmt.Sprintf("🔐 Активных сессий: %d\n", active))
		}
		if total, ok := sessionsData["total_sessions"].(int); ok {
			sb.WriteString(fmt.Sprintf("📝 Всего сессий: %d\n", total))
		}
	}

	// Активность
	if activityData, ok := stats["activity"].(map[string]interface{}); ok {
		if lastActivity, ok := activityData["last_activity"].(string); ok {
			sb.WriteString(fmt.Sprintf("🕐 Последняя активность: %s\n", lastActivity))
		}
	}

	// Время в системе
	daysInSystem := int(time.Since(user.CreatedAt).Hours() / 24)
	sb.WriteString(fmt.Sprintf("⏰ В системе: %d дней\n", daysInSystem))

	// Уведомления
	sb.WriteString(fmt.Sprintf("🔔 Уведомления: "))
	if user.NotificationsEnabled {
		sb.WriteString("✅ Включены\n")
	} else {
		sb.WriteString("❌ Выключены\n")
	}

	// Проверяем ограничения длины
	message := sb.String()
	if len(message) > 4096 {
		message = message[:4090] + "..."
	}

	return message
}

// Helper функция для проверки активной подписки
func (s *serviceImpl) isSubscriptionActive(subscription *models.UserSubscription) bool {
	if subscription == nil {
		return false
	}

	// Проверяем статус
	if subscription.Status != models.StatusActive && subscription.Status != models.StatusTrialing {
		return false
	}

	// Проверяем, не истекла ли подписка
	if subscription.CurrentPeriodEnd != nil && subscription.CurrentPeriodEnd.Before(time.Now()) {
		return false
	}

	return true
}

// safeStatus безопасно получает статус подписки
func safeStatus(subscription *models.UserSubscription) string {
	if subscription == nil {
		return "no_subscription"
	}
	return subscription.Status
}

// safePlanID безопасно получает ID плана
func safePlanID(subscription *models.UserSubscription) int {
	if subscription == nil {
		return 0
	}
	return subscription.PlanID
}
