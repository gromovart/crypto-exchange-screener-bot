// notification_guard.go
package counter

import (
	"fmt"
	"sync"
	"time"
)

// SymbolNotificationGuard реализует rate limiting для уведомлений по символам
// Ограничение: максимум limit уведомлений за период НА КАЖДЫЙ СИМВОЛ
type SymbolNotificationGuard struct {
	mu sync.RWMutex
	// Ключ: "userID:symbol:periodMinutes" → []time.Time
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

// Check проверяет, можно ли отправить уведомление для конкретного символа
// Логика: не чаще чем limit уведомлений за period для данного символа
func (g *SymbolNotificationGuard) Check(userID int64, symbol string, period time.Duration) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	key := g.generateKey(userID, symbol, period)
	timestamps, exists := g.cache[key]

	if !exists {
		return true // Нет истории по этому символу - можно отправлять
	}

	// 1. Очищаем старые записи (старше периода)
	now := time.Now()
	cutoffTime := now.Add(-period)

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

	// 2. Проверяем общий лимит (максимум limit уведомлений за период для этого символа)
	if validCount >= g.limit {
		// Лимит достигнут для этого символа
		return false
	}

	// 3. Вычисляем МИНИМАЛЬНЫЙ ИНТЕРВАЛ между уведомлениями для РАВНОМЕРНОСТИ
	// Например: период 5 минут, лимит 5 → минимальный интервал 1 минута
	minInterval := period / time.Duration(g.limit)

	// Если есть предыдущие уведомления, проверяем интервал с последним
	lastTimestamp := validTimestamps[validCount-1]
	timeSinceLast := now.Sub(lastTimestamp)

	// Если с последнего уведомления по ЭТОМУ СИМВОЛУ прошло меньше минимального интервала - блокируем
	if timeSinceLast < minInterval {
		return false
	}

	return true
}

// Record регистрирует отправку уведомления по символу
func (g *SymbolNotificationGuard) Record(userID int64, symbol string, period time.Duration) {
	g.mu.Lock()
	defer g.mu.Unlock()

	key := g.generateKey(userID, symbol, period)
	now := time.Now()

	g.cache[key] = append(g.cache[key], now)

	// Ограничиваем размер истории
	if len(g.cache[key]) > g.limit*3 {
		g.cache[key] = g.cache[key][len(g.cache[key])-g.limit*3:]
	}
}

// GetCount возвращает количество уведомлений за период для символа
func (g *SymbolNotificationGuard) GetCount(userID int64, symbol string, period time.Duration) int {
	g.mu.RLock()
	defer g.mu.RUnlock()

	key := g.generateKey(userID, symbol, period)
	timestamps, exists := g.cache[key]
	if !exists {
		return 0
	}

	now := time.Now()
	cutoffTime := now.Add(-period)
	count := 0

	for _, ts := range timestamps {
		if ts.After(cutoffTime) {
			count++
		}
	}

	return count
}

// GetNextAllowedTime возвращает время, когда можно будет отправить следующее уведомление по этому символу
func (g *SymbolNotificationGuard) GetNextAllowedTime(userID int64, symbol string, period time.Duration) time.Time {
	g.mu.RLock()
	defer g.mu.RUnlock()

	key := g.generateKey(userID, symbol, period)
	timestamps, exists := g.cache[key]
	if !exists {
		return time.Now()
	}

	now := time.Now()
	cutoffTime := now.Add(-period)

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
	minInterval := period / time.Duration(g.limit)

	// Берём время последнего уведомления
	lastTimestamp := validTimestamps[len(validTimestamps)-1]
	nextAllowed := lastTimestamp.Add(minInterval)

	// Если следующее разрешенное время уже прошло, можно отправлять сейчас
	if nextAllowed.Before(now) {
		return now
	}

	return nextAllowed
}

// generateKey генерирует ключ для кэша
func (g *SymbolNotificationGuard) generateKey(userID int64, symbol string, period time.Duration) string {
	periodMinutes := int(period.Minutes())
	return fmt.Sprintf("%d:%s:%d", userID, symbol, periodMinutes)
}

// GetTimeUntilNextAllowed возвращает оставшееся время до возможности отправки
func (g *SymbolNotificationGuard) GetTimeUntilNextAllowed(userID int64, symbol string, period time.Duration) time.Duration {
	nextTime := g.GetNextAllowedTime(userID, symbol, period)
	now := time.Now()

	if nextTime.Before(now) {
		return 0
	}
	return nextTime.Sub(now)
}

// Clear удаляет все записи для пользователя и символа
func (g *SymbolNotificationGuard) Clear(userID int64, symbol string, period time.Duration) {
	g.mu.Lock()
	defer g.mu.Unlock()

	key := g.generateKey(userID, symbol, period)
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

// CleanupOldEntries удаляет все старые записи для всех пользователей
func (g *SymbolNotificationGuard) CleanupOldEntries() {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now()
	for key, timestamps := range g.cache {
		// Парсим период из ключа
		var periodMinutes int
		fmt.Sscanf(key, "%*d:%*s:%d", &periodMinutes)
		period := time.Duration(periodMinutes) * time.Minute

		// Удаляем записи старше 2 периодов
		cutoffTime := now.Add(-period * 2)

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
