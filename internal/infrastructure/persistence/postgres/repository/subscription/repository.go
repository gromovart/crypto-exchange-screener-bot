// persistence/postgres/repository/subscription_repository.go
package repository

// import (
// 	"context"
// 	"database/sql"
// 	"encoding/json"
// 	"fmt"
// 	"time"

// 	"crypto-exchange-screener-bot/internal/subscription"

// 	"github.com/jmoiron/sqlx"
// 	"github.com/lib/pq"
// )

// // SubscriptionRepository управляет подписками пользователей
// type SubscriptionRepository struct {
// 	db *sqlx.DB
// }

// // NewSubscriptionRepository создает новый репозиторий подписок
// func NewSubscriptionRepository(db *sqlx.DB) *SubscriptionRepository {
// 	return &SubscriptionRepository{db: db}
// }

// // Планы подписок

// // CreatePlan создает новый тарифный план
// func (r *SubscriptionRepository) CreatePlan(plan *subscription.Plan) error {
// 	query := `
//     INSERT INTO subscription_plans (
//         name, code, description, price_monthly, price_yearly,
//         max_symbols, max_signals_per_day, features, is_active
//     ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
//     RETURNING id, created_at
//     `

// 	featuresJSON, err := json.Marshal(plan.Features)
// 	if err != nil {
// 		return fmt.Errorf("failed to marshal features: %w", err)
// 	}

// 	return r.db.QueryRow(
// 		query,
// 		plan.Name, plan.Code, plan.Description,
// 		plan.PriceMonthly, plan.PriceYearly,
// 		plan.MaxSymbols, plan.MaxSignalsPerDay,
// 		featuresJSON, plan.IsActive,
// 	).Scan(&plan.ID, &plan.CreatedAt)
// }

// // GetPlanByCode получает план по коду
// func (r *SubscriptionRepository) GetPlanByCode(code string) (*subscription.Plan, error) {
// 	query := `
//     SELECT id, name, code, description, price_monthly, price_yearly,
//            max_symbols, max_signals_per_day, features, is_active, created_at
//     FROM subscription_plans
//     WHERE code = $1 AND is_active = TRUE
//     `

// 	var plan subscription.Plan
// 	var featuresJSON []byte

// 	err := r.db.QueryRow(query, code).Scan(
// 		&plan.ID, &plan.Name, &plan.Code, &plan.Description,
// 		&plan.PriceMonthly, &plan.PriceYearly,
// 		&plan.MaxSymbols, &plan.MaxSignalsPerDay,
// 		&featuresJSON, &plan.IsActive, &plan.CreatedAt,
// 	)

// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			return nil, nil
// 		}
// 		return nil, err
// 	}

// 	// Декодируем JSON с фичами
// 	if err := json.Unmarshal(featuresJSON, &plan.Features); err != nil {
// 		return nil, fmt.Errorf("failed to unmarshal features: %w", err)
// 	}

// 	return &plan, nil
// }

// // GetAllPlans возвращает все активные планы
// func (r *SubscriptionRepository) GetAllPlans() ([]*subscription.Plan, error) {
// 	query := `
//     SELECT id, name, code, description, price_monthly, price_yearly,
//            max_symbols, max_signals_per_day, features, is_active, created_at
//     FROM subscription_plans
//     WHERE is_active = TRUE
//     ORDER BY price_monthly ASC
//     `

// 	rows, err := r.db.Query(query)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	var plans []*subscription.Plan

// 	for rows.Next() {
// 		var plan subscription.Plan
// 		var featuresJSON []byte

// 		err := rows.Scan(
// 			&plan.ID, &plan.Name, &plan.Code, &plan.Description,
// 			&plan.PriceMonthly, &plan.PriceYearly,
// 			&plan.MaxSymbols, &plan.MaxSignalsPerDay,
// 			&featuresJSON, &plan.IsActive, &plan.CreatedAt,
// 		)

// 		if err != nil {
// 			return nil, err
// 		}

// 		// Декодируем JSON с фичами
// 		if err := json.Unmarshal(featuresJSON, &plan.Features); err != nil {
// 			return nil, fmt.Errorf("failed to unmarshal features: %w", err)
// 		}

// 		plans = append(plans, &plan)
// 	}

// 	return plans, nil
// }

// // User Subscriptions

// // CreateSubscription создает новую подписку
// func (r *SubscriptionRepository) CreateSubscription(sub *subscription.UserSubscription) error {
// 	query := `
//     INSERT INTO user_subscriptions (
//         user_id, plan_id, stripe_subscription_id, status,
//         current_period_start, current_period_end,
//         cancel_at_period_end, metadata
//     ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
//     RETURNING id, created_at
//     `

// 	metadataJSON, err := json.Marshal(sub.Metadata)
// 	if err != nil {
// 		return fmt.Errorf("failed to marshal metadata: %w", err)
// 	}

// 	return r.db.QueryRow(
// 		query,
// 		sub.UserID, sub.PlanID, sub.StripeSubscriptionID, sub.Status,
// 		sub.CurrentPeriodStart, sub.CurrentPeriodEnd,
// 		sub.CancelAtPeriodEnd, metadataJSON,
// 	).Scan(&sub.ID, &sub.CreatedAt)
// }

// // GetActiveSubscription получает активную подписку пользователя
// func (r *SubscriptionRepository) GetActiveSubscription(userID int) (*subscription.UserSubscription, error) {
// 	query := `
//     SELECT us.id, us.user_id, us.plan_id, us.stripe_subscription_id,
//            us.status, us.current_period_start, us.current_period_end,
//            us.cancel_at_period_end, us.metadata, us.created_at,
//            sp.name as plan_name, sp.code as plan_code
//     FROM user_subscriptions us
//     JOIN subscription_plans sp ON us.plan_id = sp.id
//     WHERE us.user_id = $1
//       AND us.status IN ('active', 'trialing')
//       AND us.current_period_end > NOW()
//     ORDER BY us.created_at DESC
//     LIMIT 1
//     `

// 	var sub subscription.UserSubscription
// 	var metadataJSON []byte

// 	err := r.db.QueryRow(query, userID).Scan(
// 		&sub.ID, &sub.UserID, &sub.PlanID, &sub.StripeSubscriptionID,
// 		&sub.Status, &sub.CurrentPeriodStart, &sub.CurrentPeriodEnd,
// 		&sub.CancelAtPeriodEnd, &metadataJSON, &sub.CreatedAt,
// 		&sub.PlanName, &sub.PlanCode,
// 	)

// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			return nil, nil
// 		}
// 		return nil, err
// 	}

// 	// Декодируем JSON с метаданными
// 	if err := json.Unmarshal(metadataJSON, &sub.Metadata); err != nil {
// 		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
// 	}

// 	return &sub, nil
// }

// // UpdateSubscriptionStatus обновляет статус подписки
// func (r *SubscriptionRepository) UpdateSubscriptionStatus(
// 	subscriptionID, stripeSubscriptionID string,
// 	status string, currentPeriodEnd time.Time,
// ) error {

// 	query := `
//     UPDATE user_subscriptions
//     SET status = $1,
//         current_period_end = $2,
//         updated_at = NOW()
//     WHERE id = $3 OR stripe_subscription_id = $4
//     `

// 	result, err := r.db.Exec(
// 		query, status, currentPeriodEnd, subscriptionID, stripeSubscriptionID,
// 	)

// 	if err != nil {
// 		return err
// 	}

// 	rowsAffected, _ := result.RowsAffected()
// 	if rowsAffected == 0 {
// 		return sql.ErrNoRows
// 	}

// 	return nil
// }

// // CancelSubscription отменяет подписку
// func (r *SubscriptionRepository) CancelSubscription(
// 	userID int, cancelAtPeriodEnd bool,
// ) error {

// 	query := `
//     UPDATE user_subscriptions
//     SET cancel_at_period_end = $1,
//         updated_at = NOW()
//     WHERE user_id = $2 AND status = 'active'
//     `

// 	result, err := r.db.Exec(query, cancelAtPeriodEnd, userID)
// 	if err != nil {
// 		return err
// 	}

// 	rowsAffected, _ := result.RowsAffected()
// 	if rowsAffected == 0 {
// 		return sql.ErrNoRows
// 	}

// 	return nil
// }

// // GetExpiringSubscriptions возвращает подписки, срок которых истекает в течение N дней
// func (r *SubscriptionRepository) GetExpiringSubscriptions(days int) ([]*subscription.UserSubscription, error) {
// 	query := `
//     SELECT us.id, us.user_id, us.plan_id, us.stripe_subscription_id,
//            us.status, us.current_period_start, us.current_period_end,
//            us.cancel_at_period_end, us.metadata, us.created_at,
//            sp.name as plan_name, sp.code as plan_code,
//            u.telegram_id, u.chat_id, u.first_name
//     FROM user_subscriptions us
//     JOIN subscription_plans sp ON us.plan_id = sp.id
//     JOIN users u ON us.user_id = u.id
//     WHERE us.status IN ('active', 'trialing')
//       AND us.current_period_end BETWEEN NOW() AND NOW() + INTERVAL '1 day' * $1
//       AND us.cancel_at_period_end = FALSE
//     ORDER BY us.current_period_end ASC
//     `

// 	rows, err := r.db.Query(query, days)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	var subscriptions []*subscription.UserSubscription

// 	for rows.Next() {
// 		var sub subscription.UserSubscription
// 		var metadataJSON []byte

// 		err := rows.Scan(
// 			&sub.ID, &sub.UserID, &sub.PlanID, &sub.StripeSubscriptionID,
// 			&sub.Status, &sub.CurrentPeriodStart, &sub.CurrentPeriodEnd,
// 			&sub.CancelAtPeriodEnd, &metadataJSON, &sub.CreatedAt,
// 			&sub.PlanName, &sub.PlanCode,
// 			&sub.TelegramID, &sub.ChatID, &sub.UserFirstName,
// 		)

// 		if err != nil {
// 			return nil, err
// 		}

// 		// Декодируем JSON с метаданными
// 		if err := json.Unmarshal(metadataJSON, &sub.Metadata); err != nil {
// 			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
// 		}

// 		subscriptions = append(subscriptions, &sub)
// 	}

// 	return subscriptions, nil
// }

// // GetSubscriptionHistory возвращает историю подписок пользователя
// func (r *SubscriptionRepository) GetSubscriptionHistory(userID int) ([]*subscription.UserSubscription, error) {
// 	query := `
//     SELECT us.id, us.user_id, us.plan_id, us.stripe_subscription_id,
//            us.status, us.current_period_start, us.current_period_end,
//            us.cancel_at_period_end, us.metadata, us.created_at,
//            sp.name as plan_name, sp.code as plan_code
//     FROM user_subscriptions us
//     JOIN subscription_plans sp ON us.plan_id = sp.id
//     WHERE us.user_id = $1
//     ORDER BY us.created_at DESC
//     `

// 	rows, err := r.db.Query(query, userID)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	var subscriptions []*subscription.UserSubscription

// 	for rows.Next() {
// 		var sub subscription.UserSubscription
// 		var metadataJSON []byte

// 		err := rows.Scan(
// 			&sub.ID, &sub.UserID, &sub.PlanID, &sub.StripeSubscriptionID,
// 			&sub.Status, &sub.CurrentPeriodStart, &sub.CurrentPeriodEnd,
// 			&sub.CancelAtPeriodEnd, &metadataJSON, &sub.CreatedAt,
// 			&sub.PlanName, &sub.PlanCode,
// 		)

// 		if err != nil {
// 			return nil, err
// 		}

// 		// Декодируем JSON с метаданными
// 		if err := json.Unmarshal(metadataJSON, &sub.Metadata); err != nil {
// 			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
// 		}

// 		subscriptions = append(subscriptions, &sub)
// 	}

// 	return subscriptions, nil
// }

// // GetSubscriptionsByStatus возвращает подписки по статусу
// func (r *SubscriptionRepository) GetSubscriptionsByStatus(status string) ([]*subscription.UserSubscription, error) {
// 	query := `
//     SELECT us.id, us.user_id, us.plan_id, us.stripe_subscription_id,
//            us.status, us.current_period_start, us.current_period_end,
//            us.cancel_at_period_end, us.metadata, us.created_at,
//            sp.name as plan_name, sp.code as plan_code,
//            u.telegram_id, u.chat_id, u.first_name
//     FROM user_subscriptions us
//     JOIN subscription_plans sp ON us.plan_id = sp.id
//     JOIN users u ON us.user_id = u.id
//     WHERE us.status = $1
//     ORDER BY us.created_at DESC
//     `

// 	rows, err := r.db.Query(query, status)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	var subscriptions []*subscription.UserSubscription

// 	for rows.Next() {
// 		var sub subscription.UserSubscription
// 		var metadataJSON []byte

// 		err := rows.Scan(
// 			&sub.ID, &sub.UserID, &sub.PlanID, &sub.StripeSubscriptionID,
// 			&sub.Status, &sub.CurrentPeriodStart, &sub.CurrentPeriodEnd,
// 			&sub.CancelAtPeriodEnd, &metadataJSON, &sub.CreatedAt,
// 			&sub.PlanName, &sub.PlanCode,
// 			&sub.TelegramID, &sub.ChatID, &sub.UserFirstName,
// 		)

// 		if err != nil {
// 			return nil, err
// 		}

// 		// Декодируем JSON с метаданными
// 		if err := json.Unmarshal(metadataJSON, &sub.Metadata); err != nil {
// 			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
// 		}

// 		subscriptions = append(subscriptions, &sub)
// 	}

// 	return subscriptions, nil
// }

// // UpdateUserSubscriptionTier обновляет тариф пользователя
// func (r *SubscriptionRepository) UpdateUserSubscriptionTier(userID int, tier string) error {
// 	query := `
//     UPDATE users
//     SET subscription_tier = $1,
//         updated_at = NOW()
//     WHERE id = $2
//     `

// 	result, err := r.db.Exec(query, tier, userID)
// 	if err != nil {
// 		return err
// 	}

// 	rowsAffected, _ := result.RowsAffected()
// 	if rowsAffected == 0 {
// 		return sql.ErrNoRows
// 	}

// 	return nil
// }

// // GetSubscriptionStats возвращает статистику по подпискам
// func (r *SubscriptionRepository) GetSubscriptionStats(ctx context.Context) (map[string]interface{}, error) {
// 	stats := make(map[string]interface{})

// 	// Общее количество подписок
// 	var totalSubs int
// 	err := r.db.QueryRowContext(ctx,
// 		"SELECT COUNT(*) FROM user_subscriptions").Scan(&totalSubs)
// 	if err != nil {
// 		return nil, err
// 	}
// 	stats["total_subscriptions"] = totalSubs

// 	// Активные подписки
// 	var activeSubs int
// 	err = r.db.QueryRowContext(ctx,
// 		"SELECT COUNT(*) FROM user_subscriptions WHERE status = 'active'").Scan(&activeSubs)
// 	if err != nil {
// 		return nil, err
// 	}
// 	stats["active_subscriptions"] = activeSubs

// 	// Пробные подписки
// 	var trialSubs int
// 	err = r.db.QueryRowContext(ctx,
// 		"SELECT COUNT(*) FROM user_subscriptions WHERE status = 'trialing'").Scan(&trialSubs)
// 	if err != nil {
// 		return nil, err
// 	}
// 	stats["trial_subscriptions"] = trialSubs

// 	// Ежемесячный доход
// 	var monthlyRevenue float64
// 	err = r.db.QueryRowContext(ctx, `
//         SELECT COALESCE(SUM(sp.price_monthly), 0)
//         FROM user_subscriptions us
//         JOIN subscription_plans sp ON us.plan_id = sp.id
//         WHERE us.status = 'active'
//     `).Scan(&monthlyRevenue)
// 	if err != nil {
// 		return nil, err
// 	}
// 	stats["monthly_revenue"] = monthlyRevenue

// 	// Распределение по тарифам
// 	rows, err := r.db.QueryContext(ctx, `
//         SELECT sp.name, COUNT(*) as count
//         FROM user_subscriptions us
//         JOIN subscription_plans sp ON us.plan_id = sp.id
//         WHERE us.status = 'active'
//         GROUP BY sp.name
//         ORDER BY count DESC
//     `)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	planDistribution := make(map[string]int)
// 	for rows.Next() {
// 		var planName string
// 		var count int
// 		if err := rows.Scan(&planName, &count); err != nil {
// 			return nil, err
// 		}
// 		planDistribution[planName] = count
// 	}
// 	stats["plan_distribution"] = planDistribution

// 	// Новые подписки за месяц
// 	var newSubsThisMonth int
// 	monthStart := time.Now().AddDate(0, 0, -30)
// 	err = r.db.QueryRowContext(ctx, `
//         SELECT COUNT(*)
//         FROM user_subscriptions
//         WHERE created_at >= $1
//     `, monthStart).Scan(&newSubsThisMonth)
// 	if err != nil {
// 		return nil, err
// 	}
// 	stats["new_subscriptions_this_month"] = newSubsThisMonth

// 	// Отток (отмененные подписки за месяц)
// 	var churnedSubs int
// 	err = r.db.QueryRowContext(ctx, `
//         SELECT COUNT(*)
//         FROM user_subscriptions
//         WHERE status = 'canceled'
//           AND updated_at >= $1
//     `, monthStart).Scan(&churnedSubs)
// 	if err != nil {
// 		return nil, err
// 	}
// 	stats["churned_subscriptions"] = churnedSubs

// 	// Коэффициент оттока
// 	if activeSubs > 0 {
// 		churnRate := float64(churnedSubs) / float64(activeSubs) * 100
// 		stats["churn_rate_percent"] = churnRate
// 	} else {
// 		stats["churn_rate_percent"] = 0
// 	}

// 	return stats, nil
// }

// // ProcessExpiredSubscriptions обрабатывает истекшие подписки
// func (r *SubscriptionRepository) ProcessExpiredSubscriptions(ctx context.Context) (int, error) {
// 	// Начинаем транзакцию
// 	tx, err := r.db.BeginTx(ctx, nil)
// 	if err != nil {
// 		return 0, err
// 	}
// 	defer tx.Rollback()

// 	// 1. Находим истекшие подписки
// 	query := `
//     SELECT us.id, us.user_id, us.plan_id
//     FROM user_subscriptions us
//     WHERE us.status IN ('active', 'trialing')
//       AND us.current_period_end < NOW()
//       AND us.cancel_at_period_end = FALSE
//     FOR UPDATE
//     `

// 	rows, err := tx.QueryContext(ctx, query)
// 	if err != nil {
// 		return 0, err
// 	}
// 	defer rows.Close()

// 	var expiredSubs []struct {
// 		ID     int
// 		UserID int
// 		PlanID int
// 	}

// 	for rows.Next() {
// 		var sub struct {
// 			ID     int
// 			UserID int
// 			PlanID int
// 		}
// 		if err := rows.Scan(&sub.ID, &sub.UserID, &sub.PlanID); err != nil {
// 			return 0, err
// 		}
// 		expiredSubs = append(expiredSubs, sub)
// 	}

// 	if len(expiredSubs) == 0 {
// 		return 0, nil
// 	}

// 	// 2. Обновляем статус подписок
// 	updateQuery := `
//     UPDATE user_subscriptions
//     SET status = 'expired',
//         updated_at = NOW()
//     WHERE id = ANY($1)
//     `

// 	expiredIDs := make([]int, len(expiredSubs))
// 	for i, sub := range expiredSubs {
// 		expiredIDs[i] = sub.ID
// 	}

// 	_, err = tx.ExecContext(ctx, updateQuery, pq.Array(expiredIDs))
// 	if err != nil {
// 		return 0, err
// 	}

// 	// 3. Обновляем тариф пользователей на free
// 	for _, sub := range expiredSubs {
// 		userUpdateQuery := `
//         UPDATE users
//         SET subscription_tier = 'free',
//             max_signals_per_day = 50,
//             updated_at = NOW()
//         WHERE id = $1
//         `
// 		_, err = tx.ExecContext(ctx, userUpdateQuery, sub.UserID)
// 		if err != nil {
// 			return 0, err
// 		}
// 	}

// 	// Коммитим транзакцию
// 	if err := tx.Commit(); err != nil {
// 		return 0, err
// 	}

// 	return len(expiredSubs), nil
// }

// // CleanupOldSubscriptions удаляет старые неактивные подписки
// func (r *SubscriptionRepository) CleanupOldSubscriptions(ctx context.Context, olderThanDays int) (int, error) {
// 	query := `
//     DELETE FROM user_subscriptions
//     WHERE status IN ('canceled', 'expired')
//       AND updated_at < NOW() - INTERVAL '1 day' * $1
//     RETURNING id
//     `

// 	rows, err := r.db.QueryContext(ctx, query, olderThanDays)
// 	if err != nil {
// 		return 0, err
// 	}
// 	defer rows.Close()

// 	var deletedIDs []int
// 	for rows.Next() {
// 		var id int
// 		if err := rows.Scan(&id); err != nil {
// 			return 0, err
// 		}
// 		deletedIDs = append(deletedIDs, id)
// 	}

// 	return len(deletedIDs), nil
// }

// // GetRevenueReport возвращает отчет по доходам за период
// func (r *SubscriptionRepository) GetRevenueReport(startDate, endDate time.Time) (*subscription.RevenueReport, error) {
// 	query := `
//     SELECT
//         -- Общий доход
//         COALESCE(SUM(
//             CASE
//                 WHEN sp.price_monthly > 0 THEN sp.price_monthly
//                 ELSE sp.price_yearly / 12
//             END
//         ), 0) as total_revenue,

//         -- Количество новых подписок
//         COUNT(DISTINCT us.id) as new_subscriptions,

//         -- Средний доход на пользователя (ARPU)
//         COALESCE(AVG(
//             CASE
//                 WHEN sp.price_monthly > 0 THEN sp.price_monthly
//                 ELSE sp.price_yearly / 12
//             END
//         ), 0) as arpu,

//         -- Самый популярный план
//         (SELECT sp.name
//          FROM user_subscriptions us2
//          JOIN subscription_plans sp ON us2.plan_id = sp.id
//          WHERE us2.created_at BETWEEN $1 AND $2
//          GROUP BY sp.name
//          ORDER BY COUNT(*) DESC
//          LIMIT 1) as most_popular_plan,

//         -- Распределение по месяцам (для графика)
//         json_agg(
//             json_build_object(
//                 'month', DATE_TRUNC('month', us.created_at),
//                 'revenue', SUM(
//                     CASE
//                         WHEN sp.price_monthly > 0 THEN sp.price_monthly
//                         ELSE sp.price_yearly / 12
//                     END
//                 ),
//                 'subscribers', COUNT(*)
//             )
//         ) as monthly_breakdown

//     FROM user_subscriptions us
//     JOIN subscription_plans sp ON us.plan_id = sp.id
//     WHERE us.created_at BETWEEN $1 AND $2
//       AND us.status IN ('active', 'trialing')
//     `

// 	var report subscription.RevenueReport
// 	var monthlyBreakdownJSON []byte

// 	err := r.db.QueryRow(query, startDate, endDate).Scan(
// 		&report.TotalRevenue,
// 		&report.NewSubscriptions,
// 		&report.ARPU,
// 		&report.MostPopularPlan,
// 		&monthlyBreakdownJSON,
// 	)

// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			return &subscription.RevenueReport{}, nil
// 		}
// 		return nil, err
// 	}

// 	// Декодируем JSON с распределением по месяцам
// 	if err := json.Unmarshal(monthlyBreakdownJSON, &report.MonthlyBreakdown); err != nil {
// 		return nil, fmt.Errorf("failed to unmarshal monthly breakdown: %w", err)
// 	}

// 	report.PeriodStart = startDate
// 	report.PeriodEnd = endDate

// 	return &report, nil
// }
