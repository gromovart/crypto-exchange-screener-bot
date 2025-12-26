// internal/users/repository.go
package users

// import (
// 	"context"
// 	"crypto-exchange-screener-bot/persistence/redis"
// 	"database/sql"
// 	"encoding/json"
// 	"fmt"
// 	"log"
// 	"strings"
// 	"time"

// 	"github.com/jmoiron/sqlx"
// 	"github.com/lib/pq"
// )

// // UserRepository интерфейс для работы с данными пользователей
// type UserRepository interface {
// 	FindByID(id int) (*User, error)
// 	FindByTelegramID(telegramID int64) (*User, error)
// 	FindByEmail(email string) (*User, error)
// 	FindByChatID(chatID string) (*User, error)
// 	Create(user *User) error
// 	Update(user *User) error
// 	Delete(id int) error
// 	UpdateLastLogin(userID int) error
// 	GetAllActive() ([]*User, error) // Добавляем этот метод
// 	SearchUsers(query string, limit, offset int) ([]*User, error)
// 	GetTotalCount(ctx context.Context) (int, error)
// 	IncrementSignalsCount(userID int) error
// 	ResetDailyCounters(ctx context.Context) error
// }

// // UserRepositoryImpl реализация репозитория пользователей
// type UserRepositoryImpl struct {
// 	db    *sqlx.DB
// 	cache *redis.Cache
// }

// // NewUserRepository создает новый репозиторий пользователей
// func NewUserRepository(db *sqlx.DB, cache *redis.Cache) *UserRepositoryImpl {
// 	return &UserRepositoryImpl{db: db, cache: cache}
// }

// // Получение всех активных пользователей
// // GetAllActive возвращает всех активных пользователей
// func (r *UserRepositoryImpl) GetAllActive() ([]*User, error) {
// 	query := `
//     SELECT
//         id, telegram_id, username, first_name, last_name, chat_id,
//         email, phone,
//         notifications, settings,
//         role, is_active, is_verified, subscription_tier,
//         signals_today, max_signals_per_day,
//         created_at, updated_at, last_login_at, last_signal_at
//     FROM users
//     WHERE is_active = TRUE
//     ORDER BY created_at DESC
//     `

// 	rows, err := r.db.Query(query)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	var users []*User

// 	for rows.Next() {
// 		var user User
// 		var notificationsJSON, settingsJSON []byte

// 		err := rows.Scan(
// 			&user.ID, &user.TelegramID, &user.Username, &user.FirstName,
// 			&user.LastName, &user.ChatID, &user.Email, &user.Phone,
// 			&notificationsJSON, &settingsJSON,
// 			&user.Role, &user.IsActive, &user.IsVerified, &user.SubscriptionTier,
// 			&user.SignalsToday, &user.MaxSignalsPerDay,
// 			&user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt, &user.LastSignalAt,
// 		)

// 		if err != nil {
// 			return nil, err
// 		}

// 		// Декодируем JSON поля
// 		if err := json.Unmarshal(notificationsJSON, &user.Notifications); err != nil {
// 			return nil, fmt.Errorf("failed to unmarshal notifications: %w", err)
// 		}

// 		if err := json.Unmarshal(settingsJSON, &user.Settings); err != nil {
// 			return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
// 		}

// 		users = append(users, &user)
// 	}

// 	return users, nil
// }

// func (r *UserRepositoryImpl) Create(user *User) error {
// 	// Начинаем транзакцию
// 	tx, err := r.db.Begin()
// 	if err != nil {
// 		return fmt.Errorf("failed to begin transaction: %w", err)
// 	}
// 	defer tx.Rollback()

// 	// Сохраняем пользователя
// 	query := `
// 		INSERT INTO users (
// 			telegram_id, username, first_name, last_name, chat_id,
// 			email, phone, role, is_active, is_verified,
// 			subscription_tier, max_signals_per_day,
// 			created_at, updated_at
// 		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
// 		RETURNING id
// 	`

// 	err = tx.QueryRow(
// 		query,
// 		user.TelegramID, user.Username, user.FirstName, user.LastName, user.ChatID,
// 		user.Email, user.Phone, user.Role, user.IsActive, user.IsVerified,
// 		user.SubscriptionTier, user.MaxSignalsPerDay,
// 		user.CreatedAt, user.UpdatedAt,
// 	).Scan(&user.ID)

// 	if err != nil {
// 		return fmt.Errorf("failed to create user: %w", err)
// 	}

// 	// Сохраняем настройки пользователя
// 	if err := r.saveUserSettings(tx, user.ID, &user.Settings); err != nil {
// 		return err
// 	}

// 	// Сохраняем настройки уведомлений
// 	if err := r.saveNotificationSettings(tx, user.ID, &user.Notifications); err != nil {
// 		return err
// 	}

// 	// Фиксируем транзакцию
// 	if err := tx.Commit(); err != nil {
// 		return fmt.Errorf("failed to commit transaction: %w", err)
// 	}

// 	return nil
// }

// func (r *UserRepositoryImpl) FindByID(id int) (*User, error) {
// 	query := `
// 		SELECT
// 			id, telegram_id, username, first_name, last_name, chat_id,
// 			email, phone, role, is_active, is_verified, subscription_tier,
// 			signals_today, max_signals_per_day,
// 			created_at, updated_at, last_login_at, last_signal_at
// 		FROM users
// 		WHERE id = $1
// 	`

// 	var user User
// 	var lastLoginAt, lastSignalAt sql.NullTime

// 	err := r.db.QueryRow(query, id).Scan(
// 		&user.ID, &user.TelegramID, &user.Username, &user.FirstName, &user.LastName,
// 		&user.ChatID, &user.Email, &user.Phone, &user.Role, &user.IsActive,
// 		&user.IsVerified, &user.SubscriptionTier, &user.SignalsToday,
// 		&user.MaxSignalsPerDay, &user.CreatedAt, &user.UpdatedAt,
// 		&lastLoginAt, &lastSignalAt,
// 	)

// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			return nil, nil
// 		}
// 		return nil, err
// 	}

// 	// Устанавливаем временные метки
// 	if lastLoginAt.Valid {
// 		user.LastLoginAt = lastLoginAt.Time
// 	}
// 	if lastSignalAt.Valid {
// 		user.LastSignalAt = lastSignalAt.Time
// 	}

// 	// Загружаем настройки
// 	settings, err := r.loadUserSettings(id)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to load user settings: %w", err)
// 	}
// 	user.Settings = *settings

// 	// Загружаем настройки уведомлений
// 	notifications, err := r.loadNotificationSettings(id)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to load notification settings: %w", err)
// 	}
// 	user.Notifications = *notifications

// 	return &user, nil
// }

// func (r *UserRepositoryImpl) FindByTelegramID(telegramID int64) (*User, error) {
// 	query := `
// 		SELECT
// 			id, telegram_id, username, first_name, last_name, chat_id,
// 			email, phone, role, is_active, is_verified, subscription_tier,
// 			signals_today, max_signals_per_day,
// 			created_at, updated_at, last_login_at, last_signal_at
// 		FROM users
// 		WHERE telegram_id = $1
// 	`

// 	var user User
// 	var lastLoginAt, lastSignalAt sql.NullTime

// 	err := r.db.QueryRow(query, telegramID).Scan(
// 		&user.ID, &user.TelegramID, &user.Username, &user.FirstName, &user.LastName,
// 		&user.ChatID, &user.Email, &user.Phone, &user.Role, &user.IsActive,
// 		&user.IsVerified, &user.SubscriptionTier, &user.SignalsToday,
// 		&user.MaxSignalsPerDay, &user.CreatedAt, &user.UpdatedAt,
// 		&lastLoginAt, &lastSignalAt,
// 	)

// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			return nil, nil
// 		}
// 		return nil, err
// 	}

// 	// Устанавливаем временные метки
// 	if lastLoginAt.Valid {
// 		user.LastLoginAt = lastLoginAt.Time
// 	}
// 	if lastSignalAt.Valid {
// 		user.LastSignalAt = lastSignalAt.Time
// 	}

// 	// Загружаем настройки
// 	settings, err := r.loadUserSettings(user.ID)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to load user settings: %w", err)
// 	}
// 	user.Settings = *settings

// 	// Загружаем настройки уведомлений
// 	notifications, err := r.loadNotificationSettings(user.ID)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to load notification settings: %w", err)
// 	}
// 	user.Notifications = *notifications

// 	return &user, nil
// }

// func (r *UserRepositoryImpl) FindByChatID(chatID string) (*User, error) {
// 	query := `
// 		SELECT
// 			id, telegram_id, username, first_name, last_name, chat_id,
// 			email, phone, role, is_active, is_verified, subscription_tier,
// 			signals_today, max_signals_per_day,
// 			created_at, updated_at, last_login_at, last_signal_at
// 		FROM users
// 		WHERE chat_id = $1
// 	`

// 	var user User
// 	var lastLoginAt, lastSignalAt sql.NullTime

// 	err := r.db.QueryRow(query, chatID).Scan(
// 		&user.ID, &user.TelegramID, &user.Username, &user.FirstName, &user.LastName,
// 		&user.ChatID, &user.Email, &user.Phone, &user.Role, &user.IsActive,
// 		&user.IsVerified, &user.SubscriptionTier, &user.SignalsToday,
// 		&user.MaxSignalsPerDay, &user.CreatedAt, &user.UpdatedAt,
// 		&lastLoginAt, &lastSignalAt,
// 	)

// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			return nil, nil
// 		}
// 		return nil, err
// 	}

// 	// Устанавливаем временные метки
// 	if lastLoginAt.Valid {
// 		user.LastLoginAt = lastLoginAt.Time
// 	}
// 	if lastSignalAt.Valid {
// 		user.LastSignalAt = lastSignalAt.Time
// 	}

// 	// Загружаем настройки
// 	settings, err := r.loadUserSettings(user.ID)
// 	if err != nil {
// 		return nil, err
// 	}
// 	user.Settings = *settings

// 	// Загружаем настройки уведомлений
// 	notifications, err := r.loadNotificationSettings(user.ID)
// 	if err != nil {
// 		return nil, err
// 	}
// 	user.Notifications = *notifications

// 	return &user, nil
// }

// func (r *UserRepositoryImpl) FindByEmail(email string) (*User, error) {
// 	query := `
// 		SELECT
// 			id, telegram_id, username, first_name, last_name, chat_id,
// 			email, phone, role, is_active, is_verified, subscription_tier,
// 			signals_today, max_signals_per_day,
// 			created_at, updated_at, last_login_at, last_signal_at
// 		FROM users
// 		WHERE email = $1
// 	`

// 	var user User
// 	var lastLoginAt, lastSignalAt sql.NullTime

// 	err := r.db.QueryRow(query, email).Scan(
// 		&user.ID, &user.TelegramID, &user.Username, &user.FirstName, &user.LastName,
// 		&user.ChatID, &user.Email, &user.Phone, &user.Role, &user.IsActive,
// 		&user.IsVerified, &user.SubscriptionTier, &user.SignalsToday,
// 		&user.MaxSignalsPerDay, &user.CreatedAt, &user.UpdatedAt,
// 		&lastLoginAt, &lastSignalAt,
// 	)

// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			return nil, nil
// 		}
// 		return nil, err
// 	}

// 	// Устанавливаем временные метки
// 	if lastLoginAt.Valid {
// 		user.LastLoginAt = lastLoginAt.Time
// 	}
// 	if lastSignalAt.Valid {
// 		user.LastSignalAt = lastSignalAt.Time
// 	}

// 	// Загружаем настройки
// 	settings, err := r.loadUserSettings(user.ID)
// 	if err != nil {
// 		return nil, err
// 	}
// 	user.Settings = *settings

// 	// Загружаем настройки уведомлений
// 	notifications, err := r.loadNotificationSettings(user.ID)
// 	if err != nil {
// 		return nil, err
// 	}
// 	user.Notifications = *notifications

// 	return &user, nil
// }

// func (r *UserRepositoryImpl) Update(user *User) error {
// 	// Начинаем транзакцию
// 	tx, err := r.db.Begin()
// 	if err != nil {
// 		return fmt.Errorf("failed to begin transaction: %w", err)
// 	}
// 	defer tx.Rollback()

// 	// Обновляем пользователя
// 	query := `
// 		UPDATE users SET
// 			telegram_id = $1,
// 			username = $2,
// 			first_name = $3,
// 			last_name = $4,
// 			chat_id = $5,
// 			email = $6,
// 			phone = $7,
// 			role = $8,
// 			is_active = $9,
// 			is_verified = $10,
// 			subscription_tier = $11,
// 			signals_today = $12,
// 			max_signals_per_day = $13,
// 			last_login_at = $14,
// 			last_signal_at = $15,
// 			updated_at = $16
// 		WHERE id = $17
// 	`

// 	result, err := tx.Exec(query,
// 		user.TelegramID, user.Username, user.FirstName, user.LastName, user.ChatID,
// 		user.Email, user.Phone, user.Role, user.IsActive, user.IsVerified,
// 		user.SubscriptionTier, user.SignalsToday, user.MaxSignalsPerDay,
// 		getNullTime(user.LastLoginAt), getNullTime(user.LastSignalAt),
// 		time.Now(), user.ID,
// 	)

// 	if err != nil {
// 		return fmt.Errorf("failed to update user: %w", err)
// 	}

// 	rowsAffected, _ := result.RowsAffected()
// 	if rowsAffected == 0 {
// 		return sql.ErrNoRows
// 	}

// 	// Обновляем настройки
// 	if err := r.updateUserSettings(tx, user.ID, &user.Settings); err != nil {
// 		return err
// 	}

// 	// Обновляем настройки уведомлений
// 	if err := r.updateNotificationSettings(tx, user.ID, &user.Notifications); err != nil {
// 		return err
// 	}

// 	// Фиксируем транзакцию
// 	if err := tx.Commit(); err != nil {
// 		return fmt.Errorf("failed to commit transaction: %w", err)
// 	}

// 	// Инвалидируем кэш
// 	r.invalidateUserCache(user.ID, user.TelegramID, user.ChatID)

// 	return nil
// }

// func (r *UserRepositoryImpl) Delete(id int) error {
// 	// Сначала получаем пользователя для инвалидации кэша
// 	user, err := r.FindByID(id)
// 	if err != nil {
// 		return err
// 	}

// 	// Удаляем пользователя
// 	query := `DELETE FROM users WHERE id = $1`
// 	result, err := r.db.Exec(query, id)
// 	if err != nil {
// 		return err
// 	}

// 	rowsAffected, _ := result.RowsAffected()
// 	if rowsAffected == 0 {
// 		return sql.ErrNoRows
// 	}

// 	// Инвалидируем кэш
// 	r.invalidateUserCache(user.ID, user.TelegramID, user.ChatID)

// 	return nil
// }

// func (r *UserRepositoryImpl) GetActiveUsers() ([]*User, error) {
// 	query := `
// 		SELECT
// 			id, telegram_id, username, first_name, last_name, chat_id,
// 			email, phone, role, is_active, is_verified, subscription_tier,
// 			signals_today, max_signals_per_day,
// 			created_at, updated_at, last_login_at, last_signal_at
// 		FROM users
// 		WHERE is_active = TRUE
// 		ORDER BY created_at DESC
// 	`

// 	rows, err := r.db.Query(query)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	var users []*User
// 	for rows.Next() {
// 		var user User
// 		var lastLoginAt, lastSignalAt sql.NullTime

// 		err := rows.Scan(
// 			&user.ID, &user.TelegramID, &user.Username, &user.FirstName, &user.LastName,
// 			&user.ChatID, &user.Email, &user.Phone, &user.Role, &user.IsActive,
// 			&user.IsVerified, &user.SubscriptionTier, &user.SignalsToday,
// 			&user.MaxSignalsPerDay, &user.CreatedAt, &user.UpdatedAt,
// 			&lastLoginAt, &lastSignalAt,
// 		)

// 		if err != nil {
// 			return nil, err
// 		}

// 		// Устанавливаем временные метки
// 		if lastLoginAt.Valid {
// 			user.LastLoginAt = lastLoginAt.Time
// 		}
// 		if lastSignalAt.Valid {
// 			user.LastSignalAt = lastSignalAt.Time
// 		}

// 		users = append(users, &user)
// 	}

// 	// Загружаем настройки для каждого пользователя
// 	for _, user := range users {
// 		settings, err := r.loadUserSettings(user.ID)
// 		if err == nil {
// 			user.Settings = *settings
// 		}

// 		notifications, err := r.loadNotificationSettings(user.ID)
// 		if err == nil {
// 			user.Notifications = *notifications
// 		}
// 	}

// 	return users, nil
// }

// func (r *UserRepositoryImpl) GetUsersByStatus(status bool) ([]*User, error) {
// 	query := `
// 		SELECT
// 			id, telegram_id, username, first_name, last_name, chat_id,
// 			email, phone, role, is_active, is_verified, subscription_tier,
// 			signals_today, max_signals_per_day,
// 			created_at, updated_at, last_login_at, last_signal_at
// 		FROM users
// 		WHERE is_active = $1
// 		ORDER BY created_at DESC
// 	`

// 	rows, err := r.db.Query(query, status)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	var users []*User
// 	for rows.Next() {
// 		var user User
// 		var lastLoginAt, lastSignalAt sql.NullTime

// 		err := rows.Scan(
// 			&user.ID, &user.TelegramID, &user.Username, &user.FirstName, &user.LastName,
// 			&user.ChatID, &user.Email, &user.Phone, &user.Role, &user.IsActive,
// 			&user.IsVerified, &user.SubscriptionTier, &user.SignalsToday,
// 			&user.MaxSignalsPerDay, &user.CreatedAt, &user.UpdatedAt,
// 			&lastLoginAt, &lastSignalAt,
// 		)

// 		if err != nil {
// 			return nil, err
// 		}

// 		// Устанавливаем временные метки
// 		if lastLoginAt.Valid {
// 			user.LastLoginAt = lastLoginAt.Time
// 		}
// 		if lastSignalAt.Valid {
// 			user.LastSignalAt = lastSignalAt.Time
// 		}

// 		users = append(users, &user)
// 	}

// 	// Загружаем настройки для каждого пользователя
// 	for _, user := range users {
// 		settings, err := r.loadUserSettings(user.ID)
// 		if err == nil {
// 			user.Settings = *settings
// 		}

// 		notifications, err := r.loadNotificationSettings(user.ID)
// 		if err == nil {
// 			user.Notifications = *notifications
// 		}
// 	}

// 	return users, nil
// }

// func (r *UserRepositoryImpl) GetStatistics(ctx context.Context) (map[string]interface{}, error) {
// 	stats := make(map[string]interface{})

// 	// Получаем общую статистику
// 	query := `
// 		SELECT
// 			COUNT(*) as total_users,
// 			COUNT(CASE WHEN is_active THEN 1 END) as active_users,
// 			COUNT(CASE WHEN created_at >= CURRENT_DATE THEN 1 END) as new_users_today,
// 			COUNT(CASE WHEN role = 'admin' THEN 1 END) as admin_count,
// 			COUNT(CASE WHEN role = 'premium' THEN 1 END) as premium_count,
// 			COUNT(CASE WHEN subscription_tier = 'pro' THEN 1 END) as pro_count,
// 			COALESCE(SUM(signals_today), 0) as total_signals_today,
// 			COALESCE(AVG(signals_today), 0) as avg_signals_per_user
// 		FROM users
// 	`

// 	var (
// 		totalUsers        int
// 		activeUsers       int
// 		newUsersToday     int
// 		adminCount        int
// 		premiumCount      int
// 		proCount          int
// 		totalSignalsToday int64
// 		avgSignalsPerUser float64
// 	)

// 	row := r.db.QueryRowContext(ctx, query)

// 	err := row.Scan(
// 		&totalUsers,
// 		&activeUsers,
// 		&newUsersToday,
// 		&adminCount,
// 		&premiumCount,
// 		&proCount,
// 		&totalSignalsToday,
// 		&avgSignalsPerUser,
// 	)

// 	if err != nil {
// 		return nil, err
// 	}

// 	// Заполняем карту
// 	stats["total_users"] = totalUsers
// 	stats["active_users"] = activeUsers
// 	stats["new_users_today"] = newUsersToday
// 	stats["admin_count"] = adminCount
// 	stats["premium_count"] = premiumCount
// 	stats["pro_count"] = proCount
// 	stats["total_signals_today"] = totalSignalsToday
// 	stats["avg_signals_per_user"] = avgSignalsPerUser

// 	// Статистика по географии (timezone)
// 	query = `
// 		SELECT timezone, COUNT(*) as user_count
// 		FROM users
// 		WHERE timezone IS NOT NULL
// 		GROUP BY timezone
// 		ORDER BY user_count DESC
// 		LIMIT 10
// 	`

// 	rows, err := r.db.QueryContext(ctx, query)
// 	if err == nil {
// 		defer rows.Close()

// 		var timezoneStats []map[string]interface{}
// 		for rows.Next() {
// 			var timezone string
// 			var count int
// 			rows.Scan(&timezone, &count)
// 			timezoneStats = append(timezoneStats, map[string]interface{}{
// 				"timezone": timezone,
// 				"count":    count,
// 			})
// 		}
// 		stats["timezone_distribution"] = timezoneStats
// 	}

// 	return stats, nil
// }

// func (r *UserRepositoryImpl) IncrementSignalsCount(userID int) error {
// 	query := `
// 		UPDATE users
// 		SET signals_today = signals_today + 1,
// 			last_signal_at = NOW(),
// 			updated_at = NOW()
// 		WHERE id = $1
// 	`

// 	result, err := r.db.Exec(query, userID)
// 	if err != nil {
// 		return err
// 	}

// 	rowsAffected, _ := result.RowsAffected()
// 	if rowsAffected == 0 {
// 		return sql.ErrNoRows
// 	}

// 	return nil
// }

// func (r *UserRepositoryImpl) ResetDailyCounters(ctx context.Context) error {
// 	query := `
// 		UPDATE users
// 		SET signals_today = 0,
// 			updated_at = NOW()
// 		WHERE signals_today > 0
// 	`

// 	result, err := r.db.ExecContext(ctx, query)
// 	if err != nil {
// 		return err
// 	}

// 	rowsAffected, _ := result.RowsAffected()
// 	log.Printf("Reset daily counters for %d users", rowsAffected)

// 	return nil
// }

// func (r *UserRepositoryImpl) UpdateLastLogin(userID int) error {
// 	query := `
// 		UPDATE users
// 		SET last_login_at = NOW(),
// 			updated_at = NOW()
// 		WHERE id = $1
// 	`

// 	result, err := r.db.Exec(query, userID)
// 	if err != nil {
// 		return err
// 	}

// 	rowsAffected, _ := result.RowsAffected()
// 	if rowsAffected == 0 {
// 		return sql.ErrNoRows
// 	}

// 	// Инвалидируем кэш
// 	r.invalidateUserCache(userID, 0, "")

// 	return nil
// }

// func (r *UserRepositoryImpl) UpdateSubscriptionTier(userID int, tier string) error {
// 	query := `
// 		UPDATE users
// 		SET subscription_tier = $1,
// 			updated_at = NOW()
// 		WHERE id = $2
// 	`

// 	result, err := r.db.Exec(query, tier, userID)
// 	if err != nil {
// 		return err
// 	}

// 	rowsAffected, _ := result.RowsAffected()
// 	if rowsAffected == 0 {
// 		return sql.ErrNoRows
// 	}

// 	// Инвалидируем кэш
// 	r.invalidateUserCache(userID, 0, "")

// 	return nil
// }

// func (r *UserRepositoryImpl) SearchUsers(query string, limit, offset int) ([]*User, error) {
// 	sqlQuery := `
// 		SELECT
// 			id, telegram_id, username, first_name, last_name, chat_id,
// 			email, phone, role, is_active, is_verified, subscription_tier,
// 			signals_today, max_signals_per_day,
// 			created_at, updated_at, last_login_at, last_signal_at
// 		FROM users
// 		WHERE username ILIKE $1 OR first_name ILIKE $1 OR last_name ILIKE $1 OR email ILIKE $1
// 		ORDER BY created_at DESC
// 		LIMIT $2 OFFSET $3
// 	`

// 	searchPattern := "%" + query + "%"
// 	rows, err := r.db.Query(sqlQuery, searchPattern, limit, offset)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	var users []*User
// 	for rows.Next() {
// 		var user User
// 		var lastLoginAt, lastSignalAt sql.NullTime

// 		err := rows.Scan(
// 			&user.ID, &user.TelegramID, &user.Username, &user.FirstName, &user.LastName,
// 			&user.ChatID, &user.Email, &user.Phone, &user.Role, &user.IsActive,
// 			&user.IsVerified, &user.SubscriptionTier, &user.SignalsToday,
// 			&user.MaxSignalsPerDay, &user.CreatedAt, &user.UpdatedAt,
// 			&lastLoginAt, &lastSignalAt,
// 		)

// 		if err != nil {
// 			return nil, err
// 		}

// 		// Устанавливаем временные метки
// 		if lastLoginAt.Valid {
// 			user.LastLoginAt = lastLoginAt.Time
// 		}
// 		if lastSignalAt.Valid {
// 			user.LastSignalAt = lastSignalAt.Time
// 		}

// 		users = append(users, &user)
// 	}

// 	// Загружаем настройки для каждого пользователя
// 	for _, user := range users {
// 		settings, err := r.loadUserSettings(user.ID)
// 		if err == nil {
// 			user.Settings = *settings
// 		}

// 		notifications, err := r.loadNotificationSettings(user.ID)
// 		if err == nil {
// 			user.Notifications = *notifications
// 		}
// 	}

// 	return users, nil
// }

// func (r *UserRepositoryImpl) GetUsersByRole(role string) ([]*User, error) {
// 	query := `
// 		SELECT
// 			id, telegram_id, username, first_name, last_name, chat_id,
// 			email, phone, role, is_active, is_verified, subscription_tier,
// 			signals_today, max_signals_per_day,
// 			created_at, updated_at, last_login_at, last_signal_at
// 		FROM users
// 		WHERE role = $1
// 		ORDER BY created_at DESC
// 	`

// 	rows, err := r.db.Query(query, role)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	var users []*User
// 	for rows.Next() {
// 		var user User
// 		var lastLoginAt, lastSignalAt sql.NullTime

// 		err := rows.Scan(
// 			&user.ID, &user.TelegramID, &user.Username, &user.FirstName, &user.LastName,
// 			&user.ChatID, &user.Email, &user.Phone, &user.Role, &user.IsActive,
// 			&user.IsVerified, &user.SubscriptionTier, &user.SignalsToday,
// 			&user.MaxSignalsPerDay, &user.CreatedAt, &user.UpdatedAt,
// 			&lastLoginAt, &lastSignalAt,
// 		)

// 		if err != nil {
// 			return nil, err
// 		}

// 		// Устанавливаем временные метки
// 		if lastLoginAt.Valid {
// 			user.LastLoginAt = lastLoginAt.Time
// 		}
// 		if lastSignalAt.Valid {
// 			user.LastSignalAt = lastSignalAt.Time
// 		}

// 		users = append(users, &user)
// 	}

// 	// Загружаем настройки для каждого пользователя
// 	for _, user := range users {
// 		settings, err := r.loadUserSettings(user.ID)
// 		if err == nil {
// 			user.Settings = *settings
// 		}

// 		notifications, err := r.loadNotificationSettings(user.ID)
// 		if err == nil {
// 			user.Notifications = *notifications
// 		}
// 	}

// 	return users, nil
// }

// func (r *UserRepositoryImpl) GetUsersCreatedBetween(start, end time.Time) ([]*User, error) {
// 	query := `
// 		SELECT
// 			id, telegram_id, username, first_name, last_name, chat_id,
// 			email, phone, role, is_active, is_verified, subscription_tier,
// 			signals_today, max_signals_per_day,
// 			created_at, updated_at, last_login_at, last_signal_at
// 		FROM users
// 		WHERE created_at BETWEEN $1 AND $2
// 		ORDER BY created_at DESC
// 	`

// 	rows, err := r.db.Query(query, start, end)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	var users []*User
// 	for rows.Next() {
// 		var user User
// 		var lastLoginAt, lastSignalAt sql.NullTime

// 		err := rows.Scan(
// 			&user.ID, &user.TelegramID, &user.Username, &user.FirstName, &user.LastName,
// 			&user.ChatID, &user.Email, &user.Phone, &user.Role, &user.IsActive,
// 			&user.IsVerified, &user.SubscriptionTier, &user.SignalsToday,
// 			&user.MaxSignalsPerDay, &user.CreatedAt, &user.UpdatedAt,
// 			&lastLoginAt, &lastSignalAt,
// 		)

// 		if err != nil {
// 			return nil, err
// 		}

// 		// Устанавливаем временные метки
// 		if lastLoginAt.Valid {
// 			user.LastLoginAt = lastLoginAt.Time
// 		}
// 		if lastSignalAt.Valid {
// 			user.LastSignalAt = lastSignalAt.Time
// 		}

// 		users = append(users, &user)
// 	}

// 	// Загружаем настройки для каждого пользователя
// 	for _, user := range users {
// 		settings, err := r.loadUserSettings(user.ID)
// 		if err == nil {
// 			user.Settings = *settings
// 		}

// 		notifications, err := r.loadNotificationSettings(user.ID)
// 		if err == nil {
// 			user.Notifications = *notifications
// 		}
// 	}

// 	return users, nil
// }

// func (r *UserRepositoryImpl) BulkUpdateStatus(userIDs []int, status bool) error {
// 	if len(userIDs) == 0 {
// 		return nil
// 	}

// 	// Создаем строку с параметрами
// 	params := make([]interface{}, len(userIDs)+1)
// 	placeholders := make([]string, len(userIDs))

// 	for i, id := range userIDs {
// 		params[i] = id
// 		placeholders[i] = fmt.Sprintf("$%d", i+1)
// 	}
// 	params[len(userIDs)] = status

// 	query := fmt.Sprintf(`
// 		UPDATE users
// 		SET is_active = $%d,
// 			updated_at = NOW()
// 		WHERE id IN (%s)
// 	`, len(userIDs)+1, strings.Join(placeholders, ", "))

// 	result, err := r.db.Exec(query, params...)
// 	if err != nil {
// 		return err
// 	}

// 	rowsAffected, _ := result.RowsAffected()
// 	log.Printf("Updated status for %d users to %v", rowsAffected, status)

// 	// Инвалидируем кэш для всех пользователей
// 	for _, id := range userIDs {
// 		r.invalidateUserCache(id, 0, "")
// 	}

// 	return nil
// }

// func (r *UserRepositoryImpl) GetTotalCount(ctx context.Context) (int, error) {
// 	query := `SELECT COUNT(*) FROM users`

// 	var count int
// 	err := r.db.QueryRowContext(ctx, query).Scan(&count)
// 	if err != nil {
// 		return 0, err
// 	}

// 	return count, nil
// }

// // Вспомогательные методы для работы с настройками

// func (r *UserRepositoryImpl) loadUserSettings(userID int) (*UserSettings, error) {
// 	query := `
// 		SELECT
// 			min_growth_threshold,
// 			min_fall_threshold,
// 			preferred_periods,
// 			min_volume_filter,
// 			exclude_patterns,
// 			language,
// 			timezone,
// 			display_mode
// 		FROM users
// 		WHERE id = $1
// 	`

// 	var settings UserSettings
// 	var periods []int64
// 	var excludePatterns []string

// 	err := r.db.QueryRow(query, userID).Scan(
// 		&settings.MinGrowthThreshold,
// 		&settings.MinFallThreshold,
// 		&periods,
// 		&settings.MinVolumeFilter,
// 		&excludePatterns,
// 		&settings.Language,
// 		&settings.Timezone,
// 		&settings.DisplayMode,
// 	)

// 	if err != nil {
// 		return nil, err
// 	}

// 	// Конвертируем массивы
// 	settings.PreferredPeriods = make([]int, len(periods))
// 	for i, p := range periods {
// 		settings.PreferredPeriods[i] = int(p)
// 	}

// 	settings.ExcludePatterns = excludePatterns

// 	return &settings, nil
// }

// func (r *UserRepositoryImpl) loadNotificationSettings(userID int) (*NotificationSettings, error) {
// 	query := `
// 		SELECT
// 			notifications_enabled,
// 			notify_growth,
// 			notify_fall,
// 			notify_continuous,
// 			quiet_hours_start,
// 			quiet_hours_end
// 		FROM users
// 		WHERE id = $1
// 	`

// 	var settings NotificationSettings

// 	err := r.db.QueryRow(query, userID).Scan(
// 		&settings.Enabled,
// 		&settings.Growth,
// 		&settings.Fall,
// 		&settings.Continuous,
// 		&settings.QuietHoursStart,
// 		&settings.QuietHoursEnd,
// 	)

// 	if err != nil {
// 		return nil, err
// 	}

// 	return &settings, nil
// }

// func (r *UserRepositoryImpl) saveUserSettings(tx *sql.Tx, userID int, settings *UserSettings) error {
// 	query := `
// 		UPDATE users SET
// 			min_growth_threshold = $1,
// 			min_fall_threshold = $2,
// 			preferred_periods = $3,
// 			min_volume_filter = $4,
// 			exclude_patterns = $5,
// 			language = $6,
// 			timezone = $7,
// 			display_mode = $8
// 		WHERE id = $9
// 	`

// 	_, err := tx.Exec(query,
// 		settings.MinGrowthThreshold,
// 		settings.MinFallThreshold,
// 		pq.Array(settings.PreferredPeriods),
// 		settings.MinVolumeFilter,
// 		pq.Array(settings.ExcludePatterns),
// 		settings.Language,
// 		settings.Timezone,
// 		settings.DisplayMode,
// 		userID,
// 	)

// 	return err
// }

// func (r *UserRepositoryImpl) saveNotificationSettings(tx *sql.Tx, userID int, settings *NotificationSettings) error {
// 	query := `
// 		UPDATE users SET
// 			notifications_enabled = $1,
// 			notify_growth = $2,
// 			notify_fall = $3,
// 			notify_continuous = $4,
// 			quiet_hours_start = $5,
// 			quiet_hours_end = $6
// 		WHERE id = $7
// 	`

// 	_, err := tx.Exec(query,
// 		settings.Enabled,
// 		settings.Growth,
// 		settings.Fall,
// 		settings.Continuous,
// 		settings.QuietHoursStart,
// 		settings.QuietHoursEnd,
// 		userID,
// 	)

// 	return err
// }

// func (r *UserRepositoryImpl) updateUserSettings(tx *sql.Tx, userID int, settings *UserSettings) error {
// 	return r.saveUserSettings(tx, userID, settings)
// }

// func (r *UserRepositoryImpl) updateNotificationSettings(tx *sql.Tx, userID int, settings *NotificationSettings) error {
// 	return r.saveNotificationSettings(tx, userID, settings)
// }

// func (r *UserRepositoryImpl) invalidateUserCache(userID int, telegramID int64, chatID string) {
// 	ctx := context.Background()
// 	keys := []string{
// 		fmt.Sprintf("user:%d", userID),
// 		fmt.Sprintf("user_stats:%d", userID),
// 		"active_users",
// 		"system_stats",
// 	}

// 	if telegramID > 0 {
// 		keys = append(keys, fmt.Sprintf("user:telegram:%d", telegramID))
// 	}
// 	if chatID != "" {
// 		keys = append(keys, fmt.Sprintf("user:chat:%s", chatID))
// 	}

// 	r.cache.Del(ctx, keys...)
// }

// // Вспомогательная функция для преобразования времени в NullTime
// func getNullTime(t time.Time) sql.NullTime {
// 	if t.IsZero() {
// 		return sql.NullTime{Valid: false}
// 	}
// 	return sql.NullTime{
// 		Time:  t,
// 		Valid: true,
// 	}
// }
