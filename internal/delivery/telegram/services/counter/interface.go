// internal/delivery/telegram/services/counter/interface.go
package counter

import (
	"crypto-exchange-screener-bot/internal/core/domain/users"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/buttons"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/formatters"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/message_sender"
	"crypto-exchange-screener-bot/internal/types"
)

type Service interface {
	Exec(params interface{}) (interface{}, error)
}

// CounterParams параметры для Exec
type CounterParams struct {
	Event types.Event `json:"event"`
}

// CounterResult результат Exec
type CounterResult struct {
	Processed bool   `json:"processed"`
	Message   string `json:"message,omitempty"`
	SentTo    int    `json:"sent_to,omitempty"`
}

// NewServiceWithDependencies фабрика с зависимостями
func NewServiceWithDependencies(
	userService *users.Service,
	formatter *formatters.FormatterProvider,
	messageSender message_sender.MessageSender,
	buttonBuilder *buttons.ButtonBuilder, // ДОБАВЛЕНО
) Service {
	return NewService(userService, formatter, messageSender, buttonBuilder)
}
