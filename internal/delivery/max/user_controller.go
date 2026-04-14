// internal/delivery/max/user_controller.go
package max

import (
	"fmt"
	"math"
	"strings"
	"time"

	"crypto-exchange-screener-bot/internal/core/domain/users"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
	"crypto-exchange-screener-bot/pkg/period"
)

const (
	maxUserFetchLimit  = 1000
	maxUserFetchOffset = 0
)

// UserController рассылает сигналы персонально каждому MAX-пользователю
// с активной торговой сессией (MaxNotificationsEnabled == true).
type UserController struct {
	client      *Client
	userService *users.Service
	rateLimiter *maxRateLimiter
}

// NewUserController создаёт контроллер
func NewUserController(client *Client, userSvc *users.Service) *UserController {
	return &UserController{
		client:      client,
		userService: userSvc,
		rateLimiter: newMaxRateLimiter(),
	}
}

// GetName возвращает имя контроллера
func (c *UserController) GetName() string {
	return "max_user_controller"
}

// GetSubscribedEvents возвращает список подписанных событий
func (c *UserController) GetSubscribedEvents() []types.EventType {
	return []types.EventType{types.EventCounterSignalDetected}
}

// HandleEvent обрабатывает событие сигнала
func (c *UserController) HandleEvent(event types.Event) error {
	dataMap, ok := event.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("max user_controller: неверный формат данных события")
	}

	// Получаем всех пользователей
	allUsers, err := c.userService.GetAllUsers(maxUserFetchLimit, maxUserFetchOffset)
	if err != nil {
		return fmt.Errorf("max user_controller: ошибка получения пользователей: %w", err)
	}

	symbol := getString(dataMap, "symbol")
	text := formatSignalText(dataMap)

	// Загружаем активные MAX-сессии одним запросом → map[userID]expiresAt
	sessionMap := c.buildMaxSessionMap()
	sent := 0
	rateLimited := 0

	for _, user := range allUsers {
		if !c.shouldSendToUser(user, dataMap) {
			continue
		}

		// ── Rate limiting (аналогично Telegram) ──────────────────────────────
		rl := c.rateLimiter.check(user, dataMap)
		if !rl.Allowed {
			rateLimited++
			continue
		}

		chatID, err := maxChatIDInt64(user.MaxChatID)
		if err != nil {
			logger.Warn("⚠️ MAX UserController: невалидный MaxChatID user=%d: %v", user.ID, err)
			continue
		}

		expiresAt, hasSession := sessionMap[user.ID]
		keyboard := signalKeyboard(symbol, hasSession, expiresAt)

		if sendErr := c.client.SendMessageWithKeyboard(chatID, text, keyboard); sendErr != nil {
			logger.Warn("⚠️ MAX UserController: ошибка отправки user=%d: %v", user.ID, sendErr)
		} else {
			sent++
			// Регистрируем отправку в rate limiter
			c.rateLimiter.record(int64(user.ID), symbol,
				getString(dataMap, "direction"),
				rl.SignalPeriod, rl.RateLimitPeriod)
		}
	}

	if sent > 0 || rateLimited > 0 {
		logger.Debug("✅ MAX UserController: сигнал %s — отправлено=%d, rate_limited=%d",
			symbol, sent, rateLimited)
	}
	return nil
}

// shouldSendToUser проверяет, нужно ли отправлять сигнал конкретному пользователю
func (c *UserController) shouldSendToUser(user *models.User, data map[string]interface{}) bool {
	if user == nil {
		return false
	}

	// Базовые условия
	if !user.IsActive {
		return false
	}
	if !user.MaxNotificationsEnabled {
		return false
	}
	if user.MaxChatID == "" {
		return false
	}

	direction := getString(data, "direction")
	changePercent := getFloat64(data, "change_percent")

	// Тип сигнала и настройки пользователя
	switch direction {
	case "growth":
		if !user.NotifyGrowth {
			return false
		}
		if changePercent < user.MinGrowthThreshold {
			return false
		}
	case "fall":
		if !user.NotifyFall {
			return false
		}
		if math.Abs(changePercent) < user.MinFallThreshold {
			return false
		}
	default:
		return false
	}

	// Дневной лимит
	if user.HasReachedDailyLimit() {
		return false
	}

	// Фильтр объёма
	volume24h := getFloat64(data, "volume_24h")
	if user.MinVolumeFilter > 0 && volume24h < user.MinVolumeFilter {
		return false
	}

	// Исключённые паттерны
	symbol := getString(data, "symbol")
	for _, pattern := range user.ExcludePatterns {
		if pattern != "" && strings.Contains(symbol, pattern) {
			return false
		}
	}

	// Вотчлист: если задан — пропускаем только символы из списка
	if user.HasWatchlist() && !user.ShouldTrackSymbol(symbol) {
		return false
	}

	// Предпочтительные периоды
	periodStr := getString(data, "period")
	if !c.isPeriodAllowed(user, periodStr) {
		return false
	}

	return true
}

// isPeriodAllowed проверяет, соответствует ли период настройкам пользователя
func (c *UserController) isPeriodAllowed(user *models.User, periodStr string) bool {
	if periodStr == "" {
		return true
	}

	periodInt, err := period.StringToMinutes(periodStr)
	if err != nil {
		return false
	}

	if len(user.PreferredPeriods) == 0 {
		// Если не настроено — дефолт 15 минут
		return periodInt == 15
	}

	for _, p := range user.PreferredPeriods {
		if periodInt == p {
			return true
		}
	}
	return false
}

// buildMaxSessionMap загружает все активные MAX-сессии из БД и возвращает
// map[userID]expiresAt. Один запрос на весь цикл доставки.
func (c *UserController) buildMaxSessionMap() map[int]time.Time {
	sessions, err := c.userService.FindAllActiveTradingSessions()
	if err != nil {
		logger.Warn("⚠️ MAX UserController: ошибка загрузки сессий: %v", err)
		return nil
	}
	m := make(map[int]time.Time, len(sessions))
	for _, s := range sessions {
		if s.Platform == "max" {
			m[s.UserID] = s.ExpiresAt
		}
	}
	return m
}

// signalKeyboard формирует MAX inline-keyboard:
//   - строка 1: «🛒 Торговать» | «📊 График» (URL-кнопки)
//   - строка 2: кнопка сессии — завершить с оставшимся временем или начать
func signalKeyboard(symbol string, hasSession bool, expiresAt time.Time) interface{} {
	clean := strings.ToUpper(strings.ReplaceAll(symbol, "/", ""))
	tradeURL := fmt.Sprintf("https://www.bybit.com/trade/usdt/%s", clean)
	chartURL := fmt.Sprintf("https://www.coinglass.com/tv/ru/Bybit_%s", clean)

	var sessionBtn map[string]string
	if hasSession {
		remaining := time.Until(expiresAt)
		sessionBtn = kb.B(
			fmt.Sprintf("🔴 Завершить сессию (%s)", formatRemaining(remaining)),
			kb.CbSessionStop,
		)
	} else {
		sessionBtn = kb.B(kb.Btn.SessionStart, kb.CbSessionStart)
	}

	return kb.Keyboard([][]map[string]string{
		{
			kb.BUrl("🛒 Торговать", tradeURL),
			kb.BUrl("📊 График", chartURL),
		},
		{kb.B(kb.Btn.MainMenu, kb.CbMenuMain)},
		{sessionBtn},
	})
}

// formatRemaining форматирует оставшееся время сессии: «2ч 34м», «45м», «<1м»
func formatRemaining(d time.Duration) string {
	if d <= 0 {
		return "<1м"
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dч %dм", h, m)
	}
	return fmt.Sprintf("%dм", m)
}

// maxChatIDInt64 конвертирует строковый MaxChatID в int64
func maxChatIDInt64(chatID string) (int64, error) {
	if chatID == "" {
		return 0, fmt.Errorf("пустой MaxChatID")
	}
	var id int64
	if _, err := fmt.Sscanf(chatID, "%d", &id); err != nil {
		return 0, fmt.Errorf("не удалось распарсить MaxChatID %q: %w", chatID, err)
	}
	return id, nil
}
