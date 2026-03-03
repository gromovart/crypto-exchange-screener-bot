// internal/core/domain/users/service.go
package users

import (
	"context"
	cryptorand "crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"crypto-exchange-screener-bot/internal/infrastructure/cache/redis"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
	activity_repo "crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/repository/activity"
	session_repo "crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/repository/session"
	trading_session_repo "crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/repository/trading_session"
	users_repo "crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/repository/users"
	"crypto-exchange-screener-bot/pkg/logger"

	"github.com/jmoiron/sqlx"
)

// Config конфигурация сервиса
type Config struct {
	UserDefaults struct {
		MinGrowthThreshold float64
		MinFallThreshold   float64
		Language           string
		Timezone           string
	}
	DefaultMaxSignalsPerDay int
	SessionTTL              time.Duration
	MaxSessionsPerUser      int
}

// NotificationService интерфейс для уведомлений
type NotificationService interface {
	SendTelegramNotification(chatID, message string) error
}

// Service сервис управления пользователями
type Service struct {
	repo               users_repo.UserRepository
	sessionRepo        session_repo.SessionRepository
	activityRepo       activity_repo.ActivityRepository
	tradingSessionRepo trading_session_repo.TradingSessionRepository
	cache              *redis.Cache
	cachePrefix        string
	cacheTTL           time.Duration
	mu                 sync.RWMutex
	notifier           NotificationService
	config             Config
}

// NewService создает новый сервис пользователей
func NewService(
	db *sqlx.DB,
	cache *redis.Cache,
	notifier NotificationService,
	cfg Config, // ⭐ Передаем локальный Config
) (*Service, error) {

	userRepo := users_repo.NewUserRepository(db, cache)
	sessionRepo := session_repo.NewSessionRepository(db, cache)
	activityRepo := activity_repo.NewActivityRepository(db, cache)
	tradingSessionRepo := trading_session_repo.NewTradingSessionRepository(db)

	service := &Service{
		repo:               userRepo,
		sessionRepo:        sessionRepo,
		activityRepo:       activityRepo,
		tradingSessionRepo: tradingSessionRepo,
		cache:              cache,
		cachePrefix:        "users:",
		cacheTTL:           30 * time.Minute,
		notifier:           notifier,
		config: Config{
			UserDefaults: struct {
				MinGrowthThreshold float64
				MinFallThreshold   float64
				Language           string
				Timezone           string
			}{
				MinGrowthThreshold: cfg.UserDefaults.MinGrowthThreshold,
				MinFallThreshold:   cfg.UserDefaults.MinFallThreshold,
				Language:           cfg.UserDefaults.Language,
				Timezone:           cfg.UserDefaults.Timezone,
			},
			DefaultMaxSignalsPerDay: 1500,
			SessionTTL:              24 * time.Hour,
			MaxSessionsPerUser:      5,
		},
	}

	logger.Info("✅ User service initialized")
	return service, nil
}

// CreateUser создает нового пользователя
func (s *Service) CreateUser(telegramID int64, username, firstName, lastName string) (*models.User, error) {
	// Проверяем существование пользователя
	existing, err := s.repo.FindByTelegramID(telegramID)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}

	if existing != nil {
		return existing, nil
	}

	// Создаем нового пользователя с настройками по умолчанию из конфига
	user := &models.User{
		TelegramID:           telegramID,
		Username:             username,
		FirstName:            firstName,
		LastName:             lastName,
		IsActive:             true,
		Role:                 models.RoleUser,
		MinGrowthThreshold:   s.config.UserDefaults.MinGrowthThreshold, // ⭐ из конфига
		MinFallThreshold:     s.config.UserDefaults.MinFallThreshold,   // ⭐ из конфига
		Language:             s.config.UserDefaults.Language,           // ⭐ из конфига
		Timezone:             s.config.UserDefaults.Timezone,           // ⭐ из конфига
		MaxSignalsPerDay:     s.config.DefaultMaxSignalsPerDay,
		NotificationsEnabled: true,
		SubscriptionTier:     models.TierFree,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}

	if err := s.repo.Create(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Логируем создание пользователя
	s.logUserActivity(user, "user_created", "Пользователь создан", nil)

	log.Printf("✅ Created new user: %s (ID: %d)", username, user.ID)

	return user, nil
}

// GetOrCreateUser получает или создает пользователя
func (s *Service) GetOrCreateUser(telegramID int64, username, firstName, lastName string) (*models.User, error) {
	// Пробуем получить из кэша
	cacheKey := s.cachePrefix + fmt.Sprintf("telegram:%d", telegramID)
	var cachedUser models.User
	if err := s.cache.Get(context.Background(), cacheKey, &cachedUser); err == nil {
		return &cachedUser, nil
	}

	// Ищем в базе
	user, err := s.repo.FindByTelegramID(telegramID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Если не нашли, создаем
	if user == nil {
		user, err = s.CreateUser(telegramID, username, firstName, lastName)
		if err != nil {
			return nil, err
		}
	}

	// Кэшируем
	s.cacheUser(user)

	return user, nil
}

// GetUserByID возвращает пользователя по ID
func (s *Service) GetUserByID(id int) (*models.User, error) {
	// Пробуем получить из кэша
	cacheKey := s.cachePrefix + fmt.Sprintf("id:%d", id)
	var cachedUser models.User
	if err := s.cache.Get(context.Background(), cacheKey, &cachedUser); err == nil {
		return &cachedUser, nil
	}

	// Ищем в базе
	user, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// Кэшируем
	if user != nil {
		s.cacheUser(user)
	}

	return user, nil
}

// GetUserByTelegramID возвращает пользователя по Telegram ID
func (s *Service) GetUserByTelegramID(telegramID int64) (*models.User, error) {
	// Пробуем получить из кэша
	cacheKey := s.cachePrefix + fmt.Sprintf("telegram:%d", telegramID)
	var cachedUser models.User
	if err := s.cache.Get(context.Background(), cacheKey, &cachedUser); err == nil {
		// Проверяем и исправляем нулевые пороги
		s.fixUserDefaults(&cachedUser)
		return &cachedUser, nil
	}

	// Ищем в базе
	user, err := s.repo.FindByTelegramID(telegramID)
	if err != nil {
		return nil, err
	}

	// Проверяем и исправляем нулевые пороги
	if user != nil {
		s.fixUserDefaults(user)
		s.cacheUser(user)
	}

	return user, nil
}

// fixUserDefaults исправляет нулевые значения на значения по умолчанию из конфига
func (s *Service) fixUserDefaults(user *models.User) {
	fixed := false

	if user.MinGrowthThreshold == 0 {
		user.MinGrowthThreshold = s.config.UserDefaults.MinGrowthThreshold
		logger.Warn("⚠️ Исправлен нулевой порог роста для user %d на %.1f%%",
			user.ID, s.config.UserDefaults.MinGrowthThreshold)
		fixed = true
	}

	if user.MinFallThreshold == 0 {
		user.MinFallThreshold = s.config.UserDefaults.MinFallThreshold
		logger.Warn("⚠️ Исправлен нулевой порог падения для user %d на %.1f%%",
			user.ID, s.config.UserDefaults.MinFallThreshold)
		fixed = true
	}

	if user.Language == "" {
		user.Language = s.config.UserDefaults.Language
		logger.Warn("⚠️ Исправлен пустой язык для user %d на %s",
			user.ID, s.config.UserDefaults.Language)
		fixed = true
	}

	if user.Timezone == "" {
		user.Timezone = s.config.UserDefaults.Timezone
		logger.Warn("⚠️ Исправлен пустой часовой пояс для user %d на %s",
			user.ID, s.config.UserDefaults.Timezone)
		fixed = true
	}

	if fixed {
		// Сохраняем исправления в БД
		if err := s.repo.Update(user); err != nil {
			logger.Error("❌ Не удалось сохранить исправленные настройки для user %d: %v",
				user.ID, err)
		}
	}
}

// UpdateUser обновляет данные пользователя
// UpdateSubscriptionTier обновляет тариф пользователя
func (s *Service) UpdateSubscriptionTier(userID int, tier string) error {
	user, err := s.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("ошибка получения пользователя: %w", err)
	}
	if user == nil {
		return fmt.Errorf("пользователь %d не найден", userID)
	}
	user.SubscriptionTier = tier
	if err := s.UpdateUser(user); err != nil {
		return fmt.Errorf("ошибка обновления тарифа: %w", err)
	}
	// Инвалидируем кэш пользователей
	s.cache.Delete(context.Background(), "all_users_for_notify")
	return nil
}

func (s *Service) UpdateUser(user *models.User) error {
	user.UpdatedAt = time.Now()

	if err := s.repo.Update(user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	// Инвалидируем кэш
	s.invalidateUserCache(user)

	// Логируем обновление
	s.logUserActivity(user, "user_updated", "Данные пользователя обновлены", nil)

	return nil
}

// UpdateSettings обновляет настройки пользователя
func (s *Service) UpdateSettings(userID int, settings map[string]interface{}) error {
	// Получаем пользователя
	user, err := s.GetUserByID(userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}

	// Сохраняем старые настройки для логирования
	oldSettings := map[string]interface{}{
		"min_growth_threshold":  user.MinGrowthThreshold,
		"min_fall_threshold":    user.MinFallThreshold,
		"max_signals_per_day":   user.MaxSignalsPerDay,
		"notifications_enabled": user.NotificationsEnabled,
		"notify_growth":         user.NotifyGrowth,
		"notify_fall":           user.NotifyFall,
		"preferred_periods":     user.PreferredPeriods, // ← ДОБАВЛЯЕМ
	}

	// Применяем новые настройки
	for key, value := range settings {
		switch key {
		case "min_growth_threshold":
			if val, ok := value.(float64); ok {
				user.MinGrowthThreshold = val
			}
		case "min_fall_threshold":
			if val, ok := value.(float64); ok {
				user.MinFallThreshold = val
			}
		case "max_signals_per_day":
			if val, ok := value.(int); ok {
				user.MaxSignalsPerDay = val
			}
		case "notifications_enabled":
			if val, ok := value.(bool); ok {
				user.NotificationsEnabled = val
			}
		case "max_notifications_enabled":
			if val, ok := value.(bool); ok {
				user.MaxNotificationsEnabled = val
			}
		case "notify_growth":
			if val, ok := value.(bool); ok {
				user.NotifyGrowth = val
			}
		case "notify_fall":
			if val, ok := value.(bool); ok {
				user.NotifyFall = val
			}
		case "preferred_periods": // ← ДОБАВЛЯЕМ
			if val, ok := value.([]int); ok {
				user.PreferredPeriods = val
			}
		}
	}

	// Сохраняем изменения
	if err := s.UpdateUser(user); err != nil {
		return err
	}

	// Логируем изменение настроек
	s.logSettingsUpdate(user, settings, oldSettings)

	return nil
}

// CreateSession создает сессию для пользователя
func (s *Service) CreateSession(userID int, token, ip, userAgent string, deviceInfo map[string]interface{}) (*models.Session, error) {
	// Проверяем лимит сессий
	sessionCount, err := s.sessionRepo.GetSessionCount(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session count: %w", err)
	}

	if sessionCount >= s.config.MaxSessionsPerUser {
		// Отзываем старые сессии
		if err := s.sessionRepo.RevokeAllUserSessions(userID, "session_limit_exceeded"); err != nil {
			return nil, fmt.Errorf("failed to revoke old sessions: %w", err)
		}
	}

	// Создаем новую сессию
	now := time.Now()
	session := &models.Session{
		ID:           generateUUID(),
		UserID:       userID,
		Token:        token,
		DeviceInfo:   deviceInfo,
		ExpiresAt:    now.Add(s.config.SessionTTL),
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
		LastActivity: now,
	}

	// Создаем указатели на строки для IP и UserAgent
	if ip != "" {
		session.IPAddress = &ip
	}
	if userAgent != "" {
		session.UserAgent = &userAgent
	}

	// Сохраняем сессию
	if err := s.sessionRepo.Create(session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Логируем создание сессии
	s.logSessionActivity(session, "session_created", "Сессия создана")

	return session, nil
}

// UpdateSessionActivity обновляет время последней активности сессии
func (s *Service) UpdateSessionActivity(sessionID string) error {
	if err := s.sessionRepo.UpdateLastActivity(sessionID); err != nil {
		return fmt.Errorf("failed to update session activity: %w", err)
	}

	// Логируем активность
	activity := &models.SessionActivity{
		SessionID:    sessionID,
		ActivityType: "activity_updated",
		Details: map[string]interface{}{
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}

	if err := s.sessionRepo.LogActivity(activity); err != nil {
		log.Printf("Failed to log session activity: %v", err)
	}

	return nil
}

// ValidateSession проверяет валидность сессии
func (s *Service) ValidateSession(token string) (*models.Session, error) {
	// Находим сессию по токену
	session, err := s.sessionRepo.FindByToken(token)
	if err != nil {
		return nil, fmt.Errorf("failed to find session: %w", err)
	}

	if session == nil {
		return nil, errors.New("session not found")
	}

	// Проверяем активность
	if !session.IsActive {
		return nil, errors.New("session is not active")
	}

	// Проверяем срок действия
	if time.Now().After(session.ExpiresAt) {
		// Автоматически отзываем истекшую сессию
		s.sessionRepo.Revoke(session.ID, "session_expired")
		return nil, errors.New("session has expired")
	}

	// Обновляем время последней активности
	go s.UpdateSessionActivity(session.ID)

	return session, nil
}

// GetUserSessions возвращает сессии пользователя
func (s *Service) GetUserSessions(userID int, limit, offset int) ([]*models.Session, error) {
	return s.sessionRepo.FindByUserID(userID, limit, offset)
}

// RevokeSession отзывает сессию
func (s *Service) RevokeSession(sessionID, reason string) error {
	if err := s.sessionRepo.Revoke(sessionID, reason); err != nil {
		return fmt.Errorf("failed to revoke session: %w", err)
	}

	// Логируем отзыв сессии
	session, err := s.sessionRepo.FindByID(sessionID)
	if err == nil && session != nil {
		s.logSessionActivity(session, "session_revoked", fmt.Sprintf("Сессия отозвана: %s", reason))
	}

	return nil
}

// LogoutUser выполняет выход пользователя
// LogSignalSent логирует отправку сигнала пользователю
func (s *Service) LogSignalSent(userID int, signalType, symbol string, changePercent float64, periodMinutes int) {
	user, err := s.GetUserByID(userID)
	if err != nil || user == nil {
		logger.Warn("⚠️ LogSignalSent: пользователь %d не найден", userID)
		return
	}
	// Логируем в user_activities (заполняет activity_summary через триггер)
	if err := s.activityRepo.LogSignalReceived(user, signalType, symbol, changePercent, periodMinutes, false, ""); err != nil {
		logger.Warn("⚠️ LogSignalSent: ошибка записи user_activities: %v", err)
	}
	// Логируем в signal_activities
	signalID := fmt.Sprintf("%s_%s_%d_%d", symbol, signalType, periodMinutes, time.Now().Unix())
	if err := s.activityRepo.LogSignalActivity(userID, signalID, symbol, signalType, changePercent); err != nil {
		logger.Warn("⚠️ LogSignalSent: ошибка записи signal_activities: %v", err)
	}
}

func (s *Service) LogoutUser(userID int, sessionID, ip, userAgent string) error {
	// Отзываем конкретную сессию если указана
	if sessionID != "" {
		if err := s.RevokeSession(sessionID, "user_logout"); err != nil {
			return fmt.Errorf("failed to revoke session: %w", err)
		}
	} else {
		// Отзываем все сессии пользователя
		if err := s.sessionRepo.RevokeAllUserSessions(userID, "user_logout"); err != nil {
			return fmt.Errorf("failed to revoke all sessions: %w", err)
		}
	}

	// Получаем пользователя для логирования
	user, err := s.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Логируем выход
	s.logUserLogout(user, ip, userAgent, "user_logout")

	return nil
}

// LogoutSession выполняет выход из конкретной сессии
func (s *Service) LogoutSession(session *models.Session, ip, userAgent string) error {
	// Исправлено: получаем строки из указателей
	var ipStr, userAgentStr string
	if session.IPAddress != nil {
		ipStr = *session.IPAddress
	}
	if session.UserAgent != nil {
		userAgentStr = *session.UserAgent
	}

	// Отзываем сессию
	if err := s.RevokeSession(session.ID, "session_logout"); err != nil {
		return fmt.Errorf("failed to revoke session: %w", err)
	}

	// Получаем пользователя для логирования
	user, err := s.GetUserByID(session.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Логируем выход
	s.logUserLogout(user, ipStr, userAgentStr, "session_logout")

	return nil
}

// ResetDailyCounters сбрасывает дневные счетчики
func (s *Service) ResetDailyCounters() error {
	// Исправлено: передаем контекст
	ctx := context.Background()
	if err := s.repo.ResetDailySignals(ctx); err != nil {
		return fmt.Errorf("failed to reset daily signals: %w", err)
	}

	// Логируем сброс счетчиков
	s.logSystemEvent("counters_reset", "Дневные счетчики сброшены", nil)

	return nil
}

// GetUserStats возвращает статистику пользователя
func (s *Service) GetUserStats(userID int) (map[string]interface{}, error) {
	// Получаем базовую статистику
	stats := make(map[string]interface{})

	// Статистика пользователя
	user, err := s.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	stats["user_id"] = user.ID
	stats["telegram_id"] = user.TelegramID
	stats["username"] = user.Username
	stats["first_name"] = user.FirstName
	stats["role"] = user.Role
	stats["created_at"] = user.CreatedAt
	stats["signals_today"] = user.SignalsToday
	stats["max_signals_per_day"] = user.MaxSignalsPerDay
	stats["min_growth_threshold"] = user.MinGrowthThreshold
	stats["notifications_enabled"] = user.NotificationsEnabled

	// Статистика сессий
	sessionStats, err := s.sessionRepo.GetUserSessionStats(userID)
	if err == nil {
		stats["sessions"] = sessionStats
	}

	// Статистика активности
	activityStats, err := s.activityRepo.GetUserActivityStats(userID, 30)
	if err == nil {
		stats["activity"] = activityStats
	}

	return stats, nil
}

// SearchUsers ищет пользователей
func (s *Service) SearchUsers(query string, limit, offset int) ([]*models.User, error) {
	// Исправлено: вызываем Search
	return s.repo.Search(query, limit, offset)
}

// GetAllUsers возвращает всех пользователей с пагинацией
func (s *Service) GetAllUsers(limit, offset int) ([]*models.User, error) {
	ctx := context.Background()
	cacheKey := "all_users_for_notify"

	// Пробуем получить из кэша (TTL 1 минута)
	var cachedUsers []*models.User
	if err := s.cache.Get(ctx, cacheKey, &cachedUsers); err == nil && len(cachedUsers) > 0 {
		// logger.Info("👥 GetAllUsers: из кэша Redis (%d пользователей)", len(cachedUsers))
		return cachedUsers, nil
	}

	// Кэш пуст — идём в БД
	// logger.Info("👥 GetAllUsers: запрос к БД (кэш пуст)")
	// Кэш пуст — идём в БД
	// logger.Info("👥 GetAllUsers: запрос к БД (кэш пуст)")
	users, err := s.repo.GetAll(limit, offset)
	if err != nil {
		return nil, err
	}

	// Применяем fixUserDefaults к каждому пользователю
	for _, user := range users {
		s.fixUserDefaults(user)
	}

	// Сохраняем в кэш на 1 минуту
	_ = s.cache.Set(ctx, cacheKey, users, 1*time.Minute)

	return users, nil
}

// GetTotalUsersCount возвращает общее количество пользователей
func (s *Service) GetTotalUsersCount() (int, error) {
	// Исправлено: передаем контекст
	ctx := context.Background()
	return s.repo.GetTotalCount(ctx)
}

// GetActiveUsersCount возвращает количество активных пользователей
func (s *Service) GetActiveUsersCount() (int, error) {
	// Исправлено: передаем контекст
	ctx := context.Background()
	return s.repo.GetActiveUsersCount(ctx)
}

// BanUser блокирует пользователя
func (s *Service) BanUser(userID int, reason string) error {
	// Исправлено: вызываем UpdateStatus
	if err := s.repo.UpdateStatus(userID, false); err != nil {
		return fmt.Errorf("failed to ban user: %w", err)
	}

	// Получаем пользователя для логирования
	user, err := s.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Отзываем все сессии
	if err := s.sessionRepo.RevokeAllUserSessions(userID, fmt.Sprintf("user_banned: %s", reason)); err != nil {
		log.Printf("Failed to revoke sessions for banned user %d: %v", userID, err)
	}

	// Логируем блокировку
	s.logSecurityEvent(user, "user_banned", fmt.Sprintf("Пользователь заблокирован: %s", reason),
		models.SeverityWarning, "", "", nil)

	// Отправляем уведомление если возможно
	if user.TelegramID > 0 {
		message := fmt.Sprintf(
			"🚫 Ваш аккаунт был заблокирован.\nПричина: %s\n\nДля разблокировки обратитесь в поддержку.",
			reason,
		)
		go s.notifier.SendTelegramNotification(fmt.Sprintf("%d", user.TelegramID), message)
	}

	return nil
}

// UnbanUser разблокирует пользователя
func (s *Service) UnbanUser(userID int) error {
	// Исправлено: вызываем UpdateStatus
	if err := s.repo.UpdateStatus(userID, true); err != nil {
		return fmt.Errorf("failed to unban user: %w", err)
	}

	// Получаем пользователя для логирования
	user, err := s.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Логируем разблокировку
	s.logSecurityEvent(user, "user_unbanned", "Пользователь разблокирован",
		models.SeverityInfo, "", "", nil)

	// Отправляем уведомление если возможно
	if user.TelegramID > 0 {
		message := "✅ Ваш аккаунт был разблокирован.\nТеперь вы можете снова пользоваться сервисом."
		go s.notifier.SendTelegramNotification(fmt.Sprintf("%d", user.TelegramID), message)
	}

	return nil
}

// ChangeUserRole изменяет роль пользователя
func (s *Service) ChangeUserRole(userID int, newRole string) error {
	// Валидация роли
	validRoles := map[string]bool{
		models.RoleUser:      true,
		models.RoleAdmin:     true,
		models.RoleModerator: true, // Теперь RoleModerator определена
	}

	if !validRoles[newRole] {
		return fmt.Errorf("invalid role: %s", newRole)
	}

	// Исправлено: вызываем UpdateRole
	if err := s.repo.UpdateRole(userID, newRole); err != nil {
		return fmt.Errorf("failed to update user role: %w", err)
	}

	// Получаем пользователя для логирования
	user, err := s.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Логируем изменение роли
	oldRole := user.Role
	s.logUserActivity(user, "role_changed", fmt.Sprintf("Роль изменена с %s на %s", oldRole, newRole), nil)

	// Отправляем уведомление если возможно
	if user.TelegramID > 0 {
		message := fmt.Sprintf(
			"👑 Ваша роль изменена.\n\nСтарая роль: %s\nНовая роль: %s",
			oldRole, newRole,
		)
		go s.notifier.SendTelegramNotification(fmt.Sprintf("%d", user.TelegramID), message)
	}

	// Инвалидируем кэш
	s.invalidateUserCache(user)

	return nil
}

// Вспомогательные методы

func (s *Service) cacheUser(user *models.User) error {
	data, err := json.Marshal(user)
	if err != nil {
		return err
	}

	ctx := context.Background()
	keys := []string{
		s.cachePrefix + fmt.Sprintf("id:%d", user.ID),
		s.cachePrefix + fmt.Sprintf("telegram:%d", user.TelegramID),
	}

	for _, key := range keys {
		s.cache.Set(ctx, key, string(data), s.cacheTTL)
	}

	return nil
}

func (s *Service) invalidateUserCache(user *models.User) {
	ctx := context.Background()
	keys := []string{
		s.cachePrefix + fmt.Sprintf("id:%d", user.ID),
		s.cachePrefix + fmt.Sprintf("telegram:%d", user.TelegramID),
		"users:stats:*",
		"all_users_for_notify",
	}

	s.cache.DeleteMulti(ctx, keys...)
}

// ───────────────────────────────────────────────
// MAX мессенджер — методы объединения аккаунтов
// ───────────────────────────────────────────────

// GetOrCreateUserByMaxID ищет пользователя по max_user_id. Если не найден —
// создаёт нового пользователя, используя maxUserID как telegram_id (backward compat)
// и устанавливает max_user_id = maxUserID.
func (s *Service) GetOrCreateUserByMaxID(maxUserID int64, username, firstName, lastName string) (*models.User, error) {
	cacheKey := s.cachePrefix + fmt.Sprintf("max:%d", maxUserID)
	var cachedUser models.User
	if err := s.cache.Get(context.Background(), cacheKey, &cachedUser); err == nil {
		return &cachedUser, nil
	}

	// 1. Ищем по max_user_id
	user, err := s.repo.FindByMaxUserID(maxUserID)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("GetOrCreateUserByMaxID: FindByMaxUserID: %w", err)
	}

	if user == nil {
		// 2. Backward compat: ищем по telegram_id == maxUserID (старые MAX-записи)
		user, err = s.repo.FindByTelegramID(maxUserID)
		if err != nil && err != sql.ErrNoRows {
			return nil, fmt.Errorf("GetOrCreateUserByMaxID: FindByTelegramID: %w", err)
		}
		if user != nil && user.MaxUserID == nil {
			// Устанавливаем max_user_id на уже существующей записи
			user.MaxUserID = &maxUserID
			if err := s.repo.Update(user); err != nil {
				return nil, fmt.Errorf("GetOrCreateUserByMaxID: Update: %w", err)
			}
		}
	}

	if user == nil {
		// 3. Создаём нового MAX-пользователя
		now := time.Now()
		user = &models.User{
			TelegramID:           maxUserID,
			MaxUserID:            &maxUserID,
			Username:             username,
			FirstName:            firstName,
			LastName:             lastName,
			IsActive:             true,
			Role:                 models.RoleUser,
			SubscriptionTier:     models.TierFree,
			NotificationsEnabled: true,
			NotifyGrowth:         true,
			NotifyFall:           true,
			NotifyContinuous:     true,
			MinGrowthThreshold:   s.config.UserDefaults.MinGrowthThreshold,
			MinFallThreshold:     s.config.UserDefaults.MinFallThreshold,
			MinVolumeFilter:      100000,
			Language:             "ru",
			Timezone:             "Europe/Moscow",
			DisplayMode:          "compact",
			PreferredPeriods:     []int{5, 15, 30},
			ExcludePatterns:      []string{},
			MaxSignalsPerDay:     1500,
			CreatedAt:            now,
			UpdatedAt:            now,
		}
		if err := s.repo.Create(user); err != nil {
			return nil, fmt.Errorf("GetOrCreateUserByMaxID: Create: %w", err)
		}
		logger.Info("✅ Новый MAX-пользователь создан: id=%d maxUserID=%d", user.ID, maxUserID)
	}

	s.cacheUserByMaxID(user)
	return user, nil
}

// GenerateLinkCode генерирует 6-символьный код привязки для Telegram-пользователя.
// Код действителен 15 минут.
func (s *Service) GenerateLinkCode(telegramID int64) (string, error) {
	user, err := s.repo.FindByTelegramID(telegramID)
	if err != nil {
		return "", fmt.Errorf("GenerateLinkCode: %w", err)
	}
	if user == nil {
		return "", fmt.Errorf("GenerateLinkCode: пользователь не найден")
	}

	code := generateCode()
	expires := time.Now().Add(15 * time.Minute)
	user.LinkCode = code
	user.LinkCodeExpiresAt = &expires

	if err := s.repo.Update(user); err != nil {
		return "", fmt.Errorf("GenerateLinkCode: Update: %w", err)
	}
	s.invalidateUserCache(user)
	return code, nil
}

// LinkMaxAccount привязывает MAX-аккаунт (maxUserID) к существующему Telegram-аккаунту
// по коду linkCode. Возвращает объединённого пользователя.
func (s *Service) LinkMaxAccount(maxUserID int64, maxChatID, linkCode string) (*models.User, error) {
	// 1. Ищем TG-пользователя по коду
	tgUser, err := s.repo.FindByLinkCode(linkCode)
	if err != nil {
		return nil, fmt.Errorf("LinkMaxAccount: FindByLinkCode: %w", err)
	}
	if tgUser == nil {
		return nil, fmt.Errorf("LinkMaxAccount: неверный или истёкший код")
	}

	// 2. Ищем MAX-пользователя (может уже существовать как отдельная запись)
	maxUser, err := s.repo.FindByMaxUserID(maxUserID)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("LinkMaxAccount: FindByMaxUserID: %w", err)
	}

	// 3. Если MAX-пользователь существует как отдельная запись — удалять не будем,
	//    просто обнуляем его max_user_id (данные уйдут в TG-запись).
	//    Это избегает сложной миграции данных при слиянии записей.
	if maxUser != nil && maxUser.ID != tgUser.ID {
		maxUser.MaxUserID = nil
		maxUser.MaxChatID = ""
		_ = s.repo.Update(maxUser)
		s.invalidateUserCache(maxUser)
		// Инвалидируем кэш по MAX-ключу
		s.cache.Delete(context.Background(), s.cachePrefix+fmt.Sprintf("max:%d", maxUserID))
	}

	// 4. Привязываем MAX к TG-пользователю
	tgUser.MaxUserID = &maxUserID
	tgUser.MaxChatID = maxChatID
	tgUser.LinkCode = ""
	tgUser.LinkCodeExpiresAt = nil

	if err := s.repo.Update(tgUser); err != nil {
		return nil, fmt.Errorf("LinkMaxAccount: Update: %w", err)
	}
	s.invalidateUserCache(tgUser)
	s.cacheUserByMaxID(tgUser)

	logger.Info("🔗 MAX-аккаунт %d привязан к TG-пользователю id=%d", maxUserID, tgUser.ID)
	return tgUser, nil
}

// cacheUserByMaxID кэширует пользователя по MAX-ключу
func (s *Service) cacheUserByMaxID(user *models.User) {
	if user.MaxUserID == nil {
		return
	}
	data, err := json.Marshal(user)
	if err != nil {
		return
	}
	key := s.cachePrefix + fmt.Sprintf("max:%d", *user.MaxUserID)
	s.cache.Set(context.Background(), key, string(data), s.cacheTTL)
}

// generateCode генерирует случайный 6-символьный буквенно-цифровой код
func generateCode() string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	buf := make([]byte, 6)
	if _, err := cryptorand.Read(buf); err != nil {
		// fallback на time-based если crypto/rand недоступен
		for i := range buf {
			buf[i] = chars[time.Now().UnixNano()%int64(len(chars))]
		}
		return string(buf)
	}
	for i, b := range buf {
		buf[i] = chars[b%byte(len(chars))]
	}
	return string(buf)
}

func (s *Service) logUserActivity(user *models.User, activityType, description string, metadata map[string]interface{}) {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["description"] = description

	activity := models.NewUserActivity(
		user.ID,
		models.ActivityType(activityType),
		models.CategoryUser,
		models.SeverityInfo,
		metadata,
	)

	activity.TelegramID = user.TelegramID
	activity.Username = user.Username
	activity.FirstName = user.FirstName

	go func() {
		if err := s.activityRepo.Create(activity); err != nil {
			log.Printf("Failed to log user activity: %v", err)
		}
	}()
}

func (s *Service) logSettingsUpdate(user *models.User, newSettings, oldSettings map[string]interface{}) {
	metadata := map[string]interface{}{
		"old_settings": oldSettings,
		"new_settings": newSettings,
	}

	activity := models.NewUserActivity(
		user.ID,
		models.ActivityTypeSettingsUpdate,
		models.CategoryUser,
		models.SeverityInfo,
		metadata,
	)

	activity.TelegramID = user.TelegramID
	activity.Username = user.Username
	activity.FirstName = user.FirstName

	go func() {
		if err := s.activityRepo.Create(activity); err != nil {
			log.Printf("Failed to log settings update: %v", err)
		}
	}()
}

func (s *Service) logUserLogout(user *models.User, ip, userAgent, reason string) {
	go func() {
		if err := s.activityRepo.LogUserLogout(user, ip, userAgent, reason); err != nil {
			log.Printf("Failed to log user logout: %v", err)
		}
	}()
}

func (s *Service) logSecurityEvent(user *models.User, eventType, description string, severity models.ActivitySeverity, ip, userAgent string, metadata map[string]interface{}) {
	go func() {
		if err := s.activityRepo.LogSecurityEvent(user, eventType, description, severity, ip, userAgent, metadata); err != nil {
			log.Printf("Failed to log security event: %v", err)
		}
	}()
}

func (s *Service) logSystemEvent(eventType, description string, metadata map[string]interface{}) {
	go func() {
		if err := s.activityRepo.LogSystemEvent(eventType, description, models.SeverityInfo, metadata); err != nil {
			log.Printf("Failed to log system event: %v", err)
		}
	}()
}

func (s *Service) logSessionActivity(session *models.Session, activityType, description string) {
	activity := &models.SessionActivity{
		SessionID:    session.ID,
		ActivityType: activityType,
		Details: map[string]interface{}{
			"description": description,
			"user_id":     session.UserID,
		},
	}

	if session.IPAddress != nil {
		activity.IPAddress = session.IPAddress
	}

	go func() {
		if err := s.sessionRepo.LogActivity(activity); err != nil {
			log.Printf("Failed to log session activity: %v", err)
		}
	}()
}

// generateUUID генерирует UUID (заглушка, в реальном коде нужно использовать github.com/google/uuid)
func generateUUID() string {
	// Временная реализация, в production использовать github.com/google/uuid
	return fmt.Sprintf("session_%d", time.Now().UnixNano())
}

// SaveTradingSession сохраняет торговую сессию
func (s *Service) SaveTradingSession(session *models.TradingSession) error {
	return s.tradingSessionRepo.Save(session)
}

// DeactivateTradingSession деактивирует все торговые сессии пользователя (все платформы)
func (s *Service) DeactivateTradingSession(userID int) error {
	return s.tradingSessionRepo.Deactivate(userID)
}

// DeactivateTradingSessionByPlatform деактивирует торговую сессию пользователя на указанной платформе
func (s *Service) DeactivateTradingSessionByPlatform(userID int, platform string) error {
	return s.tradingSessionRepo.DeactivateByPlatform(userID, platform)
}

// FindAllActiveTradingSessions возвращает все активные сессии
func (s *Service) FindAllActiveTradingSessions() ([]*models.TradingSession, error) {
	return s.tradingSessionRepo.FindAllActive()
}
