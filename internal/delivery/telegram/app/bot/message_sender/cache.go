package message_sender

import (
	"sync"
	"time"
)

// MessageCache кэш сообщений для предотвращения дубликатов
type MessageCache struct {
	cache    map[string]time.Time
	mu       sync.RWMutex
	cacheTTL time.Duration
}

// NewMessageCache создает новый кэш сообщений
func NewMessageCache(ttl time.Duration) *MessageCache {
	return &MessageCache{
		cache:    make(map[string]time.Time),
		cacheTTL: ttl,
	}
}

// IsDuplicate проверяет, является ли сообщение дубликатом
func (mc *MessageCache) IsDuplicate(hash string) bool {
	mc.mu.RLock()
	lastSent, exists := mc.cache[hash]
	mc.mu.RUnlock()

	if !exists {
		return false
	}

	// Проверяем TTL
	if time.Since(lastSent) > mc.cacheTTL {
		return false
	}

	// 30 секунд между одинаковыми сообщениями
	return time.Since(lastSent) < 30*time.Second
}

// Add добавляет сообщение в кэш
func (mc *MessageCache) Add(hash string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Очищаем старые записи
	now := time.Now()
	for key, timestamp := range mc.cache {
		if now.Sub(timestamp) > mc.cacheTTL {
			delete(mc.cache, key)
		}
	}

	mc.cache[hash] = now
}

// Clear очищает кэш
func (mc *MessageCache) Clear() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.cache = make(map[string]time.Time)
}
