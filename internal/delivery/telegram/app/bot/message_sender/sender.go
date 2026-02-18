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
	// Отправка без rate limiting (для counter уведомлений)
	SendCounterMessage(chatID int64, text string, keyboard interface{}) error
	// Отправка сообщений меню (приоритетные, с защитой от блокировок)
	SendMenuMessage(chatID int64, text string, keyboard interface{}) error

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
	config          *config.Config
	httpClient      *http.Client
	baseURL         string
	rateLimiter     *RateLimiter
	menuRateLimiter *RateLimiter // Отдельный rate limiter для меню
	chatID          int64
	testMode        bool
	enabled         bool
	messageCache    *MessageCache
}

// NewMessageSender создает новый MessageSender
func NewMessageSender(cfg *config.Config) MessageSender {
	chatID := ParseChatID(cfg.TelegramChatID)

	return &MessageSenderImpl{
		config:          cfg,
		httpClient:      &http.Client{Timeout: 30 * time.Second},
		baseURL:         fmt.Sprintf("https://api.telegram.org/bot%s/", cfg.TelegramBotToken),
		rateLimiter:     NewRateLimiter(2 * time.Second),        // Для уведомлений
		menuRateLimiter: NewRateLimiter(200 * time.Millisecond), // Для меню (быстрее)
		chatID:          chatID,
		testMode:        cfg.MonitoringTestMode,
		enabled:         cfg.Telegram.Enabled,
		messageCache:    NewMessageCache(10 * time.Minute),
	}
}

// SendCounterMessage отправляет counter уведомление без rate limiting
func (ms *MessageSenderImpl) SendCounterMessage(chatID int64, text string, keyboard interface{}) error {
	return ms.sendMessageWithoutRateLimit(chatID, text, keyboard, "counter")
}

// SendMenuMessage отправляет сообщение меню (приоритетное, с отдельным rate limiter)
func (ms *MessageSenderImpl) SendMenuMessage(chatID int64, text string, keyboard interface{}) error {
	return ms.sendMenuMessage(chatID, text, keyboard)
}

// sendMessageWithoutRateLimit внутренний метод без rate limiting
func (ms *MessageSenderImpl) sendMessageWithoutRateLimit(chatID int64, text string, keyboard interface{}, msgType string) error {
	// Проверяем включен ли Telegram
	if !ms.enabled {
		log.Println("⚠️ Telegram отключен, пропуск отправки сообщения")
		return nil
	}

	// Проверяем тестовый режим
	if ms.testMode {
		log.Printf("[TEST] Send %s to %d: %s", msgType, chatID, text[:min(50, len(text))])
		return nil
	}

	// Проверяем дубликаты (защита от спама)
	messageHash := ms.getMessageHash(chatID, text, keyboard)
	if ms.messageCache.IsDuplicate(messageHash) {
		log.Printf("⚠️ Дубликат %s сообщения, пропуск", msgType)
		return nil
	}

	// Отправляем запрос
	request := map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
	}

	if keyboard != nil {
		request["reply_markup"] = keyboard
	}

	err := ms.sendTelegramRequest("sendMessage", request)
	if err == nil {
		ms.messageCache.Add(messageHash)
	} else {
		log.Printf("❌ Ошибка отправки %s сообщения: %v", msgType, err)
	}

	return err
}

// sendMenuMessage отправляет сообщение меню с отдельным rate limiter
func (ms *MessageSenderImpl) sendMenuMessage(chatID int64, text string, keyboard interface{}) error {
	// Проверяем включен ли Telegram
	if !ms.enabled {
		log.Println("⚠️ Telegram отключен, пропуск отправки сообщения")
		return nil
	}

	// Проверяем тестовый режим
	if ms.testMode {
		log.Printf("[TEST] Send menu to %d: %s", chatID, text[:min(50, len(text))])
		return nil
	}

	// Используем отдельный rate limiter для меню
	if !ms.menuRateLimiter.CanSend() {
		log.Println("⚠️ Menu rate limit, но отправляем (приоритетное сообщение)")
		// Не блокируем меню, а просто логируем и отправляем
	}

	// Проверяем дубликаты (более мягко для меню)
	messageHash := ms.getMessageHash(chatID, text, keyboard)
	if ms.messageCache.IsDuplicate(messageHash) {
		log.Println("⚠️ Дубликат menu сообщения, но обновляем (меню)")
		// Для меню разрешаем обновление даже если был дубликат
	}

	// Отправляем запрос
	request := map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
	}

	if keyboard != nil {
		request["reply_markup"] = keyboard
	}

	err := ms.sendTelegramRequest("sendMessage", request)
	if err == nil {
		ms.messageCache.Add(messageHash)
	} else {
		log.Printf("❌ Ошибка отправки menu сообщения: %v", err)
	}

	return err
}

// SendTextMessage отправляет текстовое сообщение
func (ms *MessageSenderImpl) SendTextMessage(chatID int64, text string, keyboard interface{}) error {
	return ms.sendMessage(chatID, text, keyboard, false, "Markdown")
}

// SendMessageWithKeyboard отправляет сообщение с клавиатурой
func (ms *MessageSenderImpl) SendMessageWithKeyboard(chatID int64, text string, keyboard interface{}) error {
	return ms.sendMessage(chatID, text, keyboard, false, "Markdown")
}

// EditMessageText редактирует текст сообщения
func (ms *MessageSenderImpl) EditMessageText(chatID, messageID int64, text string, keyboard interface{}) error {
	return ms.editMessage(chatID, messageID, text, keyboard, "Markdown")
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
		log.Println("⚠️ Rate limit, но отправляем (приоритетное сообщение)")
		// Не блокируем, а просто логируем
	}

	// Проверяем дубликаты
	messageHash := ms.getMessageHash(chatID, text, keyboard)
	if ms.messageCache.IsDuplicate(messageHash) {
		log.Println("⚠️ Дубликат сообщения, но отправляем (обновление)")
		// Для важных сообщений разрешаем
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
	} else {
		log.Printf("❌ Ошибка отправки сообщения: %v", err)
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
