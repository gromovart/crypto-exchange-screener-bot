// internal/delivery/telegram/app/bot/handlers/types.go
package handlers

import "crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"

// HandlerType тип хэндлера (дублирует router.HandlerType для избежания импорта)
type HandlerType string

const (
	TypeCommand  HandlerType = "command"
	TypeCallback HandlerType = "callback"
	TypeMessage  HandlerType = "message"
)

// Handler интерфейс для всех хэндлеров (должен быть совместим с router.Handler)
type Handler interface {
	Execute(params HandlerParams) (HandlerResult, error)
	GetName() string
	GetCommand() string // Может быть и командой и callback'ом
	GetType() HandlerType
}

// HandlerParams базовые параметры для всех хэндлеров
type HandlerParams struct {
	User     *models.User
	ChatID   int64
	Text     string // текст сообщения
	Data     string // для callback данных
	UpdateID string // ID обновления
}

// HandlerResult базовый результат хэндлера
type HandlerResult struct {
	Message  string                 `json:"message"`
	Keyboard interface{}            `json:"keyboard,omitempty"`
	NextStep string                 `json:"next_step,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}
