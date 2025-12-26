package telegram

import (
	"bytes"
	"crypto-exchange-screener-bot/internal/config"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

// MessageSender - отправитель сообщений
type MessageSender struct {
	config         *config.Config
	baseURL        string
	httpClient     *http.Client
	rateLimiter    *RateLimiter
	lastSendTime   time.Time
	minInterval    time.Duration
	messageCache   map[string]time.Time // Кэш отправленных сообщений
	messageCacheMu sync.RWMutex
	cacheTTL       time.Duration
	chatID         string              // Добавленное поле для текущего chat_id
	replyMarkup    ReplyKeyboardMarkup // Добавленное поле для клавиатуры
}

// NewMessageSender создает новый отправитель сообщений
func NewMessageSender(cfg *config.Config) *MessageSender {
	return &MessageSender{
		config:       cfg,
		baseURL:      fmt.Sprintf("https://api.telegram.org/bot%s/", cfg.TelegramBotToken),
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		rateLimiter:  NewRateLimiter(2 * time.Second),
		minInterval:  2 * time.Second,
		messageCache: make(map[string]time.Time),
		cacheTTL:     10 * time.Minute,      // Храним 10 минут
		chatID:       cfg.TelegramChatID,    // Используем chat_id из конфига
		replyMarkup:  ReplyKeyboardMarkup{}, // Инициализируем пустую клавиатуру
	}
}

// NewMessageSenderWithChatID создает отправитель с конкретным chat_id
func NewMessageSenderWithChatID(cfg *config.Config, chatID string) *MessageSender {
	return &MessageSender{
		config:       cfg,
		baseURL:      fmt.Sprintf("https://api.telegram.org/bot%s/", cfg.TelegramBotToken),
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		rateLimiter:  NewRateLimiter(2 * time.Second),
		minInterval:  2 * time.Second,
		messageCache: make(map[string]time.Time),
		cacheTTL:     10 * time.Minute,
		chatID:       chatID, // Устанавливаем указанный chat_id
		replyMarkup:  ReplyKeyboardMarkup{},
	}
}

// WithChatID создает копию MessageSender с другим chat_id
func (ms *MessageSender) WithChatID(chatID string) *MessageSender {
	return &MessageSender{
		config:         ms.config,
		baseURL:        ms.baseURL,
		httpClient:     ms.httpClient,
		rateLimiter:    ms.rateLimiter,
		lastSendTime:   ms.lastSendTime,
		minInterval:    ms.minInterval,
		messageCache:   ms.messageCache, // Разделяем кэш
		messageCacheMu: ms.messageCacheMu,
		cacheTTL:       ms.cacheTTL,
		chatID:         chatID,         // Устанавливаем новый chat_id
		replyMarkup:    ms.replyMarkup, // Копируем клавиатуру
	}
}

// Вспомогательная функция для создания хэша сообщения
func (ms *MessageSender) getMessageHash(chatID, text string, keyboard interface{}) string {
	// Создаем простой хэш для проверки дубликатов
	data := fmt.Sprintf("%s:%s:%v", chatID, text, keyboard)
	return data // В реальной реализации можно использовать md5 или sha256
}

// Проверяем, не отправляли ли мы уже это сообщение
func (ms *MessageSender) isDuplicateMessage(hash string) bool {
	ms.messageCacheMu.RLock()
	lastSent, exists := ms.messageCache[hash]
	ms.messageCacheMu.RUnlock()

	if !exists {
		return false
	}

	// Проверяем TTL
	if time.Since(lastSent) > ms.cacheTTL {
		return false // Истек срок жизни кэша
	}

	// Проверяем интервал между одинаковыми сообщениями
	return time.Since(lastSent) < 30*time.Second // 30 секунд между одинаковыми сообщениями
}

// Добавляем сообщение в кэш
func (ms *MessageSender) cacheMessage(hash string) {
	ms.messageCacheMu.Lock()
	defer ms.messageCacheMu.Unlock()

	// Очищаем старые записи
	now := time.Now()
	for key, timestamp := range ms.messageCache {
		if now.Sub(timestamp) > ms.cacheTTL {
			delete(ms.messageCache, key)
		}
	}

	ms.messageCache[hash] = now
}

// SendTextMessage отправляет текстовое сообщение
func (ms *MessageSender) SendTextMessage(text string, keyboard *InlineKeyboardMarkup, hideMenu bool) error {
	if !ms.config.TelegramEnabled && hideMenu {
		return nil
	}

	// Проверяем лимит частоты
	key := "message"
	if !ms.rateLimiter.CanSend(key) {
		log.Printf("⚠️ Пропуск Telegram сообщения (лимит частоты)")
		return nil // Возвращаем nil вместо ошибки
	}

	// Проверяем минимальный интервал
	now := time.Now()
	if now.Sub(ms.lastSendTime) < ms.minInterval {
		sleepTime := ms.minInterval - now.Sub(ms.lastSendTime)
		time.Sleep(sleepTime)
	}

	// Проверяем дубликаты - используем ms.chatID вместо ms.config.TelegramChatID
	messageHash := ms.getMessageHash(ms.chatID, text, keyboard)
	if ms.isDuplicateMessage(messageHash) {
		log.Printf("⚠️ Пропуск дублирующегося сообщения")
		return nil
	}

	message := TelegramMessage{
		ChatID:    ms.chatID, // Используем ms.chatID
		Text:      text,
		ParseMode: "Markdown",
	}

	if keyboard != nil {
		message.ReplyMarkup = keyboard
	}

	err := ms.SendTelegramRequest("sendMessage", message)
	if err == nil {
		ms.cacheMessage(messageHash)
		ms.lastSendTime = time.Now()
	}

	return err
}

// SendMessageToChat отправляет сообщение в указанный чат
func (ms *MessageSender) SendMessageToChat(chatID string, text string, keyboard *InlineKeyboardMarkup) error {
	message := TelegramMessage{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "Markdown",
	}

	if keyboard != nil {
		message.ReplyMarkup = keyboard
	}

	return ms.SendTelegramRequest("sendMessage", message)
}

// SendTestMessage отправляет тестовое сообщение
func (ms *MessageSender) SendTestMessage() error {
	message := "🤖 *Бот активирован!*\n\n" +
		"✅ Система мониторинга роста/падения запущена.\n" +
		"🔔 Уведомления отправляются с ограничением 1 сообщение в 10 секунд.\n" +
		"⚡ Настройки: рост=%.2f%%, падение=%.2f%%"

	// Используем настройки из конфигурации анализаторов
	growthThreshold := ms.config.Analyzers.GrowthAnalyzer.MinGrowth
	fallThreshold := ms.config.Analyzers.FallAnalyzer.MinFall

	message = fmt.Sprintf(message, growthThreshold, fallThreshold)

	return ms.SendTextMessage(message, nil, false)
}

// SetReplyKeyboard устанавливает reply клавиатуру
func (ms *MessageSender) SetReplyKeyboard(keyboard ReplyKeyboardMarkup) error {
	message := struct {
		ChatID      string              `json:"chat_id"`
		Text        string              `json:"text"`
		ReplyMarkup ReplyKeyboardMarkup `json:"reply_markup,omitempty"`
	}{
		ChatID:      ms.chatID, // Используем ms.chatID
		Text:        "⚙️ *Меню настроек активировано*\n\nВсе настройки доступны в меню ниже ⬇️",
		ReplyMarkup: keyboard,
	}

	return ms.SendTelegramRequest("sendMessage", message)
}

// SendTelegramRequest - общая функция для отправки запросов к Telegram API
func (ms *MessageSender) SendTelegramRequest(method string, payload interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	resp, err := ms.httpClient.Post(
		ms.baseURL+method,
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var telegramResp struct {
		OK          bool   `json:"ok"`
		ErrorCode   int    `json:"error_code,omitempty"`
		Description string `json:"description,omitempty"`
	}

	if err := json.Unmarshal(body, &telegramResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !telegramResp.OK {
		// Если ошибка 429, ждем указанное время
		if telegramResp.ErrorCode == 429 {
			retryAfter := 5 // по умолчанию 5 секунд
			var retryResp struct {
				Parameters struct {
					RetryAfter int `json:"retry_after"`
				} `json:"parameters"`
			}
			if json.Unmarshal(body, &retryResp) == nil && retryResp.Parameters.RetryAfter > 0 {
				retryAfter = retryResp.Parameters.RetryAfter
			}
			log.Printf("⚠️ Telegram API лимит, ждем %d секунд", retryAfter)
			time.Sleep(time.Duration(retryAfter) * time.Second)
			// Пробуем снова один раз
			return ms.SendTelegramRequest(method, payload)
		}
		return fmt.Errorf("telegram API error %d: %s", telegramResp.ErrorCode, telegramResp.Description)
	}

	ms.lastSendTime = time.Now()
	return nil
}

// GetChatID возвращает текущий chat_id
func (ms *MessageSender) GetChatID() string {
	return ms.chatID
}

// SetChatID устанавливает chat_id
func (ms *MessageSender) SetChatID(chatID string) {
	ms.chatID = chatID
}
