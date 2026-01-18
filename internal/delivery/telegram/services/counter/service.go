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

// serviceImpl реализация CounterService
type serviceImpl struct {
	userService   *users.Service
	formatter     *formatters.FormatterProvider
	messageSender message_sender.MessageSender
	buttonBuilder *buttons.ButtonBuilder

	// ДЕЛАЕМ GUARD ПОЛЕМ СЕРВИСА (один на все уведомления)
	notificationGuard *SymbolNotificationGuard
	guardMu           sync.RWMutex // для потокобезопасности
}

// NewService создает новый сервис счетчика
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

// Exec выполняет обработку события счетчика
func (s *serviceImpl) Exec(params CounterParams) (CounterResult, error) {
	// Извлекаем данные счетчика из CounterParams
	rawData, err := s.extractRawDataFromParams(params)
	if err != nil {
		return CounterResult{Processed: false},
			fmt.Errorf("ошибка извлечения данных счетчика: %w", err)
	}

	// Конвертируем в форматтер данные
	counterData := s.convertToFormatterData(rawData)

	// Получаем пользователей для отправки
	usersToNotify, err := s.getUsersToNotify(rawData)
	if err != nil {
		return CounterResult{Processed: false},
			fmt.Errorf("ошибка получения пользователей: %w", err)
	}

	// Если нет пользователей для уведомления
	if len(usersToNotify) == 0 {
		return CounterResult{
			Processed: true,
			Message:   fmt.Sprintf("Нет пользователей для уведомления по %s", rawData.Symbol),
			SentTo:    0,
		}, nil
	}

	// Отправляем уведомления с учетом rate limiting
	sentCount := 0
	rateLimitedCount := 0

	for _, user := range usersToNotify {
		// ПРОВЕРЯЕМ RATE LIMIT ПЕРЕД ОТПРАВКОЙ
		allowed, period, currentCount, timeUntilNext, minInterval := s.checkRateLimitWithDetails(user, rawData)

		if !allowed {
			rateLimitedCount++

			// Логируем детали rate limiting
			s.logRateLimitDetails(user, rawData.Symbol, period, currentCount, timeUntilNext, minInterval)
			continue
		}

		// Логируем успешную проверку
		logger.Warn("✅ Rate limit OK: user=%d (%s), symbol=%s, period=%v, count=%d/%d, min_interval=%v",
			user.ID, user.Username, rawData.Symbol, period, currentCount,
			s.notificationGuard.GetLimit(), minInterval)

		// Отправляем уведомление
		if err := s.sendNotificationWithGuard(user, counterData, period, currentCount); err != nil {
			logger.Error("❌ Ошибка отправки уведомления пользователю %s: %v", user.Username, err)
		} else {
			sentCount++
		}
	}

	// Периодически чистим старые записи (раз в 100 вызовов)
	if sentCount+rateLimitedCount > 0 && (sentCount+rateLimitedCount)%100 == 0 {
		s.cleanupOldGuardEntries()
	}

	// Логируем статистику по rate limiting
	if rateLimitedCount > 0 {
		logger.Warn("📊 Rate limiting статистика: отправлено %d, пропущено из-за лимитов %d для %s",
			sentCount, rateLimitedCount, rawData.Symbol)
	}

	return CounterResult{
		Processed: true,
		Message: fmt.Sprintf("Отправлено %d уведомлений для %s (пропущено по лимитам: %d)",
			sentCount, rawData.Symbol, rateLimitedCount),
		SentTo: sentCount,
	}, nil
}

// checkRateLimitWithDetails проверяет rate limit и возвращает детали
func (s *serviceImpl) checkRateLimitWithDetails(user *models.User, data RawCounterData) (bool, time.Duration, int, time.Duration, time.Duration) {
	s.guardMu.RLock()
	defer s.guardMu.RUnlock()

	// Получаем период для rate limiting
	period := s.getNotificationPeriod(user, data.Period)

	// Конвертируем user.ID из int в int64 для guard
	userID64 := int64(user.ID)

	// Получаем текущий счетчик
	currentCount := s.notificationGuard.GetCount(userID64, data.Symbol, period)

	// Вычисляем минимальный интервал для равномерного распределения
	limit := s.notificationGuard.GetLimit()
	minInterval := period / time.Duration(limit)

	// Время до следующего разрешенного уведомления
	timeUntilNext := s.notificationGuard.GetTimeUntilNextAllowed(userID64, data.Symbol, period)

	// Проверяем через guard
	allowed := s.notificationGuard.Check(userID64, data.Symbol, period)

	return allowed, period, currentCount, timeUntilNext, minInterval
}

// checkRateLimit проверяет rate limit для пользователя и символа
func (s *serviceImpl) checkRateLimit(user *models.User, data RawCounterData) (bool, time.Duration) {
	allowed, period, _, _, _ := s.checkRateLimitWithDetails(user, data)
	return allowed, period
}

// logRateLimitDetails логирует детали о rate limiting
func (s *serviceImpl) logRateLimitDetails(user *models.User, symbol string, period time.Duration, currentCount int, timeUntilNext time.Duration, minInterval time.Duration) {
	limit := s.notificationGuard.GetLimit()

	if currentCount >= limit {
		// Лимит по количеству достигнут
		logger.Debug("⏸️  Rate limit (количество): пользователь %d (%s) достиг лимита для %s: %d/%d за период %v",
			user.ID, user.Username, symbol, currentCount, limit, period)
	} else {
		// Лимит по времени (интервал)
		logger.Debug("⏸️  Rate limit (интервал): пользователь %d (%s) для %s: последнее уведомление было недавно, след. через %v (min_interval=%v)",
			user.ID, user.Username, symbol, timeUntilNext.Round(time.Second), minInterval)
	}
}

// sendNotificationWithGuard отправляет уведомление с записью в guard
func (s *serviceImpl) sendNotificationWithGuard(user *models.User, data formatters.CounterData, period time.Duration, currentCount int) error {
	// Форматируем сообщение
	formattedMessage := s.formatter.FormatCounterSignal(data)

	logger.Debug("📨 Отправка counter уведомления для %s пользователю %s (период лимита: %v, тариф: %s, счет ДО отправки: %d/%d)",
		data.Symbol, user.Username, period, user.SubscriptionTier, currentCount, s.notificationGuard.GetLimit())

	// Проверяем chat_id
	if user.ChatID == "" {
		return fmt.Errorf("пустой chat_id у пользователя %s", user.Username)
	}

	// Конвертируем chat_id из строки в int64
	var chatID int64
	_, err := fmt.Sscanf(user.ChatID, "%d", &chatID)
	if err != nil {
		return fmt.Errorf("неверный формат chat_id у пользователя %s: %s", user.Username, user.ChatID)
	}

	// СОЗДАЕМ КЛАВИАТУРУ С КНОПКАМИ
	var keyboard interface{} = nil
	if s.buttonBuilder != nil {
		keyboard = s.buttonBuilder.CreateSignalKeyboard(data.Symbol)
		logger.Debug("🛠️ Создана клавиатура для %s", data.Symbol)
	}

	// Отправляем через message sender
	if s.messageSender != nil {
		err := s.messageSender.SendTextMessage(chatID, formattedMessage, keyboard)
		if err != nil {
			return fmt.Errorf("ошибка отправки в Telegram: %w", err)
		}

		// ЗАПИСЫВАЕМ УСПЕШНУЮ ОТПРАВКУ В GUARD
		s.guardMu.Lock()
		userID64 := int64(user.ID)
		s.notificationGuard.Record(userID64, data.Symbol, period)
		s.guardMu.Unlock()

		// Логируем успешную отправку с новым счетчиком
		s.logSuccessfulNotification(user, data.Symbol, period, currentCount+1)

		logger.Debug("✅ Сообщение отправлено пользователю %s", user.Username)
	} else {
		logger.Error("⚠️ MessageSender не инициализирован, сообщение не отправлено")
		return fmt.Errorf("message sender not initialized")
	}

	return nil
}

// logSuccessfulNotification логирует успешную отправку уведомления
func (s *serviceImpl) logSuccessfulNotification(user *models.User, symbol string, period time.Duration, newCount int) {
	limit := s.notificationGuard.GetLimit()
	// Вычисляем минимальный интервал
	minInterval := period / time.Duration(limit)

	logger.Warn("📤 Уведомление отправлено: %s -> %s (ID: %d, тариф: %s, счет: %d/%d, период: %v, мин. интервал: %v)",
		symbol, user.Username, user.ID, user.SubscriptionTier,
		newCount, limit, period, minInterval)
}

// cleanupOldGuardEntries периодически чистит старые записи в guard
func (s *serviceImpl) cleanupOldGuardEntries() {
	s.guardMu.Lock()
	defer s.guardMu.Unlock()

	s.notificationGuard.CleanupOldEntries()
	logger.Debug("🧹 Выполнена очистка старых записей rate limiting")
}

// getNotificationPeriod определяет период для rate limiting
func (s *serviceImpl) getNotificationPeriod(user *models.User, signalPeriod string) time.Duration {
	// Основной период для rate limiting - используем период сигнала
	// Это обеспечивает связь: лимит уведомлений привязан к периоду анализа
	periodMinutes := s.periodToMinutes(signalPeriod)

	return time.Duration(periodMinutes) * time.Minute
}

// DebugGuardState возвращает отладочную информацию о состоянии guard
func (s *serviceImpl) DebugGuardState(userID int, symbol, periodStr string) string {
	s.guardMu.RLock()
	defer s.guardMu.RUnlock()

	period := s.getNotificationPeriod(&models.User{ID: userID}, periodStr)
	userID64 := int64(userID)

	count := s.notificationGuard.GetCount(userID64, symbol, period)
	allowed := s.notificationGuard.Check(userID64, symbol, period)
	timeUntilNext := s.notificationGuard.GetTimeUntilNextAllowed(userID64, symbol, period)
	limit := s.notificationGuard.GetLimit()

	return fmt.Sprintf("Guard state: user=%d, symbol=%s, period=%v, count=%d/%d, allowed=%v, next_in=%v",
		userID, symbol, period, count, limit, allowed, timeUntilNext.Round(time.Second))
}

// GetRateLimitStats возвращает статистику rate limiting для пользователя и символа
func (s *serviceImpl) GetRateLimitStats(userID int, symbol, periodStr string) map[string]interface{} {
	s.guardMu.RLock()
	defer s.guardMu.RUnlock()

	period := s.getNotificationPeriod(&models.User{ID: userID}, periodStr)
	userID64 := int64(userID)

	count := s.notificationGuard.GetCount(userID64, symbol, period)
	limit := s.notificationGuard.GetLimit()
	allowed := s.notificationGuard.Check(userID64, symbol, period)
	timeUntilNext := s.notificationGuard.GetTimeUntilNextAllowed(userID64, symbol, period)
	minInterval := period / time.Duration(limit)

	return map[string]interface{}{
		"user_id":         userID,
		"symbol":          symbol,
		"period":          period.String(),
		"current_count":   count,
		"limit":           limit,
		"allowed":         allowed,
		"time_until_next": timeUntilNext.Round(time.Second).String(),
		"min_interval":    minInterval.String(),
		"percent_used":    float64(count) / float64(limit) * 100,
	}
}
