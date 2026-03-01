// internal/delivery/max/bot/handlers/types.go
package handlers

import (
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
	"sync"
)

// HandlerType — тип хэндлера
type HandlerType string

const (
	TypeCommand  HandlerType = "command"
	TypeCallback HandlerType = "callback"
	TypeMessage  HandlerType = "message"
)

// Handler — интерфейс для всех хэндлеров MAX бота
type Handler interface {
	Execute(params HandlerParams) (HandlerResult, error)
	GetName() string
	GetCommand() string
	GetType() HandlerType
}

// HandlerParams — параметры для хэндлера
type HandlerParams struct {
	User        *models.User
	ChatID      int64
	Text        string // текст сообщения (для команд)
	Data        string // callback data
	UpdateID    string
	CallbackID  string // ID callback запроса (для AnswerCallbackQuery)
	MessageID   int64  // ID текущего сообщения (для Edit)
}

// HandlerResult — результат хэндлера
type HandlerResult struct {
	Message  string
	Keyboard interface{}
	NextStep string
	Metadata map[string]interface{}
	// EditMessage — если true, редактировать существующее сообщение вместо нового
	EditMessage bool
}

// HandlerFactory — фабрика хэндлеров
type HandlerFactory struct {
	mu      sync.RWMutex
	creators map[string]func() Handler
}

// NewHandlerFactory создаёт фабрику хэндлеров
func NewHandlerFactory() *HandlerFactory {
	return &HandlerFactory{
		creators: make(map[string]func() Handler),
	}
}

// RegisterHandlerCreator регистрирует создателя хэндлера
func (f *HandlerFactory) RegisterHandlerCreator(command string, creator func() Handler) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.creators[command] = creator
}

// CreateHandler создаёт хэндлер по команде
func (f *HandlerFactory) CreateHandler(command string) (Handler, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	creator, ok := f.creators[command]
	if !ok {
		return nil, false
	}
	return creator(), true
}

// RegisterAllHandlers создаёт все хэндлеры и возвращает их список
func (f *HandlerFactory) RegisterAllHandlers() []Handler {
	f.mu.RLock()
	defer f.mu.RUnlock()
	result := make([]Handler, 0, len(f.creators))
	for _, creator := range f.creators {
		result = append(result, creator())
	}
	return result
}
