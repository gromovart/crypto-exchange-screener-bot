// internal/core/domain/subscription/manager.go
package subscription

import (
	"crypto-exchange-screener-bot/internal/infrastructure/cache/redis"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
	"fmt"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
)

// Manager управляет всеми аспектами подписок
// Manager управляет всеми аспектами подписок
type Manager struct {
	service        *Service
	cache          *redis.Cache // Изменено: *redis.Client -> *redis.Cache
	stats          map[string]interface{}
	statsUpdatedAt time.Time
	mu             sync.RWMutex
}

// NewManager создает новый менеджер подписок
func NewManager(
	db *sqlx.DB,
	cache *redis.Cache, // Изменено: *redis.Client -> *redis.Cache
	notifier NotificationService,
	analytics AnalyticsService,
	config Config,
) (*Manager, error) {

	service, err := NewService(db, cache, notifier, analytics, config)
	if err != nil {
		return nil, err
	}

	manager := &Manager{
		service: service,
		cache:   cache,
		stats:   make(map[string]interface{}),
	}

	// Запуск обновления статистики
	go manager.updateStatsWorker()

	return manager, nil
}

// UpdateStatsWorker периодически обновляет статистику
func (m *Manager) updateStatsWorker() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		stats, err := m.service.GetSubscriptionStats()
		if err == nil {
			m.mu.Lock()
			m.stats = stats
			m.statsUpdatedAt = time.Now()
			m.mu.Unlock()
		}
	}
}

// GetStats возвращает статистику
func (m *Manager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]interface{})
	for k, v := range m.stats {
		stats[k] = v
	}
	stats["updated_at"] = m.statsUpdatedAt

	return stats
}

// HandleWebhook обрабатывает вебхуки от платежной системы
func (m *Manager) HandleWebhook(eventType string, data map[string]interface{}) error {
	switch eventType {
	case "invoice.payment_succeeded":
		return m.handlePaymentSucceeded(data)
	case "customer.subscription.updated":
		return m.handleSubscriptionUpdated(data)
	case "customer.subscription.deleted":
		return m.handleSubscriptionDeleted(data)
	case "invoice.payment_failed":
		return m.handlePaymentFailed(data)
	default:
		return fmt.Errorf("unsupported event type: %s", eventType)
	}
}

// HandlePaymentSucceeded обрабатывает успешный платеж
func (m *Manager) handlePaymentSucceeded(data map[string]interface{}) error {
	// Извлекаем данные из вебхука
	customerID, _ := data["customer"].(string)
	subscriptionID, _ := data["subscription"].(string)
	amount, _ := data["amount_paid"].(float64)

	// TODO: Найти пользователя по customerID
	// TODO: Обновить подписку

	fmt.Printf("Payment succeeded: customer=%s, subscription=%s, amount=%.2f\n",
		customerID, subscriptionID, amount)

	return nil
}

// HandleSubscriptionUpdated обрабатывает обновление подписки
func (m *Manager) handleSubscriptionUpdated(data map[string]interface{}) error {
	// TODO: Реализовать обработку обновления подписки
	fmt.Println("Subscription updated:", data)
	return nil
}

// HandleSubscriptionDeleted обрабатывает удаление подписки
func (m *Manager) handleSubscriptionDeleted(data map[string]interface{}) error {
	// TODO: Реализовать обработку удаления подписки
	fmt.Println("Subscription deleted:", data)
	return nil
}

// HandlePaymentFailed обрабатывает неудачный платеж
func (m *Manager) handlePaymentFailed(data map[string]interface{}) error {
	// TODO: Реализовать обработку неудачного платежа
	fmt.Println("Payment failed:", data)
	return nil
}

// FormatLimit форматирует лимит для отображения
func (m *Manager) FormatLimit(limit int) string {
	if limit == -1 {
		return "неограниченно"
	}
	return fmt.Sprintf("%d", limit)
}

// GetPlans возвращает все доступные планы
func (m *Manager) GetPlans() ([]*models.Plan, error) {
	return m.service.GetAllPlans()
}

// GetPlan возвращает план по коду
func (m *Manager) GetPlan(code string) (*models.Plan, error) {
	return m.service.GetPlan(code)
}

// SubscribeUser создает подписку для пользователя
func (m *Manager) SubscribeUser(userID int, planCode string, trial bool) (*models.UserSubscription, error) {
	return m.service.SubscribeUser(userID, planCode, trial)
}

// GetUserSubscription возвращает подписку пользователя
func (m *Manager) GetUserSubscription(userID int) (*models.UserSubscription, error) {
	return m.service.GetUserSubscription(userID)
}

// CancelSubscription отменяет подписку
func (m *Manager) CancelSubscription(userID int, cancelAtPeriodEnd bool) error {
	return m.service.CancelSubscription(userID, cancelAtPeriodEnd)
}

// GetUserLimits возвращает лимиты пользователя
func (m *Manager) GetUserLimits(userID int) (*models.PlanLimits, error) {
	return m.service.GetUserLimits(userID)
}

// IsSubscriptionActive проверяет активна ли подписка
func (m *Manager) IsSubscriptionActive(userID int) (bool, error) {
	return m.service.IsSubscriptionActive(userID)
}

// CheckUserLimit проверяет лимит пользователя
func (m *Manager) CheckUserLimit(userID int, limitType string, currentUsage int) (bool, int, error) {
	return m.service.CheckUserLimit(userID, limitType, currentUsage)
}
