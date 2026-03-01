// internal/delivery/max/types.go
// MAX Bot API types — совместимы с Telegram Bot API (тот же формат JSON)
package max

import "time"

// Update — входящее обновление от MAX Bot API
type Update struct {
	UpdateID      int64          `json:"update_id"`
	Message       *Message       `json:"message,omitempty"`
	CallbackQuery *CallbackQuery `json:"callback_query,omitempty"`
}

// Message — сообщение от пользователя
type Message struct {
	MessageID int64  `json:"message_id"`
	From      User   `json:"from"`
	Chat      Chat   `json:"chat"`
	Text      string `json:"text"`
	Date      int64  `json:"date"`
}

// CallbackQuery — callback от inline-кнопки
type CallbackQuery struct {
	ID      string   `json:"id"`
	From    User     `json:"from"`
	Message *Message `json:"message,omitempty"`
	Data    string   `json:"data"`
}

// User — пользователь MAX
type User struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

// Chat — чат MAX
type Chat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

// InlineKeyboardButton — кнопка inline-клавиатуры
type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
	URL          string `json:"url,omitempty"`
}

// InlineKeyboardMarkup — разметка inline-клавиатуры
type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

// BotCommand — команда бота
type BotCommand struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

// apiResponse — базовый ответ MAX Bot API
type apiResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
}

// sendMessageResponse — ответ sendMessage с message_id
type sendMessageResponse struct {
	OK     bool `json:"ok"`
	Result *struct {
		MessageID int64 `json:"message_id"`
	} `json:"result,omitempty"`
	Description string `json:"description,omitempty"`
}

// getUpdatesResponse — ответ getUpdates
type getUpdatesResponse struct {
	OK     bool     `json:"ok"`
	Result []Update `json:"result"`
}

// RateLimiter — простой ограничитель частоты запросов
type RateLimiter struct {
	lastSent map[string]time.Time
	minDelay time.Duration
}

// NewRateLimiter создаёт ограничитель частоты
func NewRateLimiter(minDelay time.Duration) *RateLimiter {
	return &RateLimiter{
		lastSent: make(map[string]time.Time),
		minDelay: minDelay,
	}
}

// CanSend проверяет, можно ли отправить сообщение
func (rl *RateLimiter) CanSend(key string) bool {
	now := time.Now()
	if last, ok := rl.lastSent[key]; ok {
		if now.Sub(last) < rl.minDelay {
			return false
		}
	}
	rl.lastSent[key] = now
	return true
}
