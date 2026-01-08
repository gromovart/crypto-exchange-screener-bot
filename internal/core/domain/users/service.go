// internal/core/domain/users/service.go
package users

import (
	"context"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
	activity_repo "crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/repository/activity"
	api_key_repo "crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/repository/api_key"
	session_repo "crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/repository/session"
	subscription_repo "crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/repository/subscription"
	user_repo "crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/repository/users"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// UserService предоставляет бизнес-логику для работы с пользователями
type UserService struct {
	userRepo         user_repo.UserRepository
	sessionRepo      session_repo.SessionRepository
	activityRepo     activity_repo.ActivityRepository
	subscriptionRepo subscription_repo.SubscriptionRepository
	apiKeyRepo       api_key_repo.APIKeyRepository
}

// NewUserService создает новый сервис пользователей
func NewUserService(
	userRepo user_repo.UserRepository,
	sessionRepo session_repo.SessionRepository,
	activityRepo activity_repo.ActivityRepository,
	subscriptionRepo subscription_repo.SubscriptionRepository,
	apiKeyRepo api_key_repo.APIKeyRepository,
) *UserService {
	return &UserService{
		userRepo:         userRepo,
		sessionRepo:      sessionRepo,
		activityRepo:     activityRepo,
		subscriptionRepo: subscriptionRepo,
		apiKeyRepo:       apiKeyRepo,
	}
}

// RegisterTelegramUser регистрирует пользователя через Telegram
func (s *UserService) RegisterTelegramUser(
	telegramID int64,
	username string,
	firstName string,
	lastName string,
	chatID string,
	email string,
	phone string,
	ipAddress string,
	userAgent string,
) (*models.User, *models.Session, error) {

	// Проверяем, не зарегистрирован ли уже пользователь
	existingUser, err := s.userRepo.FindByTelegramID(telegramID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to check existing user: %w", err)
	}

	if existingUser != nil {
		return nil, nil, fmt.Errorf("user with telegram ID %d already exists", telegramID)
	}

	// Проверяем по chat ID (на всякий случай)
	existingByChatID, err := s.userRepo.FindByChatID(chatID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to check existing user by chat ID: %w", err)
	}

	if existingByChatID != nil {
		return nil, nil, fmt.Errorf("user with chat ID %s already exists", chatID)
	}

	// Создаем нового пользователя с настройками по умолчанию
	user := models.NewUser(telegramID, username, firstName, lastName, chatID)
	user.Email = email
	user.Phone = phone

	// Сохраняем пользователя в базу данных
	if err := s.userRepo.Create(user); err != nil {
		return nil, nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Создаем сессию
	session, err := s.CreateSession(user.ID, ipAddress, userAgent)
	if err != nil {
		// Если не удалось создать сессию, все равно возвращаем пользователя
		// но без сессии
		return user, nil, fmt.Errorf("user created but session creation failed: %w", err)
	}

	// Логируем регистрацию
	s.logUserRegistration(user, ipAddress, userAgent)

	return user, session, nil
}

// LoginTelegramUser выполняет вход пользователя через Telegram
func (s *UserService) LoginTelegramUser(
	telegramID int64,
	chatID string,
	ipAddress string,
	userAgent string,
) (*models.User, *models.Session, error) {

	// Находим пользователя по telegramID
	user, err := s.userRepo.FindByTelegramID(telegramID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find user: %w", err)
	}

	if user == nil {
		return nil, nil, errors.New("user not found")
	}

	// Проверяем активность пользователя
	if !user.IsActive {
		return nil, nil, errors.New("user account is deactivated")
	}

	// Обновляем время последнего входа
	if err := s.userRepo.UpdateLastLogin(user.ID); err != nil {
		return nil, nil, fmt.Errorf("failed to update last login: %w", err)
	}

	// Создаем новую сессию
	session, err := s.CreateSession(user.ID, ipAddress, userAgent)
	if err != nil {
		return user, nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Логируем вход
	s.logUserLogin(user, ipAddress, userAgent, true, "")

	return user, session, nil
}

// CreateSession создает новую сессию для пользователя
func (s *UserService) CreateSession(userID int, ipAddress, userAgent string) (*models.Session, error) {
	session := &models.Session{
		ID:        uuid.New().String(),
		UserID:    userID,
		Token:     uuid.New().String(),
		IP:        ipAddress,
		UserAgent: userAgent,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour), // 30 дней
		CreatedAt: time.Now(),
		IsActive:  true,
	}

	// Используем метод Create из SessionRepository
	err := s.sessionRepo.Create(session)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, nil
}

// ValidateSession проверяет валидность сессии и возвращает пользователя
func (s *UserService) ValidateSession(token string) (*models.User, error) {
	// Находим сессию по токену
	session, err := s.sessionRepo.FindByToken(token)
	if err != nil {
		return nil, fmt.Errorf("failed to find session: %w", err)
	}

	if session == nil {
		return nil, errors.New("session not found")
	}

	// Проверяем, активна ли сессия
	if !session.IsActive {
		return nil, errors.New("session is not active")
	}

	// Проверяем, не истекла ли сессия
	if time.Now().After(session.ExpiresAt) {
		return nil, errors.New("session has expired")
	}

	// Обновляем время последней активности
	if err := s.sessionRepo.UpdateActivity(session.ID); err != nil {
		// Не прерываем из-за ошибки обновления активности
		fmt.Printf("Failed to update session activity: %v\n", err)
	}

	// Возвращаем пользователя
	return session.User, nil
}

// GetUserProfile возвращает профиль пользователя
func (s *UserService) GetUserProfile(userID int) (*models.UserProfile, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	if user == nil {
		return nil, errors.New("user not found")
	}

	return user.ToProfile(), nil
}

// UpdateUserProfile обновляет профиль пользователя
func (s *UserService) UpdateUserProfile(userID int, req models.UpdateProfileRequest) (*models.User, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	if user == nil {
		return nil, errors.New("user not found")
	}

	// Обновляем поля пользователя
	if req.Username != "" {
		user.Username = req.Username
	}
	if req.FirstName != "" {
		user.FirstName = req.FirstName
	}
	if req.LastName != "" {
		user.LastName = req.LastName
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Phone != "" {
		user.Phone = req.Phone
	}
	if req.Language != "" {
		user.Language = req.Language // Исправлено: user.Language
	}
	if req.Timezone != "" {
		user.Timezone = req.Timezone // Исправлено: user.Timezone
	}
	if req.DisplayMode != "" {
		user.DisplayMode = req.DisplayMode // Исправлено: user.DisplayMode
	}

	user.UpdatedAt = time.Now()

	// Сохраняем изменения
	if err := s.userRepo.Update(user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return user, nil
}

// UpdateNotificationSettings обновляет настройки уведомлений
func (s *UserService) UpdateNotificationSettings(
	userID int,
	notificationsEnabled bool,
	notifyGrowth bool,
	notifyFall bool,
	notifyContinuous bool,
	quietHoursStart int,
	quietHoursEnd int,
) (*models.User, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	if user == nil {
		return nil, errors.New("user not found")
	}

	// Обновляем настройки (плоские поля)
	user.NotificationsEnabled = notificationsEnabled
	user.NotifyGrowth = notifyGrowth
	user.NotifyFall = notifyFall
	user.NotifyContinuous = notifyContinuous
	user.QuietHoursStart = quietHoursStart
	user.QuietHoursEnd = quietHoursEnd
	user.UpdatedAt = time.Now()

	// Сохраняем изменения
	if err := s.userRepo.Update(user); err != nil {
		return nil, fmt.Errorf("failed to update notification settings: %w", err)
	}

	return user, nil
}

// UpdateUserSettings обновляет настройки пользователя
func (s *UserService) UpdateUserSettings(
	userID int,
	minGrowthThreshold float64,
	minFallThreshold float64,
	preferredPeriods []int,
	minVolumeFilter float64,
	excludePatterns []string,
	language string,
	timezone string,
	displayMode string,
) (*models.User, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	if user == nil {
		return nil, errors.New("user not found")
	}

	// Обновляем настройки (плоские поля)
	user.MinGrowthThreshold = minGrowthThreshold
	user.MinFallThreshold = minFallThreshold
	user.PreferredPeriods = preferredPeriods
	user.MinVolumeFilter = minVolumeFilter
	user.ExcludePatterns = excludePatterns
	if language != "" {
		user.Language = language
	}
	if timezone != "" {
		user.Timezone = timezone
	}
	if displayMode != "" {
		user.DisplayMode = displayMode
	}
	user.UpdatedAt = time.Now()

	// Сохраняем изменения
	if err := s.userRepo.Update(user); err != nil {
		return nil, fmt.Errorf("failed to update user settings: %w", err)
	}

	return user, nil
}

// Logout завершает сессию пользователя
func (s *UserService) Logout(sessionToken, reason string) error {
	// Находим сессию
	session, err := s.sessionRepo.FindByToken(sessionToken)
	if err != nil {
		return fmt.Errorf("failed to find session: %w", err)
	}

	if session == nil {
		return errors.New("session not found")
	}

	// Отзываем сессию
	if err := s.sessionRepo.Revoke(session.ID, reason); err != nil {
		return fmt.Errorf("failed to revoke session: %w", err)
	}

	// Логируем выход
	if session.User != nil {
		s.logUserLogout(session.User, session.IP, session.UserAgent, reason)
	}

	return nil
}

// LogoutAll завершает все сессии пользователя
func (s *UserService) LogoutAll(userID int, reason string) (int, error) {
	count, err := s.sessionRepo.RevokeAllUserSessions(userID, reason)
	if err != nil {
		return 0, fmt.Errorf("failed to revoke all sessions: %w", err)
	}

	// Логируем массовый выход
	if user, err := s.userRepo.FindByID(userID); err == nil && user != nil {
		s.activityRepo.LogUserLogout(user, "", "", fmt.Sprintf("all_sessions_%s", reason))
	}

	return count, nil
}

// CheckSignalPermission проверяет, может ли пользователь получить сигнал
func (s *UserService) CheckSignalPermission(userID int, signalType string, changePercent float64) (bool, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return false, fmt.Errorf("failed to find user: %w", err)
	}

	if user == nil {
		return false, errors.New("user not found")
	}

	// Проверяем все условия
	if !user.ShouldReceiveSignal(signalType, changePercent) {
		return false, nil
	}

	return true, nil
}

// IncrementSignalsCount увеличивает счетчик сигналов пользователя
func (s *UserService) IncrementSignalsCount(userID int) error {
	return s.userRepo.IncrementSignalsCount(userID)
}

// GetUserStats возвращает статистику пользователя
func (s *UserService) GetUserStats(userID int) (map[string]interface{}, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	if user == nil {
		return nil, errors.New("user not found")
	}

	stats := map[string]interface{}{
		"user_id":               user.ID,
		"signals_today":         user.SignalsToday,
		"max_signals_per_day":   user.MaxSignalsPerDay,
		"signals_remaining":     user.MaxSignalsPerDay - user.SignalsToday,
		"subscription_tier":     user.SubscriptionTier,
		"notifications_enabled": user.NotificationsEnabled, // Исправлено: user.NotificationsEnabled
		"in_quiet_hours":        user.IsInQuietHours(),
		"last_login":            user.LastLoginAt,
		"created_at":            user.CreatedAt,
	}

	// Получаем статистику активности
	if activityStats, err := s.activityRepo.GetUserActivityStats(userID, 7); err == nil {
		stats["activity_stats"] = activityStats
	}

	return stats, nil
}

// GetActiveSubscription возвращает активную подписку пользователя
func (s *UserService) GetActiveSubscription(userID int) (*models.UserSubscription, error) {
	return s.subscriptionRepo.GetActiveSubscription(userID)
}

// Проверяем, является ли пользователь администратором
func (s *UserService) IsAdmin(userID int) (bool, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return false, fmt.Errorf("failed to find user: %w", err)
	}

	if user == nil {
		return false, errors.New("user not found")
	}

	return user.IsAdmin(), nil
}

// Проверяем, имеет ли пользователь премиум-статус
func (s *UserService) IsPremium(userID int) (bool, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return false, fmt.Errorf("failed to find user: %w", err)
	}

	if user == nil {
		return false, errors.New("user not found")
	}

	return user.IsPremium(), nil
}

// Вспомогательные методы для логирования

func (s *UserService) logUserRegistration(user *models.User, ip, userAgent string) {
	if s.activityRepo != nil {
		// Используем готовый метод
		s.activityRepo.LogUserLogin(user, ip, userAgent, true, "registration")
	}
}

func (s *UserService) logUserLogin(user *models.User, ip, userAgent string, success bool, failureReason string) {
	if s.activityRepo != nil {
		s.activityRepo.LogUserLogin(user, ip, userAgent, success, failureReason)
	}
}

func (s *UserService) logUserLogout(user *models.User, ip, userAgent, reason string) {
	if s.activityRepo != nil {
		s.activityRepo.LogUserLogout(user, ip, userAgent, reason)
	}
}

// ResetDailyCounters сбрасывает дневные счетчики всех пользователей
func (s *UserService) ResetDailyCounters(ctx context.Context) error {
	return s.userRepo.ResetDailyCounters(ctx)
}
