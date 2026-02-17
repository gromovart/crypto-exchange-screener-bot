// internal/infrastructure/persistence/postgres/repository/subscription/repository.go
package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"

	"github.com/jmoiron/sqlx"
)

// SubscriptionRepository интерфейс репозитория подписок пользователей
type SubscriptionRepository interface {
	Create(ctx context.Context, subscription *models.UserSubscription) error
	GetByID(ctx context.Context, id int) (*models.UserSubscription, error)
	GetByUserID(ctx context.Context, userID int) (*models.UserSubscription, error)
	GetActiveByUserID(ctx context.Context, userID int) (*models.UserSubscription, error)
	Update(ctx context.Context, subscription *models.UserSubscription) error
	UpdateStatus(ctx context.Context, id int, status string) error
	Cancel(ctx context.Context, id int, cancelAtPeriodEnd bool) error
	GetExpiringSubscriptions(ctx context.Context, daysBefore int) ([]*models.UserSubscription, error)
	GetExpiredSubscriptions(ctx context.Context) ([]*models.UserSubscription, error)
	GetByPaymentID(ctx context.Context, paymentID int64) (*models.UserSubscription, error)
}

// subscriptionRepositoryImpl реализация SubscriptionRepository
type subscriptionRepositoryImpl struct {
	db *sqlx.DB
}

// NewSubscriptionRepository создает новый репозиторий подписок
func NewSubscriptionRepository(db *sqlx.DB) SubscriptionRepository {
	return &subscriptionRepositoryImpl{db: db}
}

// ✅ ИСПРАВЛЕНО: убраны поля plan_name и plan_code
// Create создает новую подписку
func (r *subscriptionRepositoryImpl) Create(ctx context.Context, subscription *models.UserSubscription) error {
	// Сериализуем Metadata в JSON
	metadataJSON, err := json.Marshal(subscription.Metadata)
	if err != nil {
		return fmt.Errorf("ошибка сериализации metadata: %w", err)
	}

	query := `
	INSERT INTO user_subscriptions (
		user_id, plan_id, payment_id,
		status,
		current_period_start, current_period_end,
		cancel_at_period_end, metadata
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8
	) RETURNING id, created_at, updated_at
	`

	err = r.db.QueryRowContext(ctx, query,
		subscription.UserID,
		subscription.PlanID,
		subscription.PaymentID,
		subscription.Status,
		subscription.CurrentPeriodStart,
		subscription.CurrentPeriodEnd,
		subscription.CancelAtPeriodEnd,
		metadataJSON,
	).Scan(&subscription.ID, &subscription.CreatedAt, &subscription.UpdatedAt)

	if err != nil {
		return fmt.Errorf("ошибка создания подписки: %w", err)
	}

	return nil
}

// ✅ БЕЗ ИЗМЕНЕНИЙ
// GetByID получает подписку по ID
func (r *subscriptionRepositoryImpl) GetByID(ctx context.Context, id int) (*models.UserSubscription, error) {
	query := `
	SELECT
		us.*,
		sp.name as plan_name,
		sp.code as plan_code
	FROM user_subscriptions us
	LEFT JOIN subscription_plans sp ON us.plan_id = sp.id
	WHERE us.id = $1
	`

	var subscription models.UserSubscription
	var metadataJSON []byte
	var planName, planCode sql.NullString

	if err := r.db.QueryRowContext(ctx, query, id).Scan(
		&subscription.ID,
		&subscription.UserID,
		&subscription.PlanID,
		&subscription.PaymentID,
		&subscription.StripeSubscriptionID,
		&subscription.Status,
		&subscription.CurrentPeriodStart,
		&subscription.CurrentPeriodEnd,
		&subscription.CancelAtPeriodEnd,
		&metadataJSON,
		&subscription.CreatedAt,
		&subscription.UpdatedAt,
		&planName,
		&planCode,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("ошибка получения подписки по ID %d: %w", id, err)
	}

	// Восстанавливаем metadata
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &subscription.Metadata); err != nil {
			return nil, fmt.Errorf("ошибка десериализации metadata: %w", err)
		}
	}

	// Восстанавливаем plan_name и plan_code
	if planName.Valid {
		subscription.PlanName = planName.String
	}
	if planCode.Valid {
		subscription.PlanCode = planCode.String
	}

	return &subscription, nil
}

// ✅ БЕЗ ИЗМЕНЕНИЙ
// GetByUserID получает последнюю подписку пользователя
func (r *subscriptionRepositoryImpl) GetByUserID(ctx context.Context, userID int) (*models.UserSubscription, error) {
	query := `
	SELECT
		us.*,
		sp.name as plan_name,
		sp.code as plan_code
	FROM user_subscriptions us
	LEFT JOIN subscription_plans sp ON us.plan_id = sp.id
	WHERE us.user_id = $1
	ORDER BY us.created_at DESC
	LIMIT 1
	`

	var subscription models.UserSubscription
	var metadataJSON []byte
	var planName, planCode sql.NullString

	if err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&subscription.ID,
		&subscription.UserID,
		&subscription.PlanID,
		&subscription.PaymentID,
		&subscription.StripeSubscriptionID,
		&subscription.Status,
		&subscription.CurrentPeriodStart,
		&subscription.CurrentPeriodEnd,
		&subscription.CancelAtPeriodEnd,
		&metadataJSON,
		&subscription.CreatedAt,
		&subscription.UpdatedAt,
		&planName,
		&planCode,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("ошибка получения подписки пользователя %d: %w", userID, err)
	}

	// Восстанавливаем metadata
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &subscription.Metadata); err != nil {
			return nil, fmt.Errorf("ошибка десериализации metadata: %w", err)
		}
	}

	// Восстанавливаем plan_name и plan_code
	if planName.Valid {
		subscription.PlanName = planName.String
	}
	if planCode.Valid {
		subscription.PlanCode = planCode.String
	}

	return &subscription, nil
}

// ✅ БЕЗ ИЗМЕНЕНИЙ
// GetActiveByUserID получает активную подписку пользователя
func (r *subscriptionRepositoryImpl) GetActiveByUserID(ctx context.Context, userID int) (*models.UserSubscription, error) {
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
	AND us.status IN ('active', 'trialing')
	AND (us.current_period_end IS NULL OR us.current_period_end > NOW())
	ORDER BY us.created_at DESC
	LIMIT 1
	`

	var subscription models.UserSubscription
	var metadataJSON []byte
	var planName, planCode sql.NullString

	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&subscription.ID,
		&subscription.UserID,
		&subscription.PlanID,
		&subscription.PaymentID,
		&subscription.StripeSubscriptionID,
		&subscription.Status,
		&subscription.CurrentPeriodStart,
		&subscription.CurrentPeriodEnd,
		&subscription.CancelAtPeriodEnd,
		&metadataJSON,
		&subscription.CreatedAt,
		&subscription.UpdatedAt,
		&planName,
		&planCode,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("ошибка получения активной подписки пользователя %d: %w", userID, err)
	}

	// Восстанавливаем metadata
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &subscription.Metadata); err != nil {
			return nil, fmt.Errorf("ошибка десериализации metadata: %w", err)
		}
	}

	// Восстанавливаем plan_name и plan_code
	if planName.Valid {
		subscription.PlanName = planName.String
	}
	if planCode.Valid {
		subscription.PlanCode = planCode.String
	}

	return &subscription, nil
}

// ✅ БЕЗ ИЗМЕНЕНИЙ
// Update обновляет подписку
func (r *subscriptionRepositoryImpl) Update(ctx context.Context, subscription *models.UserSubscription) error {
	query := `
	UPDATE user_subscriptions SET
		user_id = :user_id,
		plan_id = :plan_id,
		payment_id = :payment_id,
		stripe_subscription_id = :stripe_subscription_id,
		status = :status,
		current_period_start = :current_period_start,
		current_period_end = :current_period_end,
		cancel_at_period_end = :cancel_at_period_end,
		metadata = :metadata,
		updated_at = NOW()
	WHERE id = :id
	`

	// Преобразуем metadata в JSON
	metadataJSON, err := json.Marshal(subscription.Metadata)
	if err != nil {
		return fmt.Errorf("ошибка сериализации metadata: %w", err)
	}

	result, err := sqlx.NamedExecContext(ctx, r.db, query, map[string]interface{}{
		"id":                     subscription.ID,
		"user_id":                subscription.UserID,
		"plan_id":                subscription.PlanID,
		"payment_id":             subscription.PaymentID,
		"stripe_subscription_id": subscription.StripeSubscriptionID,
		"status":                 subscription.Status,
		"current_period_start":   subscription.CurrentPeriodStart,
		"current_period_end":     subscription.CurrentPeriodEnd,
		"cancel_at_period_end":   subscription.CancelAtPeriodEnd,
		"metadata":               metadataJSON,
	})
	if err != nil {
		return fmt.Errorf("ошибка обновления подписки %d: %w", subscription.ID, err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("подписка с ID %d не найдена", subscription.ID)
	}

	return nil
}

// ✅ БЕЗ ИЗМЕНЕНИЙ
// UpdateStatus обновляет статус подписки
func (r *subscriptionRepositoryImpl) UpdateStatus(ctx context.Context, id int, status string) error {
	query := `
	UPDATE user_subscriptions
	SET status = $1, updated_at = NOW()
	WHERE id = $2
	`

	result, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("ошибка обновления статуса подписки %d: %w", id, err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("подписка с ID %d не найдена", id)
	}

	return nil
}

// ✅ БЕЗ ИЗМЕНЕНИЙ
// Cancel отменяет подписку
func (r *subscriptionRepositoryImpl) Cancel(ctx context.Context, id int, cancelAtPeriodEnd bool) error {
	query := `
	UPDATE user_subscriptions
	SET cancel_at_period_end = $1, updated_at = NOW()
	WHERE id = $2
	`

	result, err := r.db.ExecContext(ctx, query, cancelAtPeriodEnd, id)
	if err != nil {
		return fmt.Errorf("ошибка отмены подписки %d: %w", id, err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("подписка с ID %d не найдена", id)
	}

	return nil
}

// ✅ БЕЗ ИЗМЕНЕНИЙ
// GetExpiringSubscriptions получает подписки, срок которых истекает в течение N дней
func (r *subscriptionRepositoryImpl) GetExpiringSubscriptions(ctx context.Context, daysBefore int) ([]*models.UserSubscription, error) {
	query := `
	SELECT us.*
	FROM user_subscriptions us
	WHERE us.status IN ('active', 'trialing')
	AND us.current_period_end IS NOT NULL
	AND us.current_period_end BETWEEN NOW() AND NOW() + INTERVAL '1 day' * $1
	AND us.cancel_at_period_end = false
	ORDER BY us.current_period_end ASC
	`

	rows, err := r.db.QueryContext(ctx, query, daysBefore)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения истекающих подписок: %w", err)
	}
	defer rows.Close()

	var subscriptions []*models.UserSubscription
	for rows.Next() {
		var subscription models.UserSubscription
		var metadataJSON []byte

		if err := rows.Scan(
			&subscription.ID,
			&subscription.UserID,
			&subscription.PlanID,
			&subscription.PaymentID,
			&subscription.StripeSubscriptionID,
			&subscription.Status,
			&subscription.CurrentPeriodStart,
			&subscription.CurrentPeriodEnd,
			&subscription.CancelAtPeriodEnd,
			&metadataJSON,
			&subscription.CreatedAt,
			&subscription.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("ошибка сканирования подписки: %w", err)
		}

		// Восстанавливаем metadata
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &subscription.Metadata); err != nil {
				return nil, fmt.Errorf("ошибка десериализации metadata: %w", err)
			}
		}

		subscriptions = append(subscriptions, &subscription)
	}

	return subscriptions, nil
}

// ✅ БЕЗ ИЗМЕНЕНИЙ
// GetExpiredSubscriptions получает истекшие подписки
func (r *subscriptionRepositoryImpl) GetExpiredSubscriptions(ctx context.Context) ([]*models.UserSubscription, error) {
	query := `
	SELECT us.*
	FROM user_subscriptions us
	WHERE us.status IN ('active', 'trialing')
	AND us.current_period_end IS NOT NULL
	AND us.current_period_end < NOW()
	AND us.cancel_at_period_end = false
	ORDER BY us.current_period_end ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения истекших подписок: %w", err)
	}
	defer rows.Close()

	var subscriptions []*models.UserSubscription
	for rows.Next() {
		var subscription models.UserSubscription
		var metadataJSON []byte

		if err := rows.Scan(
			&subscription.ID,
			&subscription.UserID,
			&subscription.PlanID,
			&subscription.PaymentID,
			&subscription.StripeSubscriptionID,
			&subscription.Status,
			&subscription.CurrentPeriodStart,
			&subscription.CurrentPeriodEnd,
			&subscription.CancelAtPeriodEnd,
			&metadataJSON,
			&subscription.CreatedAt,
			&subscription.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("ошибка сканирования подписки: %w", err)
		}

		// Восстанавливаем metadata
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &subscription.Metadata); err != nil {
				return nil, fmt.Errorf("ошибка десериализации metadata: %w", err)
			}
		}

		subscriptions = append(subscriptions, &subscription)
	}

	return subscriptions, nil
}

// ✅ БЕЗ ИЗМЕНЕНИЙ
// GetByPaymentID получает подписку по ID платежа
func (r *subscriptionRepositoryImpl) GetByPaymentID(ctx context.Context, paymentID int64) (*models.UserSubscription, error) {
	query := `
	SELECT us.*
	FROM user_subscriptions us
	WHERE us.payment_id = $1
	ORDER BY us.created_at DESC
	LIMIT 1
	`

	var subscription models.UserSubscription
	var metadataJSON []byte

	if err := r.db.QueryRowContext(ctx, query, paymentID).Scan(
		&subscription.ID,
		&subscription.UserID,
		&subscription.PlanID,
		&subscription.PaymentID,
		&subscription.StripeSubscriptionID,
		&subscription.Status,
		&subscription.CurrentPeriodStart,
		&subscription.CurrentPeriodEnd,
		&subscription.CancelAtPeriodEnd,
		&metadataJSON,
		&subscription.CreatedAt,
		&subscription.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("ошибка получения подписки по payment_id %d: %w", paymentID, err)
	}

	// Восстанавливаем metadata
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &subscription.Metadata); err != nil {
			return nil, fmt.Errorf("ошибка десериализации metadata: %w", err)
		}
	}

	return &subscription, nil
}
