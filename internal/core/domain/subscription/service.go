// internal/core/domain/subscription/service.go
package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"crypto-exchange-screener-bot/internal/infrastructure/cache/redis"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
	subscription_repo "crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/repository/subscription"

	"github.com/jmoiron/sqlx"
)

// Config конфигурация сервиса
type Config struct {
	StripeSecretKey  string
	StripeWebhookKey string
	DefaultPlan      string
	TrialPeriodDays  int
	GracePeriodDays  int
	AutoRenew        bool
}

// NotificationService интерфейс для уведомлений
type NotificationService interface {
	SendSubscriptionNotification(userID int, message, notificationType string) error
	SendTelegramNotification(chatID, message string) error
}

// AnalyticsService интерфейс для аналитики
type AnalyticsService interface {
	TrackSubscriptionEvent(event models.SubscriptionEvent)
}

// Service сервис управления подписками
type Service struct {
	repo        subscription_repo.SubscriptionRepository
	cache       *redis.Cache
	cachePrefix string
	cacheTTL    time.Duration
	plans       map[string]*models.Plan
	mu          sync.RWMutex
	notifier    NotificationService
	analytics   AnalyticsService
}

// NewService создает новый сервис подписок
func NewService(
	db *sqlx.DB,
	cache *redis.Cache,
	notifier NotificationService,
	analytics AnalyticsService,
	config Config,
) (*Service, error) {

	repo := subscription_repo.NewSubscriptionRepository(db, cache)
	service := &Service{
		repo:        repo,
		cache:       cache,
		cachePrefix: "subscription:",
		cacheTTL:    30 * time.Minute,
		plans:       make(map[string]*models.Plan),
		notifier:    notifier,
		analytics:   analytics,
	}

	// Загружаем планы в память
	if err := service.loadPlans(); err != nil {
		return nil, fmt.Errorf("failed to load plans: %w", err)
	}

	// Запускаем планировщик проверки подписок
	go service.startSubscriptionChecker()

	log.Println("✅ Subscription service initialized")
	return service, nil
}

// loadPlans загружает тарифные планы в память
func (s *Service) loadPlans() error {
	plans, err := s.repo.GetAllPlans()
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, plan := range plans {
		s.plans[plan.Code] = plan
		log.Printf("📋 Loaded plan: %s (%s)", plan.Name, plan.Code)
	}

	return nil
}

// GetPlan возвращает план по коду
func (s *Service) GetPlan(code string) (*models.Plan, error) {
	s.mu.RLock()
	plan, exists := s.plans[code]
	s.mu.RUnlock()

	if !exists {
		// Пробуем загрузить из БД
		dbPlan, err := s.repo.GetPlanByCode(code)
		if err != nil {
			return nil, err
		}
		if dbPlan == nil {
			return nil, fmt.Errorf("plan not found: %s", code)
		}

		s.mu.Lock()
		s.plans[code] = dbPlan
		s.mu.Unlock()

		return dbPlan, nil
	}

	return plan, nil
}

// GetAllPlans возвращает все доступные планы
func (s *Service) GetAllPlans() ([]*models.Plan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*models.Plan
	for _, plan := range s.plans {
		if plan.IsActive {
			result = append(result, plan)
		}
	}

	return result, nil
}

// SubscribeUser создает подписку для пользователя
func (s *Service) SubscribeUser(userID int, planCode string, trial bool) (*models.UserSubscription, error) {
	// Проверяем существующую подписку
	existing, err := s.repo.GetActiveSubscription(userID)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check existing subscription: %w", err)
	}

	// Если уже есть активная подписку
	if existing != nil {
		// Обновляем существующую подписку
		return s.upgradeSubscription(userID, planCode, existing)
	}

	// Получаем план
	plan, err := s.GetPlan(planCode)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}

	// Создаем подписку
	now := time.Now()
	periodEnd := now.AddDate(0, 1, 0) // 1 месяц по умолчанию

	// Для пробного периода
	if trial {
		periodEnd = now.AddDate(0, 0, 7) // 7 дней пробного периода
	}

	stripeSubscriptionID := fmt.Sprintf("local_%d_%s", userID, planCode)
	subscription := &models.UserSubscription{
		UserID:               userID,
		PlanID:               plan.ID,
		PlanName:             plan.Name,
		PlanCode:             plan.Code,
		StripeSubscriptionID: &stripeSubscriptionID,
		Status:               models.StatusActive,
		CurrentPeriodStart:   &now,
		CurrentPeriodEnd:     &periodEnd,
		CancelAtPeriodEnd:    false,
		Metadata: map[string]interface{}{
			"trial":          trial,
			"trial_ends_at":  periodEnd.Format(time.RFC3339),
			"auto_renew":     true,
			"payment_method": "manual",
		},
	}

	// Сохраняем в БД
	if err := s.repo.CreateSubscription(subscription); err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	// Обновляем тариф пользователя
	if err := s.repo.UpdateUserSubscriptionTier(userID, planCode); err != nil {
		return nil, fmt.Errorf("failed to update user tier: %w", err)
	}

	// Отправляем уведомление
	s.sendSubscriptionNotification(userID, plan, trial)

	// Трекаем событие
	s.analytics.TrackSubscriptionEvent(models.SubscriptionEvent{
		Type:           "subscription_created",
		UserID:         userID,
		SubscriptionID: subscription.ID,
		PlanCode:       planCode,
		Status:         models.StatusActive,
		Timestamp:      now,
		Metadata: map[string]interface{}{
			"trial": trial,
		},
	})

	// Кэшируем
	s.cacheSubscription(subscription)

	log.Printf("✅ User %d subscribed to plan %s", userID, planCode)

	return subscription, nil
}

// upgradeSubscription обновляет подписку пользователя
func (s *Service) upgradeSubscription(userID int, newPlanCode string, existing *models.UserSubscription) (*models.UserSubscription, error) {
	newPlan, err := s.GetPlan(newPlanCode)
	if err != nil {
		return nil, fmt.Errorf("failed to get new plan: %w", err)
	}

	// Логируем апгрейд
	oldPlanCode := existing.PlanCode

	// Обновляем подписку
	now := time.Now()
	periodEnd := now.AddDate(0, 1, 0) // Новый период на 1 месяц

	existing.PlanID = newPlan.ID
	existing.PlanName = newPlan.Name
	existing.PlanCode = newPlan.Code
	existing.Status = models.StatusActive
	existing.CurrentPeriodStart = &now
	existing.CurrentPeriodEnd = &periodEnd

	// Обновляем в БД
	err = s.repo.UpdateSubscriptionStatus(
		fmt.Sprintf("%d", existing.ID),
		"",
		models.StatusActive,
		periodEnd,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update subscription: %w", err)
	}

	// Обновляем тариф пользователя
	if err := s.repo.UpdateUserSubscriptionTier(userID, newPlanCode); err != nil {
		return nil, fmt.Errorf("failed to update user tier: %w", err)
	}

	// Отправляем уведомление
	s.sendUpgradeNotification(userID, oldPlanCode, newPlanCode)

	// Трекаем событие
	s.analytics.TrackSubscriptionEvent(models.SubscriptionEvent{
		Type:           "subscription_upgraded",
		UserID:         userID,
		SubscriptionID: existing.ID,
		PlanCode:       newPlanCode,
		OldPlanCode:    oldPlanCode,
		Status:         models.StatusActive,
		Timestamp:      now,
	})

	// Инвалидируем кэш
	s.invalidateSubscriptionCache(userID)

	log.Printf("🔄 User %d upgraded from %s to %s", userID, oldPlanCode, newPlanCode)

	return existing, nil
}

// CancelSubscription отменяет подписку
func (s *Service) CancelSubscription(userID int, cancelAtPeriodEnd bool) error {
	// Получаем активную подписку
	sub, err := s.repo.GetActiveSubscription(userID)
	if err != nil {
		return fmt.Errorf("failed to get active subscription: %w", err)
	}
	if sub == nil {
		return errors.New("no active subscription found")
	}

	// Отменяем подписку
	if err := s.repo.CancelSubscription(userID, cancelAtPeriodEnd); err != nil {
		return fmt.Errorf("failed to cancel subscription: %w", err)
	}

	// Обновляем статус
	newStatus := models.StatusCanceled
	if cancelAtPeriodEnd {
		newStatus = models.StatusActive // Остается активной до конца периода
	}

	// Проверяем, что CurrentPeriodEnd не nil
	if sub.CurrentPeriodEnd == nil {
		return errors.New("subscription has no end date")
	}

	// Обновляем в БД
	err = s.repo.UpdateSubscriptionStatus(
		fmt.Sprintf("%d", sub.ID),
		"",
		newStatus,
		*sub.CurrentPeriodEnd,
	)
	if err != nil {
		return fmt.Errorf("failed to update subscription status: %w", err)
	}

	// Если немедленная отмена, переводим на бесплатный тариф
	if !cancelAtPeriodEnd {
		if err := s.repo.UpdateUserSubscriptionTier(userID, models.PlanFree); err != nil {
			return fmt.Errorf("failed to update user tier: %w", err)
		}
	}

	// Отправляем уведомление
	s.sendCancellationNotification(userID, cancelAtPeriodEnd, *sub.CurrentPeriodEnd)

	// Трекаем событие
	s.analytics.TrackSubscriptionEvent(models.SubscriptionEvent{
		Type:           "subscription_cancelled",
		UserID:         userID,
		SubscriptionID: sub.ID,
		PlanCode:       sub.PlanCode,
		Status:         newStatus,
		Timestamp:      time.Now(),
		Metadata: map[string]interface{}{
			"cancel_at_period_end": cancelAtPeriodEnd,
		},
	})

	// Инвалидируем кэш
	s.invalidateSubscriptionCache(userID)

	log.Printf("⏹️ User %d cancelled subscription (end of period: %v)", userID, cancelAtPeriodEnd)

	return nil
}

// GetUserSubscription возвращает подписку пользователя
func (s *Service) GetUserSubscription(userID int) (*models.UserSubscription, error) {
	// Пробуем получить из кэша
	cacheKey := s.cachePrefix + fmt.Sprintf("user:%d", userID)
	var subscription models.UserSubscription
	if err := s.cache.Get(context.Background(), cacheKey, &subscription); err == nil {
		return &subscription, nil
	}

	// Получаем из репозитория
	subscriptionPtr, err := s.repo.GetActiveSubscription(userID)
	if err != nil {
		return nil, err
	}

	// Кэшируем
	if subscriptionPtr != nil {
		s.cacheSubscription(subscriptionPtr)
	}

	return subscriptionPtr, nil
}

// GetUserLimits возвращает лимиты пользователя
func (s *Service) GetUserLimits(userID int) (*models.PlanLimits, error) {
	// Пробуем получить из кэша
	cacheKey := s.cachePrefix + fmt.Sprintf("limits:%d", userID)
	var limits models.PlanLimits
	if err := s.cache.Get(context.Background(), cacheKey, &limits); err == nil {
		return &limits, nil
	}

	// Получаем подписку
	subscription, err := s.GetUserSubscription(userID)
	if err != nil {
		return nil, err
	}

	var planCode string
	if subscription != nil {
		planCode = subscription.PlanCode
	} else {
		planCode = models.PlanFree
	}

	// Получаем лимиты плана
	plan, err := s.GetPlan(planCode)
	if err != nil {
		return nil, err
	}

	limits = models.PlanLimits{
		MaxSymbols:       plan.MaxSymbols,
		MaxSignalsPerDay: plan.MaxSignalsPerDay,
		Features:         plan.Features,
	}

	// Кэшируем
	if data, err := json.Marshal(limits); err == nil {
		s.cache.Set(context.Background(), cacheKey, string(data), s.cacheTTL)
	}

	return &limits, nil
}

// CheckUserLimit проверяет лимит пользователя
func (s *Service) CheckUserLimit(userID int, limitType string, currentUsage int) (bool, int, error) {
	limits, err := s.GetUserLimits(userID)
	if err != nil {
		return false, 0, err
	}

	var maxLimit int
	switch strings.ToLower(limitType) {
	case "symbols":
		maxLimit = limits.MaxSymbols
	case "signals":
		maxLimit = limits.MaxSignalsPerDay
	case "api_requests":
		// Исправлено: MaxAPIRequests не существует в PlanLimits, используем фиксированное значение
		// В будущем можно добавить это поле в модель
		maxLimit = 1000 // Фиксированное значение для API запросов
	default:
		return false, 0, fmt.Errorf("unknown limit type: %s", limitType)
	}

	// Неограниченный доступ
	if maxLimit == -1 {
		return true, -1, nil
	}

	remaining := maxLimit - currentUsage
	hasAccess := remaining > 0

	return hasAccess, remaining, nil
}

// IsSubscriptionActive проверяет активна ли подписка
func (s *Service) IsSubscriptionActive(userID int) (bool, error) {
	subscription, err := s.GetUserSubscription(userID)
	if err != nil {
		return false, err
	}

	return subscription != nil && subscription.Status == models.StatusActive, nil
}

// GetSubscriptionEndDate возвращает дату окончания подписки
func (s *Service) GetSubscriptionEndDate(userID int) (*time.Time, error) {
	subscription, err := s.GetUserSubscription(userID)
	if err != nil {
		return nil, err
	}

	if subscription == nil || subscription.CurrentPeriodEnd == nil {
		return nil, nil
	}

	return subscription.CurrentPeriodEnd, nil
}

// GetExpiringSubscriptions возвращает подписки, срок действия которых истекает
func (s *Service) GetExpiringSubscriptions(daysBefore int) ([]*models.UserSubscription, error) {
	// В реальной реализации нужно добавить метод в репозиторий
	// query := `
	// SELECT ... FROM user_subscriptions
	// WHERE current_period_end BETWEEN NOW() AND NOW() + INTERVAL '$1 days'
	// AND status = 'active'
	// `

	// Пока возвращаем пустой список
	return []*models.UserSubscription{}, nil
}

// RenewSubscription продлевает подписку
func (s *Service) RenewSubscription(userID int) error {
	subscription, err := s.GetUserSubscription(userID)
	if err != nil {
		return err
	}
	if subscription == nil {
		return errors.New("no active subscription found")
	}

	// Продлеваем на месяц
	newEndDate := time.Now().AddDate(0, 1, 0)

	// Обновляем в БД
	err = s.repo.UpdateSubscriptionStatus(
		fmt.Sprintf("%d", subscription.ID),
		"",
		models.StatusActive,
		newEndDate,
	)
	if err != nil {
		return fmt.Errorf("failed to renew subscription: %w", err)
	}

	// Отправляем уведомление
	s.sendRenewalNotification(userID, newEndDate)

	// Трекаем событие
	s.analytics.TrackSubscriptionEvent(models.SubscriptionEvent{
		Type:           "subscription_renewed",
		UserID:         userID,
		SubscriptionID: subscription.ID,
		PlanCode:       subscription.PlanCode,
		Status:         models.StatusActive,
		Timestamp:      time.Now(),
		Metadata: map[string]interface{}{
			"new_end_date": newEndDate.Format(time.RFC3339),
		},
	})

	// Инвалидируем кэш
	s.invalidateSubscriptionCache(userID)

	log.Printf("🔄 User %d subscription renewed until %s", userID, newEndDate.Format("2006-01-02"))

	return nil
}

// GetRevenueReport возвращает отчет по доходам
func (s *Service) GetRevenueReport(startDate, endDate time.Time) (*models.RevenueReport, error) {
	// В реальной реализации нужно добавить метод в репозиторий
	// Пока возвращаем заглушку
	return &models.RevenueReport{
		PeriodStart:      startDate,
		PeriodEnd:        endDate,
		TotalRevenue:     0,
		NewSubscriptions: 0,
		ARPU:             0,
		MostPopularPlan:  models.PlanFree,
		MonthlyBreakdown: []models.MonthlyBreakdown{},
	}, nil
}

// GetSubscriptionStats возвращает статистику подписок
func (s *Service) GetSubscriptionStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// В реальной реализации нужно получить статистику из БД
	// Пока возвращаем заглушку
	stats["total_subscriptions"] = 0
	stats["active_subscriptions"] = 0
	stats["trial_subscriptions"] = 0
	stats["monthly_revenue"] = 0.0
	stats["churn_rate"] = 0.0
	stats["plan_distribution"] = map[string]int{}

	return stats, nil
}

// Вспомогательные методы

func (s *Service) cacheSubscription(subscription *models.UserSubscription) error {
	data, err := json.Marshal(subscription)
	if err != nil {
		return err
	}

	ctx := context.Background()
	cacheKey := s.cachePrefix + fmt.Sprintf("user:%d", subscription.UserID)
	s.cache.Set(ctx, cacheKey, string(data), s.cacheTTL)

	return nil
}

func (s *Service) invalidateSubscriptionCache(userID int) {
	ctx := context.Background()
	keys := []string{
		s.cachePrefix + fmt.Sprintf("user:%d", userID),
		s.cachePrefix + fmt.Sprintf("limits:%d", userID),
	}

	s.cache.DeleteMulti(ctx, keys...)
}

func (s *Service) sendSubscriptionNotification(userID int, plan *models.Plan, trial bool) {
	var message string

	// Фиксированные значения для API запросов по тарифам
	var apiRequestsStr string
	switch plan.Code {
	case models.PlanFree:
		apiRequestsStr = "100" // Бесплатный тариф
	case models.PlanBasic:
		apiRequestsStr = "1000" // Базовый тариф
	case models.PlanPro:
		apiRequestsStr = "5000" // Про тариф
	default:
		apiRequestsStr = "1000" // По умолчанию
	}

	if trial {
		message = fmt.Sprintf(
			"🎉 Вы успешно подписались на тариф %s!\n\n"+
				"Пробный период: 7 дней\n"+
				"Лимиты:\n"+
				"• Символов: %d\n"+
				"• Сигналов в день: %d\n"+
				"• API запросов: %s\n\n"+
				"После окончания пробного периода подписка будет продлена автоматически.",
			plan.Name,
			plan.MaxSymbols,
			plan.MaxSignalsPerDay,
			apiRequestsStr,
		)
	} else {
		message = fmt.Sprintf(
			"✅ Подписка активирована!\n\n"+
				"Тариф: %s\n"+
				"Лимиты:\n"+
				"• Символов: %d\n"+
				"• Сигналов в день: %d\n"+
				"• API запросов: %s\n\n"+
				"Следующее списание: через 30 дней",
			plan.Name,
			plan.MaxSymbols,
			plan.MaxSignalsPerDay,
			apiRequestsStr,
		)
	}

	s.notifier.SendSubscriptionNotification(userID, message, "subscription_created")
}

func (s *Service) sendUpgradeNotification(userID int, oldPlan, newPlan string) {
	message := fmt.Sprintf(
		"🔄 Тариф изменен!\n\n"+
			"Старый тариф: %s\n"+
			"Новый тариф: %s\n\n"+
			"Изменения вступят в силу немедленно.",
		oldPlan, newPlan,
	)

	s.notifier.SendSubscriptionNotification(userID, message, "subscription_upgraded")
}

func (s *Service) sendCancellationNotification(userID int, atPeriodEnd bool, endDate time.Time) {
	var message string
	if atPeriodEnd {
		message = fmt.Sprintf(
			"⏹️ Подписка будет отменена\n\n"+
				"Ваша подписка останется активной до %s.\n"+
				"После этой даты она будет автоматически отменена.",
			endDate.Format("02.01.2006"),
		)
	} else {
		message = "⏹️ Подписка отменена\n\n" +
			"Ваша подписка была немедленно отменена.\n" +
			"Вы переведены на бесплатный тариф."
	}

	s.notifier.SendSubscriptionNotification(userID, message, "subscription_cancelled")
}

func (s *Service) sendRenewalNotification(userID int, newEndDate time.Time) {
	message := fmt.Sprintf(
		"🔄 Подписка продлена!\n\n"+
			"Ваша подписка успешно продлена.\n"+
			"Следующая дата окончания: %s",
		newEndDate.Format("02.01.2006"),
	)

	s.notifier.SendSubscriptionNotification(userID, message, "subscription_renewed")
}

func (s *Service) sendExpirationNotification(userID int, daysLeft int) {
	message := fmt.Sprintf(
		"⚠️ Подписка скоро истекает\n\n"+
			"До окончания вашей подписки осталось %d дней.\n"+
			"Пожалуйста, продлите подписку, чтобы продолжить пользоваться всеми функциями.",
		daysLeft,
	)

	s.notifier.SendSubscriptionNotification(userID, message, "subscription_expiring")
}

func (s *Service) startSubscriptionChecker() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		s.checkExpiringSubscriptions()
		s.checkExpiredSubscriptions()
	}
}

func (s *Service) checkExpiringSubscriptions() {
	// Проверяем подписки, которые истекают через 3 дня
	subscriptions, err := s.GetExpiringSubscriptions(3)
	if err != nil {
		log.Printf("Error checking expiring subscriptions: %v", err)
		return
	}

	for _, sub := range subscriptions {
		// Отправляем уведомление
		s.sendExpirationNotification(sub.UserID, 3)
	}
}

func (s *Service) checkExpiredSubscriptions() {
	// В реальной реализации нужно найти истекшие подписки
	// и перевести пользователей на бесплатный тариф
}
