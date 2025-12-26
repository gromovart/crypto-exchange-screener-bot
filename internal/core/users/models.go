// internal/users/models.go
package users

// import (
// 	"encoding/json"
// 	"math"
// 	"time"
// )

// type User struct {
// 	ID         int    `json:"id"`
// 	TelegramID int64  `json:"telegram_id"`
// 	Username   string `json:"username"`
// 	FirstName  string `json:"first_name"`
// 	LastName   string `json:"last_name,omitempty"`
// 	ChatID     string `json:"chat_id"`

// 	// Контактная информация
// 	Email string `json:"email,omitempty"`
// 	Phone string `json:"phone,omitempty"`

// 	// Настройки уведомлений
// 	Notifications NotificationSettings `json:"notifications"`

// 	// Настройки анализа
// 	Settings UserSettings `json:"settings"`

// 	// Статус
// 	Role             string `json:"role"` // user, premium, admin
// 	IsActive         bool   `json:"is_active"`
// 	IsVerified       bool   `json:"is_verified"`
// 	SubscriptionTier string `json:"subscription_tier"` // free, basic, pro

// 	// Лимиты
// 	SignalsToday     int `json:"signals_today"`
// 	MaxSignalsPerDay int `json:"max_signals_per_day"`

// 	// Временные метки
// 	CreatedAt    time.Time `json:"created_at"`
// 	UpdatedAt    time.Time `json:"updated_at"`
// 	LastLoginAt  time.Time `json:"last_login_at,omitempty"`
// 	LastSignalAt time.Time `json:"last_signal_at,omitempty"`
// }

// type UserSettings struct {
// 	// Пороги анализа
// 	MinGrowthThreshold float64 `json:"min_growth_threshold"`
// 	MinFallThreshold   float64 `json:"min_fall_threshold"`

// 	// Периоды анализа (в минутах)
// 	PreferredPeriods []int `json:"preferred_periods"`

// 	// Фильтры
// 	MinVolumeFilter float64  `json:"min_volume_filter"`
// 	ExcludePatterns []string `json:"exclude_patterns"`

// 	// Настройки отображения
// 	Language    string `json:"language"`
// 	Timezone    string `json:"timezone"`
// 	DisplayMode string `json:"display_mode"` // compact, detailed
// }

// type NotificationSettings struct {
// 	Enabled    bool `json:"enabled"`
// 	Growth     bool `json:"growth"`
// 	Fall       bool `json:"fall"`
// 	Continuous bool `json:"continuous"`

// 	// Тихие часы (0-23)
// 	QuietHoursStart int `json:"quiet_hours_start"`
// 	QuietHoursEnd   int `json:"quiet_hours_end"`

// 	// Методы уведомлений
// 	Methods []string `json:"methods"` // telegram, email
// }

// type Session struct {
// 	ID            string                 `json:"id"`
// 	UserID        int                    `json:"user_id"`
// 	Token         string                 `json:"token"`
// 	DeviceInfo    map[string]interface{} `json:"device_info,omitempty"`
// 	IP            string                 `json:"ip,omitempty"`
// 	UserAgent     string                 `json:"user_agent,omitempty"`
// 	Data          map[string]interface{} `json:"data,omitempty"`
// 	ExpiresAt     time.Time              `json:"expires_at"`
// 	CreatedAt     time.Time              `json:"created_at"`
// 	UpdatedAt     time.Time              `json:"updated_at"`
// 	LastActivity  time.Time              `json:"last_activity,omitempty"`
// 	IsActive      bool                   `json:"is_active"`
// 	RevokedAt     *time.Time             `json:"revoked_at,omitempty"`
// 	RevokedReason string                 `json:"revoked_reason,omitempty"`
// 	User          *User                  `json:"user,omitempty"`
// }

// // Константы ролей
// const (
// 	RoleUser    = "user"
// 	RolePremium = "premium"
// 	RoleAdmin   = "admin"
// )

// // Константы тарифов
// const (
// 	TierFree  = "free"
// 	TierBasic = "basic"
// 	TierPro   = "pro"
// )

// // Методы для проверки разрешений
// func (u *User) CanReceiveNotifications() bool {
// 	return u.IsActive && u.Notifications.Enabled
// }

// func (u *User) CanReceiveGrowthSignals() bool {
// 	return u.CanReceiveNotifications() && u.Notifications.Growth
// }

// func (u *User) CanReceiveFallSignals() bool {
// 	return u.CanReceiveNotifications() && u.Notifications.Fall
// }

// func (u *User) IsInQuietHours() bool {
// 	now := time.Now().UTC()
// 	hour := now.Hour()

// 	// Если тихие часы не настроены
// 	if u.Notifications.QuietHoursStart == 0 && u.Notifications.QuietHoursEnd == 0 {
// 		return false
// 	}

// 	// Обработка случая когда start > end (например, 23-8)
// 	if u.Notifications.QuietHoursStart > u.Notifications.QuietHoursEnd {
// 		return hour >= u.Notifications.QuietHoursStart || hour < u.Notifications.QuietHoursEnd
// 	}

// 	return hour >= u.Notifications.QuietHoursStart && hour < u.Notifications.QuietHoursEnd
// }

// func (u *User) HasReachedDailyLimit() bool {
// 	return u.SignalsToday >= u.MaxSignalsPerDay
// }

// func (u *User) ShouldReceiveSignal(signalType string, changePercent float64) bool {
// 	// Базовые проверки
// 	if !u.IsActive || !u.CanReceiveNotifications() {
// 		return false
// 	}

// 	// Проверка типа сигнала
// 	if signalType == "growth" && !u.CanReceiveGrowthSignals() {
// 		return false
// 	}
// 	if signalType == "fall" && !u.CanReceiveFallSignals() {
// 		return false
// 	}

// 	// Проверка порогов
// 	if signalType == "growth" && changePercent < u.Settings.MinGrowthThreshold {
// 		return false
// 	}
// 	if signalType == "fall" && math.Abs(changePercent) < u.Settings.MinFallThreshold {
// 		return false
// 	}

// 	// Проверка тихих часов
// 	if u.IsInQuietHours() {
// 		return false
// 	}

// 	// Проверка лимитов
// 	if u.HasReachedDailyLimit() {
// 		return false
// 	}

// 	return true
// }

// // JSON методы для сериализации
// func (u *User) MarshalJSON() ([]byte, error) {
// 	type Alias User
// 	return json.Marshal(&struct {
// 		*Alias
// 		CreatedAt   string `json:"created_at"`
// 		UpdatedAt   string `json:"updated_at"`
// 		LastLoginAt string `json:"last_login_at,omitempty"`
// 	}{
// 		Alias:       (*Alias)(u),
// 		CreatedAt:   u.CreatedAt.Format(time.RFC3339),
// 		UpdatedAt:   u.UpdatedAt.Format(time.RFC3339),
// 		LastLoginAt: u.LastLoginAt.Format(time.RFC3339),// internal/users/models.go
// package users

// import (
//     "encoding/json"
//     "time"
// )

// // Структуры пользователей
// type User struct {
//     ID         int    `json:"id"`
//     TelegramID int64  `json:"telegram_id"`
//     Username   string `json:"username"`
//     FirstName  string `json:"first_name"`
//     LastName   string `json:"last_name,omitempty"`
//     ChatID     string `json:"chat_id"`

//     Email string `json:"email,omitempty"`
//     Phone string `json:"phone,omitempty"`

//     Notifications NotificationSettings `json:"notifications"`
//     Settings      UserSettings         `json:"settings"`

//     Role             string `json:"role"`
//     IsActive         bool   `json:"is_active"`
//     IsVerified       bool   `json:"is_verified"`
//     SubscriptionTier string `json:"subscription_tier"`

//     SignalsToday     int `json:"signals_today"`
//     MaxSignalsPerDay int `json:"max_signals_per_day"`

//     CreatedAt    time.Time `json:"created_at"`
//     UpdatedAt    time.Time `json:"updated_at"`
//     LastLoginAt  time.Time `json:"last_login_at,omitempty"`
//     LastSignalAt time.Time `json:"last_signal_at,omitempty"`
// }

// type UserSettings struct {
//     MinGrowthThreshold float64  `json:"min_growth_threshold"`
//     MinFallThreshold   float64  `json:"min_fall_threshold"`
//     PreferredPeriods   []int    `json:"preferred_periods"`
//     MinVolumeFilter    float64  `json:"min_volume_filter"`
//     ExcludePatterns    []string `json:"exclude_patterns"`
//     Language           string   `json:"language"`
//     Timezone           string   `json:"timezone"`
//     DisplayMode        string   `json:"display_mode"`
// }

// type NotificationSettings struct {
//     Enabled         bool     `json:"enabled"`
//     Growth          bool     `json:"growth"`
//     Fall            bool     `json:"fall"`
//     Continuous      bool     `json:"continuous"`
//     QuietHoursStart int      `json:"quiet_hours_start"`
//     QuietHoursEnd   int      `json:"quiet_hours_end"`
//     Methods         []string `json:"methods"`
// }

// type Session struct {
//     ID            string                 `json:"id"`
//     UserID        int                    `json:"user_id"`
//     Token         string                 `json:"token"`
//     DeviceInfo    map[string]interface{} `json:"device_info,omitempty"`
//     IP            string                 `json:"ip,omitempty"`
//     UserAgent     string                 `json:"user_agent,omitempty"`
//     Data          map[string]interface{} `json:"data,omitempty"`
//     ExpiresAt     time.Time              `json:"expires_at"`
//     CreatedAt     time.Time              `json:"created_at"`
//     UpdatedAt     time.Time              `json:"updated_at"`
//     LastActivity  time.Time              `json:"last_activity,omitempty"`
//     IsActive      bool                   `json:"is_active"`
//     RevokedAt     *time.Time             `json:"revoked_at,omitempty"`
//     RevokedReason string                 `json:"revoked_reason,omitempty"`
//     User          *User                  `json:"user,omitempty"`
// }

// // Константы
// const (
//     RoleUser    = "user"
//     RolePremium = "premium"
//     RoleAdmin   = "admin"

//     TierFree  = "free"
//     TierBasic = "basic"
//     TierPro   = "pro"
// )
// 	})
// }
