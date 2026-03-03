// internal/delivery/max/rate_limiter.go
package max

// maxRateLimiter реализует логику rate limiting для MAX-уведомлений.
// Правила идентичны Telegram (см. internal/delivery/telegram/services/counter/service.go):
//   - Лимиты по периоду: 5м→3, 15м→4, 30м→5, 60м→6, 240м→8, 1440м→10
//   - Сигналы падения получают +20% к лимиту
//   - Сильные движения (>5%) могут обходить rate limiting (умный обход по цене)
//   - Период rate limiting = максимальный из PreferredPeriods пользователя (clamp 5м–1ч)

import (
	"sync"
	"time"

	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
	"crypto-exchange-screener-bot/pkg/logger"
	periodPkg "crypto-exchange-screener-bot/pkg/period"
)

const (
	// rateLimitFallBoost — на сколько увеличивается лимит для сигналов падения
	rateLimitFallBoost = 1.2
	// strongMoveThresholdPct — порог "сильного движения" для умного обхода (%)
	strongMoveThresholdPct = 5.0
	// rateLimitCleanupEvery — раз в N успешных отправок запускать очистку кэша
	rateLimitCleanupEvery = 200
)

// rlResult — результат проверки rate limiting
type rlResult struct {
	Allowed         bool
	SignalPeriod    time.Duration
	RateLimitPeriod time.Duration
	CurrentCount    int
	Limit           int
}

// maxRateLimiter потокобезопасный rate limiter для MAX-уведомлений
type maxRateLimiter struct {
	mu        sync.Mutex
	guard     *maxNotifGuard
	sendCount int // счётчик успешных отправок для периодической очистки
}

// newMaxRateLimiter создаёт rate limiter
func newMaxRateLimiter() *maxRateLimiter {
	return &maxRateLimiter{
		guard: newMaxNotifGuard(),
	}
}

// check проверяет, разрешено ли отправить уведомление пользователю.
// Если движение сильное и умный обход разрешён — записывает bypass сразу.
func (rl *maxRateLimiter) check(user *models.User, data map[string]interface{}) rlResult {
	symbol := getString(data, "symbol")
	direction := getString(data, "direction")
	changePercent := getFloat64(data, "change_percent")
	currentPrice := getFloat64(data, "current_price")
	periodStr := getString(data, "period")

	// Период сигнала из события
	signalPeriod, err := periodPkg.StringToDuration(periodStr)
	if err != nil {
		signalPeriod = periodPkg.DefaultDuration
	}

	// Период rate limiting из настроек пользователя
	rateLimitPeriod := rl.rateLimitPeriod(user)
	rateLimitMinutes := int(rateLimitPeriod.Minutes())

	// Лимит с учётом символа и направления
	limit := rl.symbolLimit(symbol, rateLimitMinutes, direction)
	userID := int64(user.ID)

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// ─── Умный обход для сильных движений ───────────────────────────────────
	if isStrongMove(changePercent) {
		canBypass, reason := rl.guard.canBypassWithPrice(userID, symbol, direction, currentPrice, changePercent)
		if canBypass {
			logger.Info("⚡ MAX Rate: умный обход для %s %s: %.2f%% (причина: %s)",
				symbol, direction, changePercent, reason)
			rl.guard.recordSmartBypass(userID, symbol, direction, currentPrice, changePercent)
			return rlResult{
				Allowed: true, SignalPeriod: signalPeriod,
				RateLimitPeriod: rateLimitPeriod, Limit: limit,
			}
		}
		logger.Debug("⏸️ MAX Rate: отказ в обходе %s %s %.2f%%: %s",
			symbol, direction, changePercent, reason)
		// продолжаем к обычному rate limiting
	}

	// ─── Обычный rate limiting ───────────────────────────────────────────────
	count := rl.guard.getCount(userID, symbol, direction, signalPeriod, rateLimitPeriod)
	allowed := rl.guard.check(userID, symbol, direction, signalPeriod, rateLimitPeriod)

	if !allowed {
		logger.Debug("⏸️ MAX Rate: user=%d %s %s count=%d/%d signal=%v rl=%v",
			user.ID, symbol, direction, count, limit, signalPeriod, rateLimitPeriod)
	}

	return rlResult{
		Allowed:         allowed,
		SignalPeriod:    signalPeriod,
		RateLimitPeriod: rateLimitPeriod,
		CurrentCount:    count,
		Limit:           limit,
	}
}

// record регистрирует успешную отправку уведомления в нормальный кэш.
// Вызывается только после реальной отправки.
func (rl *maxRateLimiter) record(userID int64, symbol, direction string, signalPeriod, rateLimitPeriod time.Duration) {
	rl.mu.Lock()
	rl.guard.record(userID, symbol, direction, signalPeriod, rateLimitPeriod)
	rl.sendCount++
	needCleanup := rl.sendCount%rateLimitCleanupEvery == 0
	rl.mu.Unlock()

	if needCleanup {
		rl.mu.Lock()
		rl.guard.cleanupOldEntries()
		rl.mu.Unlock()
		logger.Debug("🧹 MAX Rate: периодическая очистка кэша (отправок: %d)", rl.sendCount)
	}
}

// ──────────────────────────────────────────────
// Вспомогательные методы (зеркало Telegram service.go)
// ──────────────────────────────────────────────

// rateLimitPeriod возвращает период rate limiting из настроек пользователя.
// Берётся максимальный из PreferredPeriods, зажатый в [5м, 1ч].
func (rl *maxRateLimiter) rateLimitPeriod(user *models.User) time.Duration {
	if user == nil {
		return periodPkg.DefaultDuration
	}

	period := periodPkg.DefaultDuration
	if len(user.PreferredPeriods) > 0 {
		maxMinutes := periodPkg.GetMaxPeriod(user.PreferredPeriods)
		clamped := periodPkg.ClampPeriodStandard(maxMinutes)
		period = periodPkg.MinutesToDuration(clamped)
	}

	// Clamp: минимум 5м, максимум 1ч
	minP := periodPkg.MinutesToDuration(periodPkg.Minutes5)
	maxP := periodPkg.MinutesToDuration(periodPkg.Minutes60)
	if period < minP {
		return minP
	}
	if period > maxP {
		return maxP
	}
	return period
}

// baseLimit возвращает базовый лимит уведомлений для периода (идентично Telegram)
func (rl *maxRateLimiter) baseLimit(periodMinutes int) int {
	switch periodMinutes {
	case 5:
		return 3
	case 15:
		return 4
	case 30:
		return 5
	case 60:
		return 6
	case 240:
		return 8
	case 1440:
		return 10
	default:
		if periodMinutes <= 5 {
			return 3
		}
		return 3 * (periodMinutes / 5)
	}
}

// symbolLimit возвращает лимит с учётом направления сигнала
func (rl *maxRateLimiter) symbolLimit(symbol string, periodMinutes int, direction string) int {
	limit := rl.baseLimit(periodMinutes)
	if direction == "fall" {
		limit = int(float64(limit) * rateLimitFallBoost)
	}
	return limit
}

// isStrongMove возвращает true, если изменение > 5% (порог умного обхода)
func isStrongMove(changePercent float64) bool {
	return absFloat(changePercent) > strongMoveThresholdPct
}
