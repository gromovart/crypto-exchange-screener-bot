// internal/delivery/telegram/services/counter/notification_guard.go
package counter

import (
	"fmt"
	"sync"
	"time"
)

// BypassRecord запись об обходе с ценой
type BypassRecord struct {
	Timestamp time.Time
	Price     float64
	Change    float64 // Изменение в %
}

// SymbolNotificationGuard реализует rate limiting для уведомлений по символам, направлениям и периодам сигналов
// Ограничение: максимум limit уведомлений за rateLimitPeriod НА КАЖДЫЙ СИМВОЛ, НАПРАВЛЕНИЕ И ПЕРИОД СИГНАЛА
type SymbolNotificationGuard struct {
	mu sync.RWMutex
	// Ключ: "userID:symbol:direction:signalPeriodMinutes:rateLimitPeriodMinutes" → []time.Time
	cache map[string][]time.Time

	// ⭐ УМНЫЕ ОБХОДЫ: храним цену и время
	smartBypassCache  map[string][]BypassRecord // Ключ: "userID:symbol:direction"
	minPriceChange    float64                   // Минимальное изменение цены для нового обхода (%)
	maxBypasses       int                       // Максимум обходов за период
	bypassPeriod      time.Duration             // Период для обходов
	minBypassInterval time.Duration             // Минимальный интервал между обходами

	// Базовый лимит уведомлений (по умолчанию 5)
	limit int
}

// NewSymbolNotificationGuard создает новый guard с умными обходами
func NewSymbolNotificationGuard() *SymbolNotificationGuard {
	return &SymbolNotificationGuard{
		cache:             make(map[string][]time.Time),
		smartBypassCache:  make(map[string][]BypassRecord),
		minPriceChange:    2.0,              // ⭐ Минимум 2% изменения цены для нового обхода
		maxBypasses:       5,                // ⭐ Максимум 5 обходов
		bypassPeriod:      30 * time.Minute, // ⭐ За 30 минут
		minBypassInterval: 1 * time.Minute,  // ⭐ Минимум 1 минута между обходами
		limit:             5,
	}
}

// CanBypassWithPrice проверяет можно ли обойти rate limiting с учетом изменения цены
func (g *SymbolNotificationGuard) CanBypassWithPrice(userID int64, symbol, direction string, currentPrice, currentChange float64) (bool, string) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	key := g.generateBypassKey(userID, symbol, direction)
	records := g.smartBypassCache[key]

	if len(records) == 0 {
		// ⭐ Первый обход - всегда разрешаем для сильных движений
		return true, "первый обход для сильного движения"
	}

	// Берём последний обход
	lastRecord := records[len(records)-1]
	timeSinceLast := time.Since(lastRecord.Timestamp)

	// 1. Проверяем минимальный временной интервал
	if timeSinceLast < g.minBypassInterval {
		return false, fmt.Sprintf("слишком частый обход (прошло %v, нужно %v)",
			timeSinceLast.Round(time.Second), g.minBypassInterval)
	}

	// 2. ⭐ ПРОВЕРЯЕМ ИЗМЕНЕНИЕ ЦЕНЫ ОТ ПОСЛЕДНЕГО ОБХОДА
	// Вычисляем изменение от цены последнего обхода
	priceChangeFromLast := g.calculatePriceChange(lastRecord.Price, currentPrice)
	absPriceChange := g.absFloat(priceChangeFromLast)

	// 3. Проверяем достаточно ли изменилась цена
	if absPriceChange < g.minPriceChange {
		return false, fmt.Sprintf("цена изменилась всего на %.2f%% (нужно минимум %.1f%%)",
			absPriceChange, g.minPriceChange)
	}

	// 4. ⭐ ПРОВЕРЯЕМ НАПРАВЛЕНИЕ ИЗМЕНЕНИЯ
	// Если предыдущий обход был на рост, а сейчас падение - разрешаем
	// Если оба в одном направлении - проверяем значимость изменения
	if (lastRecord.Change > 0 && currentChange > 0) ||
		(lastRecord.Change < 0 && currentChange < 0) {
		// Одно направление - нужно большее изменение
		requiredChange := g.absFloat(lastRecord.Change) + g.minPriceChange
		if g.absFloat(currentChange) < requiredChange {
			return false, fmt.Sprintf("в том же направлении нужен больший рост: %.2f%% < %.2f%%",
				g.absFloat(currentChange), requiredChange)
		}
	}

	// 5. Лимит обходов за период
	cutoffTime := time.Now().Add(-g.bypassPeriod)
	validCount := 0
	for _, record := range records {
		if record.Timestamp.After(cutoffTime) {
			validCount++
		}
	}

	if validCount >= g.maxBypasses {
		return false, fmt.Sprintf("достигнут лимит %d обходов за %v", g.maxBypasses, g.bypassPeriod)
	}

	return true, fmt.Sprintf("цена изменилась на %.2f%% от предыдущего обхода", absPriceChange)
}

// RecordSmartBypass регистрирует умный обход с ценой
func (g *SymbolNotificationGuard) RecordSmartBypass(userID int64, symbol, direction string, price, change float64) {
	g.mu.Lock()
	defer g.mu.Unlock()

	key := g.generateBypassKey(userID, symbol, direction)
	record := BypassRecord{
		Timestamp: time.Now(),
		Price:     price,
		Change:    change,
	}

	g.smartBypassCache[key] = append(g.smartBypassCache[key], record)

	// Ограничиваем историю
	if len(g.smartBypassCache[key]) > 10 {
		g.smartBypassCache[key] = g.smartBypassCache[key][len(g.smartBypassCache[key])-10:]
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

// generateBypassKey генерирует ключ для кэша обходов (без периодов)
func (g *SymbolNotificationGuard) generateBypassKey(userID int64, symbol, direction string) string {
	return fmt.Sprintf("%d:%s:%s", userID, symbol, direction)
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

	// ⭐ Также очищаем обходы
	for key := range g.smartBypassCache {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			delete(g.smartBypassCache, key)
		}
	}
}

// CleanupOldEntries удаляет старые записи
func (g *SymbolNotificationGuard) CleanupOldEntries() {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now()

	// Очищаем обычный кэш
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

	// ⭐ Очищаем кэш обходов
	for key, records := range g.smartBypassCache {
		cutoffTime := now.Add(-g.bypassPeriod * 2)

		var validRecords []BypassRecord
		for _, record := range records {
			if record.Timestamp.After(cutoffTime) {
				validRecords = append(validRecords, record)
			}
		}

		if len(validRecords) == 0 {
			delete(g.smartBypassCache, key)
		} else {
			g.smartBypassCache[key] = validRecords
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

// GetBypassConfig возвращает конфигурацию обходов
func (g *SymbolNotificationGuard) GetBypassConfig() (float64, int, time.Duration, time.Duration) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.minPriceChange, g.maxBypasses, g.bypassPeriod, g.minBypassInterval
}

// SetBypassConfig устанавливает конфигурацию обходов
func (g *SymbolNotificationGuard) SetBypassConfig(minPriceChange float64, maxBypasses int, bypassPeriod, minInterval time.Duration) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if minPriceChange > 0 {
		g.minPriceChange = minPriceChange
	}
	if maxBypasses > 0 {
		g.maxBypasses = maxBypasses
	}
	if bypassPeriod > 0 {
		g.bypassPeriod = bypassPeriod
	}
	if minInterval > 0 {
		g.minBypassInterval = minInterval
	}
}

// calculatePriceChange рассчитывает изменение цены в процентах
func (g *SymbolNotificationGuard) calculatePriceChange(previousPrice, currentPrice float64) float64 {
	if previousPrice == 0 {
		return 0
	}
	return ((currentPrice - previousPrice) / previousPrice) * 100
}

// absFloat возвращает абсолютное значение float64
func (g *SymbolNotificationGuard) absFloat(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}
