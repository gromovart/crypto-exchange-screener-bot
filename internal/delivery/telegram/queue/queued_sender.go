// internal/delivery/telegram/queue/queued_sender.go
package queue

import (
	"context"
	"time"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/message_sender"
)

// QueuedMessageSender реализует MessageSender:
//   - SendTextMessage / SendMessageWithKeyboard / SendCounterMessage → Redis очередь
//   - SendMenuMessage / EditMessageText / DeleteMessage / AnswerCallback → прямая отправка
//
// Это позволяет держать интерактивные ответы (меню, callbacks) мгновенными,
// а сигналы и уведомления обрабатывать через rate-limited очередь.
type QueuedMessageSender struct {
	direct   message_sender.MessageSender
	producer *Producer
}

// NewQueuedMessageSender создаёт обёртку над прямым sender'ом
func NewQueuedMessageSender(direct message_sender.MessageSender, producer *Producer) *QueuedMessageSender {
	return &QueuedMessageSender{
		direct:   direct,
		producer: producer,
	}
}

// SendTextMessage → очередь normal (сигналы, уведомления)
func (q *QueuedMessageSender) SendTextMessage(chatID int64, text string, keyboard interface{}) error {
	return q.producer.Enqueue(context.Background(), QueuedMessage{
		ChatID:    chatID,
		Text:      text,
		Keyboard:  keyboard,
		Priority:  PriorityNormal,
		CreatedAt: time.Now(),
	})
}

// SendMessageWithKeyboard → очередь normal
func (q *QueuedMessageSender) SendMessageWithKeyboard(chatID int64, text string, keyboard interface{}) error {
	return q.producer.Enqueue(context.Background(), QueuedMessage{
		ChatID:    chatID,
		Text:      text,
		Keyboard:  keyboard,
		Priority:  PriorityNormal,
		CreatedAt: time.Now(),
	})
}

// SendCounterMessage → очередь high (приоритетные уведомления)
func (q *QueuedMessageSender) SendCounterMessage(chatID int64, text string, keyboard interface{}) error {
	return q.producer.Enqueue(context.Background(), QueuedMessage{
		ChatID:    chatID,
		Text:      text,
		Keyboard:  keyboard,
		Priority:  PriorityHigh,
		CreatedAt: time.Now(),
	})
}

// SendMenuMessage → прямая отправка (ответы на команды, меню)
func (q *QueuedMessageSender) SendMenuMessage(chatID int64, text string, keyboard interface{}) error {
	return q.direct.SendMenuMessage(chatID, text, keyboard)
}

// SendMenuMessageWithID → прямая отправка с возвратом message_id
func (q *QueuedMessageSender) SendMenuMessageWithID(chatID int64, text string, keyboard interface{}) (int64, error) {
	return q.direct.SendMenuMessageWithID(chatID, text, keyboard)
}

// EditMessageText → прямая отправка
func (q *QueuedMessageSender) EditMessageText(chatID, messageID int64, text string, keyboard interface{}) error {
	return q.direct.EditMessageText(chatID, messageID, text, keyboard)
}

// DeleteMessage → прямая отправка
func (q *QueuedMessageSender) DeleteMessage(chatID, messageID int64) error {
	return q.direct.DeleteMessage(chatID, messageID)
}

// AnswerCallback → прямая отправка
func (q *QueuedMessageSender) AnswerCallback(callbackID, text string, showAlert bool) error {
	return q.direct.AnswerCallback(callbackID, text, showAlert)
}

func (q *QueuedMessageSender) SetChatID(chatID int64)    { q.direct.SetChatID(chatID) }
func (q *QueuedMessageSender) GetChatID() int64          { return q.direct.GetChatID() }
func (q *QueuedMessageSender) SetTestMode(enabled bool)  { q.direct.SetTestMode(enabled) }
func (q *QueuedMessageSender) IsTestMode() bool          { return q.direct.IsTestMode() }
