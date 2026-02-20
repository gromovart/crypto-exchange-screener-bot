// internal/delivery/telegram/services/counter/service.go
package counter

import (
	"crypto-exchange-screener-bot/internal/core/domain/subscription"
	"crypto-exchange-screener-bot/internal/core/domain/users"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/buttons"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/formatters"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/message_sender"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
	"crypto-exchange-screener-bot/pkg/logger"
	periodPkg "crypto-exchange-screener-bot/pkg/period"
	"fmt"
	"math"
	"sync"
	"time"
)

type serviceImpl struct {
	userService         *users.Service
	subscriptionService *subscription.Service
	formatter           *formatters.FormatterProvider
	messageSender       message_sender.MessageSender
	buttonBuilder       *buttons.ButtonBuilder
	notificationGuard   *SymbolNotificationGuard
	guardMu             sync.RWMutex
}

func NewService(
	userService *users.Service,
	subscriptionService *subscription.Service,
	formatter *formatters.FormatterProvider,
	messageSender message_sender.MessageSender,
	buttonBuilder *buttons.ButtonBuilder,
) Service {
	return &serviceImpl{
		userService:         userService,
		subscriptionService: subscriptionService,
		formatter:           formatter,
		messageSender:       messageSender,
		buttonBuilder:       buttonBuilder,
		notificationGuard:   NewSymbolNotificationGuard(),
	}
}

func (s *serviceImpl) Exec(params CounterParams) (CounterResult, error) {
	rawData, err := s.extractRawDataFromParams(params)
	if err != nil {
		return CounterResult{Processed: false},
			fmt.Errorf("ошибка извлечения данных счетчика: %w", err)
	}

	counterData := s.convertToFormatterData(rawData)

	usersToNotify, err := s.getUsersToNotify(rawData)
	if err != nil {
		return CounterResult{Processed: false},
			fmt.Errorf("ошибка получения пользователей: %w", err)
	}

	if len(usersToNotify) == 0 {
		return CounterResult{
			Processed: true,
			Message:   fmt.Sprintf("Нет пользователей для уведомления по %s", rawData.Symbol),
			SentTo:    0,
		}, nil
	}

	sentCount := 0
	rateLimitedCount := 0
	smartBypassedCount := 0

	for _, user := range usersToNotify {
		allowed, signalPeriod, rateLimitPeriod, currentCount, timeUntilNext, limit := s.checkRateLimitWithDetails(user, rawData)

		if !allowed {
			rateLimitedCount++
			// Уменьшаем логирование - только каждый 10-й раз или если высокий счетчик
			if rateLimitedCount%10 == 0 || currentCount > limit/2 {
				logger.Debug("⏸️ Rate limit: user=%d, symbol=%s, direction=%s, count=%d/%d, next_in=%v",
					user.ID, rawData.Symbol, rawData.Direction, currentCount, limit, timeUntilNext.Round(time.Second))
			}
			continue
		}

		if err := s.sendNotificationWithGuard(user, counterData, signalPeriod, rateLimitPeriod, currentCount, limit); err != nil {
			logger.Error("❌ Ошибка отправки уведомления: %v", err)
		} else {
			sentCount++
			s.userService.LogSignalSent(user.ID, counterData.Direction, counterData.Symbol, counterData.ChangePercent, int(signalPeriod.Minutes()))
		}
	}

	if sentCount+rateLimitedCount > 0 && (sentCount+rateLimitedCount)%100 == 0 {
		s.cleanupOldGuardEntries()
	}

	if rateLimitedCount > 0 || smartBypassedCount > 0 {
		// Логируем статистику только если есть пропущенные сигналы
		logger.Debug("📊 Rate limiting статистика: %s %s - отправлено=%d, пропущено=%d, умных обходов=%d",
			rawData.Symbol, rawData.Direction, sentCount, rateLimitedCount, smartBypassedCount)
	}

	return CounterResult{
		Processed: true,
		Message:   fmt.Sprintf("Отправлено %d уведомлений для %s %s", sentCount, rawData.Symbol, rawData.Direction),
		SentTo:    sentCount,
	}, nil
}

func (s *serviceImpl) sendNotificationWithGuard(user *models.User, data formatters.CounterData, signalPeriod, rateLimitPeriod time.Duration, currentCount, limit int) error {
	formattedMessage := s.formatter.FormatCounterSignal(data)

	if user.ChatID == "" {
		return fmt.Errorf("пустой chat_id у пользователя %s", user.Username)
	}

	var chatID int64
	_, err := fmt.Sscanf(user.ChatID, "%d", &chatID)
	if err != nil {
		return fmt.Errorf("неверный формат chat_id у пользователя %s: %s", user.Username, user.ChatID)
	}

	var keyboard interface{} = nil
	if s.buttonBuilder != nil {
		keyboard = s.buttonBuilder.CreateSignalKeyboard(data.Symbol)
	}

	if s.messageSender != nil {
		err := s.messageSender.SendTextMessage(chatID, formattedMessage, keyboard)
		if err != nil {
			return fmt.Errorf("ошибка отправки в Telegram: %w", err)
		}

		// Записываем в rate limiting
		s.guardMu.Lock()
		userID64 := int64(user.ID)
		s.notificationGuard.Record(userID64, data.Symbol, data.Direction, signalPeriod, rateLimitPeriod)
		s.guardMu.Unlock()

		s.logSuccessfulNotification(user, data.Symbol, data.Direction, signalPeriod, rateLimitPeriod, currentCount+1, limit)

		return nil
	} else {
		return fmt.Errorf("message sender not initialized")
	}
}

func (s *serviceImpl) logSuccessfulNotification(user *models.User, symbol, direction string, signalPeriod, rateLimitPeriod time.Duration, newCount, limit int) {
	signalMinutes := int(signalPeriod.Minutes())
	rateLimitMinutes := int(rateLimitPeriod.Minutes())
	minInterval := rateLimitPeriod / time.Duration(limit)

	logger.Info("📤 Уведомление: %s %s (%s) → %s (ID: %d, счет: %d/%d, сигнал: %dм, rate limit: %dм, интервал: %v)",
		symbol, direction, periodPkg.MinutesToString(signalMinutes), user.Username,
		user.ID, newCount, limit, signalMinutes, rateLimitMinutes, minInterval)
}

func (s *serviceImpl) cleanupOldGuardEntries() {
	s.guardMu.Lock()
	defer s.guardMu.Unlock()

	s.notificationGuard.CleanupOldEntries()
	logger.Debug("🧹 Очистка старых записей rate limiting")
}

// getNotificationPeriod использует МАКСИМАЛЬНЫЙ период пользователя для rate limiting
func (s *serviceImpl) getNotificationPeriod(user *models.User, signalPeriod string) time.Duration {
	// 1. Получаем максимальный период из настроек пользователя
	userMaxPeriod := s.getMaxUserPeriod(user)

	// 2. Ограничиваем период для безопасности
	period := s.clampPeriod(userMaxPeriod)

	logger.Debug("🔍 Rate limit период: user=%d, maxPeriod=%v, signalPeriod=%s",
		user.ID, period, signalPeriod)

	return period
}

// getMaxUserPeriod возвращает максимальный период из настроек пользователя
func (s *serviceImpl) getMaxUserPeriod(user *models.User) time.Duration {
	if user == nil {
		return periodPkg.DefaultDuration
	}

	// Если есть предпочтительные периоды - берем максимальный
	if len(user.PreferredPeriods) > 0 {
		// Получаем максимальный период в минутах
		maxMinutes := periodPkg.GetMaxPeriod(user.PreferredPeriods)

		// Ограничиваем стандартными пределами (5м - 1день)
		clampedMinutes := periodPkg.ClampPeriodStandard(maxMinutes)
		// Конвертируем в Duration
		return periodPkg.MinutesToDuration(clampedMinutes)
	}

	// Дефолтное значение
	return periodPkg.DefaultDuration
}

// clampPeriod ограничивает период разумными пределами
func (s *serviceImpl) clampPeriod(periodDuration time.Duration) time.Duration {
	minPeriod := periodPkg.MinutesToDuration(periodPkg.Minutes5)  // Минимум 5 минут
	maxPeriod := periodPkg.MinutesToDuration(periodPkg.Minutes60) // Максимум 1 час

	if periodDuration < minPeriod {
		return minPeriod
	}
	if periodDuration > maxPeriod {
		return maxPeriod
	}
	return periodDuration
}

// getRateLimit возвращает лимит уведомлений в зависимости от периода
func (s *serviceImpl) getRateLimit(periodMinutes int) int {
	switch periodMinutes {
	case 5: // 5 минут
		return 3 // Более строгий лимит для коротких периодов
	case 15: // 15 минут
		return 4
	case 30: // 30 минут
		return 5
	case 60: // 1 час
		return 6
	case 240: // 4 часа
		return 8
	case 1440: // 1 день
		return 10
	default:
		// Для нестандартных периодов используем пропорциональный расчет
		// Базовый период 5 минут с лимитом 3
		basePeriod := 5
		baseLimit := 3

		if periodMinutes <= basePeriod {
			return baseLimit
		}

		// Увеличиваем лимит пропорционально, но с округлением вниз
		multiplier := periodMinutes / basePeriod
		return baseLimit * multiplier
	}
}

// getSymbolSpecificLimit возвращает лимит с учетом специфики символа
func (s *serviceImpl) getSymbolSpecificLimit(symbol string, periodMinutes int, direction string) int {
	baseLimit := s.getRateLimit(periodMinutes)

	// Учитываем направление - для падения можно давать больше сигналов
	// так как падения обычно быстрее и важнее для трейдинга
	if direction == SignalTypeFall {
		baseLimit = int(float64(baseLimit) * 1.2) // +20% для сигналов падения
	}

	// В будущем можно добавить проверку волатильности символа
	// if s.isHighVolatilitySymbol(symbol) {
	//     return baseLimit * 2
	// }

	return baseLimit
}

// shouldBypassRateLimit проверяет, можно ли обойти rate limiting
func (s *serviceImpl) shouldBypassRateLimit(changePercent float64) bool {
	// Сильные движения (>5%) могут обходить rate limiting
	return math.Abs(changePercent) > 5.0
}

// checkRateLimitWithDetails проверяет rate limiting с учетом умных обходов
func (s *serviceImpl) checkRateLimitWithDetails(user *models.User, data RawCounterData) (bool, time.Duration, time.Duration, int, time.Duration, int) {
	s.guardMu.RLock()
	defer s.guardMu.RUnlock()

	// Получаем период для rate limiting из настроек пользователя
	rateLimitPeriod := s.getNotificationPeriod(user, data.Period)

	// Конвертируем период сигнала из строки в Duration
	signalPeriod, err := periodPkg.StringToDuration(data.Period)
	if err != nil {
		logger.Warn("⚠️ Ошибка конвертации периода сигнала '%s', используем дефолтный: %s",
			data.Period, periodPkg.DefaultPeriod)
		signalPeriod = periodPkg.DefaultDuration
	}

	userID64 := int64(user.ID)

	// Конвертируем период в минуты для получения лимита
	rateLimitMinutes := int(rateLimitPeriod.Minutes())

	// Получаем лимит с учетом специфики символа и направления
	limit := s.getSymbolSpecificLimit(data.Symbol, rateLimitMinutes, data.Direction)

	// ⭐ ПРОВЕРЯЕМ УМНЫЙ ОБХОД для сильных движений
	if s.shouldBypassRateLimit(data.ChangePercent) {
		allowed, reason := s.notificationGuard.CanBypassWithPrice(
			userID64, data.Symbol, data.Direction,
			data.CurrentPrice, data.ChangePercent,
		)

		if allowed {
			logger.Info("⚡ Умный обход rate limiting для %s %s: %.2f%% (причина: %s)",
				data.Symbol, data.Direction, data.ChangePercent, reason)

			// ⭐ РЕГИСТРИРУЕМ УМНЫЙ ОБХОД
			s.notificationGuard.RecordSmartBypass(
				userID64, data.Symbol, data.Direction,
				data.CurrentPrice, data.ChangePercent,
			)

			return true, signalPeriod, rateLimitPeriod, 0, 0, limit
		} else {
			logger.Info("⏸️ Отказ в обходе для %s %s: %.2f%% (причина: %s)",
				data.Symbol, data.Direction, data.ChangePercent, reason)

			// Применяем обычный rate limiting
			return s.applyNormalRateLimit(userID64, data.Symbol, data.Direction, signalPeriod, rateLimitPeriod, limit)
		}
	}

	// Обычный rate limiting для несильных движений
	return s.applyNormalRateLimit(userID64, data.Symbol, data.Direction, signalPeriod, rateLimitPeriod, limit)
}

// applyNormalRateLimit применяет обычный rate limiting
func (s *serviceImpl) applyNormalRateLimit(userID int64, symbol, direction string, signalPeriod, rateLimitPeriod time.Duration, limit int) (bool, time.Duration, time.Duration, int, time.Duration, int) {
	allowed := s.notificationGuard.Check(userID, symbol, direction, signalPeriod, rateLimitPeriod)
	currentCount := s.notificationGuard.GetCount(userID, symbol, direction, signalPeriod, rateLimitPeriod)
	timeUntilNext := s.notificationGuard.GetTimeUntilNextAllowed(userID, symbol, direction, signalPeriod, rateLimitPeriod)

	// Логируем детали rate limiting
	logger.Debug("🔍 Rate limiting: user=%d, symbol=%s, direction=%s, signal=%v, rate_limit=%v, count=%d/%d, allowed=%v",
		userID, symbol, direction, signalPeriod, rateLimitPeriod, currentCount, limit, allowed)

	return allowed, signalPeriod, rateLimitPeriod, currentCount, timeUntilNext, limit
}
