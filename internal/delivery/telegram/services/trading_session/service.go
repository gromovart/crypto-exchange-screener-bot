// internal/delivery/telegram/services/trading_session/service.go
package trading_session

import (
	"fmt"
	"sync"
	"time"

	"crypto-exchange-screener-bot/internal/core/domain/users"
	"crypto-exchange-screener-bot/internal/delivery/telegram"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/message_sender"
	"crypto-exchange-screener-bot/pkg/logger"
)

// serviceImpl реализация Service с in-memory хранилищем
type serviceImpl struct {
	mu       sync.Mutex
	sessions map[int]*TradingSession // userID → session
	timers   map[int]*time.Timer     // userID → expiry timer

	userService   *users.Service
	messageSender message_sender.MessageSender
}

// NewService создает новый сервис торговых сессий
func NewService(userService *users.Service, ms message_sender.MessageSender) Service {
	return &serviceImpl{
		sessions:      make(map[int]*TradingSession),
		timers:        make(map[int]*time.Timer),
		userService:   userService,
		messageSender: ms,
	}
}

// Start запускает торговую сессию для пользователя
func (s *serviceImpl) Start(userID int, chatID int64, duration time.Duration) (*TradingSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Останавливаем предыдущую сессию если есть
	if existing, ok := s.sessions[userID]; ok {
		s.cancelTimerLocked(userID)
		logger.Info("🔄 Перезапуск торговой сессии для пользователя %d (предыдущая: %s)", userID, existing.ExpiresAt.Format("15:04:05"))
	}

	session := &TradingSession{
		UserID:    userID,
		ChatID:    chatID,
		StartedAt: time.Now(),
		ExpiresAt: time.Now().Add(duration),
	}

	s.sessions[userID] = session

	// Включаем уведомления
	if s.userService != nil {
		if err := s.userService.UpdateSettings(userID, map[string]interface{}{
			"notifications_enabled": true,
		}); err != nil {
			logger.Warn("⚠️ Не удалось включить уведомления для пользователя %d: %v", userID, err)
		}
	}

	// Запускаем таймер автозавершения
	s.scheduleExpiryLocked(session)

	logger.Info("✅ Торговая сессия запущена для пользователя %d, истекает: %s", userID, session.ExpiresAt.Format("15:04:05"))
	return session, nil
}

// Stop завершает торговую сессию пользователя
func (s *serviceImpl) Stop(userID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.sessions[userID]; !ok {
		return fmt.Errorf("активная сессия для пользователя %d не найдена", userID)
	}

	s.cancelTimerLocked(userID)
	delete(s.sessions, userID)

	// Отключаем уведомления
	if s.userService != nil {
		if err := s.userService.UpdateSettings(userID, map[string]interface{}{
			"notifications_enabled": false,
		}); err != nil {
			logger.Warn("⚠️ Не удалось выключить уведомления для пользователя %d: %v", userID, err)
		}
	}

	logger.Info("✅ Торговая сессия завершена для пользователя %d", userID)
	return nil
}

// GetActive возвращает активную сессию пользователя
func (s *serviceImpl) GetActive(userID int) (*TradingSession, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[userID]
	if !ok {
		return nil, false
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, false
	}

	return session, true
}

// IsActive проверяет наличие активной сессии
func (s *serviceImpl) IsActive(userID int) bool {
	_, ok := s.GetActive(userID)
	return ok
}

// scheduleExpiryLocked планирует автозавершение (вызывается под мьютексом)
func (s *serviceImpl) scheduleExpiryLocked(session *TradingSession) {
	delay := time.Until(session.ExpiresAt)
	if delay < 0 {
		delay = 0
	}

	userID := session.UserID
	chatID := session.ChatID

	s.timers[userID] = time.AfterFunc(delay, func() {
		s.expire(userID, chatID)
	})
}

// cancelTimerLocked отменяет таймер (вызывается под мьютексом)
func (s *serviceImpl) cancelTimerLocked(userID int) {
	if timer, ok := s.timers[userID]; ok {
		timer.Stop()
		delete(s.timers, userID)
	}
}

// FormatRemaining возвращает строку вида "1ч 45м", "45м" или "< 1м"
func FormatRemaining(expiresAt time.Time) string {
	remaining := time.Until(expiresAt)
	if remaining <= 0 {
		return "< 1м"
	}
	h := int(remaining.Hours())
	m := int(remaining.Minutes()) % 60
	if h > 0 && m > 0 {
		return fmt.Sprintf("%dч %dм", h, m)
	} else if h > 0 {
		return fmt.Sprintf("%dч", h)
	}
	if m == 0 {
		return "< 1м"
	}
	return fmt.Sprintf("%dм", m)
}

// expire автоматически завершает сессию по истечении времени
func (s *serviceImpl) expire(userID int, chatID int64) {
	logger.Info("⏰ Торговая сессия истекла для пользователя %d", userID)

	s.mu.Lock()
	delete(s.sessions, userID)
	delete(s.timers, userID)
	s.mu.Unlock()

	// Отключаем уведомления
	if s.userService != nil {
		if err := s.userService.UpdateSettings(userID, map[string]interface{}{
			"notifications_enabled": false,
		}); err != nil {
			logger.Warn("⚠️ Не удалось выключить уведомления при автозавершении сессии пользователя %d: %v", userID, err)
		}
	}

	// Отправляем уведомление с кнопкой "Начать сессию"
	if s.messageSender != nil {
		keyboard := telegram.ReplyKeyboardMarkup{
			Keyboard: [][]telegram.ReplyKeyboardButton{
				{{Text: "🟢 Начать сессию"}},
			},
			ResizeKeyboard: true,
			IsPersistent:   true,
		}
		if err := s.messageSender.SendMenuMessage(chatID, "⏰ *Сессия завершена*\n\nВремя торговой сессии истекло. Уведомления отключены.", keyboard); err != nil {
			logger.Warn("⚠️ Не удалось отправить уведомление о завершении сессии: %v", err)
		}
	}
}
