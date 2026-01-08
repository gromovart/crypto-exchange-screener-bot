// internal/infrastructure/persistence/postgres/repository/users/repository.go
package users

import (
	"context"
	"crypto-exchange-screener-bot/internal/infrastructure/cache/redis"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// UserRepository интерфейс для работы с данными пользователей
type UserRepository interface {
	FindByID(id int) (*models.User, error)
	FindByTelegramID(telegramID int64) (*models.User, error)
	FindByEmail(email string) (*models.User, error)
	FindByChatID(chatID string) (*models.User, error)
	Create(user *models.User) error
	Update(user *models.User) error
	Delete(id int) error
	UpdateLastLogin(userID int) error
	GetAllActive() ([]*models.User, error)
	SearchUsers(query string, limit, offset int) ([]*models.User, error)
	GetTotalCount(ctx context.Context) (int, error)
	IncrementSignalsCount(userID int) error
	ResetDailyCounters(ctx context.Context) error
}

// UserRepositoryImpl реализация репозитория пользователей
type UserRepositoryImpl struct {
	db    *sqlx.DB
	cache *redis.Cache
}

// NewUserRepository создает новый репозиторий пользователей
func NewUserRepository(db *sqlx.DB, cache *redis.Cache) *UserRepositoryImpl {
	return &UserRepositoryImpl{db: db, cache: cache}
}

// GetAllActive получает всех активных пользователей
func (r *UserRepositoryImpl) GetAllActive() ([]*models.User, error) {
	query := `
    SELECT
        id, telegram_id, username, first_name, last_name, chat_id,
        email, phone,
        notifications_enabled, notify_growth, notify_fall, notify_continuous,
        quiet_hours_start, quiet_hours_end,
        min_growth_threshold, min_fall_threshold,
        preferred_periods, min_volume_filter, exclude_patterns,
        language, timezone, display_mode,
        role, is_active, is_verified, subscription_tier,
        signals_today, max_signals_per_day,
        created_at, updated_at, last_login_at, last_signal_at
    FROM users
    WHERE is_active = TRUE
    ORDER BY created_at DESC
    `

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userList []*models.User

	for rows.Next() {
		user, err := r.scanUser(rows)
		if err != nil {
			return nil, err
		}
		userList = append(userList, user)
	}

	return userList, nil
}

// Create создает нового пользователя
func (r *UserRepositoryImpl) Create(user *models.User) error {
	// Начинаем транзакцию
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO users (
			telegram_id, username, first_name, last_name, chat_id,
			email, phone,
			notifications_enabled, notify_growth, notify_fall, notify_continuous,
			quiet_hours_start, quiet_hours_end,
			min_growth_threshold, min_fall_threshold,
			preferred_periods, min_volume_filter, exclude_patterns,
			language, timezone, display_mode,
			role, is_active, is_verified,
			subscription_tier, max_signals_per_day,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11,
			$12, $13,
			$14, $15,
			$16, $17, $18,
			$19, $20, $21,
			$22, $23, $24,
			$25, $26,
			$27, $28
		)
		RETURNING id
	`

	err = tx.QueryRow(
		query,
		user.TelegramID, user.Username, user.FirstName, user.LastName, user.ChatID,
		user.Email, user.Phone,
		user.NotificationsEnabled, user.NotifyGrowth, user.NotifyFall, user.NotifyContinuous,
		user.QuietHoursStart, user.QuietHoursEnd,
		user.MinGrowthThreshold, user.MinFallThreshold,
		pq.Array(user.PreferredPeriods), user.MinVolumeFilter, pq.Array(user.ExcludePatterns),
		user.Language, user.Timezone, user.DisplayMode,
		user.Role, user.IsActive, user.IsVerified,
		user.SubscriptionTier, user.MaxSignalsPerDay,
		user.CreatedAt, user.UpdatedAt,
	).Scan(&user.ID)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	// Фиксируем транзакцию
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// FindByID находит пользователя по ID
func (r *UserRepositoryImpl) FindByID(id int) (*models.User, error) {
	query := `
		SELECT
			id, telegram_id, username, first_name, last_name, chat_id,
			email, phone,
			notifications_enabled, notify_growth, notify_fall, notify_continuous,
			quiet_hours_start, quiet_hours_end,
			min_growth_threshold, min_fall_threshold,
			preferred_periods, min_volume_filter, exclude_patterns,
			language, timezone, display_mode,
			role, is_active, is_verified, subscription_tier,
			signals_today, max_signals_per_day,
			created_at, updated_at, last_login_at, last_signal_at
		FROM users
		WHERE id = $1
	`

	row := r.db.QueryRow(query, id)
	return r.scanUserRow(row)
}

// FindByTelegramID находит пользователя по Telegram ID
func (r *UserRepositoryImpl) FindByTelegramID(telegramID int64) (*models.User, error) {
	query := `
		SELECT
			id, telegram_id, username, first_name, last_name, chat_id,
			email, phone,
			notifications_enabled, notify_growth, notify_fall, notify_continuous,
			quiet_hours_start, quiet_hours_end,
			min_growth_threshold, min_fall_threshold,
			preferred_periods, min_volume_filter, exclude_patterns,
			language, timezone, display_mode,
			role, is_active, is_verified, subscription_tier,
			signals_today, max_signals_per_day,
			created_at, updated_at, last_login_at, last_signal_at
		FROM users
		WHERE telegram_id = $1
	`

	row := r.db.QueryRow(query, telegramID)
	return r.scanUserRow(row)
}

// FindByChatID находит пользователя по Chat ID
func (r *UserRepositoryImpl) FindByChatID(chatID string) (*models.User, error) {
	query := `
		SELECT
			id, telegram_id, username, first_name, last_name, chat_id,
			email, phone,
			notifications_enabled, notify_growth, notify_fall, notify_continuous,
			quiet_hours_start, quiet_hours_end,
			min_growth_threshold, min_fall_threshold,
			preferred_periods, min_volume_filter, exclude_patterns,
			language, timezone, display_mode,
			role, is_active, is_verified, subscription_tier,
			signals_today, max_signals_per_day,
			created_at, updated_at, last_login_at, last_signal_at
		FROM users
		WHERE chat_id = $1
	`

	row := r.db.QueryRow(query, chatID)
	return r.scanUserRow(row)
}

// FindByEmail находит пользователя по email
func (r *UserRepositoryImpl) FindByEmail(email string) (*models.User, error) {
	query := `
		SELECT
			id, telegram_id, username, first_name, last_name, chat_id,
			email, phone,
			notifications_enabled, notify_growth, notify_fall, notify_continuous,
			quiet_hours_start, quiet_hours_end,
			min_growth_threshold, min_fall_threshold,
			preferred_periods, min_volume_filter, exclude_patterns,
			language, timezone, display_mode,
			role, is_active, is_verified, subscription_tier,
			signals_today, max_signals_per_day,
			created_at, updated_at, last_login_at, last_signal_at
		FROM users
		WHERE email = $1
	`

	row := r.db.QueryRow(query, email)
	return r.scanUserRow(row)
}

// Update обновляет пользователя
func (r *UserRepositoryImpl) Update(user *models.User) error {
	// Начинаем транзакцию
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		UPDATE users SET
			telegram_id = $1,
			username = $2,
			first_name = $3,
			last_name = $4,
			chat_id = $5,
			email = $6,
			phone = $7,
			notifications_enabled = $8,
			notify_growth = $9,
			notify_fall = $10,
			notify_continuous = $11,
			quiet_hours_start = $12,
			quiet_hours_end = $13,
			min_growth_threshold = $14,
			min_fall_threshold = $15,
			preferred_periods = $16,
			min_volume_filter = $17,
			exclude_patterns = $18,
			language = $19,
			timezone = $20,
			display_mode = $21,
			role = $22,
			is_active = $23,
			is_verified = $24,
			subscription_tier = $25,
			signals_today = $26,
			max_signals_per_day = $27,
			last_login_at = $28,
			last_signal_at = $29,
			updated_at = $30
		WHERE id = $31
	`

	result, err := tx.Exec(query,
		user.TelegramID, user.Username, user.FirstName, user.LastName, user.ChatID,
		user.Email, user.Phone,
		user.NotificationsEnabled, user.NotifyGrowth, user.NotifyFall, user.NotifyContinuous,
		user.QuietHoursStart, user.QuietHoursEnd,
		user.MinGrowthThreshold, user.MinFallThreshold,
		pq.Array(user.PreferredPeriods), user.MinVolumeFilter, pq.Array(user.ExcludePatterns),
		user.Language, user.Timezone, user.DisplayMode,
		user.Role, user.IsActive, user.IsVerified,
		user.SubscriptionTier, user.SignalsToday, user.MaxSignalsPerDay,
		getNullTime(user.LastLoginAt), getNullTime(user.LastSignalAt),
		time.Now(), user.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
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
	r.invalidateUserCache(user.ID, user.TelegramID, user.ChatID)

	return nil
}

// Delete удаляет пользователя
func (r *UserRepositoryImpl) Delete(id int) error {
	// Сначала получаем пользователя для инвалидации кэша
	user, err := r.FindByID(id)
	if err != nil {
		return err
	}
	if user == nil {
		return sql.ErrNoRows
	}

	// Удаляем пользователя
	query := `DELETE FROM users WHERE id = $1`
	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	// Инвалидируем кэш
	r.invalidateUserCache(user.ID, user.TelegramID, user.ChatID)

	return nil
}

// UpdateLastLogin обновляет время последнего входа
func (r *UserRepositoryImpl) UpdateLastLogin(userID int) error {
	query := `
		UPDATE users
		SET last_login_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.Exec(query, userID)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	// Инвалидируем кэш
	r.invalidateUserCache(userID, 0, "")

	return nil
}

// IncrementSignalsCount увеличивает счетчик сигналов
func (r *UserRepositoryImpl) IncrementSignalsCount(userID int) error {
	query := `
		UPDATE users
		SET signals_today = signals_today + 1,
			last_signal_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.Exec(query, userID)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// ResetDailyCounters сбрасывает дневные счетчики
func (r *UserRepositoryImpl) ResetDailyCounters(ctx context.Context) error {
	query := `
		UPDATE users
		SET signals_today = 0,
			updated_at = NOW()
		WHERE signals_today > 0
	`

	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("Reset daily counters for %d users", rowsAffected)

	return nil
}

// SearchUsers ищет пользователей
func (r *UserRepositoryImpl) SearchUsers(query string, limit, offset int) ([]*models.User, error) {
	sqlQuery := `
		SELECT
			id, telegram_id, username, first_name, last_name, chat_id,
			email, phone,
			notifications_enabled, notify_growth, notify_fall, notify_continuous,
			quiet_hours_start, quiet_hours_end,
			min_growth_threshold, min_fall_threshold,
			preferred_periods, min_volume_filter, exclude_patterns,
			language, timezone, display_mode,
			role, is_active, is_verified, subscription_tier,
			signals_today, max_signals_per_day,
			created_at, updated_at, last_login_at, last_signal_at
		FROM users
		WHERE username ILIKE $1 OR first_name ILIKE $1 OR last_name ILIKE $1 OR email ILIKE $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	searchPattern := "%" + query + "%"
	rows, err := r.db.Query(sqlQuery, searchPattern, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userList []*models.User
	for rows.Next() {
		user, err := r.scanUser(rows)
		if err != nil {
			return nil, err
		}
		userList = append(userList, user)
	}

	return userList, nil
}

// GetTotalCount возвращает общее количество пользователей
func (r *UserRepositoryImpl) GetTotalCount(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM users`

	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// Вспомогательные методы

// scanUser сканирует строку из rows в User
func (r *UserRepositoryImpl) scanUser(rows *sql.Rows) (*models.User, error) {
	var user models.User
	var lastLoginAt, lastSignalAt sql.NullTime
	var preferredPeriods []sql.NullInt64
	var excludePatterns []sql.NullString

	err := rows.Scan(
		&user.ID, &user.TelegramID, &user.Username, &user.FirstName,
		&user.LastName, &user.ChatID, &user.Email, &user.Phone,
		&user.NotificationsEnabled, &user.NotifyGrowth, &user.NotifyFall, &user.NotifyContinuous,
		&user.QuietHoursStart, &user.QuietHoursEnd,
		&user.MinGrowthThreshold, &user.MinFallThreshold,
		pq.Array(&preferredPeriods), &user.MinVolumeFilter, pq.Array(&excludePatterns),
		&user.Language, &user.Timezone, &user.DisplayMode,
		&user.Role, &user.IsActive, &user.IsVerified, &user.SubscriptionTier,
		&user.SignalsToday, &user.MaxSignalsPerDay,
		&user.CreatedAt, &user.UpdatedAt, &lastLoginAt, &lastSignalAt,
	)

	if err != nil {
		return nil, err
	}

	// Устанавливаем временные метки
	if lastLoginAt.Valid {
		user.LastLoginAt = lastLoginAt.Time
	}
	if lastSignalAt.Valid {
		user.LastSignalAt = lastSignalAt.Time
	}

	// Конвертируем массивы
	user.PreferredPeriods = make([]int, len(preferredPeriods))
	for i, v := range preferredPeriods {
		if v.Valid {
			user.PreferredPeriods[i] = int(v.Int64)
		}
	}

	user.ExcludePatterns = make([]string, len(excludePatterns))
	for i, v := range excludePatterns {
		if v.Valid {
			user.ExcludePatterns[i] = v.String
		}
	}

	return &user, nil
}

// scanUserRow сканирует строку из row в User
func (r *UserRepositoryImpl) scanUserRow(row *sql.Row) (*models.User, error) {
	var user models.User
	var lastLoginAt, lastSignalAt sql.NullTime
	var preferredPeriods []sql.NullInt64
	var excludePatterns []sql.NullString

	err := row.Scan(
		&user.ID, &user.TelegramID, &user.Username, &user.FirstName,
		&user.LastName, &user.ChatID, &user.Email, &user.Phone,
		&user.NotificationsEnabled, &user.NotifyGrowth, &user.NotifyFall, &user.NotifyContinuous,
		&user.QuietHoursStart, &user.QuietHoursEnd,
		&user.MinGrowthThreshold, &user.MinFallThreshold,
		pq.Array(&preferredPeriods), &user.MinVolumeFilter, pq.Array(&excludePatterns),
		&user.Language, &user.Timezone, &user.DisplayMode,
		&user.Role, &user.IsActive, &user.IsVerified, &user.SubscriptionTier,
		&user.SignalsToday, &user.MaxSignalsPerDay,
		&user.CreatedAt, &user.UpdatedAt, &lastLoginAt, &lastSignalAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Устанавливаем временные метки
	if lastLoginAt.Valid {
		user.LastLoginAt = lastLoginAt.Time
	}
	if lastSignalAt.Valid {
		user.LastSignalAt = lastSignalAt.Time
	}

	// Конвертируем массивы
	user.PreferredPeriods = make([]int, len(preferredPeriods))
	for i, v := range preferredPeriods {
		if v.Valid {
			user.PreferredPeriods[i] = int(v.Int64)
		}
	}

	user.ExcludePatterns = make([]string, len(excludePatterns))
	for i, v := range excludePatterns {
		if v.Valid {
			user.ExcludePatterns[i] = v.String
		}
	}

	return &user, nil
}

// invalidateUserCache инвалидирует кэш пользователя
func (r *UserRepositoryImpl) invalidateUserCache(userID int, telegramID int64, chatID string) {
	ctx := context.Background()
	keys := []string{
		fmt.Sprintf("user:%d", userID),
		fmt.Sprintf("user_stats:%d", userID),
		"active_users",
		"system_stats",
	}

	if telegramID > 0 {
		keys = append(keys, fmt.Sprintf("user:telegram:%d", telegramID))
	}
	if chatID != "" {
		keys = append(keys, fmt.Sprintf("user:chat:%s", chatID))
	}

	// Удаляем все ключи разом
	_ = r.cache.DeleteMulti(ctx, keys...)
}

// getNullTime преобразует время в NullTime
func getNullTime(t time.Time) sql.NullTime {
	if t.IsZero() {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{
		Time:  t,
		Valid: true,
	}
}
