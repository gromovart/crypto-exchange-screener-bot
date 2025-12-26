// internal/subscription/repository.go
package subscription

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// Repository репозиторий подписок
type Repository struct {
	db *sqlx.DB
}

// NewRepository создает новый репозиторий
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// CreatePlan создает новый тарифный план
func (r *Repository) CreatePlan(plan *Plan) error {
	query := `
	INSERT INTO subscription_plans (
		name, code, description, price_monthly, price_yearly,
		max_symbols, max_signals_per_day, features, is_active
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	RETURNING id, created_at
	`

	featuresJSON, err := json.Marshal(plan.Features)
	if err != nil {
		return fmt.Errorf("failed to marshal features: %w", err)
	}

	return r.db.QueryRow(
		query,
		plan.Name, plan.Code, plan.Description,
		plan.PriceMonthly, plan.PriceYearly,
		plan.MaxSymbols, plan.MaxSignalsPerDay,
		featuresJSON, plan.IsActive,
	).Scan(&plan.ID, &plan.CreatedAt)
}

// GetPlanByCode получает план по коду
func (r *Repository) GetPlanByCode(code string) (*Plan, error) {
	query := `
	SELECT id, name, code, description, price_monthly, price_yearly,
		   max_symbols, max_signals_per_day, features, is_active, created_at
	FROM subscription_plans
	WHERE code = $1 AND is_active = TRUE
	`

	var plan Plan
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

	return &plan, nil
}

// GetAllPlans возвращает все активные планы
func (r *Repository) GetAllPlans() ([]*Plan, error) {
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

	var plans []*Plan

	for rows.Next() {
		var plan Plan
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

	return plans, nil
}

// CreateSubscription создает новую подписку
func (r *Repository) CreateSubscription(sub *UserSubscription) error {
	query := `
	INSERT INTO user_subscriptions (
		user_id, plan_id, stripe_subscription_id, status,
		current_period_start, current_period_end,
		cancel_at_period_end, metadata
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	RETURNING id, created_at
	`

	metadataJSON, err := json.Marshal(sub.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	return r.db.QueryRow(
		query,
		sub.UserID, sub.PlanID, sub.StripeSubscriptionID, sub.Status,
		sub.CurrentPeriodStart, sub.CurrentPeriodEnd,
		sub.CancelAtPeriodEnd, metadataJSON,
	).Scan(&sub.ID, &sub.CreatedAt)
}

// GetActiveSubscription получает активную подписку пользователя
func (r *Repository) GetActiveSubscription(userID int) (*UserSubscription, error) {
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

	var sub UserSubscription
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

	return &sub, nil
}

// UpdateSubscriptionStatus обновляет статус подписки
func (r *Repository) UpdateSubscriptionStatus(
	subscriptionID, stripeSubscriptionID string,
	status string, currentPeriodEnd time.Time,
) error {

	query := `
	UPDATE user_subscriptions
	SET status = $1,
		current_period_end = $2,
		updated_at = NOW()
	WHERE id = $3 OR stripe_subscription_id = $4
	`

	result, err := r.db.Exec(
		query, status, currentPeriodEnd, subscriptionID, stripeSubscriptionID,
	)

	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// CancelSubscription отменяет подписку
func (r *Repository) CancelSubscription(userID int, cancelAtPeriodEnd bool) error {
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

	return nil
}

// UpdateUserSubscriptionTier обновляет тариф пользователя
func (r *Repository) UpdateUserSubscriptionTier(userID int, tier string) error {
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

	return nil
}
