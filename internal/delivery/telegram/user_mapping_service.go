// internal/delivery/telegram/user_mapping_service.go
package telegram

import (
	"crypto-exchange-screener-bot/internal/core/domain/users"
	"log"
	"strconv"
	"time"
)

// UserMappingService предоставляет методы для маппинга chatID -> userID
type UserMappingService struct {
	userService *users.Service
	userCache   *UserMappingCache
}

// NewUserMappingService создает новый сервис маппинга
func NewUserMappingService(userService *users.Service) *UserMappingService {
	return &UserMappingService{
		userService: userService,
		userCache:   NewUserMappingCache(30 * time.Minute),
	}
}

// GetUserID получает userID из chatID с использованием кэша
func (ums *UserMappingService) GetUserID(chatID string) int {
	// Проверяем кэш
	if userID, found := ums.userCache.GetUserID(chatID); found {
		log.Printf("✅ Найден userID %d в кэше для chatID %s", userID, chatID)
		return userID
	}

	// Если нет сервиса пользователей, возвращаем 0
	if ums.userService == nil {
		log.Printf("ℹ️ UserService не доступен для chatID %s", chatID)
		return 0
	}

	var userID int

	// Пытаемся преобразовать chatID в int64 (Telegram chat ID обычно числовой)
	telegramID, err := strconv.ParseInt(chatID, 10, 64)
	if err != nil {
		// Если chatID не числовой, ищем по строковому chatID
		userID = ums.findUserIDByStringChatID(chatID)
	} else {
		// Ищем пользователя по telegramID
		user, err := ums.userService.GetUserByTelegramID(telegramID)
		if err != nil {
			log.Printf("⚠️ Ошибка поиска пользователя по telegramID %d: %v", telegramID, err)
			return 0
		}

		if user != nil {
			userID = user.ID
		}
	}

	// Кэшируем результат (даже если не нашли - кэшируем 0)
	ums.userCache.SetUserID(chatID, userID)

	if userID > 0 {
		log.Printf("✅ Найден пользователь ID %d для chatID %s (закэширован)", userID, chatID)
	} else {
		log.Printf("ℹ️ Пользователь не найден для chatID %s", chatID)
	}

	return userID
}

// findUserIDByStringChatID ищет пользователя по строковому chatID
func (ums *UserMappingService) findUserIDByStringChatID(chatID string) int {
	if ums.userService == nil {
		return 0
	}

	// Используем большой лимит для поиска всех пользователей
	users, err := ums.userService.GetAllUsers(1000, 0)
	if err != nil {
		log.Printf("⚠️ Ошибка получения пользователей: %v", err)
		return 0
	}

	for _, user := range users {
		if user.ChatID == chatID {
			return user.ID
		}
	}

	return 0
}

// InvalidateCache инвалидирует кэш для chatID
func (ums *UserMappingService) InvalidateCache(chatID string) {
	if ums.userCache != nil {
		ums.userCache.Invalidate(chatID)
		log.Printf("🔄 Кэш пользователя инвалидирован для chatID %s", chatID)
	}
}

// ClearCache очищает весь кэш
func (ums *UserMappingService) ClearCache() {
	if ums.userCache != nil {
		ums.userCache.Clear()
		log.Printf("🔄 Весь кэш пользователей очищен")
	}
}

// GetCacheSize возвращает размер кэша
func (ums *UserMappingService) GetCacheSize() int {
	if ums.userCache != nil {
		return ums.userCache.Size()
	}
	return 0
}
