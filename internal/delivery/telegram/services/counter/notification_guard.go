// internal/delivery/telegram/services/counter/notification_guard.go
package counter

import (
	"fmt"
	"sync"
	"time"
)

// SymbolNotificationGuard реализует rate limiting для уведомлений по символам, направлениям и периодам сигналов
// Ограничение: максимум limit уведомлений за rateLimitPeriod НА КАЖДЫЙ СИМВОЛ, НАПРАВЛЕНИЕ И ПЕРИОД СИГНАЛА
type SymbolNotificationGuard struct {
	mu sync.RWMutex
	// Ключ: "userID:symbol:direction:signalPeriodMinutes:rateLimitPeriodMinutes" → []time.Time
	cache map[string][]time.Time

	// Базовый лимит уведомлений (по умолчанию 5)
	limit int
}

// NewSymbolNotificationGuard создает новый guard
func NewSymbolNotificationGuard() *SymbolNotificationGuard {
	return &SymbolNotificationGuard{
		cache: make(map[string][]time.Time),
		limit: 5,
	}
}

// Check проверяет, можно ли отправить уведомление
// signalPeriod - период анализа сигнала (5m, 15m, etc.)
// rateLimitPeriod - период для rate limiting (из настроек пользователя)
func (g *SymbolNotificationGuard) Check(userID int64, symbol, direction string, signalPeriod, rateLimitPeriod time.Duration) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	key := g.generateKey(userID, symbol, direction, signalPeriod, rateLimitPeriod)
	timestamps, exists := g.cache[key]

	if !exists {
		return true // Нет истории - можно отправлять
	}

	// 1. Очищаем старые записи (старше rateLimitPeriod)
	now := time.Now()
	cutoffTime := now.Add(-rateLimitPeriod)

	var validTimestamps []time.Time
	validCount := 0

	for _, ts := range timestamps {
		if ts.After(cutoffTime) {
			validTimestamps = append(validTimestamps, ts)
			validCount++
		}
	}

	// Обновляем кэш
	if len(validTimestamps) == 0 {
		delete(g.cache, key)
		return true
	}
	g.cache[key] = validTimestamps

	// 2. Проверяем общий лимит
	if validCount >= g.limit {
		// Лимит достигнут
		return false
	}

	// 3. Вычисляем минимальный интервал между уведомлениями
	minInterval := rateLimitPeriod / time.Duration(g.limit)

	// Проверяем интервал с последним уведомлением
	lastTimestamp := validTimestamps[validCount-1]
	timeSinceLast := now.Sub(lastTimestamp)

	if timeSinceLast < minInterval {
		return false
	}

	return true
}

// Record регистрирует отправку уведомления
func (g *SymbolNotificationGuard) Record(userID int64, symbol, direction string, signalPeriod, rateLimitPeriod time.Duration) {
	g.mu.Lock()
	defer g.mu.Unlock()

	key := g.generateKey(userID, symbol, direction, signalPeriod, rateLimitPeriod)
	now := time.Now()

	g.cache[key] = append(g.cache[key], now)

	// Ограничиваем размер истории
	if len(g.cache[key]) > g.limit*3 {
		g.cache[key] = g.cache[key][len(g.cache[key])-g.limit*3:]
	}
}

// GetCount возвращает количество уведомлений
func (g *SymbolNotificationGuard) GetCount(userID int64, symbol, direction string, signalPeriod, rateLimitPeriod time.Duration) int {
	g.mu.RLock()
	defer g.mu.RUnlock()

	key := g.generateKey(userID, symbol, direction, signalPeriod, rateLimitPeriod)
	timestamps, exists := g.cache[key]
	if !exists {
		return 0
	}

	now := time.Now()
	cutoffTime := now.Add(-rateLimitPeriod)
	count := 0

	for _, ts := range timestamps {
		if ts.After(cutoffTime) {
			count++
		}
	}

	return count
}

// GetNextAllowedTime возвращает время следующего разрешенного уведомления
func (g *SymbolNotificationGuard) GetNextAllowedTime(userID int64, symbol, direction string, signalPeriod, rateLimitPeriod time.Duration) time.Time {
	g.mu.RLock()
	defer g.mu.RUnlock()

	key := g.generateKey(userID, symbol, direction, signalPeriod, rateLimitPeriod)
	timestamps, exists := g.cache[key]
	if !exists {
		return time.Now()
	}

	now := time.Now()
	cutoffTime := now.Add(-rateLimitPeriod)

	// Фильтруем только актуальные записи
	var validTimestamps []time.Time
	for _, ts := range timestamps {
		if ts.After(cutoffTime) {
			validTimestamps = append(validTimestamps, ts)
		}
	}

	if len(validTimestamps) == 0 {
		return now
	}

	// Вычисляем минимальный интервал
	minInterval := rateLimitPeriod / time.Duration(g.limit)

	// Берём время последнего уведомления
	lastTimestamp := validTimestamps[len(validTimestamps)-1]
	nextAllowed := lastTimestamp.Add(minInterval)

	if nextAllowed.Before(now) {
		return now
	}

	return nextAllowed
}

// generateKey генерирует ключ для кэша с учетом периода сигнала и периода rate limiting
func (g *SymbolNotificationGuard) generateKey(userID int64, symbol, direction string, signalPeriod, rateLimitPeriod time.Duration) string {
	signalMinutes := int(signalPeriod.Minutes())
	rateLimitMinutes := int(rateLimitPeriod.Minutes())
	return fmt.Sprintf("%d:%s:%s:%d:%d", userID, symbol, direction, signalMinutes, rateLimitMinutes)
}

// GetTimeUntilNextAllowed возвращает оставшееся время до возможности отправки
func (g *SymbolNotificationGuard) GetTimeUntilNextAllowed(userID int64, symbol, direction string, signalPeriod, rateLimitPeriod time.Duration) time.Duration {
	nextTime := g.GetNextAllowedTime(userID, symbol, direction, signalPeriod, rateLimitPeriod)
	now := time.Now()

	if nextTime.Before(now) {
		return 0
	}
	return nextTime.Sub(now)
}

// Clear удаляет все записи
func (g *SymbolNotificationGuard) Clear(userID int64, symbol, direction string, signalPeriod, rateLimitPeriod time.Duration) {
	g.mu.Lock()
	defer g.mu.Unlock()

	key := g.generateKey(userID, symbol, direction, signalPeriod, rateLimitPeriod)
	delete(g.cache, key)
}

// ClearUser удаляет все записи пользователя
func (g *SymbolNotificationGuard) ClearUser(userID int64) {
	g.mu.Lock()
	defer g.mu.Unlock()

	prefix := fmt.Sprintf("%d:", userID)
	for key := range g.cache {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			delete(g.cache, key)
		}
	}
}

// CleanupOldEntries удаляет старые записи
func (g *SymbolNotificationGuard) CleanupOldEntries() {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now()
	for key, timestamps := range g.cache {
		// Парсим rateLimitPeriod из ключа (последнее число)
		var rateLimitMinutes int
		// Ключ: "userID:symbol:direction:signalMinutes:rateLimitMinutes"
		fmt.Sscanf(key, "%*d:%*s:%*s:%*d:%d", &rateLimitMinutes)
		rateLimitPeriod := time.Duration(rateLimitMinutes) * time.Minute

		// Удаляем записи старше 2 периодов
		cutoffTime := now.Add(-rateLimitPeriod * 2)

		var validTimestamps []time.Time
		for _, ts := range timestamps {
			if ts.After(cutoffTime) {
				validTimestamps = append(validTimestamps, ts)
			}
		}

		if len(validTimestamps) == 0 {
			delete(g.cache, key)
		} else {
			g.cache[key] = validTimestamps
		}
	}
}

// GetLimit возвращает текущий лимит
func (g *SymbolNotificationGuard) GetLimit() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.limit
}

// SetLimit устанавливает новый лимит
func (g *SymbolNotificationGuard) SetLimit(limit int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if limit > 0 {
		g.limit = limit
	}
}
