// internal/delivery/max/bot/message_sender/sender.go
package message_sender

import (
	"crypto-exchange-screener-bot/internal/delivery/max"
	"crypto-exchange-screener-bot/pkg/logger"
	"log"
	"time"
)

// MessageSender — интерфейс отправки сообщений для MAX бота
type MessageSender interface {
	SendTextMessage(chatID int64, text string, keyboard interface{}) error
	SendMessageWithKeyboard(chatID int64, text string, keyboard interface{}) error
	SendCounterMessage(chatID int64, text string, keyboard interface{}) error
	SendMenuMessage(chatID int64, text string, keyboard interface{}) error
	SendMenuMessageWithID(chatID int64, text string, keyboard interface{}) (string, error)
	EditMessageText(mid, text string, keyboard interface{}) error
	DeleteMessage(mid string) error
	AnswerCallback(callbackID, notification string) error
	SetTestMode(enabled bool)
	IsTestMode() bool
}

type senderImpl struct {
	client          *max.Client
	rateLimiter     *max.RateLimiter
	menuRateLimiter *max.RateLimiter
	testMode        bool
	enabled         bool
}

// NewSender создаёт новый MessageSender для MAX
func NewSender(client *max.Client, enabled bool) MessageSender {
	return &senderImpl{
		client:          client,
		rateLimiter:     max.NewRateLimiter(2 * time.Second),
		menuRateLimiter: max.NewRateLimiter(200 * time.Millisecond),
		enabled:         enabled,
	}
}

func (s *senderImpl) SendTextMessage(chatID int64, text string, keyboard interface{}) error {
	return s.send(chatID, text, keyboard)
}

func (s *senderImpl) SendMessageWithKeyboard(chatID int64, text string, keyboard interface{}) error {
	return s.send(chatID, text, keyboard)
}

func (s *senderImpl) SendCounterMessage(chatID int64, text string, keyboard interface{}) error {
	return s.send(chatID, text, keyboard)
}

func (s *senderImpl) SendMenuMessage(chatID int64, text string, keyboard interface{}) error {
	return s.send(chatID, text, keyboard)
}

func (s *senderImpl) SendMenuMessageWithID(chatID int64, text string, keyboard interface{}) (string, error) {
	if !s.enabled || s.testMode {
		return "", nil
	}
	mid, err := s.client.SendMessageGetID(chatID, text, keyboard)
	if err != nil {
		logger.Warn("⚠️ MAX SendMenuMessageWithID: %v", err)
	}
	return mid, err
}

func (s *senderImpl) EditMessageText(mid, text string, keyboard interface{}) error {
	if !s.enabled || s.testMode {
		return nil
	}
	if err := s.client.EditMessageText(mid, text, keyboard); err != nil {
		logger.Warn("⚠️ MAX EditMessageText: %v", err)
		return err
	}
	return nil
}

func (s *senderImpl) DeleteMessage(mid string) error {
	if !s.enabled || s.testMode {
		return nil
	}
	return s.client.DeleteMessage(mid)
}

func (s *senderImpl) AnswerCallback(callbackID, notification string) error {
	if !s.enabled || s.testMode {
		return nil
	}
	return s.client.AnswerCallbackQuery(callbackID, notification)
}

func (s *senderImpl) SetTestMode(enabled bool) { s.testMode = enabled }
func (s *senderImpl) IsTestMode() bool          { return s.testMode }

// send — внутренний метод отправки
func (s *senderImpl) send(chatID int64, text string, keyboard interface{}) error {
	if !s.enabled {
		return nil
	}
	if s.testMode {
		preview := text
		if len(preview) > 50 {
			preview = preview[:50]
		}
		log.Printf("[MAX TEST] → chatID=%d: %s", chatID, preview)
		return nil
	}

	if !s.rateLimiter.CanSend("global") {
		// Не блокируем, просто логируем
		logger.Debug("⏸️ MAX rate limit (продолжаем)")
	}

	if err := s.client.SendMessageWithKeyboard(chatID, text, keyboard); err != nil {
		logger.Warn("⚠️ MAX send error: %v", err)
		return err
	}
	return nil
}
