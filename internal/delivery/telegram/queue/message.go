// internal/delivery/telegram/queue/message.go
package queue

import "time"

// Priority названия Redis-списков по приоритету
type Priority string

const (
	PriorityHigh   Priority = "tg:queue:high"
	PriorityNormal Priority = "tg:queue:normal"
	PriorityLow    Priority = "tg:queue:low"
)

// MessageTTL — сигнал старше этого времени не имеет смысла отправлять
const MessageTTL = 5 * time.Minute

// QueuedMessage сообщение в очереди
type QueuedMessage struct {
	ChatID     int64       `json:"chat_id"`
	Text       string      `json:"text"`
	Keyboard   interface{} `json:"keyboard,omitempty"`
	Priority   Priority    `json:"priority"`
	Attempts   int         `json:"attempts"`
	CreatedAt  time.Time   `json:"created_at"`
	RetryAfter time.Time   `json:"retry_after,omitempty"`
}
