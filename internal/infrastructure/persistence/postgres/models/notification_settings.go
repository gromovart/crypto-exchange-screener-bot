// internal/infrastructure/persistence/postgres/models/notification_settings.go
package models

// UserNotificationSettings содержит настройки уведомлений пользователя
type UserNotificationSettings struct {
	NotificationsEnabled bool    `json:"notifications_enabled"`
	NotifyGrowth         bool    `json:"notify_growth"`
	NotifyFall           bool    `json:"notify_fall"`
	NotifyContinuous     bool    `json:"notify_continuous"`
	MinGrowthThreshold   float64 `json:"min_growth_threshold"`
	MinFallThreshold     float64 `json:"min_fall_threshold"`
	QuietHoursStart      int     `json:"quiet_hours_start"`
	QuietHoursEnd        int     `json:"quiet_hours_end"`
	PreferredPeriods     []int   `json:"preferred_periods"` // В минутах: 5, 15, 30, 60, 240, 1440
}
