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

	// ID последнего сообщения-таймера кнопки (per userID) — удаляется перед отправкой нового
	timerMsgIDsMu sync.Mutex
	timerMsgIDs   map[int]int64

	userService   *users.Service
	messageSender message_sender.MessageSender
}

// NewService создает сервис с хранением в БД
func NewService(userService *users.Service, ms message_sender.MessageSender) Service {
	svc := &serviceImpl{
		timers:        make(map[int]*time.Timer),
		timerMsgIDs:   make(map[int]int64),
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

	// Включаем уведомления
	_ = s.userService.UpdateSettings(userID, map[string]interface{}{
		"notifications_enabled": true,
	})

	// Запускаем таймер автозавершения и обновление времени
	s.scheduleExpiryLocked(session)

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

	// Получаем сессию для chatID
	var chatID int64
	rows, _ := s.userService.FindAllActiveTradingSessions()
	for _, r := range rows {
		if r.UserID == userID {
			chatID = r.ChatID
			break
		}
	}

	// Удаляем сообщение-таймер если есть
	s.deleteTimerMessage(userID, chatID)

	// Деактивируем в БД через userService
	if err := s.userService.DeactivateTradingSession(userID); err != nil {
		return fmt.Errorf("не удалось деактивировать сессию: %w", err)
	}

	// Отключаем уведомления
	_ = s.userService.UpdateSettings(userID, map[string]interface{}{
		"notifications_enabled": false,
	})

	// Возвращаем кнопку "Начать сессию"
	if chatID != 0 {
		keyboard := telegram.ReplyKeyboardMarkup{
			Keyboard: [][]telegram.ReplyKeyboardButton{
				{{Text: constants.SessionButtonTexts.Start}},
			},
			ResizeKeyboard: true,
			IsPersistent:   true,
		}
		_ = s.messageSender.SendMenuMessage(chatID,
			"🔴 *Сессия завершена*\n\nУведомления отключены.",
			keyboard)
	}

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
		// При восстановлении отправляем актуальную клавиатуру
		s.updateSessionKeyboard(r.ChatID, r)
	}

	if len(rows) > 0 {
		logger.Info("✅ Восстановлено %d торговых сессий из БД", len(rows))
	}
}

// scheduleExpiryLocked планирует автозавершение и периодическое обновление
func (s *serviceImpl) scheduleExpiryLocked(session *models.TradingSession) {
	delay := time.Until(session.ExpiresAt)
	if delay < 0 {
		delay = 0
	}

	// Основной таймер на завершение
	s.timers[session.UserID] = time.AfterFunc(delay, func() {
		s.expire(session.UserID, session.ChatID)
	})

	// Запускаем горутину для обновления времени на кнопке
	go s.updateTimePeriodically(session)
}

// cancelTimerLocked отменяет таймер
func (s *serviceImpl) cancelTimerLocked(userID int) {
	if timer, ok := s.timers[userID]; ok {
		timer.Stop()
		delete(s.timers, userID)
	}
}

// updateTimePeriodically обновляет время на кнопке каждую минуту
func (s *serviceImpl) updateTimePeriodically(session *models.TradingSession) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			// Проверяем, активна ли еще сессия
			if _, exists := s.timers[session.UserID]; !exists {
				s.mu.Unlock()
				return
			}

			// Проверяем не истекла ли сессия
			if time.Now().After(session.ExpiresAt) {
				s.mu.Unlock()
				return
			}
			s.mu.Unlock()

			// Обновляем клавиатуру
			s.updateSessionKeyboard(session.ChatID, session)
		}
	}
}

// deleteTimerMessage удаляет предыдущее сообщение-таймер для пользователя
func (s *serviceImpl) deleteTimerMessage(userID int, chatID int64) {
	s.timerMsgIDsMu.Lock()
	oldMsgID, hasOld := s.timerMsgIDs[userID]
	if hasOld {
		delete(s.timerMsgIDs, userID)
	}
	s.timerMsgIDsMu.Unlock()

	if hasOld && chatID != 0 && s.messageSender != nil {
		_ = s.messageSender.DeleteMessage(chatID, oldMsgID)
	}
}

// updateSessionKeyboard обновляет кнопку таймера: удаляет старое сообщение, отправляет новое
func (s *serviceImpl) updateSessionKeyboard(chatID int64, session *models.TradingSession) {
	if s.messageSender == nil {
		return
	}

	// Удаляем предыдущее сообщение-таймер (если есть), чтобы не накапливался спам
	s.deleteTimerMessage(session.UserID, chatID)

	remaining := FormatRemaining(session.ExpiresAt)
	stopButtonText := fmt.Sprintf("%s (%s)", constants.SessionButtonTexts.Stop, remaining)

	keyboard := telegram.ReplyKeyboardMarkup{
		Keyboard: [][]telegram.ReplyKeyboardButton{
			{{Text: stopButtonText}},
		},
		ResizeKeyboard: true,
		IsPersistent:   true,
	}

	// Отправляем новое сообщение с обновлённой кнопкой и сохраняем его ID
	msgID, err := s.messageSender.SendMenuMessageWithID(chatID, fmt.Sprintf("🕐 *%s*", remaining), keyboard)
	if err != nil || msgID == 0 {
		return
	}

	s.timerMsgIDsMu.Lock()
	s.timerMsgIDs[session.UserID] = msgID
	s.timerMsgIDsMu.Unlock()
}

// expire автоматически завершает сессию по истечении времени
func (s *serviceImpl) expire(userID int, chatID int64) {
	logger.Info("⏰ Торговая сессия истекла для пользователя %d", userID)

	s.mu.Lock()
	delete(s.timers, userID)
	s.mu.Unlock()

	// Удаляем сообщение-таймер
	s.deleteTimerMessage(userID, chatID)

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

	// Возвращаем обычную кнопку "Начать сессию"
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
