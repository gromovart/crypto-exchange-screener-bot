// internal/infrastructure/persistence/postgres/models/activity.go
package models

import (
	"encoding/json"
	"time"
)

// ActivityType типы активности
type ActivityType string

const (
	ActivityTypeLogin          ActivityType = "user_login"
	ActivityTypeLogout         ActivityType = "user_logout"
	ActivityTypeProfileUpdate  ActivityType = "profile_update"
	ActivityTypeSettingsUpdate ActivityType = "settings_update"
	ActivityTypeSignalReceived ActivityType = "signal_received"
	ActivityTypeSignalFiltered ActivityType = "signal_filtered"
	ActivityTypeNotification   ActivityType = "notification_sent"
	ActivityTypeApiCall        ActivityType = "api_call"
	ActivityTypeError          ActivityType = "error_occurred"
	ActivityTypeSystem         ActivityType = "system_event"
	ActivityTypeSubscription   ActivityType = "subscription_event"
	ActivityTypeSecurity       ActivityType = "security_event"
)

// ActivitySeverity уровень серьезности
type ActivitySeverity string

const (
	SeverityInfo     ActivitySeverity = "info"
	SeverityWarning  ActivitySeverity = "warning"
	SeverityError    ActivitySeverity = "error"
	SeverityCritical ActivitySeverity = "critical"
)

// ActivityCategory категория активности
type ActivityCategory string

const (
	CategoryAuth      ActivityCategory = "authentication"
	CategoryUser      ActivityCategory = "user_actions"
	CategoryTrading   ActivityCategory = "trading"
	CategorySystem    ActivityCategory = "system"
	CategorySecurity  ActivityCategory = "security"
	CategoryBilling   ActivityCategory = "billing"
	CategoryAnalytics ActivityCategory = "analytics"
)

// JSONMap для работы с JSON полями
type JSONMap map[string]interface{}

// UserActivity запись активности пользователя
type UserActivity struct {
	ID           int64            `db:"id" json:"id"`
	UserID       int              `db:"user_id" json:"user_id"`
	ActivityType ActivityType     `db:"activity_type" json:"activity_type"`
	Category     ActivityCategory `db:"category" json:"category"` // Добавлено в БД
	EntityType   *string          `db:"entity_type" json:"entity_type,omitempty"`
	EntityID     *int             `db:"entity_id" json:"entity_id,omitempty"`
	Details      JSONMap          `db:"details" json:"details"`
	IPAddress    *string          `db:"ip_address" json:"ip_address,omitempty"`
	UserAgent    *string          `db:"user_agent" json:"user_agent,omitempty"`
	Location     *string          `db:"location" json:"location,omitempty"`
	Severity     ActivitySeverity `db:"severity" json:"severity"`
	Metadata     JSONMap          `db:"metadata" json:"metadata,omitempty"`
	CreatedAt    time.Time        `db:"created_at" json:"created_at"`

	// Дополнительные поля (не из БД)
	TelegramID int64  `db:"-" json:"telegram_id,omitempty"`
	Username   string `db:"-" json:"username,omitempty"`
	FirstName  string `db:"-" json:"first_name,omitempty"`
}

// ActivityStats статистика активности
type ActivityStats struct {
	TotalActivities      int64            `json:"total_activities"`
	ActivitiesToday      int64            `json:"activities_today"`
	UniqueUsers          int64            `json:"unique_users"`
	MostActiveUser       *UserActivity    `json:"most_active_user,omitempty"`
	ByType               map[string]int64 `json:"by_type"`
	ByCategory           map[string]int64 `json:"by_category"`
	ByHour               map[int]int64    `json:"by_hour"`
	ErrorRate            float64          `json:"error_rate"`
	AvgActivitiesPerUser float64          `json:"avg_activities_per_user"`
}

// ActivityFilter фильтр для поиска активности
type ActivityFilter struct {
	UserID       *int              `json:"user_id,omitempty"`
	TelegramID   *int64            `json:"telegram_id,omitempty"`
	ActivityType *ActivityType     `json:"activity_type,omitempty"`
	Category     *ActivityCategory `json:"category,omitempty"`
	Severity     *ActivitySeverity `json:"severity,omitempty"`
	StartDate    *time.Time        `json:"start_date,omitempty"`
	EndDate      *time.Time        `json:"end_date,omitempty"`
	IPAddress    *string           `json:"ip_address,omitempty"`
	SearchQuery  *string           `json:"search_query,omitempty"`
	Limit        int               `json:"limit,omitempty"`
	Offset       int               `json:"offset,omitempty"`
	OrderBy      string            `json:"order_by,omitempty"`
	OrderDir     string            `json:"order_dir,omitempty"`
}

// MarshalJSON кастомная сериализация для UserActivity
func (a *UserActivity) MarshalJSON() ([]byte, error) {
	type Alias UserActivity
	return json.Marshal(&struct {
		*Alias
		CreatedAt string `json:"created_at"`
	}{
		Alias:     (*Alias)(a),
		CreatedAt: a.CreatedAt.Format(time.RFC3339),
	})
}

// DefaultActivityFilter возвращает фильтр по умолчанию
func DefaultActivityFilter() ActivityFilter {
	return ActivityFilter{
		Limit:    50,
		OrderBy:  "created_at",
		OrderDir: "DESC",
	}
}

// NewUserActivity создает новую активность пользователя
func NewUserActivity(userID int, activityType ActivityType, category ActivityCategory, severity ActivitySeverity, details JSONMap) *UserActivity {
	return &UserActivity{
		UserID:       userID,
		ActivityType: activityType,
		Category:     category, // Теперь поле есть в структуре
		Severity:     severity,
		Details:      details,
		Metadata:     make(JSONMap),
		CreatedAt:    time.Now(),
	}
}
