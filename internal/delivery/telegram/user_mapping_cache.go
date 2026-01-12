// internal/delivery/telegram/user_mapping_cache.go
package telegram

import (
	"strconv"
	"sync"
	"time"
)

// UserMappingCache кэширует маппинг chatID -> userID
type UserMappingCache struct {
	mu             sync.RWMutex
	chatToUser     map[string]int       // chatID -> userID
	telegramToUser map[int]int          // telegramID -> userID
	lastUpdated    map[string]time.Time // Время последнего обновления
	ttl            time.Duration        // Время жизни кэша
}

// NewUserMappingCache создает новый кэш маппинга
func NewUserMappingCache(ttl time.Duration) *UserMappingCache {
	return &UserMappingCache{
		chatToUser:     make(map[string]int),
		telegramToUser: make(map[int]int),
		lastUpdated:    make(map[string]time.Time),
		ttl:            ttl,
	}
}

// GetUserID получает userID из кэша
func (c *UserMappingCache) GetUserID(chatID string) (int, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	userID, exists := c.chatToUser[chatID]
	if !exists {
		return 0, false
	}

	// Проверяем TTL
	if lastUpdated, ok := c.lastUpdated[chatID]; ok {
		if time.Since(lastUpdated) > c.ttl {
			delete(c.chatToUser, chatID)
			delete(c.lastUpdated, chatID)
			return 0, false
		}
	}

	return userID, true
}

// SetUserID устанавливает маппинг в кэш
func (c *UserMappingCache) SetUserID(chatID string, userID int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.chatToUser[chatID] = userID
	c.lastUpdated[chatID] = time.Now()

	// Также добавляем маппинг по telegramID если chatID числовой
	if telegramID, err := strconv.Atoi(chatID); err == nil {
		c.telegramToUser[telegramID] = userID
	}
}

// GetUserIDByTelegramID получает userID по telegramID
func (c *UserMappingCache) GetUserIDByTelegramID(telegramID int) (int, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	userID, exists := c.telegramToUser[telegramID]
	return userID, exists
}

// SetUserIDByTelegramID устанавливает маппинг по telegramID
func (c *UserMappingCache) SetUserIDByTelegramID(telegramID, userID int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.telegramToUser[telegramID] = userID
	c.lastUpdated[strconv.Itoa(telegramID)] = time.Now()
}

// Invalidate инвалидирует кэш для chatID
func (c *UserMappingCache) Invalidate(chatID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.chatToUser, chatID)
	delete(c.lastUpdated, chatID)

	if telegramID, err := strconv.Atoi(chatID); err == nil {
		delete(c.telegramToUser, telegramID)
	}
}

// Clear очищает весь кэш
func (c *UserMappingCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.chatToUser = make(map[string]int)
	c.telegramToUser = make(map[int]int)
	c.lastUpdated = make(map[string]time.Time)
}

// Size возвращает размер кэша
func (c *UserMappingCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.chatToUser)
}
