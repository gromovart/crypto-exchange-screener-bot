// internal/infrastructure/persistence/postgres/repository/invoice/repository.go
package invoice

import (
	"context"
	"fmt"
	"time"

	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"

	"github.com/jmoiron/sqlx"
)

// Repository интерфейс репозитория инвойсов
type InvoiceRepository interface {
	Create(ctx context.Context, invoice *models.Invoice) error
	GetByID(ctx context.Context, id int64) (*models.Invoice, error)
	GetByExternalID(ctx context.Context, externalID string) (*models.Invoice, error)
	GetByUserID(ctx context.Context, userID int64, filter models.InvoiceFilter) ([]*models.Invoice, error)
	GetByPayload(ctx context.Context, payload string) (*models.Invoice, error)
	UpdateStatus(ctx context.Context, id int64, status models.InvoiceStatus) error
	Update(ctx context.Context, invoice *models.Invoice) error
	Delete(ctx context.Context, id int64) error
	GetSummary(ctx context.Context, userID int64) (*models.InvoiceSummary, error)
	GetActiveInvoices(ctx context.Context, userID int64) ([]*models.Invoice, error)
	GetExpiredInvoices(ctx context.Context, expireBefore time.Time) ([]*models.Invoice, error)
}

// repositoryImpl реализация Repository
type repositoryImpl struct {
	db *sqlx.DB
}

// NewRepository создает новый репозиторий инвойсов
func NewInvoiceRepository(db *sqlx.DB) InvoiceRepository {
	return &repositoryImpl{db: db}
}

// Create создает новый инвойс
func (r *repositoryImpl) Create(ctx context.Context, invoice *models.Invoice) error {
	query := `
	INSERT INTO invoices (
		user_id, plan_id,
		external_id, title, description,
		amount_usd, stars_amount, fiat_amount, currency,
		status, provider,
		invoice_url, payload, metadata,
		expires_at
	) VALUES (
		:user_id, :plan_id,
		:external_id, :title, :description,
		:amount_usd, :stars_amount, :fiat_amount, :currency,
		:status, :provider,
		:invoice_url, :payload, :metadata,
		:expires_at
	) RETURNING id, created_at, updated_at
	`

	rows, err := sqlx.NamedQueryContext(ctx, r.db, query, invoice)
	if err != nil {
		return fmt.Errorf("ошибка создания инвойса: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		if err := rows.Scan(&invoice.ID, &invoice.CreatedAt, &invoice.UpdatedAt); err != nil {
			return fmt.Errorf("ошибка сканирования результата: %w", err)
		}
	}

	return nil
}

// GetByID получает инвойс по ID
func (r *repositoryImpl) GetByID(ctx context.Context, id int64) (*models.Invoice, error) {
	query := `
	SELECT * FROM invoices WHERE id = $1
	`

	var invoice models.Invoice
	if err := r.db.GetContext(ctx, &invoice, query, id); err != nil {
		return nil, fmt.Errorf("ошибка получения инвойса по ID %d: %w", id, err)
	}

	return &invoice, nil
}

// GetByExternalID получает инвойс по внешнему ID
func (r *repositoryImpl) GetByExternalID(ctx context.Context, externalID string) (*models.Invoice, error) {
	query := `
	SELECT * FROM invoices WHERE external_id = $1
	`

	var invoice models.Invoice
	if err := r.db.GetContext(ctx, &invoice, query, externalID); err != nil {
		return nil, fmt.Errorf("ошибка получения инвойса по external_id %s: %w", externalID, err)
	}

	return &invoice, nil
}

// GetByPayload получает инвойс по payload
func (r *repositoryImpl) GetByPayload(ctx context.Context, payload string) (*models.Invoice, error) {
	query := `
	SELECT * FROM invoices WHERE payload = $1
	`

	var invoice models.Invoice
	if err := r.db.GetContext(ctx, &invoice, query, payload); err != nil {
		return nil, fmt.Errorf("ошибка получения инвойса по payload %s: %w", payload, err)
	}

	return &invoice, nil
}

// GetByUserID получает инвойсы пользователя с фильтрацией
func (r *repositoryImpl) GetByUserID(ctx context.Context, userID int64, filter models.InvoiceFilter) ([]*models.Invoice, error) {
	query := `
	SELECT * FROM invoices
	WHERE user_id = $1
	`
	args := []interface{}{userID}
	argIndex := 2

	// Добавляем фильтры
	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, filter.Status)
		argIndex++
	}

	if filter.Provider != "" {
		query += fmt.Sprintf(" AND provider = $%d", argIndex)
		args = append(args, filter.Provider)
		argIndex++
	}

	if filter.PlanID != "" {
		query += fmt.Sprintf(" AND plan_id = $%d", argIndex)
		args = append(args, filter.PlanID)
		argIndex++
	}

	if filter.StartDate != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, filter.StartDate)
		argIndex++
	}

	if filter.EndDate != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		args = append(args, filter.EndDate)
		argIndex++
	}

	// Сортировка и лимит
	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++

		if filter.Offset > 0 {
			query += fmt.Sprintf(" OFFSET $%d", argIndex)
			args = append(args, filter.Offset)
		}
	}

	var invoices []*models.Invoice
	if err := r.db.SelectContext(ctx, &invoices, query, args...); err != nil {
		return nil, fmt.Errorf("ошибка получения инвойсов пользователя %d: %w", userID, err)
	}

	return invoices, nil
}

// UpdateStatus обновляет статус инвойса
func (r *repositoryImpl) UpdateStatus(ctx context.Context, id int64, status models.InvoiceStatus) error {
	query := `
	UPDATE invoices
	SET status = $1, updated_at = NOW()
	WHERE id = $2
	`

	result, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("ошибка обновления статуса инвойса %d: %w", id, err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("инвойс с ID %d не найден", id)
	}

	return nil
}

// Update обновляет инвойс
func (r *repositoryImpl) Update(ctx context.Context, invoice *models.Invoice) error {
	query := `
	UPDATE invoices SET
		user_id = :user_id,
		plan_id = :plan_id,
		external_id = :external_id,
		title = :title,
		description = :description,
		amount_usd = :amount_usd,
		stars_amount = :stars_amount,
		fiat_amount = :fiat_amount,
		currency = :currency,
		status = :status,
		provider = :provider,
		invoice_url = :invoice_url,
		payload = :payload,
		metadata = :metadata,
		expires_at = :expires_at,
		paid_at = :paid_at,
		updated_at = NOW()
	WHERE id = :id
	`

	result, err := sqlx.NamedExecContext(ctx, r.db, query, invoice)
	if err != nil {
		return fmt.Errorf("ошибка обновления инвойса %d: %w", invoice.ID, err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("инвойс с ID %d не найден", invoice.ID)
	}

	return nil
}

// Delete удаляет инвойс
func (r *repositoryImpl) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM invoices WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("ошибка удаления инвойса %d: %w", id, err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("инвойс с ID %d не найден", id)
	}

	return nil
}

// GetSummary получает сводку по инвойсам пользователя
func (r *repositoryImpl) GetSummary(ctx context.Context, userID int64) (*models.InvoiceSummary, error) {
	query := `
	SELECT
		COUNT(*) as total_invoices,
		COALESCE(SUM(amount_usd), 0) as total_amount_usd,
		COUNT(CASE WHEN status = 'paid' THEN 1 END) as paid_count,
		COUNT(CASE WHEN status IN ('created', 'pending') THEN 1 END) as pending_count,
		COUNT(CASE WHEN status = 'expired' THEN 1 END) as expired_count
	FROM invoices
	WHERE user_id = $1
	`

	var summary models.InvoiceSummary
	if err := r.db.GetContext(ctx, &summary, query, userID); err != nil {
		return nil, fmt.Errorf("ошибка получения сводки инвойсов пользователя %d: %w", userID, err)
	}

	return &summary, nil
}

// GetActiveInvoices получает активные инвойсы пользователя
func (r *repositoryImpl) GetActiveInvoices(ctx context.Context, userID int64) ([]*models.Invoice, error) {
	query := `
	SELECT * FROM invoices
	WHERE user_id = $1
	AND status IN ('created', 'pending')
	AND expires_at > NOW()
	ORDER BY created_at DESC
	`

	var invoices []*models.Invoice
	if err := r.db.SelectContext(ctx, &invoices, query, userID); err != nil {
		return nil, fmt.Errorf("ошибка получения активных инвойсов пользователя %d: %w", userID, err)
	}

	return invoices, nil
}

// GetExpiredInvoices получает просроченные инвойсы
func (r *repositoryImpl) GetExpiredInvoices(ctx context.Context, expireBefore time.Time) ([]*models.Invoice, error) {
	query := `
	SELECT * FROM invoices
	WHERE status IN ('created', 'pending')
	AND expires_at < $1
	ORDER BY expires_at ASC
	`

	var invoices []*models.Invoice
	if err := r.db.SelectContext(ctx, &invoices, query, expireBefore); err != nil {
		return nil, fmt.Errorf("ошибка получения просроченных инвойсов: %w", err)
	}

	return invoices, nil
}

// MarkAsPaid помечает инвойс как оплаченный
func (r *repositoryImpl) MarkAsPaid(ctx context.Context, id int64, paidAt time.Time) error {
	query := `
	UPDATE invoices
	SET status = 'paid', paid_at = $1, updated_at = NOW()
	WHERE id = $2
	`

	result, err := r.db.ExecContext(ctx, query, paidAt, id)
	if err != nil {
		return fmt.Errorf("ошибка отметки инвойса %d как оплаченного: %w", id, err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("инвойс с ID %d не найден", id)
	}

	return nil
}

// MarkAsExpired помечает инвойс как просроченный
func (r *repositoryImpl) MarkAsExpired(ctx context.Context, id int64) error {
	query := `
	UPDATE invoices
	SET status = 'expired', updated_at = NOW()
	WHERE id = $1 AND status IN ('created', 'pending')
	`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("ошибка отметки инвойса %d как просроченного: %w", id, err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("инвойс с ID %d не найден или уже не активен", id)
	}

	return nil
}
