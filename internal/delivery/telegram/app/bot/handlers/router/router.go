// internal/delivery/telegram/app/bot/handlers/router/router.go
package router

import (
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"log"
	"strings"
)

// HandlerType тип хэндлера
type HandlerType string

const (
	TypeCommand  HandlerType = "command"
	TypeCallback HandlerType = "callback"
	TypeMessage  HandlerType = "message"
)

// Handler интерфейс для всех хэндлеров (локальная версия)
type Handler interface {
	Execute(params interface{}) (interface{}, error)
	GetName() string
	GetCommand() string // Может быть и командой и callback'ом
	GetType() HandlerType
}

// HandlerParams базовые параметры для всех хэндлеров (локальная версия)
type HandlerParams struct {
	User     interface{} // *models.User
	ChatID   int64
	Text     string // текст сообщения
	Data     string // для callback данных
	UpdateID string // ID обновления
}

// HandlerResult базовый результат хэндлера (локальная версия)
type HandlerResult struct {
	Message  string                 `json:"message"`
	Keyboard interface{}            `json:"keyboard,omitempty"`
	NextStep string                 `json:"next_step,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// routerImpl реализация Router
type routerImpl struct {
	handlers map[string]Handler // ключ: команда/callback
}

// NewRouter создает новый роутер
func NewRouter() Router {
	return &routerImpl{
		handlers: make(map[string]Handler),
	}
}

// RegisterHandler регистрирует хэндлер (использует GetCommand())
func (r *routerImpl) RegisterHandler(handler Handler) {
	command := handler.GetCommand()

	// Для команд добавляем префикс /
	if handler.GetType() == TypeCommand && command[0] != '/' {
		command = "/" + command
	}

	r.handlers[command] = handler
	logger.Debug("✅ Зарегистрирован хэндлер: %s для %s: %s",
		handler.GetName(), handler.GetType(), command)
}

// RegisterCommand регистрирует команду (явно указываем команду с /)
func (r *routerImpl) RegisterCommand(command string, handler Handler) {
	// Убеждаемся, что команда начинается с /
	if command[0] != '/' {
		command = "/" + command
	}
	r.handlers[command] = handler
	logger.Debug("✅ Зарегистрирована команда: %s → %s", command, handler.GetName())
}

// RegisterCallback регистрирует callback (без префикса /)
func (r *routerImpl) RegisterCallback(callback string, handler Handler) {
	// Убеждаемся, что callback не начинается с /
	if len(callback) > 0 && callback[0] == '/' {
		callback = callback[1:]
	}
	r.handlers[callback] = handler
	logger.Debug("✅ Зарегистрирован callback: %s → %s", callback, handler.GetName())
}

// Handle обрабатывает команду/callback
func (r *routerImpl) Handle(command string, params HandlerParams) (HandlerResult, error) {
	// Пробуем найти точное совпадение
	handler, exists := r.handlers[command]
	if exists {
		return r.executeHandler(handler, command, params)
	}

	// Проверяем, является ли command параметризованным callback (содержит :)
	if strings.Contains(command, ":") {
		// Перенаправляем в универсальный обработчик with_params
		if handler, exists := r.handlers["with_params"]; exists {
			// Сохраняем полный callback data для обработки
			params.Data = command
			log.Printf("🔀 Перенаправление параметризованного callback '%s' в with_params", command)
			return r.executeHandler(handler, command, params)
		}
	}

	// Пробуем найти обработчик по префиксу (для callback-ов с параметрами)
	for key, h := range r.handlers {
		if strings.HasPrefix(command, key+":") {
			// Обновляем data в params для передачи параметров
			params.Data = command
			return r.executeHandler(h, command, params)
		}
	}

	// Пробуем найти команду без префикса /
	if command[0] == '/' {
		handler, exists = r.handlers[command[1:]]
	} else {
		handler, exists = r.handlers["/"+command]
	}

	if exists {
		return r.executeHandler(handler, command, params)
	}

	return HandlerResult{},
		fmt.Errorf("хэндлер для '%s' не найден", command)
}

// executeHandler выполняет обработчик
func (r *routerImpl) executeHandler(handler Handler, command string, params HandlerParams) (HandlerResult, error) {
	log.Printf("🔍 Вызов хэндлера: %s для: %s",
		handler.GetName(), command)

	result, err := handler.Execute(params)
	if err != nil {
		return HandlerResult{}, err
	}

	// Приводим тип результата
	handlerResult, ok := result.(HandlerResult)
	if !ok {
		return HandlerResult{}, fmt.Errorf("неверный тип результата от хэндлера")
	}

	return handlerResult, nil
}

// GetHandler возвращает хэндлер по команде/callback
func (r *routerImpl) GetHandler(command string) (Handler, bool) {
	handler, exists := r.handlers[command]
	return handler, exists
}

// GetCommands возвращает список всех команд (с /)
func (r *routerImpl) GetCommands() []string {
	commands := make([]string, 0, len(r.handlers))
	for cmd := range r.handlers {
		commands = append(commands, cmd)
	}
	return commands
}

var _ Router = (*routerImpl)(nil)
