// internal/types/telegram/telegram.go
package telegram

import (
	"crypto_exchange_screener_bot/internal/types/analysis"
	"time"
)

// Chat - чат телеграм
type Chat struct {
	ID        int64     `json:"id"`
	Type      string    `json:"type"` // "private", "group", "supergroup", "channel"
	Title     string    `json:"title,omitempty"`
	Username  string    `json:"username,omitempty"`
	FirstName string    `json:"first_name,omitempty"`
	LastName  string    `json:"last_name,omitempty"`
	Active    bool      `json:"active"`
	JoinedAt  time.Time `json:"joined_at"`
}

// Message - сообщение телеграм
type Message struct {
	ID        int                    `json:"id"`
	ChatID    int64                  `json:"chat_id"`
	Text      string                 `json:"text"`
	Timestamp time.Time              `json:"timestamp"`
	UserID    int64                  `json:"user_id,omitempty"`
	IsCommand bool                   `json:"is_command"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// BotConfig - конфигурация бота
type BotConfig struct {
	Token         string            `json:"token"`
	WebhookURL    string            `json:"webhook_url,omitempty"`
	AllowedChats  []int64           `json:"allowed_chats,omitempty"`
	AdminIDs      []int64           `json:"admin_ids"`
	UpdateTimeout int               `json:"update_timeout"`
	EnableLogging bool              `json:"enable_logging"`
	Commands      map[string]string `json:"commands,omitempty"`
}

// Notification - уведомление
type Notification struct {
	ChatID    int64                        `json:"chat_id"`
	Signal    analysis.Signal              `json:"signal,omitempty"`
	Counter   analysis.CounterNotification `json:"counter,omitempty"`
	Text      string                       `json:"text"`
	Priority  int                          `json:"priority"` // 1-5, где 5 - высший
	Timestamp time.Time                    `json:"timestamp"`
	Format    string                       `json:"format"` // "text", "html", "markdown"
}

// BotStats - статистика бота
type BotStats struct {
	TotalMessages      int           `json:"total_messages"`
	TotalCommands      int           `json:"total_commands"`
	TotalNotifications int           `json:"total_notifications"`
	ActiveChats        int           `json:"active_chats"`
	Uptime             time.Duration `json:"uptime"`
	LastActivity       time.Time     `json:"last_activity"`
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
	ChatID      string      `json:"chat_id"`
	Text        string      `json:"text"`
	ParseMode   string      `json:"parse_mode,omitempty"`
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
