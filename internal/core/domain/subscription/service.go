// internal/core/domain/subscription/service.go
package subscription

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"crypto-exchange-screener-bot/internal/infrastructure/cache/redis"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
	plan_repo "crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/repository/plan"
	subscription_repo "crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/repository/subscription"
	"crypto-exchange-screener-bot/pkg/logger"

	"github.com/jmoiron/sqlx"
)

// Config конфигурация сервиса
type Config struct {
	DefaultPlan     string
	TrialPeriodDays int  // Для free плана
	GracePeriodDays int  // Льготный период после истечения
	AutoRenew       bool // Автопродление (для платных планов)
	IsDev           bool // Только для dev окружения
}

// AnalyticsService интерфейс для аналитики
type AnalyticsService interface {
	TrackSubscriptionEvent(event models.SubscriptionEvent)
}

// Service сервис управления подписками
type Service struct {
	subRepo     subscription_repo.SubscriptionRepository
	planRepo    plan_repo.PlanRepository
	cache       *redis.Cache
	cachePrefix string
	cacheTTL    time.Duration
	plans       map[string]*models.Plan
	mu          sync.RWMutex
	analytics   AnalyticsService
	config      Config
}

// NewService создает новый сервис подписок
func NewService(
	db *sqlx.DB,
	planRepo plan_repo.PlanRepository,
	cache *redis.Cache,
	analytics AnalyticsService,
	config Config,
) (*Service, error) {

	subRepo := subscription_repo.NewSubscriptionRepository(db)
	service := &Service{
		subRepo:     subRepo,
		planRepo:    planRepo,
		cache:       cache,
		cachePrefix: "subscription:",
		cacheTTL:    30 * time.Minute,
		plans:       make(map[string]*models.Plan),
		analytics:   analytics,
		config:      config,
	}

	// Загружаем планы в память
	if err := service.loadPlans(); err != nil {
		return nil, fmt.Errorf("не удалось загрузить планы: %w", err)
	}

	// Запускаем планировщик проверки подписок
	go service.startSubscriptionChecker()

	logger.Info("✅ Сервис подписок инициализирован")
	return service, nil
}

// loadPlans загружает тарифные планы в память
func (s *Service) loadPlans() error {
	ctx := context.Background()
	plans, err := s.planRepo.GetAllActive(ctx)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, plan := range plans {
		// Тестовый план доступен только в dev окружении
		if plan.Code == models.PlanTest && !s.config.IsDev {
			logger.Info("⏭️ Пропускаем тестовый план (не dev окружение)")
			continue
		}
		s.plans[plan.Code] = plan
		logger.Info("📋 Загружен план: %s (%s)", plan.Name, plan.Code)
	}

	return nil
}

// GetPlan возвращает план по коду
func (s *Service) GetPlan(code string) (*models.Plan, error) {
	s.mu.RLock()
	plan, exists := s.plans[code]
	s.mu.RUnlock()

	if !exists {
		// Пробуем загрузить из репозитория
		ctx := context.Background()
		dbPlan, err := s.planRepo.GetByCode(ctx, code)
		if err != nil {
			return nil, fmt.Errorf("ошибка получения плана: %w", err)
		}
		if dbPlan == nil {
			return nil, fmt.Errorf("план не найден: %s", code)
		}

		s.mu.Lock()
		s.plans[code] = dbPlan
		s.mu.Unlock()

		return dbPlan, nil
	}

	return plan, nil
}

// GetSubscriptionPeriod возвращает период подписки в зависимости от плана
func (s *Service) GetSubscriptionPeriod(planCode string) (time.Duration, error) {
	switch planCode {
	case models.PlanFree:
		return 24 * time.Hour, nil // 24 часа для бесплатного
	case "test": // ⭐ ТЕСТОВЫЙ ПЛАН
		return 5 * time.Minute, nil // 5 минут для тестирования
	case models.PlanBasic:
		return 30 * 24 * time.Hour, nil // 1 месяц
	case models.PlanPro:
		return 90 * 24 * time.Hour, nil // 3 месяца
	case models.PlanEnterprise:
		return 365 * 24 * time.Hour, nil // 12 месяцев
	default:
		return 0, fmt.Errorf("неизвестный план: %s", planCode)
	}
}

// CreateSubscription создает подписку для пользователя
func (s *Service) CreateSubscription(ctx context.Context, userID int, planCode string, paymentID *int64, isTrial bool) (*models.UserSubscription, error) {
	if s == nil || s.planRepo == nil {
		return nil, fmt.Errorf("сервис подписок не инициализирован")
	}

	// Получаем ВСЕ подписки пользователя
	allSubscriptions, err := s.subRepo.GetAllByUserID(ctx, userID)
	if err != nil && !errors.Is(err, context.Canceled) {
		return nil, fmt.Errorf("ошибка получения подписок пользователя: %w", err)
	}

	// Проверяем, была ли уже бесплатная пробная подписка
	hadFreeTrial := false
	var activeSubscription *models.UserSubscription

	for _, sub := range allSubscriptions {
		// Проверяем, была ли уже free пробная подписка
		if sub.PlanCode == models.PlanFree {
			if trial, ok := sub.Metadata["trial"].(bool); ok && trial {
				hadFreeTrial = true
			}
		}


		// Проверяем, есть ли активная подписка
		if sub.IsActive() && (activeSubscription == nil || sub.CreatedAt.After(activeSubscription.CreatedAt)) {
			activeSubscription = sub
		}
	}

	// ⭐ Если это пробная free подписка, но пользователь уже имел trial - запрещаем
	if planCode == models.PlanFree && isTrial && hadFreeTrial {
		return nil, fmt.Errorf("бесплатный пробный период уже был использован")
	}

	// ⭐ Тестовая подписка: разрешаем повторную оплату (для тестирования)
	// Существующая активная тестовая подписка будет продлена

	// ⭐ Если есть активная подписка (не тестовая) и мы пытаемся создать другую (не тест) - ошибка
	if activeSubscription != nil && planCode != "test" && activeSubscription.PlanCode != "test" {
		if activeSubscription.PlanCode != planCode {
			return nil, fmt.Errorf("у пользователя уже есть активная подписка %s. Используйте UpgradeSubscription для смены плана",
				activeSubscription.PlanCode)
		}
		return nil, fmt.Errorf("у пользователя уже есть активная подписка на план %s", planCode)
	}

	// Получаем план
	plan, err := s.GetPlan(planCode)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения плана: %w", err)
	}

	// Получаем период подписки
	period, err := s.GetSubscriptionPeriod(planCode)
	if err != nil {
		return nil, err
	}

	// Создаем подписку
	now := time.Now()
	periodEnd := now.Add(period)

	subscription := &models.UserSubscription{
		UserID:             userID,
		PlanID:             plan.ID,
		PaymentID:          paymentID,
		Status:             models.StatusActive,
		CurrentPeriodStart: &now,
		CurrentPeriodEnd:   &periodEnd,
		CancelAtPeriodEnd:  false,
		Metadata: map[string]interface{}{
			"trial":          isTrial,
			"period_days":    int(period.Hours() / 24),
			"auto_renew":     s.config.AutoRenew && !isTrial && planCode != models.PlanFree && planCode != "test",
			"payment_method": "stars",
			"created_at":     now.Format(time.RFC3339),
		},
	}

	// Для free плана добавляем дополнительную метку
	if planCode == models.PlanFree {
		subscription.Metadata["type"] = "trial"
		subscription.Metadata["expires_after_hours"] = 24
	}

	// ⭐ Для тестового плана добавляем метку
	if planCode == "test" {
		subscription.Metadata["type"] = "test"
		subscription.Metadata["expires_after_minutes"] = 5
		subscription.Metadata["test_payment"] = true
	}

	// Сохраняем в БД
	if err := s.subRepo.Create(ctx, subscription); err != nil {
		return nil, fmt.Errorf("ошибка создания подписки: %w", err)
	}

	// Трекаем событие
	if s.analytics != nil {
		s.analytics.TrackSubscriptionEvent(models.SubscriptionEvent{
			Type:           "subscription_created",
			UserID:         userID,
			SubscriptionID: subscription.ID,
			PlanCode:       planCode,
			Status:         models.StatusActive,
			Timestamp:      now,
			Metadata: map[string]interface{}{
				"trial":       isTrial,
				"period_days": int(period.Hours() / 24),
			},
		})
	}

	logMsg := fmt.Sprintf("✅ Создана подписка: пользователь %d, план %s, период %d дней",
		userID, planCode, int(period.Hours()/24))

	if isTrial {
		logMsg += " (пробный период)"
	}
	if planCode == "test" {
		logMsg = fmt.Sprintf("🧪 Создана тестовая подписка: пользователь %d, план %s, период 5 минут", userID, planCode)
	}
	logger.Info(logMsg)

	return subscription, nil
}

// UpgradeSubscription обновляет подписку пользователя на новый план
func (s *Service) UpgradeSubscription(ctx context.Context, userID int, newPlanCode string, paymentID *int64) (*models.UserSubscription, error) {
	// Получаем текущую подписку
	existing, err := s.subRepo.GetActiveByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения текущей подписки: %w", err)
	}
	if existing == nil {
		return nil, errors.New("активная подписка не найдена")
	}

	// Получаем новый план
	newPlan, err := s.GetPlan(newPlanCode)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения нового плана: %w", err)
	}

	// Получаем период нового плана
	period, err := s.GetSubscriptionPeriod(newPlanCode)
	if err != nil {
		return nil, err
	}

	// Логируем апгрейд
	oldPlanCode := existing.PlanCode

	// Обновляем подписку
	now := time.Now()
	periodEnd := now.Add(period)

	existing.PlanID = newPlan.ID
	existing.PlanName = newPlan.Name
	existing.PlanCode = newPlan.Code
	existing.PaymentID = paymentID
	existing.Status = models.StatusActive
	existing.CurrentPeriodStart = &now
	existing.CurrentPeriodEnd = &periodEnd
	existing.CancelAtPeriodEnd = false

	// Обновляем метаданные
	if existing.Metadata == nil {
		existing.Metadata = make(map[string]interface{})
	}
	existing.Metadata["upgraded_at"] = now.Format(time.RFC3339)
	existing.Metadata["previous_plan"] = oldPlanCode
	existing.Metadata["period_days"] = int(period.Hours() / 24)
	existing.Metadata["auto_renew"] = s.config.AutoRenew && newPlanCode != models.PlanFree

	existing.Metadata["new_plan_name"] = newPlan.Name // ⭐ Сохраняем в metadata
	existing.Metadata["new_plan_code"] = newPlan.Code // ⭐ Сохраняем в metadata

	// Обновляем в БД
	if err := s.subRepo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("ошибка обновления подписки: %w", err)
	}

	// Трекаем событие
	if s.analytics != nil {
		s.analytics.TrackSubscriptionEvent(models.SubscriptionEvent{
			Type:           "subscription_upgraded",
			UserID:         userID,
			SubscriptionID: existing.ID,
			PlanCode:       newPlanCode,
			OldPlanCode:    oldPlanCode,
			Status:         models.StatusActive,
			Timestamp:      now,
			Metadata: map[string]interface{}{
				"period_days": int(period.Hours() / 24),
			},
		})
	}

	// Инвалидируем кэш
	s.invalidateSubscriptionCache(userID)

	logger.Info("🔄 Обновлена подписка: пользователь %d, с %s на %s, период %d дней",
		userID, oldPlanCode, newPlanCode, int(period.Hours()/24))

	return existing, nil
}

// CancelSubscription отменяет подписку
func (s *Service) CancelSubscription(ctx context.Context, userID int, cancelAtPeriodEnd bool) error {
	// Получаем активную подписку
	sub, err := s.subRepo.GetActiveByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("ошибка получения активной подписки: %w", err)
	}
	if sub == nil {
		return errors.New("активная подписка не найдена")
	}

	// Отменяем подписку
	if err := s.subRepo.Cancel(ctx, sub.ID, cancelAtPeriodEnd); err != nil {
		return fmt.Errorf("ошибка отмены подписки: %w", err)
	}

	// Обновляем статус
	newStatus := models.StatusCanceled
	if cancelAtPeriodEnd {
		newStatus = models.StatusActive // Остается активной до конца периода
		// Обновляем только флаг cancel_at_period_end
		sub.CancelAtPeriodEnd = true
		if err := s.subRepo.Update(ctx, sub); err != nil {
			return fmt.Errorf("ошибка обновления подписки: %w", err)
		}
	} else {
		// Немедленная отмена - обновляем статус
		if err := s.subRepo.UpdateStatus(ctx, sub.ID, newStatus); err != nil {
			return fmt.Errorf("ошибка обновления статуса подписки: %w", err)
		}
	}

	// Трекаем событие
	if s.analytics != nil {
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
	}

	// Инвалидируем кэш
	s.invalidateSubscriptionCache(userID)

	logger.Info("⏹️ Отменена подписка: пользователь %d, отмена в конце периода: %v", userID, cancelAtPeriodEnd)

	return nil
}

// GetActiveSubscription возвращает активную подписку пользователя
func (s *Service) GetActiveSubscription(ctx context.Context, userID int) (*models.UserSubscription, error) {
	// Пробуем получить из кэша
	cacheKey := s.cachePrefix + fmt.Sprintf("user:%d", userID)
	var subscription models.UserSubscription
	if err := s.cache.Get(ctx, cacheKey, &subscription); err == nil {
		return &subscription, nil
	}

	// Получаем из репозитория
	subscriptionPtr, err := s.subRepo.GetActiveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Кэшируем
	if subscriptionPtr != nil {
		s.cacheSubscription(subscriptionPtr)
	}

	return subscriptionPtr, nil
}

// CheckUserLimit проверяет лимит пользователя
func (s *Service) CheckUserLimit(ctx context.Context, userID int, limitType string, currentUsage int) (bool, int, error) {
	// Получаем активную подписку
	subscription, err := s.GetActiveSubscription(ctx, userID)
	if err != nil {
		return false, 0, err
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
		return false, 0, err
	}

	var maxLimit int
	switch strings.ToLower(limitType) {
	case "symbols":
		maxLimit = plan.MaxSymbols
	case "signals":
		maxLimit = plan.MaxSignalsPerDay
	case "api_requests":
		maxLimit = plan.GetMaxAPIRequests()
	default:
		return false, 0, fmt.Errorf("неизвестный тип лимита: %s", limitType)
	}

	// Неограниченный доступ
	if maxLimit == -1 {
		return true, -1, nil
	}

	remaining := maxLimit - currentUsage
	return remaining > 0, remaining, nil
}

// ProcessExpiredSubscriptions обрабатывает истекшие подписки
func (s *Service) ProcessExpiredSubscriptions(ctx context.Context) error {
	expiredSubs, err := s.subRepo.GetExpiredSubscriptions(ctx)
	if err != nil {
		return err
	}

	for _, sub := range expiredSubs {
		// Проверяем, есть ли у пользователя другая активная подписка
		activeSub, _ := s.GetActiveSubscription(ctx, sub.UserID)

		// Помечаем старую подписку как истекшую
		s.subRepo.UpdateStatus(ctx, sub.ID, models.StatusExpired)

		// ⭐ НИКОГДА не создаем новую free автоматически!
		if activeSub == nil && sub.PlanCode != models.PlanFree {
			// Только если нет активной подписки и это не free
			logger.Info("⚠️ Пользователь %d остался без подписки", sub.UserID)
		}
	}
	return nil
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
	}

	s.cache.DeleteMulti(ctx, keys...)
}

func (s *Service) startSubscriptionChecker() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()
		currentTime := time.Now()
		logger.Info("⏰ [CHECKER] Запуск проверки истекших подписок в %s", currentTime.Format("15:04:05"))

		if err := s.ProcessExpiredSubscriptions(ctx); err != nil {
			logger.Error("❌ [CHECKER] Ошибка обработки истекших подписок: %v", err)
		}
	}
}

// GetUserSubscription возвращает подписку пользователя (для обратной совместимости)
func (s *Service) GetUserSubscription(userID int) (*models.UserSubscription, error) {
	ctx := context.Background()
	return s.GetActiveSubscription(ctx, userID)
}

// GetRepository возвращает репозиторий подписок
func (s *Service) GetRepository() subscription_repo.SubscriptionRepository {
	return s.subRepo
}

// GetPlanByID возвращает план по ID
func (s *Service) GetPlanByID(ctx context.Context, planID int) (*models.Plan, error) {
	// Сначала ищем в кэше планов
	s.mu.RLock()
	for _, plan := range s.plans {
		if plan.ID == planID {
			s.mu.RUnlock()
			return plan, nil
		}
	}
	s.mu.RUnlock()

	// Если не нашли в памяти, ищем в БД
	plan, err := s.planRepo.GetByID(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения плана по ID %d: %w", planID, err)
	}
	if plan == nil {
		return nil, fmt.Errorf("план не найден: %d", planID)
	}

	// Сохраняем в кэш
	s.mu.Lock()
	s.plans[plan.Code] = plan
	s.mu.Unlock()

	return plan, nil
}

// GetLatestSubscription возвращает последнюю подписку пользователя (любого статуса)
func (s *Service) GetLatestSubscription(ctx context.Context, userID int) (*models.UserSubscription, error) {
	logger.Info("🔍 GetLatestSubscription: ищем подписку для user %d", userID)

	// Получаем из репозитория последнюю подписку
	subscription, err := s.subRepo.GetByUserID(ctx, userID)
	if err != nil {
		logger.Error("❌ GetLatestSubscription: ошибка получения подписки для user %d: %v", userID, err)
		return nil, err
	}

	if subscription == nil {
		logger.Info("📅 GetLatestSubscription: подписка не найдена для user %d", userID)
		return nil, nil
	}

	logger.Info("✅ GetLatestSubscription: найдена подписка для user %d, статус: %s, план: %s",
		userID, subscription.Status, subscription.PlanCode)

	return subscription, nil
}
