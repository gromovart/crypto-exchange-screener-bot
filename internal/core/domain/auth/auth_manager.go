// internal/auth/auth_manager.go
package auth

// import (
// 	"context"
// 	"crypto/rand"
// 	"encoding/base64"
// 	"encoding/json"
// 	"errors"
// 	"fmt"
// 	"log"
// 	"strconv"
// 	"time"

// 	"crypto-exchange-screener-bot/internal/telegram"
// 	"crypto-exchange-screener-bot/internal/users"
// 	"crypto-exchange-screener-bot/persistence/postgres/repository"

// 	"github.com/go-redis/redis/v8"
// )

// var (
// 	ErrUserNotFound = errors.New("user not found")
// 	ErrInvalidAuth  = errors.New("invalid authentication")
// 	ErrUserInactive = errors.New("user account is inactive")
// 	ErrRateLimit    = errors.New("rate limit exceeded")
// )

// type AuthManager struct {
// 	userRepo    repository.UserRepository
// 	sessionRepo repository.SessionRepository
// 	jwtService  *JWTService
// 	cache       *redis.Client
// }

// func NewAuthManager(
// 	userRepo repository.UserRepository,
// 	sessionRepo repository.SessionRepository,
// 	jwtSecret string,
// ) *AuthManager {

// 	return &AuthManager{
// 		userRepo:    userRepo,
// 		sessionRepo: sessionRepo,
// 		jwtService:  NewJWTService(jwtSecret),
// 		cache: redis.NewClient(&redis.Options{
// 			Addr:     "localhost:6379",
// 			Password: "",
// 			DB:       0,
// 		}),
// 	}
// }

// // Регистрация через Telegram
// func (am *AuthManager) RegisterTelegramUser(
// 	telegramID int64,
// 	username,
// 	firstName,
// 	lastName,
// 	chatID string,
// ) (*users.User, error) {

// 	// Проверяем существующего пользователя
// 	existing, err := am.userRepo.FindByTelegramID(telegramID)
// 	if err == nil && existing != nil {
// 		// Обновляем последний логин
// 		am.UpdateLastLogin(existing.ID)
// 		return existing, nil
// 	}

// 	// Создаем нового пользователя
// 	user := &users.User{
// 		TelegramID: telegramID,
// 		Username:   username,
// 		FirstName:  firstName,
// 		LastName:   lastName,
// 		ChatID:     chatID,
// 		Email:      "",
// 		Phone:      "",
// 		Role:       users.RoleUser,
// 		IsActive:   true,
// 		IsVerified: true, // Telegram уже верифицировал
// 		CreatedAt:  time.Now(),
// 		UpdatedAt:  time.Now(),
// 		Settings: users.UserSettings{
// 			MinGrowthThreshold: 2.0,
// 			MinFallThreshold:   2.0,
// 			PreferredPeriods:   []int{5, 15, 30},
// 			MinVolumeFilter:    0.0,
// 			ExcludePatterns:    []string{},
// 			Language:           "ru",
// 			Timezone:           "Europe/Moscow",
// 			DisplayMode:        "compact",
// 		},
// 		Notifications: users.NotificationSettings{
// 			Enabled:    true,
// 			Growth:     true,
// 			Fall:       true,
// 			Continuous: true,
// 		},
// 	}

// 	// Сохраняем в базу
// 	if err := am.userRepo.Create(user); err != nil {
// 		return nil, err
// 	}

// 	// Создаем сессию
// 	session := &users.Session{
// 		UserID:    user.ID,
// 		Token:     am.GenerateSessionToken(),
// 		IP:        "telegram",
// 		UserAgent: "telegram-bot",
// 		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
// 	}

// 	if err := am.sessionRepo.Create(session); err != nil {
// 		return nil, err
// 	}

// 	// Кэшируем пользователя
// 	am.CacheUser(user)

// 	// Логируем регистрацию
// 	am.LogActivity(user.ID, "user_registered", map[string]interface{}{
// 		"source":   "telegram",
// 		"username": username,
// 	})

// 	return user, nil
// }

// // Middleware для Telegram бота
// func (am *AuthManager) TelegramAuthMiddleware(next telegram.HandlerFunc) telegram.HandlerFunc {
// 	return func(ctx *telegram.Context) error {
// 		sender := ctx.Sender()
// 		chatID := strconv.FormatInt(ctx.ChatID(), 10)

// 		// Ищем пользователя
// 		user, err := am.userRepo.FindByTelegramID(sender.ID)
// 		if err != nil {
// 			// Создаем нового при первом обращении
// 			user, err = am.RegisterTelegramUser(
// 				sender.ID,
// 				sender.UserName,
// 				sender.FirstName,
// 				sender.LastName,
// 				chatID,
// 			)
// 			if err != nil {
// 				log.Printf("Failed to register user: %v", err)
// 				return ctx.Reply("❌ Ошибка авторизации. Попробуйте позже.")
// 			}
// 		}

// 		// Проверяем активность
// 		if !user.IsActive {
// 			return ctx.Reply("⚠️ Ваш аккаунт деактивирован. Обратитесь к администратору.")
// 		}

// 		// Обновляем время последнего входа
// 		am.UpdateLastLogin(user.ID)

// 		// Добавляем пользователя в контекст
// 		ctx.Set("user", user)
// 		ctx.Set("user_id", user.ID)
// 		ctx.Set("notifications_enabled", user.Notifications.Enabled)

// 		return next(ctx)
// 	}
// }

// // Проверка лимитов для пользователя
// func (am *AuthManager) CheckRateLimit(userID int, limitType string) (bool, error) {
// 	key := fmt.Sprintf("ratelimit:%s:%d", limitType, userID)

// 	// Используем Redis для rate limiting
// 	val, err := am.cache.Incr(context.Background(), key).Result()
// 	if err != nil {
// 		return true, nil // При ошибке Redis пропускаем проверку
// 	}

// 	if val == 1 {
// 		// Устанавливаем TTL
// 		var ttl time.Duration
// 		switch limitType {
// 		case "signals_per_minute":
// 			ttl = time.Minute
// 		case "commands_per_hour":
// 			ttl = time.Hour
// 		default:
// 			ttl = time.Hour
// 		}
// 		am.cache.Expire(context.Background(), key, ttl)
// 	}

// 	// Проверяем лимиты
// 	switch limitType {
// 	case "signals_per_minute":
// 		return val <= 10, nil // 10 сигналов в минуту
// 	case "commands_per_hour":
// 		return val <= 60, nil // 60 команд в час
// 	default:
// 		return true, nil
// 	}
// }

// // Генерация токена сессии
// func (am *AuthManager) GenerateSessionToken() string {
// 	b := make([]byte, 32)
// 	rand.Read(b)
// 	return base64.URLEncoding.EncodeToString(b)
// }

// // Кэширование пользователя
// func (am *AuthManager) CacheUser(user *users.User) error {
// 	key := fmt.Sprintf("user:%d", user.ID)
// 	data, err := json.Marshal(user)
// 	if err != nil {
// 		return err
// 	}

// 	return am.cache.Set(context.Background(), key, data, 15*time.Minute).Err()
// }

// // Вспомогательные методы

// func (am *AuthManager) UpdateLastLogin(userID int) error {
// 	// Здесь должна быть логика обновления last_login в базе данных
// 	// Пока просто логируем
// 	log.Printf("User %d logged in", userID)
// 	return nil
// }

// func (am *AuthManager) LogActivity(userID int, activityType string, details map[string]interface{}) {
// 	// Здесь должна быть логика записи активности
// 	log.Printf("Activity: user=%d, type=%s, details=%v", userID, activityType, details)
// }
