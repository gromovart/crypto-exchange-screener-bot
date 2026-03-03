// internal/delivery/max/notification_guard.go
package max

// maxNotifGuard — in-memory rate limiting guard для MAX-уведомлений.
// Логика идентична SymbolNotificationGuard из Telegram-доставки.
// Все методы вызываются с удержанным внешним мьютексом (maxRateLimiter.mu),
// поэтому собственной блокировки не содержат.

import (
	"fmt"
	"time"
)

// maxBypassRecord хранит запись об умном обходе rate limiting
type maxBypassRecord struct {
	Timestamp time.Time
	Price     float64
	Change    float64 // изменение в %
}

// maxNotifGuard реализует rate limiting для MAX-уведомлений.
// Ключ кэша: "userID:symbol:direction:signalMinutes:rateLimitMinutes"
type maxNotifGuard struct {
	cache map[string][]time.Time

	// Кэш умных обходов: ключ "userID:symbol:direction"
	bypassCache map[string][]maxBypassRecord

	limit             int           // базовый лимит уведомлений (по умолчанию 5)
	minPriceChange    float64       // минимальное изменение цены для обхода (%)
	maxBypasses       int           // максимум умных обходов за bypassPeriod
	bypassPeriod      time.Duration // период для подсчёта умных обходов
	minBypassInterval time.Duration // минимальный интервал между обходами
}

// newMaxNotifGuard создаёт guard с теми же параметрами, что и в Telegram
func newMaxNotifGuard() *maxNotifGuard {
	return &maxNotifGuard{
		cache:             make(map[string][]time.Time),
		bypassCache:       make(map[string][]maxBypassRecord),
		limit:             5,
		minPriceChange:    2.0,
		maxBypasses:       5,
		bypassPeriod:      30 * time.Minute,
		minBypassInterval: 1 * time.Minute,
	}
}

// check проверяет, разрешена ли отправка уведомления.
// Вызывается с удержанным внешним мьютексом.
func (g *maxNotifGuard) check(userID int64, symbol, direction string, signalPeriod, rateLimitPeriod time.Duration) bool {
	key := g.key(userID, symbol, direction, signalPeriod, rateLimitPeriod)
	now := time.Now()
	cutoff := now.Add(-rateLimitPeriod)

	// Фильтруем только актуальные записи
	var valid []time.Time
	for _, ts := range g.cache[key] {
		if ts.After(cutoff) {
			valid = append(valid, ts)
		}
	}

	if len(valid) == 0 {
		delete(g.cache, key)
		return true
	}
	g.cache[key] = valid

	// Общий лимит
	if len(valid) >= g.limit {
		return false
	}

	// Минимальный интервал между уведомлениями
	minInterval := rateLimitPeriod / time.Duration(g.limit)
	if now.Sub(valid[len(valid)-1]) < minInterval {
		return false
	}

	return true
}

// record регистрирует отправку уведомления.
// Вызывается с удержанным внешним мьютексом.
func (g *maxNotifGuard) record(userID int64, symbol, direction string, signalPeriod, rateLimitPeriod time.Duration) {
	key := g.key(userID, symbol, direction, signalPeriod, rateLimitPeriod)
	g.cache[key] = append(g.cache[key], time.Now())
	// Ограничиваем историю
	if len(g.cache[key]) > g.limit*3 {
		g.cache[key] = g.cache[key][len(g.cache[key])-g.limit*3:]
	}
}

// getCount возвращает текущее количество уведомлений за период.
// Вызывается с удержанным внешним мьютексом.
func (g *maxNotifGuard) getCount(userID int64, symbol, direction string, signalPeriod, rateLimitPeriod time.Duration) int {
	key := g.key(userID, symbol, direction, signalPeriod, rateLimitPeriod)
	cutoff := time.Now().Add(-rateLimitPeriod)
	count := 0
	for _, ts := range g.cache[key] {
		if ts.After(cutoff) {
			count++
		}
	}
	return count
}

// canBypassWithPrice проверяет возможность умного обхода rate limiting
// для сильных движений (>5%). Вызывается с удержанным внешним мьютексом.
func (g *maxNotifGuard) canBypassWithPrice(userID int64, symbol, direction string, currentPrice, currentChange float64) (bool, string) {
	bkey := g.bypassKey(userID, symbol, direction)
	records := g.bypassCache[bkey]

	if len(records) == 0 {
		return true, "первый обход для сильного движения"
	}

	last := records[len(records)-1]
	timeSinceLast := time.Since(last.Timestamp)

	// 1. Минимальный интервал
	if timeSinceLast < g.minBypassInterval {
		return false, fmt.Sprintf("слишком частый обход (прошло %v, нужно %v)",
			timeSinceLast.Round(time.Second), g.minBypassInterval)
	}

	// 2. Изменение цены от последнего обхода
	priceChange := g.priceChangePct(last.Price, currentPrice)
	if absFloat(priceChange) < g.minPriceChange {
		return false, fmt.Sprintf("цена изменилась всего на %.2f%% (нужно минимум %.1f%%)",
			absFloat(priceChange), g.minPriceChange)
	}

	// 3. В одном направлении — нужно большее изменение
	if (last.Change > 0 && currentChange > 0) || (last.Change < 0 && currentChange < 0) {
		required := absFloat(last.Change) + g.minPriceChange
		if absFloat(currentChange) < required {
			return false, fmt.Sprintf("в том же направлении нужен больший рост: %.2f%% < %.2f%%",
				absFloat(currentChange), required)
		}
	}

	// 4. Лимит умных обходов за период
	cutoff := time.Now().Add(-g.bypassPeriod)
	validCount := 0
	for _, r := range records {
		if r.Timestamp.After(cutoff) {
			validCount++
		}
	}
	if validCount >= g.maxBypasses {
		return false, fmt.Sprintf("достигнут лимит %d обходов за %v", g.maxBypasses, g.bypassPeriod)
	}

	return true, fmt.Sprintf("цена изменилась на %.2f%% от предыдущего обхода", absFloat(priceChange))
}

// recordSmartBypass регистрирует умный обход.
// Вызывается с удержанным внешним мьютексом.
func (g *maxNotifGuard) recordSmartBypass(userID int64, symbol, direction string, price, change float64) {
	bkey := g.bypassKey(userID, symbol, direction)
	g.bypassCache[bkey] = append(g.bypassCache[bkey], maxBypassRecord{
		Timestamp: time.Now(),
		Price:     price,
		Change:    change,
	})
	// Ограничиваем историю
	if len(g.bypassCache[bkey]) > 10 {
		g.bypassCache[bkey] = g.bypassCache[bkey][len(g.bypassCache[bkey])-10:]
	}
}

// cleanupOldEntries удаляет устаревшие записи из обоих кэшей.
// Вызывается с удержанным внешним мьютексом.
func (g *maxNotifGuard) cleanupOldEntries() {
	now := time.Now()

	for key, tss := range g.cache {
		// Извлекаем rateLimitMinutes из конца ключа
		var rateLimitMinutes int
		fmt.Sscanf(key, "%*d:%*s:%*s:%*d:%d", &rateLimitMinutes)
		cutoff := now.Add(-time.Duration(rateLimitMinutes) * time.Minute * 2)

		var valid []time.Time
		for _, ts := range tss {
			if ts.After(cutoff) {
				valid = append(valid, ts)
			}
		}
		if len(valid) == 0 {
			delete(g.cache, key)
		} else {
			g.cache[key] = valid
		}
	}

	cutoffBypass := now.Add(-g.bypassPeriod * 2)
	for bkey, records := range g.bypassCache {
		var valid []maxBypassRecord
		for _, r := range records {
			if r.Timestamp.After(cutoffBypass) {
				valid = append(valid, r)
			}
		}
		if len(valid) == 0 {
			delete(g.bypassCache, bkey)
		} else {
			g.bypassCache[bkey] = valid
		}
	}
}

// ──────────────────────────────────────────────
// Внутренние вспомогательные методы
// ──────────────────────────────────────────────

func (g *maxNotifGuard) key(userID int64, symbol, direction string, signalPeriod, rateLimitPeriod time.Duration) string {
	return fmt.Sprintf("%d:%s:%s:%d:%d",
		userID, symbol, direction,
		int(signalPeriod.Minutes()), int(rateLimitPeriod.Minutes()))
}

func (g *maxNotifGuard) bypassKey(userID int64, symbol, direction string) string {
	return fmt.Sprintf("%d:%s:%s", userID, symbol, direction)
}

func (g *maxNotifGuard) priceChangePct(prev, curr float64) float64 {
	if prev == 0 {
		return 0
	}
	return ((curr - prev) / prev) * 100
}

// absFloat возвращает модуль float64
func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
