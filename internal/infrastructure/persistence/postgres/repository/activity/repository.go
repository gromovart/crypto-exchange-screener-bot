// persistence/postgres/repository/activity_repository.go
package activity

// import (
// 	"context"
// 	"database/sql"
// 	"encoding/json"
// 	"fmt"
// 	"time"

// 	"crypto-exchange-screener-bot/persistence/postgres/repository/api_key"
// 	"crypto-exchange-screener-bot/persistence/postgres/repository/users"

// 	"github.com/jmoiron/sqlx"
// )

// // ActivityRepository управляет активностью пользователей
// type ActivityRepository struct {
// 	db *sqlx.DB
// }

// // ActivityType типы активности
// type ActivityType string

// const (
// 	ActivityTypeLogin          ActivityType = "user_login"
// 	ActivityTypeLogout         ActivityType = "user_logout"
// 	ActivityTypeProfileUpdate  ActivityType = "profile_update"
// 	ActivityTypeSettingsUpdate ActivityType = "settings_update"
// 	ActivityTypeSignalReceived ActivityType = "signal_received"
// 	ActivityTypeSignalFiltered ActivityType = "signal_filtered"
// 	ActivityTypeNotification   ActivityType = "notification_sent"
// 	ActivityTypeApiCall        ActivityType = "api_call"
// 	ActivityTypeError          ActivityType = "error_occurred"
// 	ActivityTypeSystem         ActivityType = "system_event"
// 	ActivityTypeSubscription   ActivityType = "subscription_event"
// 	ActivityTypeSecurity       ActivityType = "security_event"
// )

// // ActivitySeverity уровень серьезности
// type ActivitySeverity string

// const (
// 	SeverityInfo     ActivitySeverity = "info"
// 	SeverityWarning  ActivitySeverity = "warning"
// 	SeverityError    ActivitySeverity = "error"
// 	SeverityCritical ActivitySeverity = "critical"
// )

// // ActivityCategory категория активности
// type ActivityCategory string

// const (
// 	CategoryAuth      ActivityCategory = "authentication"
// 	CategoryUser      ActivityCategory = "user_actions"
// 	CategoryTrading   ActivityCategory = "trading"
// 	CategorySystem    ActivityCategory = "system"
// 	CategorySecurity  ActivityCategory = "security"
// 	CategoryBilling   ActivityCategory = "billing"
// 	CategoryAnalytics ActivityCategory = "analytics"
// )

// // UserActivity запись активности пользователя
// type UserActivity struct {
// 	ID           int64            `db:"id" json:"id"`
// 	UserID       int              `db:"user_id" json:"user_id"`
// 	TelegramID   int64            `db:"telegram_id" json:"telegram_id,omitempty"`
// 	Username     string           `db:"username" json:"username,omitempty"`
// 	FirstName    string           `db:"first_name" json:"first_name,omitempty"`
// 	ActivityType ActivityType     `db:"activity_type" json:"activity_type"`
// 	Category     ActivityCategory `db:"category" json:"category"`
// 	Severity     ActivitySeverity `db:"severity" json:"severity"`
// 	Details      api_key.JSONMap  `db:"details" json:"details"`
// 	IPAddress    string           `db:"ip_address" json:"ip_address,omitempty"`
// 	UserAgent    string           `db:"user_agent" json:"user_agent,omitempty"`
// 	Metadata     api_key.JSONMap  `db:"metadata" json:"metadata,omitempty"`
// 	CreatedAt    time.Time        `db:"created_at" json:"created_at"`
// }

// // ActivityStats статистика активности
// type ActivityStats struct {
// 	TotalActivities      int64            `json:"total_activities"`
// 	ActivitiesToday      int64            `json:"activities_today"`
// 	UniqueUsers          int64            `json:"unique_users"`
// 	MostActiveUser       *UserActivity    `json:"most_active_user,omitempty"`
// 	ByType               map[string]int64 `json:"by_type"`
// 	ByCategory           map[string]int64 `json:"by_category"`
// 	ByHour               map[int]int64    `json:"by_hour"`
// 	ErrorRate            float64          `json:"error_rate"`
// 	AvgActivitiesPerUser float64          `json:"avg_activities_per_user"`
// }

// // ActivityFilter фильтр для поиска активности
// type ActivityFilter struct {
// 	UserID       *int              `json:"user_id,omitempty"`
// 	TelegramID   *int64            `json:"telegram_id,omitempty"`
// 	ActivityType *ActivityType     `json:"activity_type,omitempty"`
// 	Category     *ActivityCategory `json:"category,omitempty"`
// 	Severity     *ActivitySeverity `json:"severity,omitempty"`
// 	StartDate    *time.Time        `json:"start_date,omitempty"`
// 	EndDate      *time.Time        `json:"end_date,omitempty"`
// 	IPAddress    *string           `json:"ip_address,omitempty"`
// 	SearchQuery  *string           `json:"search_query,omitempty"`
// 	Limit        int               `json:"limit,omitempty"`
// 	Offset       int               `json:"offset,omitempty"`
// 	OrderBy      string            `json:"order_by,omitempty"`
// 	OrderDir     string            `json:"order_dir,omitempty"`
// }

// // NewActivityRepository создает новый репозиторий активности
// func NewActivityRepository(db *sqlx.DB) *ActivityRepository {
// 	return &ActivityRepository{db: db}
// }

// // Create создает запись активности
// func (r *ActivityRepository) Create(activity *UserActivity) error {
// 	query := `
// 	INSERT INTO user_activities (
// 		user_id, activity_type, category, severity, details,
// 		ip_address, user_agent, metadata
// 	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
// 	RETURNING id, created_at
// 	`

// 	detailsJSON, err := json.Marshal(activity.Details)
// 	if err != nil {
// 		return fmt.Errorf("failed to marshal details: %w", err)
// 	}

// 	metadataJSON, err := json.Marshal(activity.Metadata)
// 	if err != nil {
// 		return fmt.Errorf("failed to marshal metadata: %w", err)
// 	}

// 	return r.db.QueryRow(
// 		query,
// 		activity.UserID,
// 		activity.ActivityType,
// 		activity.Category,
// 		activity.Severity,
// 		detailsJSON,
// 		activity.IPAddress,
// 		activity.UserAgent,
// 		metadataJSON,
// 	).Scan(&activity.ID, &activity.CreatedAt)
// }

// // CreateWithUser создает активность с информацией о пользователе
// func (r *ActivityRepository) CreateWithUser(activity *UserActivity, user *users.User) error {
// 	activity.TelegramID = user.TelegramID
// 	activity.Username = user.Username
// 	activity.FirstName = user.FirstName
// 	return r.Create(activity)
// }

// // LogUserLogin логирует вход пользователя
// func (r *ActivityRepository) LogUserLogin(user *users.User, ip, userAgent string, success bool, failureReason string) error {
// 	severity := SeverityInfo
// 	if !success {
// 		severity = SeverityWarning
// 	}

// 	activity := &UserActivity{
// 		UserID:       user.ID,
// 		TelegramID:   user.TelegramID,
// 		Username:     user.Username,
// 		FirstName:    user.FirstName,
// 		ActivityType: ActivityTypeLogin,
// 		Category:     CategoryAuth,
// 		Severity:     severity,
// 		Details: api_key.JSONMap{
// 			"success":        success,
// 			"failure_reason": failureReason,
// 			"method":         "telegram",
// 		},
// 		IPAddress: ip,
// 		UserAgent: userAgent,
// 		Metadata: api_key.JSONMap{
// 			"user_role":   user.Role,
// 			"user_status": user.IsActive,
// 		},
// 	}

// 	return r.Create(activity)
// }

// // LogUserLogout логирует выход пользователя
// func (r *ActivityRepository) LogUserLogout(user *users.User, ip, userAgent, reason string) error {
// 	activity := &UserActivity{
// 		UserID:       user.ID,
// 		TelegramID:   user.TelegramID,
// 		Username:     user.Username,
// 		FirstName:    user.FirstName,
// 		ActivityType: ActivityTypeLogout,
// 		Category:     CategoryAuth,
// 		Severity:     SeverityInfo,
// 		Details: api_key.JSONMap{
// 			"reason": reason,
// 		},
// 		IPAddress: ip,
// 		UserAgent: userAgent,
// 	}

// 	return r.Create(activity)
// }

// // LogSignalReceived логирует получение сигнала
// func (r *ActivityRepository) LogSignalReceived(user *users.User, signalType string, symbol string, changePercent float64, period int, filtered bool, filterReason string) error {
// 	activity := &UserActivity{
// 		UserID:       user.ID,
// 		TelegramID:   user.TelegramID,
// 		Username:     user.Username,
// 		FirstName:    user.FirstName,
// 		ActivityType: ActivityTypeSignalReceived,
// 		Category:     CategoryTrading,
// 		Severity:     SeverityInfo,
// 		Details: api_key.JSONMap{
// 			"signal_type":    signalType,
// 			"symbol":         symbol,
// 			"change_percent": changePercent,
// 			"period_minutes": period,
// 			"filtered":       filtered,
// 			"filter_reason":  filterReason,
// 			"user_threshold": user.Settings.MinGrowthThreshold,
// 		},
// 		Metadata: api_key.JSONMap{
// 			"signals_today":         user.SignalsToday,
// 			"max_signals":           user.MaxSignalsPerDay,
// 			"notifications_enabled": user.Notifications.Enabled,
// 		},
// 	}

// 	return r.Create(activity)
// }

// // LogSettingsUpdate логирует изменение настроек
// func (r *ActivityRepository) LogSettingsUpdate(user *users.User, settingType string, oldValue, newValue interface{}, ip, userAgent string) error {
// 	activity := &UserActivity{
// 		UserID:       user.ID,
// 		TelegramID:   user.TelegramID,
// 		Username:     user.Username,
// 		FirstName:    user.FirstName,
// 		ActivityType: ActivityTypeSettingsUpdate,
// 		Category:     CategoryUser,
// 		Severity:     SeverityInfo,
// 		Details: api_key.JSONMap{
// 			"setting_type": settingType,
// 			"old_value":    oldValue,
// 			"new_value":    newValue,
// 			"source":       "telegram_bot",
// 		},
// 		IPAddress: ip,
// 		UserAgent: userAgent,
// 	}

// 	return r.Create(activity)
// }

// // LogSecurityEvent логирует события безопасности
// func (r *ActivityRepository) LogSecurityEvent(user *users.User, eventType, description string, severity ActivitySeverity, ip, userAgent string, metadata api_key.JSONMap) error {
// 	activity := &UserActivity{
// 		UserID:       user.ID,
// 		TelegramID:   user.TelegramID,
// 		Username:     user.Username,
// 		FirstName:    user.FirstName,
// 		ActivityType: ActivityTypeSecurity,
// 		Category:     CategorySecurity,
// 		Severity:     severity,
// 		Details: api_key.JSONMap{
// 			"event_type":  eventType,
// 			"description": description,
// 		},
// 		IPAddress: ip,
// 		UserAgent: userAgent,
// 		Metadata:  metadata,
// 	}

// 	return r.Create(activity)
// }

// // LogError логирует ошибки
// func (r *ActivityRepository) LogError(user *users.User, errorType, errorMessage, stackTrace string, severity ActivitySeverity, additionalData api_key.JSONMap) error {
// 	activity := &UserActivity{
// 		UserID:       user.ID,
// 		TelegramID:   user.TelegramID,
// 		Username:     user.Username,
// 		FirstName:    user.FirstName,
// 		ActivityType: ActivityTypeError,
// 		Category:     CategorySystem,
// 		Severity:     severity,
// 		Details: api_key.JSONMap{
// 			"error_type":    errorType,
// 			"error_message": errorMessage,
// 			"stack_trace":   stackTrace,
// 		},
// 		Metadata: additionalData,
// 	}

// 	return r.Create(activity)
// }

// // LogSystemEvent логирует системные события
// func (r *ActivityRepository) LogSystemEvent(eventType, description string, severity ActivitySeverity, metadata api_key.JSONMap) error {
// 	activity := &UserActivity{
// 		UserID:       0, // System user
// 		ActivityType: ActivityTypeSystem,
// 		Category:     CategorySystem,
// 		Severity:     severity,
// 		Details: api_key.JSONMap{
// 			"event_type":  eventType,
// 			"description": description,
// 		},
// 		Metadata: metadata,
// 	}

// 	return r.Create(activity)
// }

// // FindByID находит активность по ID
// func (r *ActivityRepository) FindByID(id int64) (*UserActivity, error) {
// 	query := `
// 	SELECT
// 		a.id, a.user_id, a.activity_type, a.category, a.severity,
// 		a.details, a.ip_address, a.user_agent, a.metadata, a.created_at,
// 		u.telegram_id, u.username, u.first_name
// 	FROM user_activities a
// 	LEFT JOIN users u ON a.user_id = u.id
// 	WHERE a.id = $1
// 	`

// 	var activity UserActivity
// 	var detailsJSON, metadataJSON []byte

// 	err := r.db.QueryRow(query, id).Scan(
// 		&activity.ID,
// 		&activity.UserID,
// 		&activity.ActivityType,
// 		&activity.Category,
// 		&activity.Severity,
// 		&detailsJSON,
// 		&activity.IPAddress,
// 		&activity.UserAgent,
// 		&metadataJSON,
// 		&activity.CreatedAt,
// 		&activity.TelegramID,
// 		&activity.Username,
// 		&activity.FirstName,
// 	)

// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			return nil, nil
// 		}
// 		return nil, err
// 	}

// 	// Декодируем JSON
// 	if err := json.Unmarshal(detailsJSON, &activity.Details); err != nil {
// 		return nil, fmt.Errorf("failed to unmarshal details: %w", err)
// 	}

// 	if err := json.Unmarshal(metadataJSON, &activity.Metadata); err != nil {
// 		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
// 	}

// 	return &activity, nil
// }

// // FindByUserID находит активность пользователя
// func (r *ActivityRepository) FindByUserID(userID int, limit, offset int) ([]*UserActivity, error) {
// 	query := `
// 	SELECT
// 		a.id, a.user_id, a.activity_type, a.category, a.severity,
// 		a.details, a.ip_address, a.user_agent, a.metadata, a.created_at,
// 		u.telegram_id, u.username, u.first_name
// 	FROM user_activities a
// 	JOIN users u ON a.user_id = u.id
// 	WHERE a.user_id = $1
// 	ORDER BY a.created_at DESC
// 	LIMIT $2 OFFSET $3
// 	`

// 	return r.queryActivities(query, userID, limit, offset)
// }

// // FindByFilter находит активность по фильтру
// func (r *ActivityRepository) FindByFilter(filter ActivityFilter) ([]*UserActivity, int64, error) {
// 	// Строим запрос
// 	query, args, err := r.buildFilterQuery(filter)
// 	if err != nil {
// 		return nil, 0, err
// 	}

// 	// Получаем данные
// 	activities, err := r.queryActivities(query, args...)
// 	if err != nil {
// 		return nil, 0, err
// 	}

// 	// Получаем общее количество
// 	countQuery, countArgs := r.buildCountQuery(filter)
// 	var total int64
// 	err = r.db.QueryRow(countQuery, countArgs...).Scan(&total)
// 	if err != nil {
// 		return nil, 0, err
// 	}

// 	return activities, total, nil
// }

// // GetRecentActivities возвращает последние активности
// func (r *ActivityRepository) GetRecentActivities(limit int) ([]*UserActivity, error) {
// 	query := `
// 	SELECT
// 		a.id, a.user_id, a.activity_type, a.category, a.severity,
// 		a.details, a.ip_address, a.user_agent, a.metadata, a.created_at,
// 		u.telegram_id, u.username, u.first_name
// 	FROM user_activities a
// 	LEFT JOIN users u ON a.user_id = u.id
// 	WHERE a.created_at >= NOW() - INTERVAL '24 hours'
// 	ORDER BY a.created_at DESC
// 	LIMIT $1
// 	`

// 	return r.queryActivities(query, limit)
// }

// // GetUserActivityStats возвращает статистику активности пользователя (ИСПРАВЛЕННЫЙ МЕТОД)
// func (r *ActivityRepository) GetUserActivityStats(userID int, days int) (map[string]interface{}, error) {
// 	query := `
// 	SELECT
// 		COUNT(*) as total_activities,
// 		COUNT(DISTINCT DATE(created_at)) as active_days,
// 		MIN(created_at) as first_activity,
// 		MAX(created_at) as last_activity,
// 		COUNT(CASE WHEN severity = 'error' OR severity = 'critical' THEN 1 END) as error_count,
// 		COUNT(CASE WHEN activity_type = 'signal_received' THEN 1 END) as signal_count,
// 		COUNT(CASE WHEN activity_type = 'user_login' THEN 1 END) as login_count
// 	FROM user_activities
// 	WHERE user_id = $1
// 	  AND created_at >= NOW() - INTERVAL '1 day' * $2
// 	`

// 	// Используем отдельные переменные для сканирования
// 	var (
// 		totalActivities int
// 		activeDays      int
// 		firstActivity   sql.NullTime
// 		lastActivity    sql.NullTime
// 		errorCount      int
// 		signalCount     int
// 		loginCount      int
// 	)

// 	err := r.db.QueryRow(query, userID, days).Scan(
// 		&totalActivities,
// 		&activeDays,
// 		&firstActivity,
// 		&lastActivity,
// 		&errorCount,
// 		&signalCount,
// 		&loginCount,
// 	)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Заполняем map после сканирования
// 	stats := make(map[string]interface{})
// 	stats["total_activities"] = totalActivities
// 	stats["active_days"] = activeDays
// 	stats["error_count"] = errorCount
// 	stats["signal_count"] = signalCount
// 	stats["login_count"] = loginCount

// 	if firstActivity.Valid {
// 		stats["first_activity"] = firstActivity.Time
// 	} else {
// 		stats["first_activity"] = nil
// 	}

// 	if lastActivity.Valid {
// 		stats["last_activity"] = lastActivity.Time
// 	} else {
// 		stats["last_activity"] = nil
// 	}

// 	// Распределение по типам
// 	typeStats, err := r.getActivityTypeStats(userID, days)
// 	if err == nil {
// 		stats["by_type"] = typeStats
// 	}

// 	// Распределение по часам
// 	hourStats, err := r.getActivityHourStats(userID, days)
// 	if err == nil {
// 		stats["by_hour"] = hourStats
// 	}

// 	return stats, nil
// }

// // GetStatistics возвращает статистику для API (ИСПРАВЛЕННЫЙ МЕТОД, если он существует)
// func (r *ActivityRepository) GetStatistics(ctx context.Context, userID int) (map[string]interface{}, error) {
// 	stats := make(map[string]interface{})

// 	query := `
// 	SELECT
// 		COUNT(*) as total_activities,
// 		COUNT(DISTINCT DATE(created_at)) as active_days,
// 		COUNT(CASE WHEN activity_type = 'error' THEN 1 END) as error_count,
// 		COUNT(CASE WHEN activity_type = 'signal_sent' THEN 1 END) as signal_count,
// 		COUNT(CASE WHEN activity_type = 'user_login' THEN 1 END) as login_count
// 	FROM user_activities
// 	WHERE user_id = $1
// 	`

// 	// Используем отдельные переменные для сканирования
// 	var (
// 		totalActivities int
// 		activeDays      int
// 		errorCount      int
// 		signalCount     int
// 		loginCount      int
// 	)

// 	err := r.db.QueryRowContext(ctx, query, userID).Scan(
// 		&totalActivities,
// 		&activeDays,
// 		&errorCount,
// 		&signalCount,
// 		&loginCount,
// 	)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Затем заполняем map
// 	stats["total_activities"] = totalActivities
// 	stats["active_days"] = activeDays
// 	stats["error_count"] = errorCount
// 	stats["signal_count"] = signalCount
// 	stats["login_count"] = loginCount

// 	return stats, nil
// }

// // GetSystemActivityStats возвращает системную статистику активности
// func (r *ActivityRepository) GetSystemActivityStats(days int) (*ActivityStats, error) {
// 	stats := &ActivityStats{
// 		ByType:     make(map[string]int64),
// 		ByCategory: make(map[string]int64),
// 		ByHour:     make(map[int]int64),
// 	}

// 	// Основная статистика
// 	query := `
// 	SELECT
// 		COUNT(*) as total_activities,
// 		COUNT(DISTINCT user_id) as unique_users,
// 		COUNT(CASE WHEN severity IN ('error', 'critical') THEN 1 END) as error_count
// 	FROM user_activities
// 	WHERE created_at >= NOW() - INTERVAL '1 day' * $1
// 	  AND user_id > 0
// 	`

// 	var totalActivities, uniqueUsers, errorCount int64
// 	err := r.db.QueryRow(query, days).Scan(&totalActivities, &uniqueUsers, &errorCount)
// 	if err != nil {
// 		return nil, err
// 	}

// 	stats.TotalActivities = totalActivities
// 	stats.UniqueUsers = uniqueUsers

// 	if totalActivities > 0 {
// 		stats.ErrorRate = float64(errorCount) / float64(totalActivities) * 100
// 		stats.AvgActivitiesPerUser = float64(totalActivities) / float64(uniqueUsers)
// 	}

// 	// Активность за сегодня
// 	todayQuery := `
// 	SELECT COUNT(*)
// 	FROM user_activities
// 	WHERE created_at >= CURRENT_DATE
// 	  AND user_id > 0
// 	`

// 	err = r.db.QueryRow(todayQuery).Scan(&stats.ActivitiesToday)
// 	if err != nil && err != sql.ErrNoRows {
// 		return nil, err
// 	}

// 	// Самый активный пользователь
// 	mostActiveQuery := `
// 	SELECT
// 		u.id, u.telegram_id, u.username, u.first_name,
// 		COUNT(*) as activity_count
// 	FROM user_activities a
// 	JOIN users u ON a.user_id = u.id
// 	WHERE a.created_at >= NOW() - INTERVAL '1 day' * $1
// 	GROUP BY u.id, u.telegram_id, u.username, u.first_name
// 	ORDER BY activity_count DESC
// 	LIMIT 1
// 	`

// 	var mostActiveUser UserActivity
// 	var activityCount int64
// 	err = r.db.QueryRow(mostActiveQuery, days).Scan(
// 		&mostActiveUser.UserID,
// 		&mostActiveUser.TelegramID,
// 		&mostActiveUser.Username,
// 		&mostActiveUser.FirstName,
// 		&activityCount,
// 	)

// 	if err == nil {
// 		stats.MostActiveUser = &mostActiveUser
// 	}

// 	// Распределение по типам
// 	typeQuery := `
// 	SELECT activity_type, COUNT(*)
// 	FROM user_activities
// 	WHERE created_at >= NOW() - INTERVAL '1 day' * $1
// 	  AND user_id > 0
// 	GROUP BY activity_type
// 	`

// 	rows, err := r.db.Query(typeQuery, days)
// 	if err == nil {
// 		defer rows.Close()
// 		for rows.Next() {
// 			var activityType string
// 			var count int64
// 			rows.Scan(&activityType, &count)
// 			stats.ByType[activityType] = count
// 		}
// 	}

// 	// Распределение по категориям
// 	categoryQuery := `
// 	SELECT category, COUNT(*)
// 	FROM user_activities
// 	WHERE created_at >= NOW() - INTERVAL '1 day' * $1
// 	  AND user_id > 0
// 	GROUP BY category
// 	`

// 	rows, err = r.db.Query(categoryQuery, days)
// 	if err == nil {
// 		defer rows.Close()
// 		for rows.Next() {
// 			var category string
// 			var count int64
// 			rows.Scan(&category, &count)
// 			stats.ByCategory[category] = count
// 		}
// 	}

// 	// Распределение по часам
// 	hourQuery := `
// 	SELECT EXTRACT(HOUR FROM created_at) as hour, COUNT(*)
// 	FROM user_activities
// 	WHERE created_at >= NOW() - INTERVAL '1 day' * $1
// 	  AND user_id > 0
// 	GROUP BY EXTRACT(HOUR FROM created_at)
// 	ORDER BY hour
// 	`

// 	rows, err = r.db.Query(hourQuery, days)
// 	if err == nil {
// 		defer rows.Close()
// 		for rows.Next() {
// 			var hour int
// 			var count int64
// 			rows.Scan(&hour, &count)
// 			stats.ByHour[hour] = count
// 		}
// 	}

// 	return stats, nil
// }

// // GetSuspiciousActivities возвращает подозрительную активность
// func (r *ActivityRepository) GetSuspiciousActivities(limit int) ([]*UserActivity, error) {
// 	query := `
// 	SELECT
// 		a.id, a.user_id, a.activity_type, a.category, a.severity,
// 		a.details, a.ip_address, a.user_agent, a.metadata, a.created_at,
// 		u.telegram_id, u.username, u.first_name
// 	FROM user_activities a
// 	JOIN users u ON a.user_id = u.id
// 	WHERE a.severity IN ('warning', 'error', 'critical')
// 	   OR (a.activity_type = 'user_login' AND a.details->>'success' = 'false')
// 	   OR (a.activity_type = 'security_event')
// 	ORDER BY a.created_at DESC
// 	LIMIT $1
// 	`

// 	return r.queryActivities(query, limit)
// }

// // GetFailedLoginAttempts возвращает неудачные попытки входа
// func (r *ActivityRepository) GetFailedLoginAttempts(ip string, minutes int) (int, error) {
// 	query := `
// 	SELECT COUNT(*)
// 	FROM user_activities
// 	WHERE activity_type = 'user_login'
// 	  AND details->>'success' = 'false'
// 	  AND ip_address = $1
// 	  AND created_at >= NOW() - INTERVAL '1 minute' * $2
// 	`

// 	var count int
// 	err := r.db.QueryRow(query, ip, minutes).Scan(&count)
// 	if err != nil {
// 		return 0, err
// 	}

// 	return count, nil
// }

// // CleanupOldActivities очищает старые записи активности
// func (r *ActivityRepository) CleanupOldActivities(ctx context.Context, olderThanDays int) (int64, error) {
// 	query := `
// 	DELETE FROM user_activities
// 	WHERE created_at < NOW() - INTERVAL '1 day' * $1
// 	  AND severity = 'info'
// 	RETURNING id
// 	`

// 	rows, err := r.db.QueryContext(ctx, query, olderThanDays)
// 	if err != nil {
// 		return 0, err
// 	}
// 	defer rows.Close()

// 	var deletedIDs []int64
// 	for rows.Next() {
// 		var id int64
// 		if err := rows.Scan(&id); err != nil {
// 			return 0, err
// 		}
// 		deletedIDs = append(deletedIDs, id)
// 	}

// 	return int64(len(deletedIDs)), nil
// }

// // ArchiveActivities архивирует старые записи в отдельную таблицу
// func (r *ActivityRepository) ArchiveActivities(ctx context.Context, olderThanDays int) (int64, error) {
// 	tx, err := r.db.BeginTx(ctx, nil)
// 	if err != nil {
// 		return 0, err
// 	}
// 	defer tx.Rollback()

// 	// Копируем в архивную таблицу
// 	copyQuery := `
// 	INSERT INTO user_activities_archive
// 	SELECT * FROM user_activities
// 	WHERE created_at < NOW() - INTERVAL '1 day' * $1
// 	`

// 	result, err := tx.ExecContext(ctx, copyQuery, olderThanDays)
// 	if err != nil {
// 		return 0, err
// 	}

// 	rowsAffected, _ := result.RowsAffected()

// 	// Удаляем из основной таблицы
// 	deleteQuery := `
// 	DELETE FROM user_activities
// 	WHERE created_at < NOW() - INTERVAL '1 day' * $1
// 	`

// 	_, err = tx.ExecContext(ctx, deleteQuery, olderThanDays)
// 	if err != nil {
// 		return 0, err
// 	}

// 	if err := tx.Commit(); err != nil {
// 		return 0, err
// 	}

// 	return rowsAffected, nil
// }

// // Вспомогательные методы

// func (r *ActivityRepository) queryActivities(query string, args ...interface{}) ([]*UserActivity, error) {
// 	rows, err := r.db.Query(query, args...)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	var activities []*UserActivity

// 	for rows.Next() {
// 		var activity UserActivity
// 		var detailsJSON, metadataJSON []byte

// 		err := rows.Scan(
// 			&activity.ID,
// 			&activity.UserID,
// 			&activity.ActivityType,
// 			&activity.Category,
// 			&activity.Severity,
// 			&detailsJSON,
// 			&activity.IPAddress,
// 			&activity.UserAgent,
// 			&metadataJSON,
// 			&activity.CreatedAt,
// 			&activity.TelegramID,
// 			&activity.Username,
// 			&activity.FirstName,
// 		)

// 		if err != nil {
// 			return nil, err
// 		}

// 		// Декодируем JSON
// 		if err := json.Unmarshal(detailsJSON, &activity.Details); err != nil {
// 			return nil, fmt.Errorf("failed to unmarshal details: %w", err)
// 		}

// 		if err := json.Unmarshal(metadataJSON, &activity.Metadata); err != nil {
// 			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
// 		}

// 		activities = append(activities, &activity)
// 	}

// 	return activities, nil
// }

// func (r *ActivityRepository) buildFilterQuery(filter ActivityFilter) (string, []interface{}, error) {
// 	baseQuery := `
// 	SELECT
// 		a.id, a.user_id, a.activity_type, a.category, a.severity,
// 		a.details, a.ip_address, a.user_agent, a.metadata, a.created_at,
// 		u.telegram_id, u.username, u.first_name
// 	FROM user_activities a
// 	LEFT JOIN users u ON a.user_id = u.id
// 	`

// 	whereClauses := []string{"1=1"}
// 	args := []interface{}{}
// 	argIndex := 1

// 	if filter.UserID != nil {
// 		whereClauses = append(whereClauses, fmt.Sprintf("a.user_id = $%d", argIndex))
// 		args = append(args, *filter.UserID)
// 		argIndex++
// 	}

// 	if filter.TelegramID != nil {
// 		whereClauses = append(whereClauses, fmt.Sprintf("u.telegram_id = $%d", argIndex))
// 		args = append(args, *filter.TelegramID)
// 		argIndex++
// 	}

// 	if filter.ActivityType != nil {
// 		whereClauses = append(whereClauses, fmt.Sprintf("a.activity_type = $%d", argIndex))
// 		args = append(args, *filter.ActivityType)
// 		argIndex++
// 	}

// 	if filter.Category != nil {
// 		whereClauses = append(whereClauses, fmt.Sprintf("a.category = $%d", argIndex))
// 		args = append(args, *filter.Category)
// 		argIndex++
// 	}

// 	if filter.Severity != nil {
// 		whereClauses = append(whereClauses, fmt.Sprintf("a.severity = $%d", argIndex))
// 		args = append(args, *filter.Severity)
// 		argIndex++
// 	}

// 	if filter.StartDate != nil {
// 		whereClauses = append(whereClauses, fmt.Sprintf("a.created_at >= $%d", argIndex))
// 		args = append(args, *filter.StartDate)
// 		argIndex++
// 	}

// 	if filter.EndDate != nil {
// 		whereClauses = append(whereClauses, fmt.Sprintf("a.created_at <= $%d", argIndex))
// 		args = append(args, *filter.EndDate)
// 		argIndex++
// 	}

// 	if filter.IPAddress != nil {
// 		whereClauses = append(whereClauses, fmt.Sprintf("a.ip_address = $%d", argIndex))
// 		args = append(args, *filter.IPAddress)
// 		argIndex++
// 	}

// 	if filter.SearchQuery != nil {
// 		whereClauses = append(whereClauses,
// 			fmt.Sprintf(`(u.username ILIKE '%%' || $%d || '%%' OR
// 			               u.first_name ILIKE '%%' || $%d || '%%' OR
// 			               a.details::text ILIKE '%%' || $%d || '%%')`,
// 				argIndex, argIndex, argIndex))
// 		args = append(args, *filter.SearchQuery)
// 		argIndex++
// 	}

// 	// Собираем запрос
// 	query := baseQuery + " WHERE " + joinStrings(whereClauses, " AND ")

// 	// Сортировка
// 	orderBy := "a.created_at"
// 	orderDir := "DESC"

// 	if filter.OrderBy != "" {
// 		validOrderFields := map[string]bool{
// 			"created_at": true,
// 			"user_id":    true,
// 			"severity":   true,
// 			"category":   true,
// 		}
// 		if validOrderFields[filter.OrderBy] {
// 			orderBy = "a." + filter.OrderBy
// 		}
// 	}

// 	if filter.OrderDir != "" && (filter.OrderDir == "ASC" || filter.OrderDir == "DESC") {
// 		orderDir = filter.OrderDir
// 	}

// 	query += fmt.Sprintf(" ORDER BY %s %s", orderBy, orderDir)

// 	// Лимит и оффсет
// 	if filter.Limit > 0 {
// 		query += fmt.Sprintf(" LIMIT $%d", argIndex)
// 		args = append(args, filter.Limit)
// 		argIndex++

// 		if filter.Offset > 0 {
// 			query += fmt.Sprintf(" OFFSET $%d", argIndex)
// 			args = append(args, filter.Offset)
// 		}
// 	}

// 	return query, args, nil
// }

// func (r *ActivityRepository) buildCountQuery(filter ActivityFilter) (string, []interface{}) {
// 	baseQuery := "SELECT COUNT(*) FROM user_activities a LEFT JOIN users u ON a.user_id = u.id"

// 	whereClauses := []string{"1=1"}
// 	args := []interface{}{}

// 	// Повторяем те же условия что и в основном запросе
// 	// (упрощенная версия, в реальном коде нужно вынести в общую функцию)

// 	query := baseQuery + " WHERE " + joinStrings(whereClauses, " AND ")
// 	return query, args
// }

// func (r *ActivityRepository) getActivityTypeStats(userID int, days int) (map[string]int64, error) {
// 	query := `
// 	SELECT activity_type, COUNT(*)
// 	FROM user_activities
// 	WHERE user_id = $1
// 	  AND created_at >= NOW() - INTERVAL '1 day' * $2
// 	GROUP BY activity_type
// 	`

// 	rows, err := r.db.Query(query, userID, days)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	stats := make(map[string]int64)
// 	for rows.Next() {
// 		var activityType string
// 		var count int64
// 		if err := rows.Scan(&activityType, &count); err != nil {
// 			return nil, err
// 		}
// 		stats[activityType] = count
// 	}

// 	return stats, nil
// }

// func (r *ActivityRepository) getActivityHourStats(userID int, days int) (map[int]int64, error) {
// 	query := `
// 	SELECT EXTRACT(HOUR FROM created_at) as hour, COUNT(*)
// 	FROM user_activities
// 	WHERE user_id = $1
// 	  AND created_at >= NOW() - INTERVAL '1 day' * $2
// 	GROUP BY EXTRACT(HOUR FROM created_at)
// 	ORDER BY hour
// 	`

// 	rows, err := r.db.Query(query, userID, days)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	stats := make(map[int]int64)
// 	for rows.Next() {
// 		var hour int
// 		var count int64
// 		if err := rows.Scan(&hour, &count); err != nil {
// 			return nil, err
// 		}
// 		stats[hour] = count
// 	}

// 	return stats, nil
// }

// func joinStrings(strs []string, sep string) string {
// 	if len(strs) == 0 {
// 		return ""
// 	}
// 	if len(strs) == 1 {
// 		return strs[0]
// 	}
// 	result := strs[0]
// 	for _, s := range strs[1:] {
// 		result += sep + s
// 	}
// 	return result
// }
