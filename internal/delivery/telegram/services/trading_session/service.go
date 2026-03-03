// internal/delivery/telegram/services/trading_session/service.go
package trading_session

import (
	"fmt"
	"sync"
	"time"

	"crypto-exchange-screener-bot/internal/core/domain/users"
	"crypto-exchange-screener-bot/internal/delivery/telegram"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/message_sender"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
	"crypto-exchange-screener-bot/pkg/logger"
)

// timerKey возвращает строковый ключ таймера: "userID:platform"
func timerKey(userID int, platform string) string {
	return fmt.Sprintf("%d:%s", userID, platform)
}

// notifKey возвращает ключ настройки уведомлений для платформы.
// "telegram" → "notifications_enabled"
// "max"      → "max_notifications_enabled"
func notifKey(platform string) string {
	if platform == "max" {
		return "max_notifications_enabled"
	}
	return "notifications_enabled"
}

// serviceImpl реализация Service с хранением в БД
type serviceImpl struct {
	mu     sync.Mutex
	timers map[string]*time.Timer // ключ: "userID:platform"

	userService   *users.Service
	messageSender message_sender.MessageSender
}

// NewService создает сервис с хранением в БД
func NewService(userService *users.Service, ms message_sender.MessageSender) Service {
	svc := &serviceImpl{
		timers:        make(map[string]*time.Timer),
		userService:   userService,
		messageSender: ms,
	}
	svc.restore()
	return svc
}

// toDTO конвертирует models.TradingSession в TradingSession
func (s *serviceImpl) toDTO(m *models.TradingSession) *TradingSession {
	if m == nil {
		return nil
	}
	return &TradingSession{
		UserID:    m.UserID,
		ChatID:    m.ChatID,
		Platform:  m.Platform,
		StartedAt: m.StartedAt,
		ExpiresAt: m.ExpiresAt,
	}
}

// Start запускает торговую сессию для пользователя на указанной платформе.
// Каждая платформа управляет своим флагом уведомлений независимо:
//   - "telegram" → notifications_enabled
//   - "max"      → max_notifications_enabled
func (s *serviceImpl) Start(userID int, chatID int64, duration time.Duration, platform string) (*TradingSession, error) {
	if platform == "" {
		platform = "telegram"
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.userService == nil {
		return nil, fmt.Errorf("userService не доступен")
	}

	// Останавливаем предыдущий таймер этой платформы
	s.cancelTimerLocked(userID, platform)

	session := &models.TradingSession{
		UserID:    userID,
		ChatID:    chatID,
		Platform:  platform,
		StartedAt: time.Now(),
		ExpiresAt: time.Now().Add(duration),
		IsActive:  true,
	}

	if err := s.userService.SaveTradingSession(session); err != nil {
		return nil, fmt.Errorf("не удалось сохранить сессию: %w", err)
	}

	s.scheduleExpiryLocked(session)

	// Включаем уведомления для этой платформы
	_ = s.userService.UpdateSettings(userID, map[string]interface{}{
		notifKey(platform): true,
	})

	logger.Info("✅ Торговая сессия запущена: user=%d platform=%s", userID, platform)
	return s.toDTO(session), nil
}

// Stop завершает торговую сессию пользователя на указанной платформе
// и отключает уведомления ТОЛЬКО для этой платформы.
func (s *serviceImpl) Stop(userID int, platform string) error {
	if platform == "" {
		platform = "telegram"
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.userService == nil {
		return fmt.Errorf("userService не доступен")
	}

	s.cancelTimerLocked(userID, platform)

	if err := s.userService.DeactivateTradingSessionByPlatform(userID, platform); err != nil {
		return fmt.Errorf("не удалось деактивировать сессию: %w", err)
	}

	// Отключаем уведомления только этой платформы
	_ = s.userService.UpdateSettings(userID, map[string]interface{}{
		notifKey(platform): false,
	})

	logger.Info("✅ Торговая сессия завершена: user=%d platform=%s", userID, platform)
	return nil
}

// GetActive возвращает активную сессию пользователя на указанной платформе
func (s *serviceImpl) GetActive(userID int, platform string) (*TradingSession, bool) {
	if platform == "" {
		platform = "telegram"
	}
	if s.userService == nil {
		logger.Warn("⚠️ userService не доступен")
		return nil, false
	}

	rows, err := s.userService.FindAllActiveTradingSessions()
	if err != nil {
		logger.Warn("⚠️ Не удалось получить активные сессии: %v", err)
		return nil, false
	}

	for _, session := range rows {
		if session.UserID == userID && session.Platform == platform {
			return s.toDTO(session), true
		}
	}
	return nil, false
}

// IsActive проверяет наличие активной сессии на указанной платформе
func (s *serviceImpl) IsActive(userID int, platform string) bool {
	_, ok := s.GetActive(userID, platform)
	return ok
}

// restore восстанавливает активные сессии из БД при старте
func (s *serviceImpl) restore() {
	if s.userService == nil {
		logger.Warn("⚠️ userService не доступен для восстановления")
		return
	}

	rows, err := s.userService.FindAllActiveTradingSessions()
	if err != nil {
		logger.Warn("⚠️ Не удалось загрузить сессии из БД: %v", err)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, r := range rows {
		s.scheduleExpiryLocked(r)
	}

	if len(rows) > 0 {
		logger.Info("✅ Восстановлено %d торговых сессий из БД", len(rows))
	}
}

// scheduleExpiryLocked планирует автозавершение (mu уже захвачен)
func (s *serviceImpl) scheduleExpiryLocked(session *models.TradingSession) {
	delay := time.Until(session.ExpiresAt)
	if delay < 0 {
		delay = 0
	}

	key := timerKey(session.UserID, session.Platform)
	s.timers[key] = time.AfterFunc(delay, func() {
		s.expire(session.UserID, session.ChatID, session.Platform)
	})
}

// cancelTimerLocked отменяет таймер для конкретной платформы (mu уже захвачен)
func (s *serviceImpl) cancelTimerLocked(userID int, platform string) {
	key := timerKey(userID, platform)
	if timer, ok := s.timers[key]; ok {
		timer.Stop()
		delete(s.timers, key)
	}
}

// expire автоматически завершает сессию по истечении времени
func (s *serviceImpl) expire(userID int, chatID int64, platform string) {
	logger.Info("⏰ Торговая сессия истекла: user=%d platform=%s", userID, platform)

	s.mu.Lock()
	delete(s.timers, timerKey(userID, platform))
	s.mu.Unlock()

	if s.userService != nil {
		if err := s.userService.DeactivateTradingSessionByPlatform(userID, platform); err != nil {
			logger.Warn("⚠️ Не удалось деактивировать истекшую сессию: %v", err)
		}

		// Отключаем уведомления только этой платформы
		_ = s.userService.UpdateSettings(userID, map[string]interface{}{
			notifKey(platform): false,
		})
	}

	// Отправляем уведомление только для Telegram (MAX уведомляет через UserController)
	if s.messageSender != nil && platform == "telegram" {
		keyboard := telegram.ReplyKeyboardMarkup{
			Keyboard: [][]telegram.ReplyKeyboardButton{
				{{Text: constants.SessionButtonTexts.Start}},
			},
			ResizeKeyboard: true,
			IsPersistent:   true,
		}
		_ = s.messageSender.SendMenuMessage(chatID,
			"⏰ *Сессия завершена*\n\nВремя торговой сессии истекло. Уведомления отключены.",
			keyboard)
	}
}
