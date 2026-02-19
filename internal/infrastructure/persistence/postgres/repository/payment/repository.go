// internal/infrastructure/persistence/postgres/repository/payment/repository.go
package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"crypto-exchange-screener-bot/internal/core/domain/subscription"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"

	"github.com/jmoiron/sqlx"
)

// PaymentRepository интерфейс репозитория платежей
type PaymentRepository interface {
	Create(ctx context.Context, payment *models.Payment) error
	GetByID(ctx context.Context, id int64) (*models.Payment, error)
	GetByExternalID(ctx context.Context, externalID string) (*models.Payment, error)
	GetByUserID(ctx context.Context, userID int64, filter models.PaymentFilter) ([]*models.Payment, error)
	UpdateStatus(ctx context.Context, id int64, status models.PaymentStatus) error
	Update(ctx context.Context, payment *models.Payment) error
	Delete(ctx context.Context, id int64) error
	GetSummary(ctx context.Context, userID int64) (*models.PaymentSummary, error)
	GetSuccessfulPayments(ctx context.Context, days int) ([]*subscription.PaymentData, error)
}

// paymentRepositoryImpl реализация PaymentRepository
type paymentRepositoryImpl struct {
	db *sqlx.DB
}

// NewPaymentRepository создает новый репозиторий платежей
func NewPaymentRepository(db *sqlx.DB) PaymentRepository {
	return &paymentRepositoryImpl{db: db}
}

// Create создает новый платеж
func (r *paymentRepositoryImpl) Create(ctx context.Context, payment *models.Payment) error {
	// Сериализуем Metadata в JSON если есть
	var metadataJSON []byte
	if payment.Metadata != nil {
		var err error
		metadataJSON, err = json.Marshal(payment.Metadata)
		if err != nil {
			return fmt.Errorf("ошибка сериализации metadata: %w", err)
		}
	} else {
		metadataJSON = []byte("{}")
	}

	query := `
	INSERT INTO payments (
		user_id, subscription_id, invoice_id,
		external_id, amount, currency, stars_amount, fiat_amount,
		payment_type, status, provider,
		description, payload, metadata,
		paid_at, expires_at
	) VALUES (
		$1, $2, $3,
		$4, $5, $6, $7, $8,
		$9, $10, $11,
		$12, $13, $14,
		$15, $16
	) RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		payment.UserID,
		payment.SubscriptionID,
		payment.InvoiceID,
		payment.ExternalID,
		payment.Amount,
		payment.Currency,
		payment.StarsAmount,
		payment.FiatAmount,
		payment.PaymentType,
		payment.Status,
		payment.Provider,
		payment.Description,
		payment.Payload,
		metadataJSON,
		payment.PaidAt,
		payment.ExpiresAt,
	).Scan(&payment.ID, &payment.CreatedAt, &payment.UpdatedAt)

	if err != nil {
		return fmt.Errorf("ошибка создания платежа: %w", err)
	}

	return nil
}

// GetByID получает платеж по ID
func (r *paymentRepositoryImpl) GetByID(ctx context.Context, id int64) (*models.Payment, error) {
	query := `
	SELECT * FROM payments WHERE id = $1
	`

	var payment models.Payment
	if err := r.db.GetContext(ctx, &payment, query, id); err != nil {
		return nil, fmt.Errorf("ошибка получения платежа по ID %d: %w", id, err)
	}

	return &payment, nil
}

// GetByExternalID получает платеж по внешнему ID (TelegramPaymentID)
func (r *paymentRepositoryImpl) GetByExternalID(ctx context.Context, externalID string) (*models.Payment, error) {
	query := `
	SELECT * FROM payments WHERE external_id = $1
	`

	var payment models.Payment
	if err := r.db.GetContext(ctx, &payment, query, externalID); err != nil {
		return nil, fmt.Errorf("ошибка получения платежа по external_id %s: %w", externalID, err)
	}

	return &payment, nil
}

// GetByUserID получает платежи пользователя с фильтрацией
func (r *paymentRepositoryImpl) GetByUserID(ctx context.Context, userID int64, filter models.PaymentFilter) ([]*models.Payment, error) {
	query := `
	SELECT * FROM payments
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

	if filter.PaymentType != "" {
		query += fmt.Sprintf(" AND payment_type = $%d", argIndex)
		args = append(args, filter.PaymentType)
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

	var payments []*models.Payment
	if err := r.db.SelectContext(ctx, &payments, query, args...); err != nil {
		return nil, fmt.Errorf("ошибка получения платежей пользователя %d: %w", userID, err)
	}

	return payments, nil
}

// UpdateStatus обновляет статус платежа
func (r *paymentRepositoryImpl) UpdateStatus(ctx context.Context, id int64, status models.PaymentStatus) error {
	query := `
	UPDATE payments
	SET status = $1, updated_at = NOW()
	WHERE id = $2
	`

	result, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("ошибка обновления статуса платежа %d: %w", id, err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("платеж с ID %d не найден", id)
	}

	return nil
}

// Update обновляет платеж
func (r *paymentRepositoryImpl) Update(ctx context.Context, payment *models.Payment) error {
	query := `
	UPDATE payments SET
		user_id = :user_id,
		subscription_id = :subscription_id,
		invoice_id = :invoice_id,
		external_id = :external_id,
		amount = :amount,
		currency = :currency,
		stars_amount = :stars_amount,
		fiat_amount = :fiat_amount,
		payment_type = :payment_type,
		status = :status,
		provider = :provider,
		description = :description,
		payload = :payload,
		metadata = :metadata,
		paid_at = :paid_at,
		expires_at = :expires_at,
		updated_at = NOW()
	WHERE id = :id
	`

	result, err := sqlx.NamedExecContext(ctx, r.db, query, payment)
	if err != nil {
		return fmt.Errorf("ошибка обновления платежа %d: %w", payment.ID, err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("платеж с ID %d не найден", payment.ID)
	}

	return nil
}

// Delete удаляет платеж
func (r *paymentRepositoryImpl) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM payments WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("ошибка удаления платежа %d: %w", id, err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("платеж с ID %d не найден", id)
	}

	return nil
}

// GetSummary получает сводку по платежам пользователя
func (r *paymentRepositoryImpl) GetSummary(ctx context.Context, userID int64) (*models.PaymentSummary, error) {
	query := `
	SELECT
		COUNT(*) as total_payments,
		COALESCE(SUM(amount), 0) as total_amount,
		COUNT(CASE WHEN status = 'completed' THEN 1 END) as successful_count,
		COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed_count,
		COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending_count
	FROM payments
	WHERE user_id = $1
	`

	var summary models.PaymentSummary
	if err := r.db.GetContext(ctx, &summary, query, userID); err != nil {
		return nil, fmt.Errorf("ошибка получения сводки платежей пользователя %d: %w", userID, err)
	}

	return &summary, nil
}

// MarkAsPaid помечает платеж как оплаченный
func (r *paymentRepositoryImpl) MarkAsPaid(ctx context.Context, id int64, paidAt time.Time) error {
	query := `
	UPDATE payments
	SET status = 'completed', paid_at = $1, updated_at = NOW()
	WHERE id = $2
	`

	result, err := r.db.ExecContext(ctx, query, paidAt, id)
	if err != nil {
		return fmt.Errorf("ошибка отметки платежа %d как оплаченного: %w", id, err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("платеж с ID %d не найден", id)
	}

	return nil
}

// GetPendingPayments получает ожидающие платежи
func (r *paymentRepositoryImpl) GetPendingPayments(ctx context.Context, expireBefore time.Time) ([]*models.Payment, error) {
	query := `
	SELECT * FROM payments
	WHERE status = 'pending'
	AND (expires_at IS NULL OR expires_at < $1)
	ORDER BY created_at ASC
	`

	var payments []*models.Payment
	if err := r.db.SelectContext(ctx, &payments, query, expireBefore); err != nil {
		return nil, fmt.Errorf("ошибка получения ожидающих платежей: %w", err)
	}

	return payments, nil
}

// GetByInvoiceID получает платеж по ID инвойса
func (r *paymentRepositoryImpl) GetByInvoiceID(ctx context.Context, invoiceID int64) (*models.Payment, error) {
	query := `
	SELECT * FROM payments WHERE invoice_id = $1
	`

	var payment models.Payment
	if err := r.db.GetContext(ctx, &payment, query, invoiceID); err != nil {
		return nil, fmt.Errorf("ошибка получения платежа по invoice_id %d: %w", invoiceID, err)
	}

	return &payment, nil
}

// GetSuccessfulPayments получает успешные платежи за последние N дней
func (r *paymentRepositoryImpl) GetSuccessfulPayments(ctx context.Context, days int) ([]*subscription.PaymentData, error) {
	query := `
	SELECT
		id,
		user_id,
		metadata->>'plan_code' as plan_code,
		amount,
		created_at
	FROM payments
	WHERE status = 'completed'
	AND created_at >= NOW() - INTERVAL '1 day' * $1
	ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, days)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения успешных платежей за %d дней: %w", days, err)
	}
	defer rows.Close()

	var result []*subscription.PaymentData
	for rows.Next() {
		var data subscription.PaymentData
		var planCode string

		if err := rows.Scan(
			&data.ID,
			&data.UserID,
			&planCode,
			&data.Amount,
			&data.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("ошибка сканирования платежа: %w", err)
		}

		data.PlanCode = planCode
		result = append(result, &data)
	}

	return result, nil
}
