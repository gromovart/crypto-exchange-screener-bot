// internal/delivery/telegram/app/bot/message_sender/sender.go
package message_sender

import (
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"fmt"
	"log"
	"net/http"
	"time"
)

// MessageSender интерфейс для отправки сообщений
type MessageSender interface {
	// Основные методы отправки
	SendTextMessage(chatID int64, text string, keyboard interface{}) error
	SendMessageWithKeyboard(chatID int64, text string, keyboard interface{}) error

	// Управление сообщениями
	EditMessageText(chatID, messageID int64, text string, keyboard interface{}) error
	DeleteMessage(chatID, messageID int64) error
	AnswerCallback(callbackID, text string, showAlert bool) error

	// Утилиты
	SetChatID(chatID int64)
	GetChatID() int64
	SetTestMode(enabled bool)
	IsTestMode() bool
}

// MessageSenderImpl реализация MessageSender
type MessageSenderImpl struct {
	config       *config.Config
	httpClient   *http.Client
	baseURL      string
	rateLimiter  *RateLimiter
	chatID       int64
	testMode     bool
	enabled      bool
	messageCache *MessageCache
}

// NewMessageSender создает новый MessageSender
func NewMessageSender(cfg *config.Config) MessageSender {
	chatID := ParseChatID(cfg.TelegramChatID)

	return &MessageSenderImpl{
		config:       cfg,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		baseURL:      fmt.Sprintf("https://api.telegram.org/bot%s/", cfg.TelegramBotToken),
		rateLimiter:  NewRateLimiter(2 * time.Second),
		chatID:       chatID,
		testMode:     cfg.MonitoringTestMode,
		enabled:      cfg.TelegramEnabled,
		messageCache: NewMessageCache(10 * time.Minute),
	}
}

// SendTextMessage отправляет текстовое сообщение
func (ms *MessageSenderImpl) SendTextMessage(chatID int64, text string, keyboard interface{}) error {
	return ms.sendMessage(chatID, text, keyboard, false, "")
}

// SendMessageWithKeyboard отправляет сообщение с клавиатурой
func (ms *MessageSenderImpl) SendMessageWithKeyboard(chatID int64, text string, keyboard interface{}) error {
	return ms.sendMessage(chatID, text, keyboard, false, "")
}

// EditMessageText редактирует текст сообщения
func (ms *MessageSenderImpl) EditMessageText(chatID, messageID int64, text string, keyboard interface{}) error {
	return ms.editMessage(chatID, messageID, text, keyboard, "")
}

// DeleteMessage удаляет сообщение
func (ms *MessageSenderImpl) DeleteMessage(chatID, messageID int64) error {
	if !ms.enabled || ms.testMode {
		return nil
	}

	request := map[string]interface{}{
		"chat_id":    chatID,
		"message_id": messageID,
	}

	return ms.sendTelegramRequest("deleteMessage", request)
}

// AnswerCallback отвечает на callback запрос
func (ms *MessageSenderImpl) AnswerCallback(callbackID, text string, showAlert bool) error {
	if !ms.enabled || ms.testMode {
		return nil
	}

	request := map[string]interface{}{
		"callback_query_id": callbackID,
		"text":              text,
		"show_alert":        showAlert,
	}

	return ms.sendTelegramRequest("answerCallbackQuery", request)
}

// SetChatID устанавливает chat ID
func (ms *MessageSenderImpl) SetChatID(chatID int64) {
	ms.chatID = chatID
}

// GetChatID возвращает текущий chat ID
func (ms *MessageSenderImpl) GetChatID() int64 {
	return ms.chatID
}

// SetTestMode включает/выключает тестовый режим
func (ms *MessageSenderImpl) SetTestMode(enabled bool) {
	ms.testMode = enabled
}

// IsTestMode возвращает статус тестового режима
func (ms *MessageSenderImpl) IsTestMode() bool {
	return ms.testMode
}

// sendMessage внутренний метод отправки сообщения
func (ms *MessageSenderImpl) sendMessage(chatID int64, text string, keyboard interface{}, edit bool, parseMode string) error {
	// Проверяем включен ли Telegram
	if !ms.enabled {
		log.Println("⚠️ Telegram отключен, пропуск отправки сообщения")
		return nil
	}

	// Проверяем тестовый режим
	if ms.testMode {
		log.Printf("[TEST] Send to %d: %s", chatID, text[:min(50, len(text))])
		return nil
	}

	// Проверяем rate limiting
	if !ms.rateLimiter.CanSend() {
		log.Println("⚠️ Rate limit, пропуск сообщения")
		return nil
	}

	// Проверяем дубликаты
	messageHash := ms.getMessageHash(chatID, text, keyboard)
	if ms.messageCache.IsDuplicate(messageHash) {
		log.Println("⚠️ Дубликат сообщения, пропуск")
		return nil
	}

	// Определяем метод
	method := "sendMessage"
	if edit {
		method = "editMessageText"
	}

	// Подготавливаем запрос
	request := map[string]interface{}{
		"chat_id": chatID,
		"text":    text,
	}

	// Добавляем parse_mode если указан
	if parseMode != "" {
		request["parse_mode"] = parseMode
	}

	// Добавляем клавиатуру если есть
	if keyboard != nil {
		request["reply_markup"] = keyboard
	}

	// Отправляем запрос
	err := ms.sendTelegramRequest(method, request)
	if err == nil {
		ms.messageCache.Add(messageHash)
	}

	return err
}

// editMessage редактирует сообщение
func (ms *MessageSenderImpl) editMessage(chatID, messageID int64, text string, keyboard interface{}, parseMode string) error {
	if !ms.enabled {
		return nil
	}

	if ms.testMode {
		log.Printf("[TEST] Edit message %d in chat %d: %s", messageID, chatID, text[:min(50, len(text))])
		return nil
	}

	request := map[string]interface{}{
		"chat_id":    chatID,
		"message_id": messageID,
		"text":       text,
	}

	// Добавляем parse_mode если указан
	if parseMode != "" {
		request["parse_mode"] = parseMode
	}

	if keyboard != nil {
		request["reply_markup"] = keyboard
	}

	return ms.sendTelegramRequest("editMessageText", request)
}

// getMessageHash создает хэш для проверки дубликатов
func (ms *MessageSenderImpl) getMessageHash(chatID int64, text string, keyboard interface{}) string {
	return GetMessageHash(chatID, text, keyboard)
}
