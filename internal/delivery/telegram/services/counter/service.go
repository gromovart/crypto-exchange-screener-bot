// internal/delivery/telegram/services/counter/service.go
package counter

import (
	"crypto-exchange-screener-bot/internal/core/domain/users"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/buttons"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/formatters"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/message_sender"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"sync"
	"time"
)

type serviceImpl struct {
	userService       *users.Service
	formatter         *formatters.FormatterProvider
	messageSender     message_sender.MessageSender
	buttonBuilder     *buttons.ButtonBuilder
	notificationGuard *SymbolNotificationGuard
	guardMu           sync.RWMutex
}

func NewService(
	userService *users.Service,
	formatter *formatters.FormatterProvider,
	messageSender message_sender.MessageSender,
	buttonBuilder *buttons.ButtonBuilder,
) Service {
	return &serviceImpl{
		userService:       userService,
		formatter:         formatter,
		messageSender:     messageSender,
		buttonBuilder:     buttonBuilder,
		notificationGuard: NewSymbolNotificationGuard(),
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

	for _, user := range usersToNotify {
		allowed, period, currentCount, timeUntilNext, _ := s.checkRateLimitWithDetails(user, rawData)

		if !allowed {
			rateLimitedCount++
			logger.Info("⏸️ Rate limit: user=%d, symbol=%s, count=%d/5, next_in=%v",
				user.ID, rawData.Symbol, currentCount, timeUntilNext.Round(time.Second))
			continue
		}

		if err := s.sendNotificationWithGuard(user, counterData, period, currentCount); err != nil {
			logger.Error("❌ Ошибка отправки уведомления: %v", err)
		} else {
			sentCount++
		}
	}

	if sentCount+rateLimitedCount > 0 && (sentCount+rateLimitedCount)%100 == 0 {
		s.cleanupOldGuardEntries()
	}

	if rateLimitedCount > 0 {
		logger.Info("📊 Rate limiting: %s - отправлено=%d, пропущено=%d",
			rawData.Symbol, sentCount, rateLimitedCount)
	}

	return CounterResult{
		Processed: true,
		Message:   fmt.Sprintf("Отправлено %d уведомлений для %s", sentCount, rawData.Symbol),
		SentTo:    sentCount,
	}, nil
}

func (s *serviceImpl) checkRateLimitWithDetails(user *models.User, data RawCounterData) (bool, time.Duration, int, time.Duration, time.Duration) {
	s.guardMu.RLock()
	defer s.guardMu.RUnlock()

	period := s.getNotificationPeriod(user, data.Period)
	userID64 := int64(user.ID)

	currentCount := s.notificationGuard.GetCount(userID64, data.Symbol, period)
	limit := s.notificationGuard.GetLimit()
	minInterval := period / time.Duration(limit)
	timeUntilNext := s.notificationGuard.GetTimeUntilNextAllowed(userID64, data.Symbol, period)
	allowed := s.notificationGuard.Check(userID64, data.Symbol, period)

	return allowed, period, currentCount, timeUntilNext, minInterval
}

func (s *serviceImpl) sendNotificationWithGuard(user *models.User, data formatters.CounterData, period time.Duration, currentCount int) error {
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

		s.guardMu.Lock()
		userID64 := int64(user.ID)
		s.notificationGuard.Record(userID64, data.Symbol, period)
		s.guardMu.Unlock()

		s.logSuccessfulNotification(user, data.Symbol, period, currentCount+1)

		return nil
	} else {
		return fmt.Errorf("message sender not initialized")
	}
}

func (s *serviceImpl) logSuccessfulNotification(user *models.User, symbol string, period time.Duration, newCount int) {
	limit := s.notificationGuard.GetLimit()
	minInterval := period / time.Duration(limit)

	logger.Info("📤 Уведомление: %s -> %s (ID: %d, счет: %d/%d, период: %v, интервал: %v)",
		symbol, user.Username, user.ID, newCount, limit, period, minInterval)
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
		return 5 * time.Minute
	}

	// Если есть предпочтительные периоды - берем максимальный
	if len(user.PreferredPeriods) > 0 {
		maxPeriodMin := 0
		for _, periodMin := range user.PreferredPeriods {
			if periodMin > maxPeriodMin {
				maxPeriodMin = periodMin
			}
		}

		// Конвертируем минуты в Duration
		if maxPeriodMin >= 5 {
			return time.Duration(maxPeriodMin) * time.Minute
		}
	}

	// Дефолтное значение
	return 5 * time.Minute
}

// clampPeriod ограничивает период разумными пределами
func (s *serviceImpl) clampPeriod(period time.Duration) time.Duration {
	minPeriod := 5 * time.Minute  // Минимум 5 минут
	maxPeriod := 60 * time.Minute // Максимум 1 час

	if period < minPeriod {
		return minPeriod
	}
	if period > maxPeriod {
		return maxPeriod
	}
	return period
}
