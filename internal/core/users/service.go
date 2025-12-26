// internal/users/service.go
package users

// import (
// 	"context"
// 	"database/sql"
// 	"encoding/json"
// 	"fmt"
// 	"log"
// 	"math"
// 	"strings"
// 	"time"

// 	"github.com/go-redis/redis/v8"
// )

// // Определяем интерфейсы репозиториев
// type UserRepository interface {
// 	FindByID(id int) (*usersUser, error)
// 	FindByTelegramID(telegramID int64) (*users.User, error)
// 	FindByEmail(email string) (*users.User, error)
// 	FindByChatID(chatID string) (*users.User, error)
// 	Create(user *users.User) error
// 	Update(user *users.User) error
// 	Delete(id int) error
// 	UpdateLastLogin(userID int) error
// 	GetAllActive() ([]*users.User, error)
// 	SearchUsers(query string, limit, offset int) ([]*users.User, error)
// 	GetTotalCount(ctx context.Context) (int, error)
// 	IncrementSignalsCount(userID int) error
// 	ResetDailyCounters(ctx context.Context) error
// }

// // UserService предоставляет бизнес-логику для работы с пользователями
// type UserService struct {
// 	userRepo     UserRepository
// 	settingsRepo SettingsRepository
// 	cache        *redis.Client
// 	analytics    AnalyticsService
// 	notifier     NotificationService
// }

// // NewUserService создает новый сервис пользователей
// func NewUserService(
// 	userRepo UserRepository,
// 	settingsRepo SettingsRepository,
// 	cache *redis.Client,
// 	analytics AnalyticsService,
// 	notifier NotificationService,
// ) *UserService {

// 	return &UserService{
// 		userRepo:     userRepo,
// 		settingsRepo: settingsRepo,
// 		cache:        cache,
// 		analytics:    analytics,
// 		notifier:     notifier,
// 	}
// }

// // SettingsRepository интерфейс для работы с настройками
// type SettingsRepository interface {
// 	GetSettings(userID int) (*users.UserSettings, error)
// 	UpdateSettings(userID int, settings *users.UserSettings) error
// 	GetNotificationPreferences(userID int) (*users.NotificationSettings, error)
// 	UpdateNotificationPreferences(userID int, prefs *users.NotificationSettings) error
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

// // UserStats статистика пользователя
// type UserStats struct {
// 	UserID           int       `json:"user_id"`
// 	TotalSignals     int       `json:"total_signals"`
// 	SignalsToday     int       `json:"signals_today"`
// 	AvgSignalsPerDay float64   `json:"avg_signals_per_day"`
// 	LastSignalAt     time.Time `json:"last_signal_at"`
// 	FavoriteSymbol   string    `json:"favorite_symbol"`
// 	SuccessRate      float64   `json:"success_rate"`
// 	ActiveDays       int       `json:"active_days"`
// 	FirstActivity    time.Time `json:"first_activity"`
// 	LastActivity     time.Time `json:"last_activity"`
// }

// // SystemStats статистика системы
// type SystemStats struct {
// 	TotalUsers          int     `json:"total_users"`
// 	ActiveUsers         int     `json:"active_users"`
// 	NewUsersToday       int     `json:"new_users_today"`
// 	TotalSignalsSent    int64   `json:"total_signals_sent"`
// 	AvgSignalsPerUser   float64 `json:"avg_signals_per_user"`
// 	MostActiveHour      int     `json:"most_active_hour"`
// 	PeakConcurrentUsers int     `json:"peak_concurrent_users"`
// }

// // RegisterRequest запрос на регистрацию
// type RegisterRequest struct {
// 	TelegramID   int64  `json:"telegram_id" validate:"required"`
// 	Username     string `json:"username"`
// 	FirstName    string `json:"first_name" validate:"required"`
// 	LastName     string `json:"last_name"`
// 	ChatID       string `json:"chat_id" validate:"required"`
// 	Email        string `json:"email" validate:"email"`
// 	Phone        string `json:"phone"`
// 	Language     string `json:"language"`
// 	Timezone     string `json:"timezone"`
// 	ReferralCode string `json:"referral_code"`
// }

// // UpdateProfileRequest запрос на обновление профиля
// type UpdateProfileRequest struct {
// 	FirstName *string `json:"first_name,omitempty"`
// 	LastName  *string `json:"last_name,omitempty"`
// 	Email     *string `json:"email,omitempty" validate:"omitempty,email"`
// 	Phone     *string `json:"phone,omitempty"`
// 	Language  *string `json:"language,omitempty"`
// 	Timezone  *string `json:"timezone,omitempty"`
// }

// // SearchCriteria критерии поиска пользователей
// type SearchCriteria struct {
// 	Query      string    `json:"query"`
// 	Role       string    `json:"role,omitempty"`
// 	Status     *bool     `json:"status,omitempty"`
// 	StartDate  time.Time `json:"start_date,omitempty"`
// 	EndDate    time.Time `json:"end_date,omitempty"`
// 	MinSignals int       `json:"min_signals,omitempty"`
// 	MaxSignals int       `json:"max_signals,omitempty"`
// 	SortBy     string    `json:"sort_by,omitempty"`
// 	SortOrder  string    `json:"sort_order,omitempty"` // asc, desc
// 	Limit      int       `json:"limit,omitempty"`
// 	Offset     int       `json:"offset,omitempty"`
// }

// // SearchResult результат поиска пользователей
// type SearchResult struct {
// 	Users      []*users.User          `json:"users"`
// 	Total      int                    `json:"total"`
// 	Page       int                    `json:"page"`
// 	PageSize   int                    `json:"page_size"`
// 	TotalPages int                    `json:"total_pages"`
// 	Stats      map[string]interface{} `json:"stats,omitempty"`
// }

// // Регистрация нового пользователя
// func (s *UserService) RegisterUser(req RegisterRequest) (*users.User, error) {
// 	// Валидация
// 	if req.TelegramID == 0 {
// 		return nil, fmt.Errorf("telegram_id is required")
// 	}
// 	if req.FirstName == "" {
// 		return nil, fmt.Errorf("first_name is required")
// 	}

// 	// Проверка существующего пользователя
// 	existing, err := s.userRepo.FindByTelegramID(req.TelegramID)
// 	if err != nil && err != sql.ErrNoRows {
// 		return nil, fmt.Errorf("failed to check existing user: %w", err)
// 	}

// 	if existing != nil {
// 		// Обновляем последний логин и возвращаем существующего
// 		s.userRepo.UpdateLastLogin(existing.ID)
// 		return existing, nil
// 	}

// 	// Нормализуем язык и часовой пояс
// 	language := normalizeLanguage(req.Language)
// 	timezone := normalizeTimezone(req.Timezone)

// 	// Создаем нового пользователя
// 	user := &users.User{
// 		TelegramID: req.TelegramID,
// 		Username:   req.Username,
// 		FirstName:  req.FirstName,
// 		LastName:   req.LastName,
// 		ChatID:     req.ChatID,
// 		Email:      req.Email,
// 		Phone:      req.Phone,
// 		Role:       users.RoleUser,
// 		IsActive:   true,
// 		IsVerified: false, // Требуется верификация
// 		CreatedAt:  time.Now(),
// 		UpdatedAt:  time.Now(),
// 		Settings: users.UserSettings{
// 			MinGrowthThreshold: 2.0,
// 			MinFallThreshold:   2.0,
// 			PreferredPeriods:   []int{5, 15, 30},
// 			Language:           language,  // Установили язык в настройках
// 			Timezone:           timezone,  // Установили часовой пояс в настройках
// 			DisplayMode:        "compact", // Значение по умолчанию
// 		},
// 		Notifications: users.NotificationSettings{
// 			Enabled:    true,
// 			Growth:     true,
// 			Fall:       true,
// 			Continuous: true,
// 		},
// 	}

// 	// Сохраняем пользователя
// 	if err := s.userRepo.Create(user); err != nil {
// 		return nil, fmt.Errorf("failed to create user: %w", err)
// 	}

// 	// Отправляем приветственное сообщение
// 	s.sendWelcomeMessage(user)

// 	// Трекаем активность
// 	s.analytics.TrackUserActivity(user.ID, "user_registered", map[string]interface{}{
// 		"source":        "telegram",
// 		"referral_code": req.ReferralCode,
// 		"telegram_id":   req.TelegramID,
// 		"username":      req.Username,
// 	})

// 	// Кэшируем пользователя
// 	s.cacheUser(user)

// 	log.Printf("✅ New user registered: %s (ID: %d)", user.FirstName, user.ID)

// 	return user, nil
// }

// // Регистрация через Telegram
// func (s *UserService) RegisterTelegramUser(telegramID int64, username, firstName, lastName, chatID string) (*users.User, error) {
// 	req := RegisterRequest{
// 		TelegramID: telegramID,
// 		Username:   username,
// 		FirstName:  firstName,
// 		LastName:   lastName,
// 		ChatID:     chatID,
// 		Language:   "ru", // По умолчанию русский
// 		Timezone:   "Europe/Moscow",
// 	}

// 	return s.RegisterUser(req)
// }

// // Получение пользователя по ID
// func (s *UserService) GetUserByID(userID int) (*users.User, error) {
// 	// Пробуем получить из кэша
// 	cacheKey := fmt.Sprintf("user:%d", userID)
// 	if cached, err := s.cache.Get(context.Background(), cacheKey).Result(); err == nil {
// 		var user users.User
// 		if err := json.Unmarshal([]byte(cached), &user); err == nil {
// 			return &user, nil
// 		}
// 	}

// 	// Получаем из репозитория
// 	user, err := s.userRepo.FindByID(userID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Кэшируем
// 	s.cacheUser(user)

// 	return user, nil
// }

// // Получение пользователя по Telegram ID
// func (s *UserService) GetUserByTelegramID(telegramID int64) (*users.User, error) {
// 	cacheKey := fmt.Sprintf("user:telegram:%d", telegramID)

// 	if cached, err := s.cache.Get(context.Background(), cacheKey).Result(); err == nil {
// 		var user users.User
// 		if err := json.Unmarshal([]byte(cached), &user); err == nil {
// 			return &user, nil
// 		}
// 	}

// 	user, err := s.userRepo.FindByTelegramID(telegramID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	if user != nil {
// 		s.cacheUser(user)
// 	}

// 	return user, nil
// }

// // Обновление профиля пользователя
// func (s *UserService) UpdateProfile(userID int, req UpdateProfileRequest) (*users.User, error) {
// 	// Получаем пользователя
// 	user, err := s.userRepo.FindByID(userID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Применяем изменения
// 	updated := false

// 	if req.FirstName != nil && *req.FirstName != "" && *req.FirstName != user.FirstName {
// 		user.FirstName = *req.FirstName
// 		updated = true
// 	}

// 	if req.LastName != nil && *req.LastName != user.LastName {
// 		user.LastName = *req.LastName
// 		updated = true
// 	}

// 	if req.Email != nil && *req.Email != user.Email {
// 		// Проверяем уникальность email
// 		if existing, _ := s.userRepo.FindByEmail(*req.Email); existing != nil && existing.ID != userID {
// 			return nil, fmt.Errorf("email already in use")
// 		}
// 		user.Email = *req.Email
// 		updated = true
// 	}

// 	if req.Phone != nil && *req.Phone != user.Phone {
// 		user.Phone = *req.Phone
// 		updated = true
// 	}

// 	// Исправление: обновляем Language и Timezone в Settings
// 	if req.Language != nil && *req.Language != user.Settings.Language {
// 		if !isValidLanguage(*req.Language) {
// 			return nil, fmt.Errorf("invalid language: %s", *req.Language)
// 		}
// 		user.Settings.Language = *req.Language // Исправлено
// 		updated = true
// 	}

// 	if req.Timezone != nil && *req.Timezone != user.Settings.Timezone {
// 		if !isValidTimezone(*req.Timezone) {
// 			return nil, fmt.Errorf("invalid timezone: %s", *req.Timezone)
// 		}
// 		user.Settings.Timezone = *req.Timezone // Исправлено
// 		updated = true
// 	}

// 	// Если нет изменений, возвращаем текущего пользователя
// 	if !updated {
// 		return user, nil
// 	}

// 	// Обновляем в базе
// 	user.UpdatedAt = time.Now()
// 	if err := s.userRepo.Update(user); err != nil {
// 		return nil, fmt.Errorf("failed to update user: %w", err)
// 	}

// 	// Инвалидируем кэш
// 	s.invalidateUserCache(user.ID)

// 	// Трекаем активность
// 	s.analytics.TrackUserActivity(user.ID, "profile_updated", map[string]interface{}{
// 		"updated_fields": getUpdatedFields(req),
// 		"user_id":        userID,
// 	})

// 	// Отправляем уведомление
// 	s.notifier.SendUserNotification(user.ID,
// 		"Ваш профиль был успешно обновлен",
// 		"profile_updated")

// 	return user, nil
// }

// // Обновление роли пользователя (только для администраторов)
// func (s *UserService) UpdateUserRole(userID int, newRole string, updatedBy int) error {
// 	// Получаем пользователя
// 	user, err := s.userRepo.FindByID(userID)
// 	if err != nil {
// 		return err
// 	}

// 	// Проверяем валидность роли
// 	validRoles := map[string]bool{
// 		users.RoleUser:    true,
// 		users.RolePremium: true,
// 		users.RoleAdmin:   true,
// 	}

// 	if !validRoles[newRole] {
// 		return fmt.Errorf("invalid role: %s", newRole)
// 	}

// 	// Сохраняем старую роль для логов
// 	oldRole := user.Role

// 	// Обновляем роль
// 	user.Role = newRole
// 	user.UpdatedAt = time.Now()

// 	if err := s.userRepo.Update(user); err != nil {
// 		return fmt.Errorf("failed to update user role: %w", err)
// 	}

// 	// Инвалидируем кэш
// 	s.invalidateUserCache(user.ID)

// 	// Трекаем активность
// 	s.analytics.TrackUserActivity(user.ID, "role_updated", map[string]interface{}{
// 		"old_role":   oldRole,
// 		"new_role":   newRole,
// 		"updated_by": updatedBy,
// 	})

// 	// Отправляем уведомление пользователю
// 	if newRole == users.RoleAdmin {
// 		s.notifier.SendTelegramNotification(user.ChatID,
// 			"🎉 Вам назначена роль администратора! Теперь у вас есть доступ к панели администрирования.")
// 	}

// 	return nil
// }

// // Активация/деактивация пользователя
// func (s *UserService) ToggleUserStatus(userID int, status bool, updatedBy int) error {
// 	user, err := s.userRepo.FindByID(userID)
// 	if err != nil {
// 		return err
// 	}

// 	// Если статус не меняется
// 	if user.IsActive == status {
// 		return nil
// 	}

// 	user.IsActive = status
// 	user.UpdatedAt = time.Now()

// 	if err := s.userRepo.Update(user); err != nil {
// 		return err
// 	}

// 	// Инвалидируем кэш
// 	s.invalidateUserCache(user.ID)

// 	// Трекаем активность
// 	s.analytics.TrackUserActivity(user.ID, "status_updated", map[string]interface{}{
// 		"new_status": status,
// 		"updated_by": updatedBy,
// 	})

// 	// Отправляем уведомление
// 	if status {
// 		s.notifier.SendTelegramNotification(user.ChatID,
// 			"✅ Ваш аккаунт активирован. Добро пожаловать обратно!")
// 	} else {
// 		s.notifier.SendTelegramNotification(user.ChatID,
// 			"⚠️ Ваш аккаунт деактивирован. Для активации обратитесь к администратору.")
// 	}

// 	return nil
// }

// // Получение статистики пользователя
// func (s *UserService) GetUserStats(userID int) (*UserStats, error) {
// 	cacheKey := fmt.Sprintf("user_stats:%d", userID)

// 	// Пробуем получить из кэша
// 	if cached, err := s.cache.Get(context.Background(), cacheKey).Result(); err == nil {
// 		var stats UserStats
// 		if err := json.Unmarshal([]byte(cached), &stats); err == nil {
// 			return &stats, nil
// 		}
// 	}

// 	// Получаем статистику через сервис аналитики
// 	stats, err := s.analytics.GetUserStats(userID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Кэшируем
// 	if data, err := json.Marshal(stats); err == nil {
// 		s.cache.Set(context.Background(), cacheKey, data, 10*time.Minute)
// 	}

// 	return stats, nil
// }

// // Получение всех активных пользователей
// func (s *UserService) GetActiveUsers() ([]*users.User, error) {
// 	cacheKey := "active_users"

// 	if cached, err := s.cache.Get(context.Background(), cacheKey).Result(); err == nil {
// 		var users []*users.User
// 		if err := json.Unmarshal([]byte(cached), &users); err == nil {
// 			return users, nil
// 		}
// 	}

// 	users, err := s.userRepo.GetAllActive()
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Кэшируем
// 	if data, err := json.Marshal(users); err == nil {
// 		s.cache.Set(context.Background(), cacheKey, data, 5*time.Minute)
// 	}

// 	return users, nil
// }

// // Поиск пользователей
// func (s *UserService) SearchUsers(criteria SearchCriteria) (*SearchResult, error) {
// 	// Построение запроса
// 	query := criteria.Query
// 	limit := criteria.Limit
// 	if limit == 0 {
// 		limit = 50
// 	}
// 	offset := criteria.Offset

// 	// Поиск по репозиторию
// 	users, err := s.userRepo.SearchUsers(query, limit, offset)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Получаем общее количество
// 	ctx := context.Background()
// 	total, err := s.userRepo.GetTotalCount(ctx)
// 	if err != nil {
// 		total = len(users) // fallback
// 	}

// 	// Вычисляем статистику
// 	stats := s.calculateSearchStats(users)

// 	// Вычисляем пагинацию
// 	pageSize := limit
// 	page := offset/limit + 1
// 	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

// 	return &SearchResult{
// 		Users:      users,
// 		Total:      total,
// 		Page:       page,
// 		PageSize:   pageSize,
// 		TotalPages: totalPages,
// 		Stats:      stats,
// 	}, nil
// }

// // Увеличение счетчика сигналов
// func (s *UserService) IncrementSignalsCount(userID int) error {
// 	if err := s.userRepo.IncrementSignalsCount(userID); err != nil {
// 		return err
// 	}

// 	// Инвалидируем кэш статистики
// 	cacheKey := fmt.Sprintf("user_stats:%d", userID)
// 	s.cache.Del(context.Background(), cacheKey)

// 	return nil
// }

// // Сброс дневных счетчиков
// func (s *UserService) ResetDailyCounters() error {
// 	ctx := context.Background()

// 	if err := s.userRepo.ResetDailyCounters(ctx); err != nil {
// 		return err
// 	}

// 	// Инвалидируем кэш статистики
// 	pattern := "user_stats:*"
// 	keys, err := s.cache.Keys(ctx, pattern).Result()
// 	if err == nil {
// 		for _, key := range keys {
// 			s.cache.Del(ctx, key)
// 		}
// 	}

// 	// Инвалидируем кэш активных пользователей
// 	s.cache.Del(ctx, "active_users")

// 	log.Println("✅ Daily counters reset")

// 	return nil
// }

// // Получение системной статистики
// func (s *UserService) GetSystemStats() (*SystemStats, error) {
// 	cacheKey := "system_stats"

// 	if cached, err := s.cache.Get(context.Background(), cacheKey).Result(); err == nil {
// 		var stats SystemStats
// 		if err := json.Unmarshal([]byte(cached), &stats); err == nil {
// 			return &stats, nil
// 		}
// 	}

// 	stats, err := s.analytics.GetSystemStats()
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Кэшируем
// 	if data, err := json.Marshal(stats); err == nil {
// 		s.cache.Set(context.Background(), cacheKey, data, 2*time.Minute)
// 	}

// 	return stats, nil
// }

// // Отправка массового уведомления
// func (s *UserService) SendBulkNotification(message string, userIDs []int, notificationType string) (int, []error) {
// 	var errors []error
// 	sentCount := 0

// 	for _, userID := range userIDs {
// 		if err := s.notifier.SendUserNotification(userID, message, notificationType); err != nil {
// 			errors = append(errors, fmt.Errorf("user %d: %w", userID, err))
// 		} else {
// 			sentCount++
// 		}
// 	}

// 	// Логируем массовую рассылку
// 	if len(userIDs) > 0 {
// 		s.analytics.TrackUserActivity(0, "bulk_notification_sent", map[string]interface{}{
// 			"total_recipients":  len(userIDs),
// 			"sent_count":        sentCount,
// 			"failed_count":      len(errors),
// 			"notification_type": notificationType,
// 		})
// 	}

// 	return sentCount, errors
// }

// // Импорт пользователей
// func (s *UserService) ImportUsers(users []*users.User, overwrite bool) (*ImportResult, error) {
// 	result := &ImportResult{
// 		Total:   len(users),
// 		Created: 0,
// 		Updated: 0,
// 		Skipped: 0,
// 		Errors:  make([]ImportError, 0),
// 	}

// 	for i, user := range users {
// 		// Валидация пользователя
// 		if err := s.validateUserForImport(user); err != nil {
// 			result.Errors = append(result.Errors, ImportError{
// 				Index: i,
// 				Email: user.Email,
// 				Error: err.Error(),
// 			})
// 			result.Skipped++
// 			continue
// 		}

// 		// Проверка существующего пользователя
// 		existing, _ := s.userRepo.FindByEmail(user.Email)
// 		if existing != nil {
// 			if overwrite {
// 				// Обновляем существующего
// 				user.ID = existing.ID
// 				user.UpdatedAt = time.Now()
// 				if err := s.userRepo.Update(user); err != nil {
// 					result.Errors = append(result.Errors, ImportError{
// 						Index: i,
// 						Email: user.Email,
// 						Error: fmt.Sprintf("update failed: %v", err),
// 					})
// 					result.Skipped++
// 				} else {
// 					result.Updated++
// 					s.invalidateUserCache(user.ID)
// 				}
// 			} else {
// 				result.Skipped++
// 			}
// 		} else {
// 			// Создаем нового
// 			user.CreatedAt = time.Now()
// 			user.UpdatedAt = time.Now()
// 			if err := s.userRepo.Create(user); err != nil {
// 				result.Errors = append(result.Errors, ImportError{
// 					Index: i,
// 					Email: user.Email,
// 					Error: fmt.Sprintf("create failed: %v", err),
// 				})
// 				result.Skipped++
// 			} else {
// 				result.Created++
// 				s.sendWelcomeMessage(user)
// 				s.cacheUser(user)
// 			}
// 		}
// 	}

// 	return result, nil
// }

// // Экспорт пользователей
// func (s *UserService) ExportUsers(criteria SearchCriteria) ([]*users.User, error) {
// 	result, err := s.SearchUsers(criteria)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return result.Users, nil
// }

// // Вспомогательные методы

// func (s *UserService) cacheUser(user *users.User) error {
// 	data, err := json.Marshal(user)
// 	if err != nil {
// 		return err
// 	}

// 	ctx := context.Background()

// 	// Кэшируем по разным ключам для быстрого доступа
// 	keys := map[string]string{
// 		fmt.Sprintf("user:%d", user.ID):                  string(data),
// 		fmt.Sprintf("user:telegram:%d", user.TelegramID): string(data),
// 		fmt.Sprintf("user:chat:%s", user.ChatID):         string(data),
// 	}

// 	for key, value := range keys {
// 		s.cache.Set(ctx, key, value, 30*time.Minute)
// 	}

// 	return nil
// }

// func (s *UserService) invalidateUserCache(userID int) {
// 	ctx := context.Background()
// 	user, err := s.userRepo.FindByID(userID)
// 	if err != nil {
// 		return
// 	}

// 	keys := []string{
// 		fmt.Sprintf("user:%d", userID),
// 		fmt.Sprintf("user:telegram:%d", user.TelegramID),
// 		fmt.Sprintf("user:chat:%s", user.ChatID),
// 		fmt.Sprintf("user_stats:%d", userID),
// 	}

// 	s.cache.Del(ctx, keys...)
// }

// func (s *UserService) sendWelcomeMessage(user *users.User) {
// 	message := fmt.Sprintf(
// 		"👋 Добро пожаловать, %s!\n\n"+
// 			"✅ Вы успешно зарегистрированы в Crypto Growth Monitor\n\n"+
// 			"📊 Ваши возможности:\n"+
// 			"• Получать сигналы о росте/падении криптовалют\n"+
// 			"• Настраивать персональные пороги\n"+
// 			"• Управлять уведомлениями\n\n"+
// 			"⚙️ Начните с настройки:\n"+
// 			"1. Используйте /settings для основных настроек\n"+
// 			"2. /notifications для управления уведомлениями\n"+
// 			"3. /help для получения справки\n\n"+
// 			"Удачи в трейдинге! 🚀",
// 		user.FirstName,
// 	)

// 	s.notifier.SendTelegramNotification(user.ChatID, message)

// 	// Также отправляем email если указан
// 	if user.Email != "" {
// 		s.notifier.SendEmailNotification(
// 			user.Email,
// 			"Добро пожаловать в Crypto Growth Monitor",
// 			message,
// 		)
// 	}
// }

// func (s *UserService) calculateSearchStats(users []*users.User) map[string]interface{} {
// 	if len(users) == 0 {
// 		return nil
// 	}

// 	stats := map[string]interface{}{
// 		"total_users":   len(users),
// 		"active_count":  0,
// 		"premium_count": 0,
// 		"admin_count":   0,
// 		"avg_signals":   0,
// 		"new_today":     0,
// 	}

// 	totalSignals := 0
// 	today := time.Now().Format("2006-01-02")

// 	for _, user := range users {
// 		if user.IsActive {
// 			stats["active_count"] = stats["active_count"].(int) + 1
// 		}

// 		if user.Role == users.RolePremium {
// 			stats["premium_count"] = stats["premium_count"].(int) + 1
// 		}

// 		if user.Role == users.RoleAdmin {
// 			stats["admin_count"] = stats["admin_count"].(int) + 1
// 		}

// 		totalSignals += user.SignalsToday

// 		if user.CreatedAt.Format("2006-01-02") == today {
// 			stats["new_today"] = stats["new_today"].(int) + 1
// 		}
// 	}

// 	if len(users) > 0 {
// 		stats["avg_signals"] = float64(totalSignals) / float64(len(users))
// 	}

// 	return stats
// }

// func (s *UserService) validateUserForImport(user *users.User) error {
// 	if user.Email == "" {
// 		return fmt.Errorf("email is required")
// 	}

// 	if user.FirstName == "" {
// 		return fmt.Errorf("first_name is required")
// 	}

// 	if !isValidLanguage(user.Settings.Language) {
// 		return fmt.Errorf("invalid language: %s", user.Settings.Language)
// 	}

// 	if !isValidTimezone(user.Settings.Timezone) {
// 		return fmt.Errorf("invalid timezone: %s", user.Settings.Timezone)
// 	}

// 	return nil
// }

// // Вспомогательные функции
// func normalizeLanguage(lang string) string {
// 	if lang == "" {
// 		return "ru"
// 	}

// 	lang = strings.ToLower(lang)
// 	if strings.HasPrefix(lang, "ru") {
// 		return "ru"
// 	}
// 	if strings.HasPrefix(lang, "en") {
// 		return "en"
// 	}

// 	return "ru" // по умолчанию
// }

// func normalizeTimezone(tz string) string {
// 	if tz == "" {
// 		return "Europe/Moscow"
// 	}

// 	// Простая нормализация
// 	tzMap := map[string]string{
// 		"msk": "Europe/Moscow",
// 		"utc": "UTC",
// 		"est": "America/New_York",
// 		"pst": "America/Los_Angeles",
// 		"gmt": "Europe/London",
// 	}

// 	if normalized, ok := tzMap[strings.ToLower(tz)]; ok {
// 		return normalized
// 	}

// 	return tz
// }

// func isValidLanguage(lang string) bool {
// 	validLanguages := []string{"ru", "en", "es", "zh", "de", "fr", "it", "ja", "ko"}
// 	for _, valid := range validLanguages {
// 		if lang == valid {
// 			return true
// 		}
// 	}
// 	return false
// }

// func isValidTimezone(tz string) bool {
// 	// В реальном приложении используйте time.LoadLocation для проверки
// 	// Здесь упрощенная проверка
// 	knownTimezones := []string{
// 		"Europe/Moscow", "UTC", "America/New_York", "Europe/London",
// 		"Asia/Tokyo", "Australia/Sydney", "Europe/Berlin", "Asia/Shanghai",
// 	}

// 	for _, known := range knownTimezones {
// 		if tz == known {
// 			return true
// 		}
// 	}

// 	return false
// }

// // Вспомогательная функция для получения обновленных полей
// func getUpdatedFields(req UpdateProfileRequest) []string {
// 	fields := []string{}

// 	if req.FirstName != nil {
// 		fields = append(fields, "first_name")
// 	}
// 	if req.LastName != nil {
// 		fields = append(fields, "last_name")
// 	}
// 	if req.Email != nil {
// 		fields = append(fields, "email")
// 	}
// 	if req.Phone != nil {
// 		fields = append(fields, "phone")
// 	}
// 	if req.Language != nil {
// 		fields = append(fields, "language")
// 	}
// 	if req.Timezone != nil {
// 		fields = append(fields, "timezone")
// 	}

// 	return fields
// }

// // Структуры для импорта/экспорта
// type ImportResult struct {
// 	Total   int           `json:"total"`
// 	Created int           `json:"created"`
// 	Updated int           `json:"updated"`
// 	Skipped int           `json:"skipped"`
// 	Errors  []ImportError `json:"errors"`
// }

// type ImportError struct {
// 	Index int    `json:"index"`
// 	Email string `json:"email"`
// 	Error string `json:"error"`
// }

// // Метод для планировщика задач
// func (s *UserService) StartScheduledTasks() {
// 	// Ежедневный сброс счетчиков
// 	go s.startDailyResetScheduler()

// 	// Очистка старых кэшей
// 	go s.startCacheCleanupScheduler()

// 	// Проверка неактивных пользователей
// 	go s.startInactiveUsersCheckScheduler()
// }

// func (s *UserService) startDailyResetScheduler() {
// 	for {
// 		now := time.Now()
// 		// Вычисляем время до следующей полночи
// 		nextMidnight := time.Date(
// 			now.Year(), now.Month(), now.Day()+1,
// 			0, 0, 0, 0, now.Location(),
// 		)

// 		durationUntilMidnight := nextMidnight.Sub(now)
// 		time.Sleep(durationUntilMidnight)

// 		if err := s.ResetDailyCounters(); err != nil {
// 			log.Printf("Error resetting daily counters: %v", err)
// 		}
// 	}
// }

// func (s *UserService) startCacheCleanupScheduler() {
// 	ticker := time.NewTicker(1 * time.Hour)
// 	defer ticker.Stop()

// 	for range ticker.C {
// 		s.cleanupOldCache()
// 	}
// }

// func (s *UserService) startInactiveUsersCheckScheduler() {
// 	ticker := time.NewTicker(24 * time.Hour)
// 	defer ticker.Stop()

// 	for range ticker.C {
// 		s.checkInactiveUsers()
// 	}
// }

// func (s *UserService) cleanupOldCache() {
// 	ctx := context.Background()

// 	// Удаляем старые кэши статистики
// 	pattern := "user_stats:*"
// 	keys, err := s.cache.Keys(ctx, pattern).Result()
// 	if err != nil {
// 		return
// 	}

// 	for _, key := range keys {
// 		if ttl, err := s.cache.TTL(ctx, key).Result(); err == nil && ttl < 0 {
// 			s.cache.Del(ctx, key)
// 		}
// 	}
// }

// func (s *UserService) checkInactiveUsers() {
// 	// Находим пользователей, которые не заходили более 30 дней
// 	// thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

// 	// В реальной реализации нужен метод репозитория для поиска неактивных
// 	// users, err := s.userRepo.GetInactiveUsers(thirtyDaysAgo)
// 	// Отправляем уведомления о возвращении
// }
