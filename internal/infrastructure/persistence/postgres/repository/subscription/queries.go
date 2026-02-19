// internal/infrastructure/persistence/postgres/repository/subscription/queries.go
package subscription

import (
	"context"
	"database/sql"
	"fmt"

	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
)

// GetExpiringSubscriptions получает подписки, срок которых истекает в течение N дней
func (r *subscriptionRepositoryImpl) GetExpiringSubscriptions(ctx context.Context, daysBefore int) ([]*models.UserSubscription, error) {
	query := `
	SELECT
		us.id, us.user_id, us.plan_id, us.payment_id,
		us.stripe_subscription_id, us.status,
		us.current_period_start, us.current_period_end,
		us.cancel_at_period_end, us.metadata,
		us.created_at, us.updated_at
	FROM user_subscriptions us
	WHERE us.status IN ('active', 'trialing')
	AND us.current_period_end IS NOT NULL
	AND us.current_period_end BETWEEN NOW() AND NOW() + INTERVAL '1 day' * $1
	AND us.cancel_at_period_end = false
	ORDER BY us.current_period_end ASC
	`

	return r.scanRows(ctx, query, daysBefore)
}

// GetExpiredSubscriptions получает истекшие подписки
func (r *subscriptionRepositoryImpl) GetExpiredSubscriptions(ctx context.Context) ([]*models.UserSubscription, error) {
	query := `
	SELECT
		us.id, us.user_id, us.plan_id, us.payment_id,
		us.stripe_subscription_id, us.status,
		us.current_period_start, us.current_period_end,
		us.cancel_at_period_end, us.metadata,
		us.created_at, us.updated_at
	FROM user_subscriptions us
	WHERE us.status IN ('active', 'trialing')
	AND us.current_period_end IS NOT NULL
	AND us.current_period_end < NOW()
	AND us.cancel_at_period_end = false
	ORDER BY us.current_period_end ASC
	`

	return r.scanRows(ctx, query)
}

// GetByPaymentID получает подписку по ID платежа
func (r *subscriptionRepositoryImpl) GetByPaymentID(ctx context.Context, paymentID int64) (*models.UserSubscription, error) {
	query := `
	SELECT
		us.id, us.user_id, us.plan_id, us.payment_id,
		us.stripe_subscription_id, us.status,
		us.current_period_start, us.current_period_end,
		us.cancel_at_period_end, us.metadata,
		us.created_at, us.updated_at
	FROM user_subscriptions us
	WHERE us.payment_id = $1
	ORDER BY us.created_at DESC
	LIMIT 1
	`

	var sub models.UserSubscription
	var metadataJSON []byte

	if err := r.db.QueryRowContext(ctx, query, paymentID).Scan(
		&sub.ID, &sub.UserID, &sub.PlanID, &sub.PaymentID,
		&sub.StripeSubscriptionID, &sub.Status,
		&sub.CurrentPeriodStart, &sub.CurrentPeriodEnd,
		&sub.CancelAtPeriodEnd, &metadataJSON,
		&sub.CreatedAt, &sub.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("ошибка получения подписки по payment_id %d: %w", paymentID, err)
	}

	return scanMetaAndPlan(&sub, metadataJSON, sql.NullString{}, sql.NullString{})
}

// GetAllByUserID получает ВСЕ подписки пользователя (любого статуса)
func (r *subscriptionRepositoryImpl) GetAllByUserID(ctx context.Context, userID int) ([]*models.UserSubscription, error) {
	query := `
	SELECT
		us.id, us.user_id, us.plan_id, us.payment_id,
		us.stripe_subscription_id, us.status,
		us.current_period_start, us.current_period_end,
		us.cancel_at_period_end, us.metadata,
		us.created_at, us.updated_at,
		sp.name as plan_name, sp.code as plan_code
	FROM user_subscriptions us
	LEFT JOIN subscription_plans sp ON us.plan_id = sp.id
	WHERE us.user_id = $1
	ORDER BY us.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения подписок пользователя %d: %w", userID, err)
	}
	defer rows.Close()

	var subscriptions []*models.UserSubscription
	for rows.Next() {
		var sub models.UserSubscription
		var metadataJSON []byte
		var planName, planCode sql.NullString

		if err := rows.Scan(
			&sub.ID, &sub.UserID, &sub.PlanID, &sub.PaymentID,
			&sub.StripeSubscriptionID, &sub.Status,
			&sub.CurrentPeriodStart, &sub.CurrentPeriodEnd,
			&sub.CancelAtPeriodEnd, &metadataJSON,
			&sub.CreatedAt, &sub.UpdatedAt,
			&planName, &planCode,
		); err != nil {
			return nil, fmt.Errorf("ошибка сканирования подписки: %w", err)
		}

		result, err := scanMetaAndPlan(&sub, metadataJSON, planName, planCode)
		if err != nil {
			return nil, err
		}
		subscriptions = append(subscriptions, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка чтения подписок пользователя %d: %w", userID, err)
	}

	return subscriptions, nil
}

// scanRows сканирует строки без join (для GetExpiring/GetExpired)
func (r *subscriptionRepositoryImpl) scanRows(ctx context.Context, query string, args ...interface{}) ([]*models.UserSubscription, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer rows.Close()

	var subscriptions []*models.UserSubscription
	for rows.Next() {
		var sub models.UserSubscription
		var metadataJSON []byte

		if err := rows.Scan(
			&sub.ID, &sub.UserID, &sub.PlanID, &sub.PaymentID,
			&sub.StripeSubscriptionID, &sub.Status,
			&sub.CurrentPeriodStart, &sub.CurrentPeriodEnd,
			&sub.CancelAtPeriodEnd, &metadataJSON,
			&sub.CreatedAt, &sub.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("ошибка сканирования подписки: %w", err)
		}

		result, err := scanMetaAndPlan(&sub, metadataJSON, sql.NullString{}, sql.NullString{})
		if err != nil {
			return nil, err
		}
		subscriptions = append(subscriptions, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка чтения строк: %w", err)
	}

	return subscriptions, nil
}
