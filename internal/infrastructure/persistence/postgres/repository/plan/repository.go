// internal/infrastructure/persistence/postgres/repository/plan/repository.go
package plan

import (
	"context"
	"encoding/json"
	"fmt"

	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"

	"github.com/jmoiron/sqlx"
)

// PlanRepository интерфейс репозитория планов подписки
type PlanRepository interface {
	GetByID(ctx context.Context, id int) (*models.Plan, error)
	GetByCode(ctx context.Context, code string) (*models.Plan, error)
	GetAllActive(ctx context.Context) ([]*models.Plan, error)
	Create(ctx context.Context, plan *models.Plan) error
	Update(ctx context.Context, plan *models.Plan) error
	Delete(ctx context.Context, id int) error
	GetByPriceRange(ctx context.Context, minPrice, maxPrice float64) ([]*models.Plan, error)
	GetDefaultPlan(ctx context.Context) (*models.Plan, error)
	GetFeaturesByPlanCode(ctx context.Context, code string) (map[string]interface{}, error)
	UpdateStarsPrices(ctx context.Context, id int, monthlyStars, yearlyStars int) error
	GetPlanLimits(ctx context.Context, code string) (*models.PlanLimits, error)
}

// planRepositoryImpl реализация PlanRepository
type planRepositoryImpl struct {
	db *sqlx.DB
}

// NewPlanRepository создает новый репозиторий планов
func NewPlanRepository(db *sqlx.DB) PlanRepository {
	return &planRepositoryImpl{db: db}
}

// GetByID получает план по ID
func (r *planRepositoryImpl) GetByID(ctx context.Context, id int) (*models.Plan, error) {
	query := `
	SELECT * FROM subscription_plans
	WHERE id = $1 AND is_active = true
	`

	var plan models.Plan
	if err := r.db.GetContext(ctx, &plan, query, id); err != nil {
		return nil, fmt.Errorf("ошибка получения плана по ID %d: %w", id, err)
	}

	return &plan, nil
}

// GetByCode получает план по коду
func (r *planRepositoryImpl) GetByCode(ctx context.Context, code string) (*models.Plan, error) {
	query := `
	SELECT * FROM subscription_plans
	WHERE code = $1 AND is_active = true
	`

	var plan models.Plan
	if err := r.db.GetContext(ctx, &plan, query, code); err != nil {
		return nil, fmt.Errorf("ошибка получения плана по коду %s: %w", code, err)
	}

	return &plan, nil
}

// GetAllActive получает все активные планы
func (r *planRepositoryImpl) GetAllActive(ctx context.Context) ([]*models.Plan, error) {
	query := `
	SELECT * FROM subscription_plans
	WHERE is_active = true
	ORDER BY price_monthly ASC
	`

	var plans []*models.Plan
	if err := r.db.SelectContext(ctx, &plans, query); err != nil {
		return nil, fmt.Errorf("ошибка получения активных планов: %w", err)
	}

	return plans, nil
}

// Create создает новый план
func (r *planRepositoryImpl) Create(ctx context.Context, plan *models.Plan) error {
	// Сериализуем features в JSON
	featuresJSON, err := json.Marshal(plan.Features)
	if err != nil {
		return fmt.Errorf("ошибка сериализации features: %w", err)
	}

	query := `
    INSERT INTO subscription_plans (
        name, code, description,
        price_monthly, price_yearly,
        stars_price_monthly, stars_price_yearly,
        max_symbols, max_signals_per_day,
        features, is_active
    ) VALUES (
        :name, :code, :description,
        :price_monthly, :price_yearly,
        :stars_price_monthly, :stars_price_yearly,
        :max_symbols, :max_signals_per_day,
        :features, :is_active
    ) RETURNING id, created_at
    `

	// Используем map для передачи параметров
	params := map[string]interface{}{
		"name":                plan.Name,
		"code":                plan.Code,
		"description":         plan.Description,
		"price_monthly":       plan.PriceMonthly,
		"price_yearly":        plan.PriceYearly,
		"stars_price_monthly": plan.StarsPriceMonthly,
		"stars_price_yearly":  plan.StarsPriceYearly,
		"max_symbols":         plan.MaxSymbols,
		"max_signals_per_day": plan.MaxSignalsPerDay,
		"features":            featuresJSON,
		"is_active":           plan.IsActive,
	}

	rows, err := sqlx.NamedQueryContext(ctx, r.db, query, params)
	if err != nil {
		return fmt.Errorf("ошибка создания плана: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		if err := rows.Scan(&plan.ID, &plan.CreatedAt); err != nil {
			return fmt.Errorf("ошибка сканирования результата: %w", err)
		}
	}

	return nil
}

// Update обновляет план
func (r *planRepositoryImpl) Update(ctx context.Context, plan *models.Plan) error {
	query := `
	UPDATE subscription_plans SET
		name = :name,
		description = :description,
		price_monthly = :price_monthly,
		price_yearly = :price_yearly,
		stars_price_monthly = :stars_price_monthly,
		stars_price_yearly = :stars_price_yearly,
		max_symbols = :max_symbols,
		max_signals_per_day = :max_signals_per_day,
		features = :features,
		is_active = :is_active
	WHERE id = :id
	`

	result, err := sqlx.NamedExecContext(ctx, r.db, query, plan)
	if err != nil {
		return fmt.Errorf("ошибка обновления плана %d: %w", plan.ID, err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("план с ID %d не найден", plan.ID)
	}

	return nil
}

// Delete удаляет план (помечает как неактивный)
func (r *planRepositoryImpl) Delete(ctx context.Context, id int) error {
	query := `
	UPDATE subscription_plans
	SET is_active = false
	WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("ошибка удаления плана %d: %w", id, err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("план с ID %d не найден", id)
	}

	return nil
}

// GetByPriceRange получает планы в диапазоне цен
func (r *planRepositoryImpl) GetByPriceRange(ctx context.Context, minPrice, maxPrice float64) ([]*models.Plan, error) {
	query := `
	SELECT * FROM subscription_plans
	WHERE price_monthly BETWEEN $1 AND $2
	AND is_active = true
	ORDER BY price_monthly ASC
	`

	var plans []*models.Plan
	if err := r.db.SelectContext(ctx, &plans, query, minPrice, maxPrice); err != nil {
		return nil, fmt.Errorf("ошибка получения планов по диапазону цен: %w", err)
	}

	return plans, nil
}

// GetDefaultPlan получает план по умолчанию (самый дешевый)
func (r *planRepositoryImpl) GetDefaultPlan(ctx context.Context) (*models.Plan, error) {
	query := `
	SELECT * FROM subscription_plans
	WHERE is_active = true
	ORDER BY price_monthly ASC
	LIMIT 1
	`

	var plan models.Plan
	if err := r.db.GetContext(ctx, &plan, query); err != nil {
		return nil, fmt.Errorf("ошибка получения плана по умолчанию: %w", err)
	}

	return &plan, nil
}

// GetFeaturesByPlanCode получает фичи плана по коду
func (r *planRepositoryImpl) GetFeaturesByPlanCode(ctx context.Context, code string) (map[string]interface{}, error) {
	query := `
	SELECT features FROM subscription_plans
	WHERE code = $1 AND is_active = true
	`

	var features map[string]interface{}
	if err := r.db.GetContext(ctx, &features, query, code); err != nil {
		return nil, fmt.Errorf("ошибка получения фич плана %s: %w", code, err)
	}

	return features, nil
}

// UpdateStarsPrices обновляет цены в Stars
func (r *planRepositoryImpl) UpdateStarsPrices(ctx context.Context, id int, monthlyStars, yearlyStars int) error {
	query := `
	UPDATE subscription_plans
	SET stars_price_monthly = $1,
	    stars_price_yearly = $2
	WHERE id = $3
	`

	result, err := r.db.ExecContext(ctx, query, monthlyStars, yearlyStars, id)
	if err != nil {
		return fmt.Errorf("ошибка обновления цен Stars плана %d: %w", id, err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("план с ID %d не найден", id)
	}

	return nil
}

// GetPlanLimits получает лимиты плана по коду
func (r *planRepositoryImpl) GetPlanLimits(ctx context.Context, code string) (*models.PlanLimits, error) {
	plan, err := r.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	limits := &models.PlanLimits{
		MaxSymbols:       plan.MaxSymbols,
		MaxSignalsPerDay: plan.MaxSignalsPerDay,
		MaxAPIRequests:   1000, // Пример значения, можно брать из features
		Features:         plan.Features,
	}

	return limits, nil
}
