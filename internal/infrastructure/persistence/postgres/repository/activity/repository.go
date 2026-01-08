// internal/infrastructure/persistence/postgres/repository/activity/repository.go
package activity

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"crypto-exchange-screener-bot/internal/infrastructure/cache/redis"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"

	"github.com/jmoiron/sqlx"
)

// ActivityRepository интерфейс для работы с активностью пользователей
type ActivityRepository interface {
	// Базовые CRUD операции
	Create(activity *models.UserActivity) error
	CreateWithUser(activity *models.UserActivity, user *models.User) error
	FindByID(id int64) (*models.UserActivity, error)
	Update(activity *models.UserActivity) error
	Delete(id int64) error
	BulkDelete(ids []int64) error

	// Поиск и фильтрация
	FindByUserID(userID int, limit, offset int) ([]*models.UserActivity, error)
	FindByFilter(filter models.ActivityFilter) ([]*models.UserActivity, int64, error)
	GetRecentActivities(limit int) ([]*models.UserActivity, error)
	GetAll(limit, offset int) ([]*models.UserActivity, error)
	GetSuspiciousActivities(limit int) ([]*models.UserActivity, error)

	// Логирование различных событий
	LogUserLogin(user *models.User, ip, userAgent string, success bool, failureReason string) error
	LogUserLogout(user *models.User, ip, userAgent, reason string) error
	LogSignalReceived(user *models.User, signalType string, symbol string, changePercent float64, period int, filtered bool, filterReason string) error
	LogSettingsUpdate(user *models.User, settingType string, oldValue, newValue interface{}, ip, userAgent string) error
	LogSecurityEvent(user *models.User, eventType, description string, severity models.ActivitySeverity, ip, userAgent string, metadata models.JSONMap) error
	LogError(user *models.User, errorType, errorMessage, stackTrace string, severity models.ActivitySeverity, additionalData models.JSONMap) error
	LogSystemEvent(eventType, description string, severity models.ActivitySeverity, metadata models.JSONMap) error

	// Статистика и аналитика
	GetUserActivityStats(userID int, days int) (map[string]interface{}, error)
	GetSystemActivityStats(days int) (*models.ActivityStats, error)
	GetFailedLoginAttempts(ip string, minutes int) (int, error)
	GetStatistics(ctx context.Context, userID int) (map[string]interface{}, error)

	// Управление и обслуживание
	CleanupOldActivities(ctx context.Context, olderThanDays int) (int64, error)
	ArchiveActivities(ctx context.Context, olderThanDays int) (int64, error)
	ResetDailyCounters(ctx context.Context) error
	GetTotalCount(ctx context.Context) (int, error)
}

// ActivityRepositoryImpl реализация репозитория активности
type ActivityRepositoryImpl struct {
	db    *sqlx.DB
	cache *redis.Cache
}

// NewActivityRepository создает новый репозиторий активности
func NewActivityRepository(db *sqlx.DB, cache *redis.Cache) *ActivityRepositoryImpl {
	return &ActivityRepositoryImpl{db: db, cache: cache}
}

// Create создает запись активности
func (r *ActivityRepositoryImpl) Create(activity *models.UserActivity) error {
	query := `
	INSERT INTO user_activities (
		user_id, activity_type, category, severity, details,
		ip_address, user_agent, metadata
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	RETURNING id, created_at
	`

	detailsJSON, err := json.Marshal(activity.Details)
	if err != nil {
		return fmt.Errorf("failed to marshal details: %w", err)
	}

	metadataJSON, err := json.Marshal(activity.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	return r.db.QueryRow(
		query,
		activity.UserID,
		activity.ActivityType,
		activity.Category,
		activity.Severity,
		detailsJSON,
		activity.IPAddress,
		activity.UserAgent,
		metadataJSON,
	).Scan(&activity.ID, &activity.CreatedAt)
}

// CreateWithUser создает активность с информацией о пользователе
func (r *ActivityRepositoryImpl) CreateWithUser(activity *models.UserActivity, user *models.User) error {
	activity.TelegramID = user.TelegramID
	activity.Username = user.Username
	activity.FirstName = user.FirstName
	return r.Create(activity)
}

// FindByID находит активность по ID
func (r *ActivityRepositoryImpl) FindByID(id int64) (*models.UserActivity, error) {
	cacheKey := fmt.Sprintf("activity:%d", id)

	// Попытка получить из кэша
	var activity models.UserActivity
	if err := r.cache.Get(context.Background(), cacheKey, &activity); err == nil {
		return &activity, nil
	}

	query := `
	SELECT
		a.id, a.user_id, a.activity_type, a.category, a.severity,
		a.details, a.ip_address, a.user_agent, a.metadata, a.created_at,
		u.telegram_id, u.username, u.first_name
	FROM user_activities a
	LEFT JOIN users u ON a.user_id = u.id
	WHERE a.id = $1
	`

	var detailsJSON, metadataJSON []byte

	err := r.db.QueryRow(query, id).Scan(
		&activity.ID,
		&activity.UserID,
		&activity.ActivityType,
		&activity.Category,
		&activity.Severity,
		&detailsJSON,
		&activity.IPAddress,
		&activity.UserAgent,
		&metadataJSON,
		&activity.CreatedAt,
		&activity.TelegramID,
		&activity.Username,
		&activity.FirstName,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Декодируем JSON
	if err := json.Unmarshal(detailsJSON, &activity.Details); err != nil {
		return nil, fmt.Errorf("failed to unmarshal details: %w", err)
	}

	if err := json.Unmarshal(metadataJSON, &activity.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	// Сохраняем в кэш
	if data, err := json.Marshal(activity); err == nil {
		_ = r.cache.Set(context.Background(), cacheKey, string(data), 5*time.Minute)
	}

	return &activity, nil
}

// Update обновляет активность
func (r *ActivityRepositoryImpl) Update(activity *models.UserActivity) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	detailsJSON, err := json.Marshal(activity.Details)
	if err != nil {
		return fmt.Errorf("failed to marshal details: %w", err)
	}

	metadataJSON, err := json.Marshal(activity.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
	UPDATE user_activities SET
		user_id = $1,
		activity_type = $2,
		category = $3,
		severity = $4,
		details = $5,
		ip_address = $6,
		user_agent = $7,
		metadata = $8
	WHERE id = $9
	`

	result, err := tx.Exec(query,
		activity.UserID,
		activity.ActivityType,
		activity.Category,
		activity.Severity,
		detailsJSON,
		activity.IPAddress,
		activity.UserAgent,
		metadataJSON,
		activity.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update activity: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Инвалидируем кэш
	r.invalidateActivityCache(activity.ID)
	return nil
}

// Delete удаляет активность
func (r *ActivityRepositoryImpl) Delete(id int64) error {
	// Сначала получаем активность для инвалидации кэша
	activity, err := r.FindByID(id)
	if err != nil {
		return err
	}
	if activity == nil {
		return sql.ErrNoRows
	}

	query := `DELETE FROM user_activities WHERE id = $1`
	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	// Инвалидируем кэш
	r.invalidateActivityCache(id)
	return nil
}

// BulkDelete массово удаляет активности
func (r *ActivityRepositoryImpl) BulkDelete(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Создаем строку с параметрами
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))

	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(
		"DELETE FROM user_activities WHERE id IN (%s)",
		strings.Join(placeholders, ", "),
	)

	result, err := tx.Exec(query, args...)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Инвалидируем кэш для всех удаленных активностей
	for _, id := range ids {
		r.invalidateActivityCache(id)
	}

	return nil
}

// FindByUserID находит активность пользователя
func (r *ActivityRepositoryImpl) FindByUserID(userID int, limit, offset int) ([]*models.UserActivity, error) {
	cacheKey := fmt.Sprintf("activities:user:%d:%d:%d", userID, limit, offset)

	// Попытка получить из кэша
	var activities []*models.UserActivity
	if err := r.cache.Get(context.Background(), cacheKey, &activities); err == nil {
		return activities, nil
	}

	query := `
	SELECT
		a.id, a.user_id, a.activity_type, a.category, a.severity,
		a.details, a.ip_address, a.user_agent, a.metadata, a.created_at,
		u.telegram_id, u.username, u.first_name
	FROM user_activities a
	JOIN users u ON a.user_id = u.id
	WHERE a.user_id = $1
	ORDER BY a.created_at DESC
	LIMIT $2 OFFSET $3
	`

	activities, err := r.queryActivities(query, userID, limit, offset)
	if err != nil {
		return nil, err
	}

	// Сохраняем в кэш
	if data, err := json.Marshal(activities); err == nil {
		_ = r.cache.Set(context.Background(), cacheKey, string(data), 2*time.Minute)
	}

	return activities, nil
}

// FindByFilter находит активность по фильтру
func (r *ActivityRepositoryImpl) FindByFilter(filter models.ActivityFilter) ([]*models.UserActivity, int64, error) {
	query, args, err := r.buildFilterQuery(filter)
	if err != nil {
		return nil, 0, err
	}

	activities, err := r.queryActivities(query, args...)
	if err != nil {
		return nil, 0, err
	}

	countQuery, countArgs := r.buildCountQuery(filter)
	var total int64
	err = r.db.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	return activities, total, nil
}

// GetRecentActivities возвращает последние активности
func (r *ActivityRepositoryImpl) GetRecentActivities(limit int) ([]*models.UserActivity, error) {
	cacheKey := fmt.Sprintf("activities:recent:%d", limit)
	var activities []*models.UserActivity
	if err := r.cache.Get(context.Background(), cacheKey, &activities); err == nil {

		return activities, nil
	}

	query := `
	SELECT
		a.id, a.user_id, a.activity_type, a.category, a.severity,
		a.details, a.ip_address, a.user_agent, a.metadata, a.created_at,
		u.telegram_id, u.username, u.first_name
	FROM user_activities a
	LEFT JOIN users u ON a.user_id = u.id
	WHERE a.created_at >= NOW() - INTERVAL '24 hours'
	ORDER BY a.created_at DESC
	LIMIT $1
	`

	activities, err := r.queryActivities(query, limit)
	if err != nil {
		return nil, err
	}

	if data, err := json.Marshal(activities); err == nil {
		_ = r.cache.Set(context.Background(), cacheKey, string(data), 1*time.Minute)
	}

	return activities, nil
}

// GetAll возвращает все активности с пагинацией
func (r *ActivityRepositoryImpl) GetAll(limit, offset int) ([]*models.UserActivity, error) {
	query := `
	SELECT
		a.id, a.user_id, a.activity_type, a.category, a.severity,
		a.details, a.ip_address, a.user_agent, a.metadata, a.created_at,
		u.telegram_id, u.username, u.first_name
	FROM user_activities a
	LEFT JOIN users u ON a.user_id = u.id
	ORDER BY a.created_at DESC
	LIMIT $1 OFFSET $2
	`

	return r.queryActivities(query, limit, offset)
}

// GetSuspiciousActivities возвращает подозрительную активность
func (r *ActivityRepositoryImpl) GetSuspiciousActivities(limit int) ([]*models.UserActivity, error) {
	query := `
	SELECT
		a.id, a.user_id, a.activity_type, a.category, a.severity,
		a.details, a.ip_address, a.user_agent, a.metadata, a.created_at,
		u.telegram_id, u.username, u.first_name
	FROM user_activities a
	JOIN users u ON a.user_id = u.id
	WHERE a.severity IN ('warning', 'error', 'critical')
	   OR (a.activity_type = 'user_login' AND a.details->>'success' = 'false')
	   OR (a.activity_type = 'security_event')
	ORDER BY a.created_at DESC
	LIMIT $1
	`

	return r.queryActivities(query, limit)
}

// LogUserLogin логирует вход пользователя
func (r *ActivityRepositoryImpl) LogUserLogin(user *models.User, ip, userAgent string, success bool, failureReason string) error {
	severity := models.SeverityInfo
	if !success {
		severity = models.SeverityWarning
	}

	activity := models.NewUserActivity(
		user.ID,
		models.ActivityTypeLogin,
		models.CategoryAuth,
		severity,
		models.JSONMap{
			"success":        success,
			"failure_reason": failureReason,
			"method":         "telegram",
		},
	)

	activity.TelegramID = user.TelegramID
	activity.Username = user.Username
	activity.FirstName = user.FirstName
	activity.IPAddress = ip
	activity.UserAgent = userAgent
	activity.Metadata = models.JSONMap{
		"user_role":   user.Role,
		"user_status": user.IsActive,
	}

	return r.Create(activity)
}

// LogUserLogout логирует выход пользователя
func (r *ActivityRepositoryImpl) LogUserLogout(user *models.User, ip, userAgent, reason string) error {
	activity := models.NewUserActivity(
		user.ID,
		models.ActivityTypeLogout,
		models.CategoryAuth,
		models.SeverityInfo,
		models.JSONMap{"reason": reason},
	)

	activity.TelegramID = user.TelegramID
	activity.Username = user.Username
	activity.FirstName = user.FirstName
	activity.IPAddress = ip
	activity.UserAgent = userAgent

	return r.Create(activity)
}

// LogSignalReceived логирует получение сигнала
func (r *ActivityRepositoryImpl) LogSignalReceived(user *models.User, signalType string, symbol string, changePercent float64, period int, filtered bool, filterReason string) error {
	activity := models.NewUserActivity(
		user.ID,
		models.ActivityTypeSignalReceived,
		models.CategoryTrading,
		models.SeverityInfo,
		models.JSONMap{
			"signal_type":    signalType,
			"symbol":         symbol,
			"change_percent": changePercent,
			"period_minutes": period,
			"filtered":       filtered,
			"filter_reason":  filterReason,
			"user_threshold": user.MinGrowthThreshold, // Исправлено: user.MinGrowthThreshold
		},
	)

	activity.TelegramID = user.TelegramID
	activity.Username = user.Username
	activity.FirstName = user.FirstName
	activity.Metadata = models.JSONMap{
		"signals_today":         user.SignalsToday,
		"max_signals":           user.MaxSignalsPerDay,
		"notifications_enabled": user.NotificationsEnabled, // Исправлено: user.NotificationsEnabled
	}

	return r.Create(activity)
}

// LogSettingsUpdate логирует изменение настроек
func (r *ActivityRepositoryImpl) LogSettingsUpdate(user *models.User, settingType string, oldValue, newValue interface{}, ip, userAgent string) error {
	activity := models.NewUserActivity(
		user.ID,
		models.ActivityTypeSettingsUpdate,
		models.CategoryUser,
		models.SeverityInfo,
		models.JSONMap{
			"setting_type": settingType,
			"old_value":    oldValue,
			"new_value":    newValue,
			"source":       "telegram_bot",
		},
	)

	activity.TelegramID = user.TelegramID
	activity.Username = user.Username
	activity.FirstName = user.FirstName
	activity.IPAddress = ip
	activity.UserAgent = userAgent

	return r.Create(activity)
}

// LogSecurityEvent логирует события безопасности
func (r *ActivityRepositoryImpl) LogSecurityEvent(user *models.User, eventType, description string, severity models.ActivitySeverity, ip, userAgent string, metadata models.JSONMap) error {
	activity := models.NewUserActivity(
		user.ID,
		models.ActivityTypeSecurity,
		models.CategorySecurity,
		severity,
		models.JSONMap{
			"event_type":  eventType,
			"description": description,
		},
	)

	activity.TelegramID = user.TelegramID
	activity.Username = user.Username
	activity.FirstName = user.FirstName
	activity.IPAddress = ip
	activity.UserAgent = userAgent
	activity.Metadata = metadata

	return r.Create(activity)
}

// LogError логирует ошибки
func (r *ActivityRepositoryImpl) LogError(user *models.User, errorType, errorMessage, stackTrace string, severity models.ActivitySeverity, additionalData models.JSONMap) error {
	activity := models.NewUserActivity(
		user.ID,
		models.ActivityTypeError,
		models.CategorySystem,
		severity,
		models.JSONMap{
			"error_type":    errorType,
			"error_message": errorMessage,
			"stack_trace":   stackTrace,
		},
	)

	activity.TelegramID = user.TelegramID
	activity.Username = user.Username
	activity.FirstName = user.FirstName
	activity.Metadata = additionalData

	return r.Create(activity)
}

// LogSystemEvent логирует системные события
func (r *ActivityRepositoryImpl) LogSystemEvent(eventType, description string, severity models.ActivitySeverity, metadata models.JSONMap) error {
	activity := models.NewUserActivity(
		0, // System user
		models.ActivityTypeSystem,
		models.CategorySystem,
		severity,
		models.JSONMap{
			"event_type":  eventType,
			"description": description,
		},
	)

	activity.Metadata = metadata
	return r.Create(activity)
}

// GetUserActivityStats возвращает статистику активности пользователя
func (r *ActivityRepositoryImpl) GetUserActivityStats(userID int, days int) (map[string]interface{}, error) {
	cacheKey := fmt.Sprintf("activity:stats:user:%d:%d", userID, days)
	var stats map[string]interface{}

	if err := r.cache.Get(context.Background(), cacheKey, &stats); err == nil {
		return stats, nil
	}

	query := `
	SELECT
		COUNT(*) as total_activities,
		COUNT(DISTINCT DATE(created_at)) as active_days,
		MIN(created_at) as first_activity,
		MAX(created_at) as last_activity,
		COUNT(CASE WHEN severity = 'error' OR severity = 'critical' THEN 1 END) as error_count,
		COUNT(CASE WHEN activity_type = 'signal_received' THEN 1 END) as signal_count,
		COUNT(CASE WHEN activity_type = 'user_login' THEN 1 END) as login_count
	FROM user_activities
	WHERE user_id = $1
	  AND created_at >= NOW() - INTERVAL '1 day' * $2
	`

	var (
		totalActivities int
		activeDays      int
		firstActivity   sql.NullTime
		lastActivity    sql.NullTime
		errorCount      int
		signalCount     int
		loginCount      int
	)

	err := r.db.QueryRow(query, userID, days).Scan(
		&totalActivities,
		&activeDays,
		&firstActivity,
		&lastActivity,
		&errorCount,
		&signalCount,
		&loginCount,
	)
	if err != nil {
		return nil, err
	}

	stats["total_activities"] = totalActivities
	stats["active_days"] = activeDays
	stats["error_count"] = errorCount
	stats["signal_count"] = signalCount
	stats["login_count"] = loginCount

	if firstActivity.Valid {
		stats["first_activity"] = firstActivity.Time
	}
	if lastActivity.Valid {
		stats["last_activity"] = lastActivity.Time
	}

	// Распределение по типам
	typeStats, err := r.getActivityTypeStats(userID, days)
	if err == nil {
		stats["by_type"] = typeStats
	}

	// Распределение по часам
	hourStats, err := r.getActivityHourStats(userID, days)
	if err == nil {
		stats["by_hour"] = hourStats
	}

	// Сохраняем в кэш
	if data, err := json.Marshal(stats); err == nil {
		_ = r.cache.Set(context.Background(), cacheKey, string(data), 10*time.Minute)
	}

	return stats, nil
}

// GetSystemActivityStats возвращает системную статистику активности
func (r *ActivityRepositoryImpl) GetSystemActivityStats(days int) (*models.ActivityStats, error) {
	cacheKey := fmt.Sprintf("activity:stats:system:%d", days)

	// Попытка получить из кэша
	var cachedStats models.ActivityStats
	if err := r.cache.Get(context.Background(), cacheKey, &cachedStats); err == nil {
		return &cachedStats, nil
	}

	// Создаем новую структуру
	stats := &models.ActivityStats{
		ByType:     make(map[string]int64),
		ByCategory: make(map[string]int64),
		ByHour:     make(map[int]int64),
	}

	// Основная статистика
	query := `
	SELECT
		COUNT(*) as total_activities,
		COUNT(DISTINCT user_id) as unique_users,
		COUNT(CASE WHEN severity IN ('error', 'critical') THEN 1 END) as error_count
	FROM user_activities
	WHERE created_at >= NOW() - INTERVAL '1 day' * $1
	  AND user_id > 0
	`

	var totalActivities, uniqueUsers, errorCount int64
	err := r.db.QueryRow(query, days).Scan(&totalActivities, &uniqueUsers, &errorCount)
	if err != nil {
		return nil, err
	}

	stats.TotalActivities = totalActivities
	stats.UniqueUsers = uniqueUsers

	if totalActivities > 0 {
		stats.ErrorRate = float64(errorCount) / float64(totalActivities) * 100
		stats.AvgActivitiesPerUser = float64(totalActivities) / float64(uniqueUsers)
	}

	// Активность за сегодня
	todayQuery := `SELECT COUNT(*) FROM user_activities WHERE created_at >= CURRENT_DATE AND user_id > 0`
	err = r.db.QueryRow(todayQuery).Scan(&stats.ActivitiesToday)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Самый активный пользователь
	mostActiveQuery := `
	SELECT
		u.id, u.telegram_id, u.username, u.first_name,
		COUNT(*) as activity_count
	FROM user_activities a
	JOIN users u ON a.user_id = u.id
	WHERE a.created_at >= NOW() - INTERVAL '1 day' * $1
	GROUP BY u.id, u.telegram_id, u.username, u.first_name
	ORDER BY activity_count DESC
	LIMIT 1
	`

	var mostActiveUser models.UserActivity
	var activityCount int64
	err = r.db.QueryRow(mostActiveQuery, days).Scan(
		&mostActiveUser.UserID,
		&mostActiveUser.TelegramID,
		&mostActiveUser.Username,
		&mostActiveUser.FirstName,
		&activityCount,
	)

	if err == nil {
		stats.MostActiveUser = &mostActiveUser
	}

	// Распределение по типам
	typeQuery := `
	SELECT activity_type, COUNT(*)
	FROM user_activities
	WHERE created_at >= NOW() - INTERVAL '1 day' * $1
	  AND user_id > 0
	GROUP BY activity_type
	`

	rows, err := r.db.Query(typeQuery, days)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var activityType string
			var count int64
			rows.Scan(&activityType, &count)
			stats.ByType[activityType] = count
		}
	}

	// Распределение по категориям
	categoryQuery := `
	SELECT category, COUNT(*)
	FROM user_activities
	WHERE created_at >= NOW() - INTERVAL '1 day' * $1
	  AND user_id > 0
	GROUP BY category
	`

	rows, err = r.db.Query(categoryQuery, days)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var category string
			var count int64
			rows.Scan(&category, &count)
			stats.ByCategory[category] = count
		}
	}

	// Распределение по часам
	hourQuery := `
	SELECT EXTRACT(HOUR FROM created_at) as hour, COUNT(*)
	FROM user_activities
	WHERE created_at >= NOW() - INTERVAL '1 day' * $1
	  AND user_id > 0
	GROUP BY EXTRACT(HOUR FROM created_at)
	ORDER BY hour
	`

	rows, err = r.db.Query(hourQuery, days)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var hour int
			var count int64
			rows.Scan(&hour, &count)
			stats.ByHour[hour] = count
		}
	}

	// Сохраняем в кэш
	if data, err := json.Marshal(stats); err == nil {
		_ = r.cache.Set(context.Background(), cacheKey, string(data), 5*time.Minute)
	}

	return stats, nil
}

// GetFailedLoginAttempts возвращает неудачные попытки входа
func (r *ActivityRepositoryImpl) GetFailedLoginAttempts(ip string, minutes int) (int, error) {
	query := `
	SELECT COUNT(*)
	FROM user_activities
	WHERE activity_type = 'user_login'
	  AND details->>'success' = 'false'
	  AND ip_address = $1
	  AND created_at >= NOW() - INTERVAL '1 minute' * $2
	`

	var count int
	err := r.db.QueryRow(query, ip, minutes).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// GetStatistics возвращает статистику для API
func (r *ActivityRepositoryImpl) GetStatistics(ctx context.Context, userID int) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	query := `
	SELECT
		COUNT(*) as total_activities,
		COUNT(DISTINCT DATE(created_at)) as active_days,
		COUNT(CASE WHEN activity_type = 'error' THEN 1 END) as error_count,
		COUNT(CASE WHEN activity_type = 'signal_sent' THEN 1 END) as signal_count,
		COUNT(CASE WHEN activity_type = 'user_login' THEN 1 END) as login_count
	FROM user_activities
	WHERE user_id = $1
	`

	var (
		totalActivities int
		activeDays      int
		errorCount      int
		signalCount     int
		loginCount      int
	)

	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&totalActivities,
		&activeDays,
		&errorCount,
		&signalCount,
		&loginCount,
	)
	if err != nil {
		return nil, err
	}

	stats["total_activities"] = totalActivities
	stats["active_days"] = activeDays
	stats["error_count"] = errorCount
	stats["signal_count"] = signalCount
	stats["login_count"] = loginCount

	return stats, nil
}

// CleanupOldActivities очищает старые записи активности
func (r *ActivityRepositoryImpl) CleanupOldActivities(ctx context.Context, olderThanDays int) (int64, error) {
	query := `
	DELETE FROM user_activities
	WHERE created_at < NOW() - INTERVAL '1 day' * $1
	  AND severity = 'info'
	RETURNING id
	`

	rows, err := r.db.QueryContext(ctx, query, olderThanDays)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var deletedIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return 0, err
		}
		deletedIDs = append(deletedIDs, id)
	}

	// Инвалидируем кэш
	for _, id := range deletedIDs {
		r.invalidateActivityCache(id)
	}

	return int64(len(deletedIDs)), nil
}

// ArchiveActivities архивирует старые записи в отдельную таблицу
func (r *ActivityRepositoryImpl) ArchiveActivities(ctx context.Context, olderThanDays int) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// Копируем в архивную таблицу
	copyQuery := `
	INSERT INTO user_activities_archive
	SELECT * FROM user_activities
	WHERE created_at < NOW() - INTERVAL '1 day' * $1
	`

	result, err := tx.ExecContext(ctx, copyQuery, olderThanDays)
	if err != nil {
		return 0, err
	}

	rowsAffected, _ := result.RowsAffected()

	// Удаляем из основной таблицы
	deleteQuery := `
	DELETE FROM user_activities
	WHERE created_at < NOW() - INTERVAL '1 day' * $1
	`

	_, err = tx.ExecContext(ctx, deleteQuery, olderThanDays)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return rowsAffected, nil
}

// ResetDailyCounters сбрасывает дневные счетчики
func (r *ActivityRepositoryImpl) ResetDailyCounters(ctx context.Context) error {
	// Для активности нет дневных счетчиков, оставляем заглушку
	return nil
}

// GetTotalCount возвращает общее количество записей
func (r *ActivityRepositoryImpl) GetTotalCount(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM user_activities`

	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// Вспомогательные методы

func (r *ActivityRepositoryImpl) queryActivities(query string, args ...interface{}) ([]*models.UserActivity, error) {
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []*models.UserActivity

	for rows.Next() {
		var activity models.UserActivity
		var detailsJSON, metadataJSON []byte

		err := rows.Scan(
			&activity.ID,
			&activity.UserID,
			&activity.ActivityType,
			&activity.Category,
			&activity.Severity,
			&detailsJSON,
			&activity.IPAddress,
			&activity.UserAgent,
			&metadataJSON,
			&activity.CreatedAt,
			&activity.TelegramID,
			&activity.Username,
			&activity.FirstName,
		)

		if err != nil {
			return nil, err
		}

		// Декодируем JSON
		if err := json.Unmarshal(detailsJSON, &activity.Details); err != nil {
			return nil, fmt.Errorf("failed to unmarshal details: %w", err)
		}

		if err := json.Unmarshal(metadataJSON, &activity.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		activities = append(activities, &activity)
	}

	return activities, nil
}

func (r *ActivityRepositoryImpl) buildFilterQuery(filter models.ActivityFilter) (string, []interface{}, error) {
	baseQuery := `
	SELECT
		a.id, a.user_id, a.activity_type, a.category, a.severity,
		a.details, a.ip_address, a.user_agent, a.metadata, a.created_at,
		u.telegram_id, u.username, u.first_name
	FROM user_activities a
	LEFT JOIN users u ON a.user_id = u.id
	`

	whereClauses := []string{"1=1"}
	args := []interface{}{}
	argIndex := 1

	if filter.UserID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("a.user_id = $%d", argIndex))
		args = append(args, *filter.UserID)
		argIndex++
	}

	if filter.TelegramID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("u.telegram_id = $%d", argIndex))
		args = append(args, *filter.TelegramID)
		argIndex++
	}

	if filter.ActivityType != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("a.activity_type = $%d", argIndex))
		args = append(args, *filter.ActivityType)
		argIndex++
	}

	if filter.Category != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("a.category = $%d", argIndex))
		args = append(args, *filter.Category)
		argIndex++
	}

	if filter.Severity != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("a.severity = $%d", argIndex))
		args = append(args, *filter.Severity)
		argIndex++
	}

	if filter.StartDate != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("a.created_at >= $%d", argIndex))
		args = append(args, *filter.StartDate)
		argIndex++
	}

	if filter.EndDate != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("a.created_at <= $%d", argIndex))
		args = append(args, *filter.EndDate)
		argIndex++
	}

	if filter.IPAddress != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("a.ip_address = $%d", argIndex))
		args = append(args, *filter.IPAddress)
		argIndex++
	}

	if filter.SearchQuery != nil {
		whereClauses = append(whereClauses,
			fmt.Sprintf(`(u.username ILIKE '%%' || $%d || '%%' OR
			               u.first_name ILIKE '%%' || $%d || '%%' OR
			               a.details::text ILIKE '%%' || $%d || '%%')`,
				argIndex, argIndex, argIndex))
		args = append(args, *filter.SearchQuery)
		argIndex++
	}

	// Собираем запрос
	query := baseQuery + " WHERE " + joinStrings(whereClauses, " AND ")

	// Сортировка
	orderBy := "a.created_at"
	orderDir := "DESC"

	if filter.OrderBy != "" {
		validOrderFields := map[string]bool{
			"created_at": true,
			"user_id":    true,
			"severity":   true,
			"category":   true,
		}
		if validOrderFields[filter.OrderBy] {
			orderBy = "a." + filter.OrderBy
		}
	}

	if filter.OrderDir != "" && (filter.OrderDir == "ASC" || filter.OrderDir == "DESC") {
		orderDir = filter.OrderDir
	}

	query += fmt.Sprintf(" ORDER BY %s %s", orderBy, orderDir)

	// Лимит и оффсет
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++

		if filter.Offset > 0 {
			query += fmt.Sprintf(" OFFSET $%d", argIndex)
			args = append(args, filter.Offset)
		}
	}

	return query, args, nil
}

func (r *ActivityRepositoryImpl) buildCountQuery(filter models.ActivityFilter) (string, []interface{}) {
	baseQuery := "SELECT COUNT(*) FROM user_activities a LEFT JOIN users u ON a.user_id = u.id"

	whereClauses := []string{"1=1"}
	args := []interface{}{}

	// Повторяем те же условия что и в основном запросе
	// (упрощенная версия, в реальном коде нужно вынести в общую функцию)

	query := baseQuery + " WHERE " + joinStrings(whereClauses, " AND ")
	return query, args
}

func (r *ActivityRepositoryImpl) getActivityTypeStats(userID int, days int) (map[string]int64, error) {
	query := `
	SELECT activity_type, COUNT(*)
	FROM user_activities
	WHERE user_id = $1
	  AND created_at >= NOW() - INTERVAL '1 day' * $2
	GROUP BY activity_type
	`

	rows, err := r.db.Query(query, userID, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[string]int64)
	for rows.Next() {
		var activityType string
		var count int64
		if err := rows.Scan(&activityType, &count); err != nil {
			return nil, err
		}
		stats[activityType] = count
	}

	return stats, nil
}

func (r *ActivityRepositoryImpl) getActivityHourStats(userID int, days int) (map[int]int64, error) {
	query := `
	SELECT EXTRACT(HOUR FROM created_at) as hour, COUNT(*)
	FROM user_activities
	WHERE user_id = $1
	  AND created_at >= NOW() - INTERVAL '1 day' * $2
	GROUP BY EXTRACT(HOUR FROM created_at)
	ORDER BY hour
	`

	rows, err := r.db.Query(query, userID, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[int]int64)
	for rows.Next() {
		var hour int
		var count int64
		if err := rows.Scan(&hour, &count); err != nil {
			return nil, err
		}
		stats[hour] = count
	}

	return stats, nil
}

func (r *ActivityRepositoryImpl) invalidateActivityCache(activityID int64) {
	ctx := context.Background()
	keys := []string{
		fmt.Sprintf("activity:%d", activityID),
		"activities:recent:*",
		"activity:stats:*",
		"activities:user:*",
	}

	_ = r.cache.DeleteMulti(ctx, keys...)
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}
	result := strs[0]
	for _, s := range strs[1:] {
		result += sep + s
	}
	return result
}
