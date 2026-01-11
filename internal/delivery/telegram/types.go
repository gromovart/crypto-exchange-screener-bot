// internal/delivery/telegram/types.go
package telegram

import (
	"sync"
	"time"
)

// RateLimiter - ограничитель частоты запросов
type RateLimiter struct {
	mu       sync.Mutex
	lastSent map[string]time.Time
	minDelay time.Duration
}

// TelegramResponse - ответ от Telegram API
type TelegramResponse struct {
	OK     bool `json:"ok"`
	Result struct {
		MessageID int `json:"message_id"`
	} `json:"result"`
}

// InlineKeyboardButton - кнопка inline клавиатуры
type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
	URL          string `json:"url,omitempty"`
}

// InlineKeyboardMarkup - разметка inline клавиатуры
type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

// TelegramMessage - сообщение с клавиатурой
type TelegramMessage struct {
	ChatID string `json:"chat_id"`
	Text   string `json:"text"`
	// ParseMode   string      `json:"parse_mode,omitempty"`
	ReplyMarkup interface{} `json:"reply_markup,omitempty"`
}

// ReplyKeyboardButton - кнопка reply клавиатуры
type ReplyKeyboardButton struct {
	Text string `json:"text"`
}

// ReplyKeyboardMarkup - разметка reply клавиатуры
type ReplyKeyboardMarkup struct {
	Keyboard        [][]ReplyKeyboardButton `json:"keyboard"`
	ResizeKeyboard  bool                    `json:"resize_keyboard,omitempty"`
	OneTimeKeyboard bool                    `json:"one_time_keyboard,omitempty"`
	RemoveKeyboard  bool                    `json:"remove_keyboard,omitempty"`
	Selective       bool                    `json:"selective,omitempty"`
	IsPersistent    bool                    `json:"is_persistent,omitempty"` // Новая опция
}

// NewRateLimiter создает новый ограничитель частоты
func NewRateLimiter(minDelay time.Duration) *RateLimiter {
	return &RateLimiter{
		lastSent: make(map[string]time.Time),
		minDelay: minDelay,
	}
}

// CanSend проверяет, можно ли отправить сообщение
func (rl *RateLimiter) CanSend(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	if last, exists := rl.lastSent[key]; exists {
		if now.Sub(last) < rl.minDelay {
			return false
		}
	}
	rl.lastSent[key] = now
	return true
}
