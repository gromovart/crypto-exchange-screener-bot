// internal/infrastructure/persistence/postgres/repository/subscription/repository.go
package subscription

import (
	"context"
	"crypto-exchange-screener-bot/internal/infrastructure/cache/redis"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// SubscriptionRepository интерфейс для работы с подписками
type SubscriptionRepository interface {
	// Планы подписок
	CreatePlan(plan *models.Plan) error
	GetPlanByCode(code string) (*models.Plan, error)
	GetAllPlans() ([]*models.Plan, error)

	// Подписки пользователей
	CreateSubscription(sub *models.UserSubscription) error
	GetActiveSubscription(userID int) (*models.UserSubscription, error)
	UpdateSubscriptionStatus(subscriptionID, stripeSubscriptionID string, status string, currentPeriodEnd time.Time) error
	CancelSubscription(userID int, cancelAtPeriodEnd bool) error
	GetExpiringSubscriptions(days int) ([]*models.UserSubscription, error)
	GetSubscriptionHistory(userID int) ([]*models.UserSubscription, error)
	GetSubscriptionsByStatus(status string) ([]*models.UserSubscription, error)
	UpdateUserSubscriptionTier(userID int, tier string) error
	GetSubscriptionStats(ctx context.Context) (map[string]interface{}, error)
	ProcessExpiredSubscriptions(ctx context.Context) (int, error)
	CleanupOldSubscriptions(ctx context.Context, olderThanDays int) (int, error)
	GetRevenueReport(startDate, endDate time.Time) (*models.RevenueReport, error)
}

// JSONMap для работы с JSON полями
type JSONMap map[string]interface{}

// SubscriptionRepositoryImpl реализация репозитория подписок
type SubscriptionRepositoryImpl struct {
	db    *sqlx.DB
	cache *redis.Cache
}

// NewSubscriptionRepository создает новый репозиторий подписок
func NewSubscriptionRepository(db *sqlx.DB, cache *redis.Cache) *SubscriptionRepositoryImpl {
	return &SubscriptionRepositoryImpl{db: db, cache: cache}
}

// CreatePlan создает новый тарифный план
func (r *SubscriptionRepositoryImpl) CreatePlan(plan *models.Plan) error {
	// Начинаем транзакцию
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	featuresJSON, err := json.Marshal(plan.Features)
	if err != nil {
		return fmt.Errorf("failed to marshal features: %w", err)
	}

	query := `
    INSERT INTO subscription_plans (
        name, code, description, price_monthly, price_yearly,
        max_symbols, max_signals_per_day, features, is_active
    ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    RETURNING id, created_at
    `

	err = tx.QueryRow(
		query,
		plan.Name, plan.Code, plan.Description,
		plan.PriceMonthly, plan.PriceYearly,
		plan.MaxSymbols, plan.MaxSignalsPerDay,
		featuresJSON, plan.IsActive,
	).Scan(&plan.ID, &plan.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create plan: %w", err)
	}

	// Фиксируем транзакцию
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Инвалидируем кэш
	r.invalidatePlanCache(plan.ID, plan.Code)

	return nil
}

// GetPlanByCode получает план по коду
func (r *SubscriptionRepositoryImpl) GetPlanByCode(code string) (*models.Plan, error) {
	// Попробуем получить из кэша
	cacheKey := fmt.Sprintf("plan:code:%s", code)
	var cachedPlan models.Plan
	if err := r.cache.Get(context.Background(), cacheKey, &cachedPlan); err == nil {
		return &cachedPlan, nil
	}

	query := `
    SELECT id, name, code, description, price_monthly, price_yearly,
           max_symbols, max_signals_per_day, features, is_active, created_at
    FROM subscription_plans
    WHERE code = $1 AND is_active = TRUE
    `

	var plan models.Plan
	var featuresJSON []byte

	err := r.db.QueryRow(query, code).Scan(
		&plan.ID, &plan.Name, &plan.Code, &plan.Description,
		&plan.PriceMonthly, &plan.PriceYearly,
		&plan.MaxSymbols, &plan.MaxSignalsPerDay,
		&featuresJSON, &plan.IsActive, &plan.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Декодируем JSON с фичами
	if err := json.Unmarshal(featuresJSON, &plan.Features); err != nil {
		return nil, fmt.Errorf("failed to unmarshal features: %w", err)
	}

	// Сохраняем в кэш
	r.cache.Set(context.Background(), cacheKey, plan, 1*time.Hour)

	return &plan, nil
}

// GetAllPlans возвращает все активные планы
func (r *SubscriptionRepositoryImpl) GetAllPlans() ([]*models.Plan, error) {
	// Попробуем получить из кэша
	cacheKey := "plans:all"
	var cachedPlans []*models.Plan
	if err := r.cache.Get(context.Background(), cacheKey, &cachedPlans); err == nil {
		return cachedPlans, nil
	}

	query := `
    SELECT id, name, code, description, price_monthly, price_yearly,
           max_symbols, max_signals_per_day, features, is_active, created_at
    FROM subscription_plans
    WHERE is_active = TRUE
    ORDER BY price_monthly ASC
    `

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plans []*models.Plan

	for rows.Next() {
		var plan models.Plan
		var featuresJSON []byte

		err := rows.Scan(
			&plan.ID, &plan.Name, &plan.Code, &plan.Description,
			&plan.PriceMonthly, &plan.PriceYearly,
			&plan.MaxSymbols, &plan.MaxSignalsPerDay,
			&featuresJSON, &plan.IsActive, &plan.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		// Декодируем JSON с фичами
		if err := json.Unmarshal(featuresJSON, &plan.Features); err != nil {
			return nil, fmt.Errorf("failed to unmarshal features: %w", err)
		}

		plans = append(plans, &plan)
	}

	// Сохраняем в кэш
	r.cache.Set(context.Background(), cacheKey, plans, 30*time.Minute)

	return plans, nil
}

// CreateSubscription создает новую подписку
func (r *SubscriptionRepositoryImpl) CreateSubscription(sub *models.UserSubscription) error {
	// Начинаем транзакцию
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	metadataJSON, err := json.Marshal(sub.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
    INSERT INTO user_subscriptions (
        user_id, plan_id, stripe_subscription_id, status,
        current_period_start, current_period_end,
        cancel_at_period_end, metadata
    ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    RETURNING id, created_at
    `

	err = tx.QueryRow(
		query,
		sub.UserID, sub.PlanID, sub.StripeSubscriptionID, sub.Status,
		sub.CurrentPeriodStart, sub.CurrentPeriodEnd,
		sub.CancelAtPeriodEnd, metadataJSON,
	).Scan(&sub.ID, &sub.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create subscription: %w", err)
	}

	// Фиксируем транзакцию
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Инвалидируем кэш
	r.invalidateSubscriptionCache(sub.ID, sub.UserID)

	return nil
}

// GetActiveSubscription получает активную подписку пользователя
func (r *SubscriptionRepositoryImpl) GetActiveSubscription(userID int) (*models.UserSubscription, error) {
	// Попробуем получить из кэша
	cacheKey := fmt.Sprintf("subscription:active:user:%d", userID)
	var cachedSub models.UserSubscription
	if err := r.cache.Get(context.Background(), cacheKey, &cachedSub); err == nil {
		return &cachedSub, nil
	}

	query := `
    SELECT us.id, us.user_id, us.plan_id, us.stripe_subscription_id,
           us.status, us.current_period_start, us.current_period_end,
           us.cancel_at_period_end, us.metadata, us.created_at,
           sp.name as plan_name, sp.code as plan_code
    FROM user_subscriptions us
    JOIN subscription_plans sp ON us.plan_id = sp.id
    WHERE us.user_id = $1
      AND us.status IN ('active', 'trialing')
      AND us.current_period_end > NOW()
    ORDER BY us.created_at DESC
    LIMIT 1
    `

	var sub models.UserSubscription
	var metadataJSON []byte

	err := r.db.QueryRow(query, userID).Scan(
		&sub.ID, &sub.UserID, &sub.PlanID, &sub.StripeSubscriptionID,
		&sub.Status, &sub.CurrentPeriodStart, &sub.CurrentPeriodEnd,
		&sub.CancelAtPeriodEnd, &metadataJSON, &sub.CreatedAt,
		&sub.PlanName, &sub.PlanCode,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Декодируем JSON с метаданными
	if err := json.Unmarshal(metadataJSON, &sub.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	// Сохраняем в кэш
	r.cache.Set(context.Background(), cacheKey, sub, 5*time.Minute)

	return &sub, nil
}

// UpdateSubscriptionStatus обновляет статус подписки
func (r *SubscriptionRepositoryImpl) UpdateSubscriptionStatus(
	subscriptionID, stripeSubscriptionID string,
	status string, currentPeriodEnd time.Time,
) error {
	// Начинаем транзакцию
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
    UPDATE user_subscriptions
    SET status = $1,
        current_period_end = $2,
        updated_at = NOW()
    WHERE id = $3 OR stripe_subscription_id = $4
    `

	result, err := tx.Exec(
		query, status, currentPeriodEnd, subscriptionID, stripeSubscriptionID,
	)

	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	// Фиксируем транзакцию
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Инвалидируем кэш
	// Для инвалидации нужно получить user_id подписки
	var userID int
	err = r.db.QueryRow(
		"SELECT user_id FROM user_subscriptions WHERE id = $1 OR stripe_subscription_id = $2",
		subscriptionID, stripeSubscriptionID,
	).Scan(&userID)

	if err == nil {
		r.invalidateSubscriptionCache(0, userID)
	}

	return nil
}

// CancelSubscription отменяет подписку
func (r *SubscriptionRepositoryImpl) CancelSubscription(
	userID int, cancelAtPeriodEnd bool,
) error {
	query := `
    UPDATE user_subscriptions
    SET cancel_at_period_end = $1,
        updated_at = NOW()
    WHERE user_id = $2 AND status = 'active'
    `

	result, err := r.db.Exec(query, cancelAtPeriodEnd, userID)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	// Инвалидируем кэш
	r.invalidateSubscriptionCache(0, userID)

	return nil
}

// GetExpiringSubscriptions возвращает подписки, срок которых истекает в течение N дней
func (r *SubscriptionRepositoryImpl) GetExpiringSubscriptions(days int) ([]*models.UserSubscription, error) {
	query := `
    SELECT us.id, us.user_id, us.plan_id, us.stripe_subscription_id,
           us.status, us.current_period_start, us.current_period_end,
           us.cancel_at_period_end, us.metadata, us.created_at,
           sp.name as plan_name, sp.code as plan_code,
           u.telegram_id, u.chat_id, u.first_name
    FROM user_sessions us
    JOIN subscription_plans sp ON us.plan_id = sp.id
    JOIN users u ON us.user_id = u.id
    WHERE us.status IN ('active', 'trialing')
      AND us.current_period_end BETWEEN NOW() AND NOW() + INTERVAL '1 day' * $1
      AND us.cancel_at_period_end = FALSE
    ORDER BY us.current_period_end ASC
    `

	rows, err := r.db.Query(query, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subscriptions []*models.UserSubscription

	for rows.Next() {
		var sub models.UserSubscription
		var metadataJSON []byte

		err := rows.Scan(
			&sub.ID, &sub.UserID, &sub.PlanID, &sub.StripeSubscriptionID,
			&sub.Status, &sub.CurrentPeriodStart, &sub.CurrentPeriodEnd,
			&sub.CancelAtPeriodEnd, &metadataJSON, &sub.CreatedAt,
			&sub.PlanName, &sub.PlanCode,
			&sub.TelegramID, &sub.ChatID, &sub.UserFirstName,
		)

		if err != nil {
			return nil, err
		}

		// Декодируем JSON с метаданными
		if err := json.Unmarshal(metadataJSON, &sub.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		subscriptions = append(subscriptions, &sub)
	}

	return subscriptions, nil
}

// GetSubscriptionHistory возвращает историю подписок пользователя
func (r *SubscriptionRepositoryImpl) GetSubscriptionHistory(userID int) ([]*models.UserSubscription, error) {
	// Попробуем получить из кэша
	cacheKey := fmt.Sprintf("subscription:history:user:%d", userID)
	var cachedHistory []*models.UserSubscription
	if err := r.cache.Get(context.Background(), cacheKey, &cachedHistory); err == nil {
		return cachedHistory, nil
	}

	query := `
    SELECT us.id, us.user_id, us.plan_id, us.stripe_subscription_id,
           us.status, us.current_period_start, us.current_period_end,
           us.cancel_at_period_end, us.metadata, us.created_at,
           sp.name as plan_name, sp.code as plan_code
    FROM user_subscriptions us
    JOIN subscription_plans sp ON us.plan_id = sp.id
    WHERE us.user_id = $1
    ORDER BY us.created_at DESC
    `

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subscriptions []*models.UserSubscription

	for rows.Next() {
		var sub models.UserSubscription
		var metadataJSON []byte

		err := rows.Scan(
			&sub.ID, &sub.UserID, &sub.PlanID, &sub.StripeSubscriptionID,
			&sub.Status, &sub.CurrentPeriodStart, &sub.CurrentPeriodEnd,
			&sub.CancelAtPeriodEnd, &metadataJSON, &sub.CreatedAt,
			&sub.PlanName, &sub.PlanCode,
		)

		if err != nil {
			return nil, err
		}

		// Декодируем JSON с метаданными
		if err := json.Unmarshal(metadataJSON, &sub.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		subscriptions = append(subscriptions, &sub)
	}

	// Сохраняем в кэш
	r.cache.Set(context.Background(), cacheKey, subscriptions, 15*time.Minute)

	return subscriptions, nil
}

// GetSubscriptionsByStatus возвращает подписки по статусу
func (r *SubscriptionRepositoryImpl) GetSubscriptionsByStatus(status string) ([]*models.UserSubscription, error) {
	query := `
    SELECT us.id, us.user_id, us.plan_id, us.stripe_subscription_id,
           us.status, us.current_period_start, us.current_period_end,
           us.cancel_at_period_end, us.metadata, us.created_at,
           sp.name as plan_name, sp.code as plan_code,
           u.telegram_id, u.chat_id, u.first_name
    FROM user_subscriptions us
    JOIN subscription_plans sp ON us.plan_id = sp.id
    JOIN users u ON us.user_id = u.id
    WHERE us.status = $1
    ORDER BY us.created_at DESC
    `

	rows, err := r.db.Query(query, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subscriptions []*models.UserSubscription

	for rows.Next() {
		var sub models.UserSubscription
		var metadataJSON []byte

		err := rows.Scan(
			&sub.ID, &sub.UserID, &sub.PlanID, &sub.StripeSubscriptionID,
			&sub.Status, &sub.CurrentPeriodStart, &sub.CurrentPeriodEnd,
			&sub.CancelAtPeriodEnd, &metadataJSON, &sub.CreatedAt,
			&sub.PlanName, &sub.PlanCode,
			&sub.TelegramID, &sub.ChatID, &sub.UserFirstName,
		)

		if err != nil {
			return nil, err
		}

		// Декодируем JSON с метаданными
		if err := json.Unmarshal(metadataJSON, &sub.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		subscriptions = append(subscriptions, &sub)
	}

	return subscriptions, nil
}

// UpdateUserSubscriptionTier обновляет тариф пользователя
func (r *SubscriptionRepositoryImpl) UpdateUserSubscriptionTier(userID int, tier string) error {
	query := `
    UPDATE users
    SET subscription_tier = $1,
        updated_at = NOW()
    WHERE id = $2
    `

	result, err := r.db.Exec(query, tier, userID)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	// Инвалидируем кэш пользователя
	r.invalidateUserCache(userID)

	return nil
}

// GetSubscriptionStats возвращает статистику по подпискам
func (r *SubscriptionRepositoryImpl) GetSubscriptionStats(ctx context.Context) (map[string]interface{}, error) {
	// Попробуем получить из кэша
	cacheKey := "subscription:stats"
	var cachedStats map[string]interface{}
	if err := r.cache.Get(ctx, cacheKey, &cachedStats); err == nil {
		return cachedStats, nil
	}

	stats := make(map[string]interface{})

	// Общее количество подписок
	var totalSubs int
	err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM user_subscriptions").Scan(&totalSubs)
	if err != nil {
		return nil, err
	}
	stats["total_subscriptions"] = totalSubs

	// Активные подписки
	var activeSubs int
	err = r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM user_subscriptions WHERE status = 'active'").Scan(&activeSubs)
	if err != nil {
		return nil, err
	}
	stats["active_subscriptions"] = activeSubs

	// Пробные подписки
	var trialSubs int
	err = r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM user_subscriptions WHERE status = 'trialing'").Scan(&trialSubs)
	if err != nil {
		return nil, err
	}
	stats["trial_subscriptions"] = trialSubs

	// Ежемесячный доход
	var monthlyRevenue float64
	err = r.db.QueryRowContext(ctx, `
        SELECT COALESCE(SUM(sp.price_monthly), 0)
        FROM user_subscriptions us
        JOIN subscription_plans sp ON us.plan_id = sp.id
        WHERE us.status = 'active'
    `).Scan(&monthlyRevenue)
	if err != nil {
		return nil, err
	}
	stats["monthly_revenue"] = monthlyRevenue

	// Распределение по тарифам
	rows, err := r.db.QueryContext(ctx, `
        SELECT sp.name, COUNT(*) as count
        FROM user_subscriptions us
        JOIN subscription_plans sp ON us.plan_id = sp.id
        WHERE us.status = 'active'
        GROUP BY sp.name
        ORDER BY count DESC
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	planDistribution := make(map[string]int)
	for rows.Next() {
		var planName string
		var count int
		if err := rows.Scan(&planName, &count); err != nil {
			return nil, err
		}
		planDistribution[planName] = count
	}
	stats["plan_distribution"] = planDistribution

	// Новые подписки за месяц
	var newSubsThisMonth int
	monthStart := time.Now().AddDate(0, 0, -30)
	err = r.db.QueryRowContext(ctx, `
        SELECT COUNT(*)
        FROM user_subscriptions
        WHERE created_at >= $1
    `, monthStart).Scan(&newSubsThisMonth)
	if err != nil {
		return nil, err
	}
	stats["new_subscriptions_this_month"] = newSubsThisMonth

	// Отток (отмененные подписки за месяц)
	var churnedSubs int
	err = r.db.QueryRowContext(ctx, `
        SELECT COUNT(*)
        FROM user_subscriptions
        WHERE status = 'canceled'
          AND updated_at >= $1
    `, monthStart).Scan(&churnedSubs)
	if err != nil {
		return nil, err
	}
	stats["churned_subscriptions"] = churnedSubs

	// Коэффициент оттока
	if activeSubs > 0 {
		churnRate := float64(churnedSubs) / float64(activeSubs) * 100
		stats["churn_rate_percent"] = churnRate
	} else {
		stats["churn_rate_percent"] = 0
	}

	// Сохраняем в кэш
	r.cache.Set(ctx, cacheKey, stats, 10*time.Minute)

	return stats, nil
}

// ProcessExpiredSubscriptions обрабатывает истекшие подписки
func (r *SubscriptionRepositoryImpl) ProcessExpiredSubscriptions(ctx context.Context) (int, error) {
	// Начинаем транзакцию
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// 1. Находим истекшие подписки
	query := `
    SELECT us.id, us.user_id, us.plan_id
    FROM user_subscriptions us
    WHERE us.status IN ('active', 'trialing')
      AND us.current_period_end < NOW()
      AND us.cancel_at_period_end = FALSE
    FOR UPDATE
    `

	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var expiredSubs []struct {
		ID     int
		UserID int
		PlanID int
	}

	for rows.Next() {
		var sub struct {
			ID     int
			UserID int
			PlanID int
		}
		if err := rows.Scan(&sub.ID, &sub.UserID, &sub.PlanID); err != nil {
			return 0, err
		}
		expiredSubs = append(expiredSubs, sub)
	}

	if len(expiredSubs) == 0 {
		return 0, nil
	}

	// 2. Обновляем статус подписок
	updateQuery := `
    UPDATE user_subscriptions
    SET status = 'expired',
        updated_at = NOW()
    WHERE id = ANY($1)
    `

	expiredIDs := make([]int, len(expiredSubs))
	for i, sub := range expiredSubs {
		expiredIDs[i] = sub.ID
	}

	_, err = tx.ExecContext(ctx, updateQuery, pq.Array(expiredIDs))
	if err != nil {
		return 0, err
	}

	// 3. Обновляем тариф пользователей на free
	for _, sub := range expiredSubs {
		userUpdateQuery := `
        UPDATE users
        SET subscription_tier = 'free',
            max_signals_per_day = 50,
            updated_at = NOW()
        WHERE id = $1
        `
		_, err = tx.ExecContext(ctx, userUpdateQuery, sub.UserID)
		if err != nil {
			return 0, err
		}
	}

	// Коммитим транзакцию
	if err := tx.Commit(); err != nil {
		return 0, err
	}

	// Инвалидируем кэш для всех пользователей
	for _, sub := range expiredSubs {
		r.invalidateSubscriptionCache(sub.ID, sub.UserID)
		r.invalidateUserCache(sub.UserID)
	}

	return len(expiredSubs), nil
}

// CleanupOldSubscriptions удаляет старые неактивные подписки
func (r *SubscriptionRepositoryImpl) CleanupOldSubscriptions(ctx context.Context, olderThanDays int) (int, error) {
	query := `
    DELETE FROM user_subscriptions
    WHERE status IN ('canceled', 'expired')
      AND updated_at < NOW() - INTERVAL '1 day' * $1
    RETURNING id, user_id
    `

	rows, err := r.db.QueryContext(ctx, query, olderThanDays)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var deletedCount int
	var userIDs []int

	for rows.Next() {
		var id, userID int
		if err := rows.Scan(&id, &userID); err != nil {
			return 0, err
		}
		deletedCount++
		userIDs = append(userIDs, userID)
	}

	// Инвалидируем кэш для затронутых пользователей
	for _, userID := range userIDs {
		r.invalidateSubscriptionCache(0, userID)
	}

	return deletedCount, nil
}

// GetRevenueReport возвращает отчет по доходам за период
func (r *SubscriptionRepositoryImpl) GetRevenueReport(startDate, endDate time.Time) (*models.RevenueReport, error) {
	cacheKey := fmt.Sprintf("revenue:report:%s:%s",
		startDate.Format("2006-01-02"),
		endDate.Format("2006-01-02"))

	// Попробуем получить из кэша
	var cachedReport models.RevenueReport
	if err := r.cache.Get(context.Background(), cacheKey, &cachedReport); err == nil {
		return &cachedReport, nil
	}

	query := `
    SELECT
        -- Общий доход
        COALESCE(SUM(
            CASE
                WHEN sp.price_monthly > 0 THEN sp.price_monthly
                ELSE sp.price_yearly / 12
            END
        ), 0) as total_revenue,

        -- Количество новых подписок
        COUNT(DISTINCT us.id) as new_subscriptions,

        -- Средний доход на пользователя (ARPU)
        COALESCE(AVG(
            CASE
                WHEN sp.price_monthly > 0 THEN sp.price_monthly
                ELSE sp.price_yearly / 12
            END
        ), 0) as arpu,

        -- Самый популярный план
        (SELECT sp.name
         FROM user_subscriptions us2
         JOIN subscription_plans sp ON us2.plan_id = sp.id
         WHERE us2.created_at BETWEEN $1 AND $2
         GROUP BY sp.name
         ORDER BY COUNT(*) DESC
         LIMIT 1) as most_popular_plan,

        -- Распределение по месяцам (для графика)
        json_agg(
            json_build_object(
                'month', DATE_TRUNC('month', us.created_at),
                'revenue', SUM(
                    CASE
                        WHEN sp.price_monthly > 0 THEN sp.price_monthly
                        ELSE sp.price_yearly / 12
                    END
                ),
                'subscribers', COUNT(*)
            )
        ) as monthly_breakdown

    FROM user_subscriptions us
    JOIN subscription_plans sp ON us.plan_id = sp.id
    WHERE us.created_at BETWEEN $1 AND $2
      AND us.status IN ('active', 'trialing')
    `

	var report models.RevenueReport
	var monthlyBreakdownJSON []byte

	err := r.db.QueryRow(query, startDate, endDate).Scan(
		&report.TotalRevenue,
		&report.NewSubscriptions,
		&report.ARPU,
		&report.MostPopularPlan,
		&monthlyBreakdownJSON,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return &models.RevenueReport{}, nil
		}
		return nil, err
	}

	// Декодируем JSON с распределением по месяцам
	if err := json.Unmarshal(monthlyBreakdownJSON, &report.MonthlyBreakdown); err != nil {
		return nil, fmt.Errorf("failed to unmarshal monthly breakdown: %w", err)
	}

	report.PeriodStart = startDate
	report.PeriodEnd = endDate

	// Сохраняем в кэш
	r.cache.Set(context.Background(), cacheKey, report, 1*time.Hour)

	return &report, nil
}

// Вспомогательные методы для инвалидации кэша

// invalidatePlanCache инвалидирует кэш планов
func (r *SubscriptionRepositoryImpl) invalidatePlanCache(planID int, planCode string) {
	ctx := context.Background()
	keys := []string{
		"plans:all",
		fmt.Sprintf("plan:id:%d", planID),
		fmt.Sprintf("plan:code:%s", planCode),
	}

	_ = r.cache.DeleteMulti(ctx, keys...)
}

// invalidateSubscriptionCache инвалидирует кэш подписок
func (r *SubscriptionRepositoryImpl) invalidateSubscriptionCache(subscriptionID int, userID int) {
	ctx := context.Background()
	keys := []string{
		"subscription:stats",
	}

	if subscriptionID > 0 {
		keys = append(keys, fmt.Sprintf("subscription:id:%d", subscriptionID))
	}
	if userID > 0 {
		keys = append(keys, fmt.Sprintf("subscription:active:user:%d", userID))
		keys = append(keys, fmt.Sprintf("subscription:history:user:%d", userID))
	}

	_ = r.cache.DeleteMulti(ctx, keys...)
}

// invalidateUserCache инвалидирует кэш пользователя
func (r *SubscriptionRepositoryImpl) invalidateUserCache(userID int) {
	ctx := context.Background()
	keys := []string{
		fmt.Sprintf("user:%d", userID),
		fmt.Sprintf("user_stats:%d", userID),
	}

	_ = r.cache.DeleteMulti(ctx, keys...)
}

// Вспомогательная функция для преобразования времени в NullTime
func getNullTime(t time.Time) sql.NullTime {
	if t.IsZero() {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{
		Time:  t,
		Valid: true,
	}
}
