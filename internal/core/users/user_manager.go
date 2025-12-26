// internal/users/user_manager.go
package users

// import (
// 	"crypto-exchange-screener-bot/persistence/postgres/repository/users"
// 	"fmt"
// 	"log"
// 	"sync"
// 	"time"

// 	"github.com/go-redis/redis/v8"
// )

// // UserManager управляет пользователями в памяти
// type UserManager struct {
// 	users          map[int]*users.User
// 	telegramUsers  map[int64]*users.User
// 	chatUsers      map[string]*users.User
// 	mutex          sync.RWMutex
// 	cache          *redis.Client
// 	userService    *UserService
// 	settingsLoaded bool
// }

// // NewUserManager создает новый менеджер пользователей
// func NewUserManager(cache *redis.Client, userService *UserService) *UserManager {
// 	return &UserManager{
// 		users:         make(map[int]*users.User),
// 		telegramUsers: make(map[int64]*users.User),
// 		chatUsers:     make(map[string]*users.User),
// 		cache:         cache,
// 		userService:   userService,
// 	}
// }

// // LoadUser загружает пользователя в память
// func (um *UserManager) LoadUser(userID int) (*users.User, error) {
// 	um.mutex.RLock()
// 	if user, ok := um.users[userID]; ok {
// 		um.mutex.RUnlock()
// 		return user, nil
// 	}
// 	um.mutex.RUnlock()

// 	// Загружаем пользователя из сервиса
// 	user, err := um.userService.GetUserByID(userID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	um.mutex.Lock()
// 	um.users[user.ID] = user
// 	um.telegramUsers[user.TelegramID] = user
// 	um.chatUsers[user.ChatID] = user
// 	um.mutex.Unlock()

// 	return user, nil
// }

// // LoadUserByTelegramID загружает пользователя по Telegram ID
// func (um *UserManager) LoadUserByTelegramID(telegramID int64) (*users.User, error) {
// 	um.mutex.RLock()
// 	if user, ok := um.telegramUsers[telegramID]; ok {
// 		um.mutex.RUnlock()
// 		return user, nil
// 	}
// 	um.mutex.RUnlock()

// 	// Загружаем пользователя из сервиса
// 	user, err := um.userService.GetUserByTelegramID(telegramID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	if user != nil {
// 		um.mutex.Lock()
// 		um.users[user.ID] = user
// 		um.telegramUsers[user.TelegramID] = user
// 		um.chatUsers[user.ChatID] = user
// 		um.mutex.Unlock()
// 	}

// 	return user, nil
// }

// // LoadActiveUsers загружает всех активных пользователей
// func (um *UserManager) LoadActiveUsers() ([]*users.User, error) {
// 	users, err := um.userService.GetActiveUsers()
// 	if err != nil {
// 		return nil, err
// 	}

// 	um.mutex.Lock()
// 	for _, user := range users {
// 		um.users[user.ID] = user
// 		um.telegramUsers[user.TelegramID] = user
// 		um.chatUsers[user.ChatID] = user
// 	}
// 	um.mutex.Unlock()

// 	return users, nil
// }

// // GetUser получает пользователя из памяти
// func (um *UserManager) GetUser(userID int) (*users.User, bool) {
// 	um.mutex.RLock()
// 	user, ok := um.users[userID]
// 	um.mutex.RUnlock()
// 	return user, ok
// }

// // GetUserByTelegramID получает пользователя по Telegram ID
// func (um *UserManager) GetUserByTelegramID(telegramID int64) (*users.User, bool) {
// 	um.mutex.RLock()
// 	user, ok := um.telegramUsers[telegramID]
// 	um.mutex.RUnlock()
// 	return user, ok
// }

// // GetUserByChatID получает пользователя по Chat ID
// func (um *UserManager) GetUserByChatID(chatID string) (*users.User, bool) {
// 	um.mutex.RLock()
// 	user, ok := um.chatUsers[chatID]
// 	um.mutex.RUnlock()
// 	return user, ok
// }

// // UpdateUser обновляет пользователя в памяти
// func (um *UserManager) UpdateUser(user *users.User) {
// 	um.mutex.Lock()
// 	um.users[user.ID] = user
// 	um.telegramUsers[user.TelegramID] = user
// 	um.chatUsers[user.ChatID] = user
// 	um.mutex.Unlock()
// }

// // RemoveUser удаляет пользователя из памяти
// func (um *UserManager) RemoveUser(userID int) {
// 	um.mutex.Lock()
// 	if user, ok := um.users[userID]; ok {
// 		delete(um.telegramUsers, user.TelegramID)
// 		delete(um.chatUsers, user.ChatID)
// 		delete(um.users, userID)
// 	}
// 	um.mutex.Unlock()
// }

// // RefreshUser обновляет пользователя из базы данных
// func (um *UserManager) RefreshUser(userID int) error {
// 	user, err := um.userService.GetUserByID(userID)
// 	if err != nil {
// 		return err
// 	}

// 	um.UpdateUser(user)
// 	return nil
// }

// // GetUserLanguage получает язык пользователя (исправлено)
// func (um *UserManager) GetUserLanguage(userID int) string {
// 	um.mutex.RLock()
// 	user, ok := um.users[userID]
// 	um.mutex.RUnlock()

// 	if !ok || user.Settings.Language == "" {
// 		return "ru" // по умолчанию
// 	}

// 	return user.Settings.Language
// }

// // GetUserTimezone получает часовой пояс пользователя (исправлено)
// func (um *UserManager) GetUserTimezone(userID int) string {
// 	um.mutex.RLock()
// 	user, ok := um.users[userID]
// 	um.mutex.RUnlock()

// 	if !ok || user.Settings.Timezone == "" {
// 		return "Europe/Moscow" // по умолчанию
// 	}

// 	return user.Settings.Timezone
// }

// // ShouldReceiveSignal проверяет, должен ли пользователь получить сигнал
// func (um *UserManager) ShouldReceiveSignal(userID int, signalType string, changePercent float64) bool {
// 	user, ok := um.GetUser(userID)
// 	if !ok {
// 		return false
// 	}

// 	return user.ShouldReceiveSignal(signalType, changePercent)
// }

// // IncrementUserSignals увеличивает счетчик сигналов пользователя
// func (um *UserManager) IncrementUserSignals(userID int) error {
// 	// Увеличиваем в сервисе
// 	if err := um.userService.IncrementSignalsCount(userID); err != nil {
// 		return err
// 	}

// 	// Обновляем в памяти
// 	um.mutex.RLock()
// 	user, ok := um.users[userID]
// 	um.mutex.RUnlock()

// 	if ok {
// 		um.mutex.Lock()
// 		user.SignalsToday++
// 		user.LastSignalAt = time.Now()
// 		um.mutex.Unlock()
// 	}

// 	return nil
// }

// // ResetAllCounters сбрасывает счетчики всех пользователей
// func (um *UserManager) ResetAllCounters() error {
// 	// Сбрасываем в сервисе
// 	if err := um.userService.ResetDailyCounters(); err != nil {
// 		return err
// 	}

// 	// Сбрасываем в памяти
// 	um.mutex.Lock()
// 	for _, user := range um.users {
// 		user.SignalsToday = 0
// 	}
// 	um.mutex.Unlock()

// 	return nil
// }

// // GetUserStats получает статистику пользователя
// func (um *UserManager) GetUserStats(userID int) (map[string]interface{}, error) {
// 	stats, err := um.userService.GetUserStats(userID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Преобразуем в map
// 	result := map[string]interface{}{
// 		"user_id":             stats.UserID,
// 		"total_signals":       stats.TotalSignals,
// 		"signals_today":       stats.SignalsToday,
// 		"avg_signals_per_day": stats.AvgSignalsPerDay,
// 		"last_signal_at":      stats.LastSignalAt,
// 		"favorite_symbol":     stats.FavoriteSymbol,
// 		"success_rate":        stats.SuccessRate,
// 		"active_days":         stats.ActiveDays,
// 		"first_activity":      stats.FirstActivity,
// 		"last_activity":       stats.LastActivity,
// 	}

// 	return result, nil
// }

// // GetSystemStats получает системную статистику
// func (um *UserManager) GetSystemStats() (map[string]interface{}, error) {
// 	stats, err := um.userService.GetSystemStats()
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Преобразуем в map
// 	result := map[string]interface{}{
// 		"total_users":           stats.TotalUsers,
// 		"active_users":          stats.ActiveUsers,
// 		"new_users_today":       stats.NewUsersToday,
// 		"total_signals_sent":    stats.TotalSignalsSent,
// 		"avg_signals_per_user":  stats.AvgSignalsPerUser,
// 		"most_active_hour":      stats.MostActiveHour,
// 		"peak_concurrent_users": stats.PeakConcurrentUsers,
// 	}

// 	return result, nil
// }

// // Cleanup очищает устаревших пользователей из памяти
// func (um *UserManager) Cleanup() {
// 	um.mutex.Lock()
// 	defer um.mutex.Unlock()

// 	now := time.Now()
// 	expirationTime := now.Add(-24 * time.Hour) // Удаляем тех, кто не обновлялся 24 часа

// 	for id, user := range um.users {
// 		if user.UpdatedAt.Before(expirationTime) {
// 			delete(um.telegramUsers, user.TelegramID)
// 			delete(um.chatUsers, user.ChatID)
// 			delete(um.users, id)
// 		}
// 	}
// }

// // StartCleanupScheduler запускает планировщик очистки
// func (um *UserManager) StartCleanupScheduler() {
// 	ticker := time.NewTicker(1 * time.Hour)
// 	defer ticker.Stop()

// 	for range ticker.C {
// 		um.Cleanup()
// 		log.Println("UserManager: cleanup completed")
// 	}
// }

// // GetUsersCount получает количество пользователей в памяти
// func (um *UserManager) GetUsersCount() int {
// 	um.mutex.RLock()
// 	defer um.mutex.RUnlock()
// 	return len(um.users)
// }

// // GetAllUsers получает всех пользователей из памяти
// func (um *UserManager) GetAllUsers() []*users.User {
// 	um.mutex.RLock()
// 	defer um.mutex.RUnlock()

// 	users := make([]*users.User, 0, len(um.users))
// 	for _, user := range um.users {
// 		users = append(users, user)
// 	}

// 	return users
// }

// // GetUsersByRole получает пользователей по роли
// func (um *UserManager) GetUsersByRole(role string) []*users.User {
// 	um.mutex.RLock()
// 	defer um.mutex.RUnlock()

// 	var users []*users.User
// 	for _, user := range um.users {
// 		if user.Role == role {
// 			users = append(users, user)
// 		}
// 	}

// 	return users
// }

// // IsUserLoaded проверяет, загружен ли пользователь в память
// func (um *UserManager) IsUserLoaded(userID int) bool {
// 	um.mutex.RLock()
// 	_, ok := um.users[userID]
// 	um.mutex.RUnlock()
// 	return ok
// }

// // WarmUp загружает активных пользователей в память
// func (um *UserManager) WarmUp() error {
// 	log.Println("UserManager: warming up...")
// 	_, err := um.LoadActiveUsers()
// 	if err != nil {
// 		return fmt.Errorf("failed to warm up: %w", err)
// 	}
// 	log.Printf("UserManager: warmed up with %d users", um.GetUsersCount())
// 	return nil
// }
