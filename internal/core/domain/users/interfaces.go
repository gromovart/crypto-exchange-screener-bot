// internal/users/interfaces.go
package users

// import (
// 	"context"
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
// 	GetAllActive() ([]*User, error)
// 	SearchUsers(query string, limit, offset int) ([]*User, error)
// 	GetTotalCount(ctx context.Context) (int, error)
// 	IncrementSignalsCount(userID int) error
// 	ResetDailyCounters(ctx context.Context) error
// }

// // SettingsRepository интерфейс для работы с настройками
// type SettingsRepository interface {
// 	GetSettings(userID int) (*UserSettings, error)
// 	UpdateSettings(userID int, settings *UserSettings) error
// 	GetNotificationPreferences(userID int) (*NotificationSettings, error)
// 	UpdateNotificationPreferences(userID int, prefs *NotificationSettings) error
// 	ResetToDefault(userID int) error
// }

// // AnalyticsService интерфейс для аналитики
// type AnalyticsService interface {
// 	TrackUserActivity(userID int, activityType string, details map[string]interface{})
// 	GetUserStats(userID int) (*UserStats, error)
// 	GetSystemStats() (*SystemStats, error)
// }

// // NotificationService интерфейс для уведомлений
// type NotificationService interface {
// 	SendUserNotification(userID int, message string, notificationType string) error
// 	SendTelegramNotification(chatID, message string) error
// 	SendEmailNotification(email, subject, message string) error
// }
