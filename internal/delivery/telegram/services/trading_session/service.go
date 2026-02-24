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

// serviceImpl реализация Service с хранением в БД
type serviceImpl struct {
	mu     sync.Mutex
	timers map[int]*time.Timer

	userService   *users.Service
	messageSender message_sender.MessageSender
}

// NewService создает сервис с хранением в БД
func NewService(userService *users.Service, ms message_sender.MessageSender) Service {
	svc := &serviceImpl{
		timers:        make(map[int]*time.Timer),
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
		StartedAt: m.StartedAt,
		ExpiresAt: m.ExpiresAt,
	}
}

// toModel конвертирует TradingSession в models.TradingSession
func (s *serviceImpl) toModel(dto *TradingSession) *models.TradingSession {
	if dto == nil {
		return nil
	}
	return &models.TradingSession{
		UserID:    dto.UserID,
		ChatID:    dto.ChatID,
		StartedAt: dto.StartedAt,
		ExpiresAt: dto.ExpiresAt,
		IsActive:  true,
	}
}

// Start запускает торговую сессию для пользователя
func (s *serviceImpl) Start(userID int, chatID int64, duration time.Duration) (*TradingSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.userService == nil {
		return nil, fmt.Errorf("userService не доступен")
	}

	// Останавливаем таймер если был
	s.cancelTimerLocked(userID)

	session := &models.TradingSession{
		UserID:    userID,
		ChatID:    chatID,
		StartedAt: time.Now(),
		ExpiresAt: time.Now().Add(duration),
		IsActive:  true,
	}

	// Сохраняем в БД через userService
	if err := s.userService.SaveTradingSession(session); err != nil {
		return nil, fmt.Errorf("не удалось сохранить сессию: %w", err)
	}

	// Запускаем таймер автозавершения
	s.scheduleExpiryLocked(session)

	// Включаем уведомления
	_ = s.userService.UpdateSettings(userID, map[string]interface{}{
		"notifications_enabled": true,
	})

	logger.Info("✅ Торговая сессия запущена для пользователя %d", userID)
	return s.toDTO(session), nil
}

// Stop завершает торговую сессию пользователя
func (s *serviceImpl) Stop(userID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.userService == nil {
		return fmt.Errorf("userService не доступен")
	}

	s.cancelTimerLocked(userID)

	// Деактивируем в БД через userService
	if err := s.userService.DeactivateTradingSession(userID); err != nil {
		return fmt.Errorf("не удалось деактивировать сессию: %w", err)
	}

	// Отключаем уведомления
	_ = s.userService.UpdateSettings(userID, map[string]interface{}{
		"notifications_enabled": false,
	})

	logger.Info("✅ Торговая сессия завершена для пользователя %d", userID)
	return nil
}

// GetActive возвращает активную сессию пользователя
func (s *serviceImpl) GetActive(userID int) (*TradingSession, bool) {
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
		if session.UserID == userID {
			return s.toDTO(session), true
		}
	}
	return nil, false
}

// IsActive проверяет наличие активной сессии
func (s *serviceImpl) IsActive(userID int) bool {
	_, ok := s.GetActive(userID)
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

// scheduleExpiryLocked планирует автозавершение
func (s *serviceImpl) scheduleExpiryLocked(session *models.TradingSession) {
	delay := time.Until(session.ExpiresAt)
	if delay < 0 {
		delay = 0
	}

	s.timers[session.UserID] = time.AfterFunc(delay, func() {
		s.expire(session.UserID, session.ChatID)
	})
}

// cancelTimerLocked отменяет таймер
func (s *serviceImpl) cancelTimerLocked(userID int) {
	if timer, ok := s.timers[userID]; ok {
		timer.Stop()
		delete(s.timers, userID)
	}
}

// expire автоматически завершает сессию по истечении времени
func (s *serviceImpl) expire(userID int, chatID int64) {
	logger.Info("⏰ Торговая сессия истекла для пользователя %d", userID)

	s.mu.Lock()
	delete(s.timers, userID)
	s.mu.Unlock()

	if s.userService != nil {
		// Деактивируем в БД
		if err := s.userService.DeactivateTradingSession(userID); err != nil {
			logger.Warn("⚠️ Не удалось деактивировать истекшую сессию: %v", err)
		}

		// Отключаем уведомления
		_ = s.userService.UpdateSettings(userID, map[string]interface{}{
			"notifications_enabled": false,
		})
	}

	// Отправляем уведомление
	if s.messageSender != nil {
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
