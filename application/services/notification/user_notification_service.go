// application/services/notification/user_notification_service.go
package notification

import (
	"crypto-exchange-screener-bot/internal/core/domain/users"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
	"crypto-exchange-screener-bot/internal/types"
	"fmt"
	"log"
	"sync"
)

// UserNotificationService - сервис для создания уведомлений пользователям
type UserNotificationService struct {
	userService *users.Service
	eventBus    types.EventBus
	mu          sync.RWMutex
	enabled     bool
}

// HandleEvent - реализация интерфейса types.EventSubscriber 🔴 ДОБАВЛЕНО
func (uns *UserNotificationService) HandleEvent(event types.Event) error {
	if !uns.enabled || uns.userService == nil {
		return nil
	}

	switch event.Type {
	case types.EventCounterSignalDetected:
		return uns.HandleCounterSignal(event)
	default:
		return nil
	}
}

// NewUserNotificationService создает новый сервис
func NewUserNotificationService(
	userService *users.Service,
	eventBus types.EventBus,
) *UserNotificationService {
	return &UserNotificationService{
		userService: userService,
		eventBus:    eventBus,
		enabled:     true,
	}
}

// HandleCounterSignal обрабатывает событие счетчика
func (uns *UserNotificationService) HandleCounterSignal(event types.Event) error {
	if !uns.enabled || uns.userService == nil {
		return nil
	}

	// Извлекаем данные из события
	data, ok := event.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid event data format")
	}

	// Извлекаем данные
	symbol, _ := data["symbol"].(string)
	direction, _ := data["direction"].(string)
	count, _ := data["signal_count"].(int)
	maxSignals, _ := data["max_signals"].(int)

	if symbol == "" {
		return fmt.Errorf("symbol not specified in event")
	}

	log.Printf("📨 UserNotificationService: Processing counter signal for %s", symbol)

	// Получаем пользователей
	allUsers, err := uns.userService.GetAllUsers(1000, 0)
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	log.Printf("📊 UserNotificationService: Found %d total users", len(allUsers))

	// Создаем события для каждого подходящего пользователя
	eventCount := 0
	skippedCount := 0

	for _, user := range allUsers {
		if uns.shouldSendToUser(user, direction, count, maxSignals) {
			uns.createUserNotificationEvent(user, event, data)
			eventCount++
		} else {
			skippedCount++
			uns.logUserSkipped(user, direction, count, maxSignals)
		}
	}

	log.Printf("✅ UserNotificationService: Created %d user notification events, skipped %d users",
		eventCount, skippedCount)
	return nil
}

// shouldSendToUser проверяет, нужно ли отправлять пользователю
func (uns *UserNotificationService) shouldSendToUser(user *models.User, direction string, count, maxSignals int) bool {
	if user == nil {
		log.Printf("⚠️ User is nil")
		return false
	}

	// Проверяем ChatID - ДОЛЖЕН БЫТЬ НЕ ПУСТЫМ СТРОКОЙ
	if user.ChatID == "" {
		log.Printf("⚠️ User %d (%s) skipped: empty chat_id", user.ID, user.Username)
		return false
	}

	// Проверяем активность
	if !user.IsActive {
		log.Printf("⚠️ User %d (%s) skipped: not active", user.ID, user.Username)
		return false
	}

	// Базовые проверки из модели User
	if !user.CanReceiveNotifications() {
		log.Printf("⚠️ User %d (%s) skipped: notifications disabled", user.ID, user.Username)
		return false
	}

	// Проверяем тип сигнала
	if direction == "growth" && !user.CanReceiveGrowthSignals() {
		log.Printf("⚠️ User %d (%s) skipped: growth signals disabled", user.ID, user.Username)
		return false
	}
	if direction == "fall" && !user.CanReceiveFallSignals() {
		log.Printf("⚠️ User %d (%s) skipped: fall signals disabled", user.ID, user.Username)
		return false
	}

	// Проверяем тихие часы
	if user.IsInQuietHours() {
		log.Printf("⚠️ User %d (%s) skipped: in quiet hours (%d-%d)",
			user.ID, user.Username, user.QuietHoursStart, user.QuietHoursEnd)
		return false
	}

	// Проверяем лимиты
	if user.HasReachedDailyLimit() {
		log.Printf("⚠️ User %d (%s) skipped: daily limit reached (%d/%d)",
			user.ID, user.Username, user.SignalsToday, user.MaxSignalsPerDay)
		return false
	}

	// Проверяем пороги
	fillPercentage := float64(count) / float64(maxSignals) * 100
	if direction == "growth" && fillPercentage < user.MinGrowthThreshold {
		log.Printf("⚠️ User %d (%s) skipped: growth threshold not met (%.1f%% < %.1f%%)",
			user.ID, user.Username, fillPercentage, user.MinGrowthThreshold)
		return false
	}
	if direction == "fall" && fillPercentage < user.MinFallThreshold {
		log.Printf("⚠️ User %d (%s) skipped: fall threshold not met (%.1f%% < %.1f%%)",
			user.ID, user.Username, fillPercentage, user.MinFallThreshold)
		return false
	}

	log.Printf("✅ User %d (%s) passed all checks", user.ID, user.Username)
	return true
}

// logUserSkipped логирует причину пропуска пользователя
func (uns *UserNotificationService) logUserSkipped(user *models.User, direction string, count, maxSignals int) {
	// Уже логируется в shouldSendToUser
}

// createUserNotificationEvent создает событие для конкретного пользователя
func (uns *UserNotificationService) createUserNotificationEvent(user *models.User, originalEvent types.Event, data map[string]interface{}) {
	// Создаем копию данных с информацией пользователя
	userData := make(map[string]interface{})
	for k, v := range data {
		userData[k] = v
	}

	// Добавляем информацию пользователя
	userData["user_id"] = user.ID
	userData["chat_id"] = user.ChatID
	userData["username"] = user.Username

	// Создаем событие
	userEvent := types.Event{
		Type:      types.EventUserNotification,
		Source:    "user_notification_service",
		Timestamp: originalEvent.Timestamp,
		Data:      userData,
	}

	log.Printf("📤 Creating user notification event for %s (chat_id: %s)",
		user.Username, user.ChatID)

	// Публикуем в EventBus
	go func() {
		if err := uns.eventBus.Publish(userEvent); err != nil {
			log.Printf("❌ Failed to publish user notification event: %v", err)
		} else {
			log.Printf("✅ Published user notification event for %s", user.Username)
		}
	}()
}

// GetSubscribedEvents возвращает типы событий для подписки
func (uns *UserNotificationService) GetSubscribedEvents() []types.EventType {
	return []types.EventType{
		types.EventCounterSignalDetected,
	}
}

// GetName возвращает имя сервиса
func (uns *UserNotificationService) GetName() string {
	return "user_notification_service"
}

// SetEnabled включает/выключает сервис
func (uns *UserNotificationService) SetEnabled(enabled bool) {
	uns.mu.Lock()
	uns.enabled = enabled
	uns.mu.Unlock()
}

// IsEnabled возвращает статус сервиса
func (uns *UserNotificationService) IsEnabled() bool {
	uns.mu.RLock()
	defer uns.mu.RUnlock()
	return uns.enabled
}
