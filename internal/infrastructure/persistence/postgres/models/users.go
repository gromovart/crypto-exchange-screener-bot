// internal/infrastructure/persistence/postgres/models/users.go
package models

import (
	"encoding/json"
	"math"
	"time"
)

// / User - основная модель пользователя
type User struct {
	ID         int    `db:"id" json:"id"`
	TelegramID int64  `db:"telegram_id" json:"telegram_id"`
	Username   string `db:"username" json:"username"`
	FirstName  string `db:"first_name" json:"first_name"`
	LastName   string `db:"last_name" json:"last_name,omitempty"`
	ChatID     string `db:"chat_id" json:"chat_id"`

	// Контактная информация
	Email string `db:"email" json:"email,omitempty"`
	Phone string `db:"phone" json:"phone,omitempty"`

	// Настройки уведомлений (плоские поля для маппинга с БД)
	NotificationsEnabled bool `db:"notifications_enabled" json:"notifications_enabled"`
	NotifyGrowth         bool `db:"notify_growth" json:"notify_growth"`
	NotifyFall           bool `db:"notify_fall" json:"notify_fall"`
	NotifyContinuous     bool `db:"notify_continuous" json:"notify_continuous"`
	QuietHoursStart      int  `db:"quiet_hours_start" json:"quiet_hours_start"`
	QuietHoursEnd        int  `db:"quiet_hours_end" json:"quiet_hours_end"`

	// Настройки анализа (плоские поля для маппинга с БД)
	MinGrowthThreshold float64 `db:"min_growth_threshold" json:"min_growth_threshold"`
	MinFallThreshold   float64 `db:"min_fall_threshold" json:"min_fall_threshold"`
	// ИЗМЕНЕНИЕ: убираем db:"-", теперь sqlx будет маппить эти поля
	PreferredPeriods []int   `db:"preferred_periods" json:"preferred_periods"`
	MinVolumeFilter  float64 `db:"min_volume_filter" json:"min_volume_filter"`
	// ИЗМЕНЕНИЕ: убираем db:"-", теперь sqlx будет маппить эти поля
	ExcludePatterns []string `db:"exclude_patterns" json:"exclude_patterns"`
	Language        string   `db:"language" json:"language"`
	Timezone        string   `db:"timezone" json:"timezone"`
	DisplayMode     string   `db:"display_mode" json:"display_mode"`

	// Статус
	Role             string `db:"role" json:"role"` // user, premium, admin
	IsActive         bool   `db:"is_active" json:"is_active"`
	IsVerified       bool   `db:"is_verified" json:"is_verified"`
	SubscriptionTier string `db:"subscription_tier" json:"subscription_tier"` // free, basic, pro

	// Лимиты
	SignalsToday     int `db:"signals_today" json:"signals_today"`
	MaxSignalsPerDay int `db:"max_signals_per_day" json:"max_signals_per_day"`

	// Временные метки
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
	LastLoginAt  time.Time `db:"last_login_at" json:"last_login_at,omitempty"`
	LastSignalAt time.Time `db:"last_signal_at" json:"last_signal_at,omitempty"`
}

// Константы ролей пользователей
const (
	RoleUser      = "user"
	RolePremium   = "premium"
	RoleModerator = "moderator"
	RoleAdmin     = "admin"
	RoleSystem    = "system"
)

// Константы тарифных планов
const (
	TierFree  = "free"
	TierBasic = "basic"
	TierPro   = "pro"
)

// UserProfile - профиль пользователя для отображения
type UserProfile struct {
	ID               int       `json:"id"`
	TelegramID       int64     `json:"telegram_id"`
	Username         string    `json:"username"`
	FirstName        string    `json:"first_name"`
	LastName         string    `json:"last_name,omitempty"`
	Email            string    `json:"email,omitempty"`
	Role             string    `json:"role"`
	SubscriptionTier string    `json:"subscription_tier"`
	IsActive         bool      `json:"is_active"`
	IsVerified       bool      `json:"is_verified"`
	SignalsToday     int       `json:"signals_today"`
	MaxSignalsPerDay int       `json:"max_signals_per_day"`
	CreatedAt        time.Time `json:"created_at"`
	LastLoginAt      time.Time `json:"last_login_at,omitempty"`
}

// LoginRequest - запрос на вход пользователя
type LoginRequest struct {
	TelegramID int64  `json:"telegram_id" validate:"required"`
	ChatID     string `json:"chat_id" validate:"required"`
}

// RegisterRequest - запрос на регистрацию пользователя
type RegisterRequest struct {
	TelegramID int64  `json:"telegram_id" validate:"required"`
	Username   string `json:"username"`
	FirstName  string `json:"first_name" validate:"required"`
	LastName   string `json:"last_name"`
	ChatID     string `json:"chat_id" validate:"required"`
	Email      string `json:"email" validate:"email"`
	Phone      string `json:"phone"`
}

// UpdateProfileRequest - запрос на обновление профиля
type UpdateProfileRequest struct {
	Username    string `json:"username"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Email       string `json:"email" validate:"email"`
	Phone       string `json:"phone"`
	Language    string `json:"language"`
	Timezone    string `json:"timezone"`
	DisplayMode string `json:"display_mode"`
}

// ==================== МЕТОД ДЛЯ ИНТЕРФЕЙСА PAYMENT.USER ====================

// GetID возвращает ID пользователя (для интерфейса payment.User)
func (u *User) GetID() int {
	return u.ID
}

// ==================== МЕТОДЫ ПОЛЬЗОВАТЕЛЯ ====================

// CanReceiveNotifications проверяет, может ли пользователь получать уведомления
func (u *User) CanReceiveNotifications() bool {
	return u.IsActive && u.NotificationsEnabled
}

// CanReceiveGrowthSignals проверяет, может ли пользователь получать уведомления о росте
func (u *User) CanReceiveGrowthSignals() bool {
	return u.CanReceiveNotifications() && u.NotifyGrowth
}

// CanReceiveFallSignals проверяет, может ли пользователь получать уведомления о падении
func (u *User) CanReceiveFallSignals() bool {
	return u.CanReceiveNotifications() && u.NotifyFall
}

// IsInQuietHours проверяет, находится ли текущее время в тихих часах пользователя
func (u *User) IsInQuietHours() bool {
	now := time.Now().UTC()
	hour := now.Hour()

	// Если тихие часы не настроены
	if u.QuietHoursStart == 0 && u.QuietHoursEnd == 0 {
		return false
	}

	// Обработка случая когда start > end (например, 23-8)
	if u.QuietHoursStart > u.QuietHoursEnd {
		return hour >= u.QuietHoursStart || hour < u.QuietHoursEnd
	}

	return hour >= u.QuietHoursStart && hour < u.QuietHoursEnd
}

// HasReachedDailyLimit проверяет, достиг ли пользователь дневного лимита сигналов
func (u *User) HasReachedDailyLimit() bool {
	return u.SignalsToday >= u.MaxSignalsPerDay
}

// ShouldReceiveSignal проверяет, должен ли пользователь получить уведомление о сигнале
func (u *User) ShouldReceiveSignal(signalType string, changePercent float64) bool {
	// Базовые проверки
	if !u.IsActive || !u.CanReceiveNotifications() {
		return false
	}

	// Проверка типа сигнала
	if signalType == "growth" && !u.CanReceiveGrowthSignals() {
		return false
	}
	if signalType == "fall" && !u.CanReceiveFallSignals() {
		return false
	}

	// Проверка порогов
	if signalType == "growth" && changePercent < u.MinGrowthThreshold {
		return false
	}
	// Для падения changePercent отрицательный, сравниваем модуль с порогом
	if signalType == "fall" && math.Abs(changePercent) < u.MinFallThreshold {
		return false
	}

	// Проверка тихих часов
	if u.IsInQuietHours() {
		return false
	}

	// Проверка лимитов
	if u.HasReachedDailyLimit() {
		return false
	}

	return true
}

// IsAdmin проверяет, является ли пользователь администратором
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// IsPremium проверяет, имеет ли пользователь премиум-статус
func (u *User) IsPremium() bool {
	return u.Role == RolePremium || u.SubscriptionTier == TierPro || u.SubscriptionTier == TierBasic
}

// ToProfile преобразует пользователя в профиль для отображения
func (u *User) ToProfile() *UserProfile {
	return &UserProfile{
		ID:               u.ID,
		TelegramID:       u.TelegramID,
		Username:         u.Username,
		FirstName:        u.FirstName,
		LastName:         u.LastName,
		Email:            u.Email,
		Role:             u.Role,
		SubscriptionTier: u.SubscriptionTier,
		IsActive:         u.IsActive,
		IsVerified:       u.IsVerified,
		SignalsToday:     u.SignalsToday,
		MaxSignalsPerDay: u.MaxSignalsPerDay,
		CreatedAt:        u.CreatedAt,
		LastLoginAt:      u.LastLoginAt,
	}
}

// ==================== JSON СЕРИАЛИЗАЦИЯ ====================

// MarshalJSON кастомная сериализация для User
func (u *User) MarshalJSON() ([]byte, error) {
	type Alias User
	return json.Marshal(&struct {
		*Alias
		CreatedAt    string `json:"created_at"`
		UpdatedAt    string `json:"updated_at"`
		LastLoginAt  string `json:"last_login_at,omitempty"`
		LastSignalAt string `json:"last_signal_at,omitempty"`
	}{
		Alias:        (*Alias)(u),
		CreatedAt:    u.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    u.UpdatedAt.Format(time.RFC3339),
		LastLoginAt:  formatTime(u.LastLoginAt),
		LastSignalAt: formatTime(u.LastSignalAt),
	})
}

// NewUser создает нового пользователя с настройками по умолчанию
func NewUser(telegramID int64, username, firstName, lastName, chatID string) *User {
	now := time.Now()
	return &User{
		TelegramID:           telegramID,
		Username:             username,
		FirstName:            firstName,
		LastName:             lastName,
		ChatID:               chatID,
		NotificationsEnabled: true,
		NotifyGrowth:         true,
		NotifyFall:           true,
		NotifyContinuous:     true,
		QuietHoursStart:      23,
		QuietHoursEnd:        8,
		MinGrowthThreshold:   2.0,
		MinFallThreshold:     2.0,
		MinVolumeFilter:      100000,
		Language:             "ru",
		Timezone:             "Europe/Moscow",
		DisplayMode:          "compact",
		PreferredPeriods:     []int{5, 15, 30},
		ExcludePatterns:      []string{},
		Role:                 RoleUser,
		IsActive:             true,
		IsVerified:           false,
		SubscriptionTier:     TierFree,
		SignalsToday:         0,
		MaxSignalsPerDay:     50,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
}
