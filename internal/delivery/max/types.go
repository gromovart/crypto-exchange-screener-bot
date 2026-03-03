// internal/delivery/max/types.go
package max

import "time"

// Update — входящее обновление от MAX API (GET /updates)
type Update struct {
	UpdateType string    `json:"update_type"` // "message_created", "message_callback", "bot_started"
	Timestamp  int64     `json:"timestamp"`
	Message    *Message  `json:"message,omitempty"`
	Callback   *Callback `json:"callback,omitempty"`
	// bot_started
	ChatID int64 `json:"chat_id,omitempty"`
	User   *User `json:"user,omitempty"`
}

// Message — входящее сообщение (update_type: message_created)
type Message struct {
	Sender    User        `json:"sender"`
	Recipient Recipient   `json:"recipient"`
	Timestamp int64       `json:"timestamp"`
	Body      MessageBody `json:"body"`
}

// MessageBody — тело сообщения
type MessageBody struct {
	Mid  string `json:"mid"`
	Text string `json:"text"`
	Seq  int64  `json:"seq"`
}

// Recipient — получатель сообщения
type Recipient struct {
	ChatID   int64  `json:"chat_id"`
	ChatType string `json:"chat_type"`
	UserID   int64  `json:"user_id,omitempty"`
}

// Callback — callback от inline-кнопки (update_type: message_callback)
type Callback struct {
	Timestamp  int64    `json:"timestamp"`
	CallbackID string   `json:"callback_id"`
	User       User     `json:"user"`
	Payload    string   `json:"payload"` // данные кнопки
	Message    *Message `json:"message,omitempty"`
}

// User — пользователь MAX
type User struct {
	UserID    int64  `json:"user_id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
	IsBot     bool   `json:"is_bot,omitempty"`
}


// getUpdatesResponse — ответ GET /updates
type getUpdatesResponse struct {
	Updates []Update `json:"updates"`
	Marker  int64    `json:"marker"`
}

// sendMessageResponse — ответ POST /messages
type sendMessageResponse struct {
	Recipient Recipient   `json:"recipient"`
	Timestamp int64       `json:"timestamp"`
	Body      MessageBody `json:"body"`
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
