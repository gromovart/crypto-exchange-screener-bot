// internal/delivery/telegram/app/bot/handlers/router/router.go
package router

import (
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
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
	logger.Info("✅ Зарегистрирована команда: %s → %s", command, handler.GetName())
}

// RegisterCallback регистрирует callback (без префикса /)
func (r *routerImpl) RegisterCallback(callback string, handler Handler) {
	// Убеждаемся, что callback не начинается с /
	if len(callback) > 0 && callback[0] == '/' {
		callback = callback[1:]
	}
	r.handlers[callback] = handler
	logger.Info("✅ Зарегистрирован callback: %s → %s", callback, handler.GetName())
}

// RegisterEvent регистрирует эвент
func (r *routerImpl) RegisterEvent(eventName string, handler Handler) {
	r.handlers[eventName] = handler
	logger.Info("✅ Зарегистрирован эвент: %s → %s", eventName, handler.GetName())
}

// Handle обрабатывает команду/callback
func (r *routerImpl) Handle(command string, params HandlerParams) (HandlerResult, error) {
	// ОТЛАДКА: выводим полученную команду
	logger.Debug("🚀 Router.Handle вызван с command='%s'", command)

	// ⭐ СПЕЦИАЛЬНАЯ ОБРАБОТКА ДЛЯ PRE-CHECKOUT QUERY
	if strings.HasPrefix(command, "pre_checkout_query") {
		logger.Debug("💰 Обнаружен pre_checkout_query: %s", command)
		if handler, exists := r.handlers["pre_checkout_query"]; exists {
			// ⚠️ ВАЖНО: params.Data уже содержит полную строку с параметрами
			// Не нужно ничего менять, просто вызываем хэндлер
			logger.Debug("💰 Вызов хэндлера pre_checkout_query с data='%s'", params.Data)
			return r.executeHandler(handler, command, params)
		}
	}

	// ⭐ СПЕЦИАЛЬНАЯ ОБРАБОТКА ДЛЯ SUCCESSFUL PAYMENT
	if strings.HasPrefix(command, "successful_payment") {
		logger.Debug("💰 Обнаружен successful_payment: %s", command)
		if handler, exists := r.handlers["successful_payment"]; exists {
			// ⚠️ ВАЖНО: params.Data уже содержит полную строку с параметрами
			logger.Debug("💰 Вызов хэндлера successful_payment с data='%s'", params.Data)
			return r.executeHandler(handler, command, params)
		}
	}

	// Если команда начинается с / и содержит пробел (параметры)
	if strings.HasPrefix(command, "/") && strings.Contains(command, " ") {
		// Разделяем команду и параметры
		parts := strings.SplitN(command, " ", 2)
		baseCommand := parts[0]
		payload := parts[1]

		// Пробуем найти обработчик для базовой команды
		if handler, exists := r.handlers[baseCommand]; exists {
			// Передаем полный текст в params для обработки параметров
			params.Text = command
			params.Data = payload // Дополнительно сохраняем payload в Data
			logger.Debug("Обработка команды с параметрами: %s → %s (payload: %s)",
				command, baseCommand, payload)
			return r.executeHandler(handler, command, params)
		}
	}

	// Пробуем найти точное совпадение
	handler, exists := r.handlers[command]
	if exists {
		logger.Debug("✅ Найдено точное совпадение для command='%s'", command)
		return r.executeHandler(handler, command, params)
	}

	// Проверяем, является ли command параметризованным callback (содержит :)
	if strings.Contains(command, ":") {
		// Определяем префикс (часть до :)
		prefix := strings.Split(command, ":")[0]

		logger.Debug("🔄 Обработка параметризованного callback: '%s', префикс: '%s'", command, prefix)

		// Проверяем специальные платежные префиксы
		if prefix == "payment_plan" || prefix == "payment_confirm" {
			paymentKey := prefix + ":"
			logger.Debug("💰 Проверка платежного обработчика для ключа: '%s'", paymentKey)

			// Пробуем найти обработчик по префиксу
			if handler, exists := r.handlers[paymentKey]; exists {
				params.Data = command
				logger.Debug("✅ Перенаправление платежного callback '%s' в %s", command, paymentKey)
				return r.executeHandler(handler, command, params)
			} else {
				logger.Debug("❌ Платежный обработчик не найден для ключа: '%s'", paymentKey)
				// Выводим все зарегистрированные обработчики для отладки
				r.debugRegisteredHandlers()
			}
		}

		// Перенаправляем в универсальный обработчик with_params
		if handler, exists := r.handlers["with_params"]; exists {
			// Сохраняем полный callback data для обработки
			params.Data = command
			logger.Debug("🔄 Перенаправление параметризованного callback '%s' в with_params", command)
			return r.executeHandler(handler, command, params)
		}
	}

	// Проверяем префиксы для периодов (period_5m, period_15m и т.д.)
	if strings.HasPrefix(command, "period_") {
		// Пробуем найти обработчик period_select
		if handler, exists := r.handlers["period_select"]; exists {
			params.Data = command
			logger.Debug("🔄 Перенаправление периода '%s' в period_select", command)
			return r.executeHandler(handler, command, params)
		}
		// Или пробуем найти обработчик с префиксом period_
		for key, h := range r.handlers {
			if strings.HasPrefix(key, "period_") && strings.HasPrefix(command, key) {
				params.Data = command
				logger.Debug("🔄 Перенаправление периода '%s' в %s", command, key)
				return r.executeHandler(h, command, params)
			}
		}
	}

	// Пробуем найти обработчик по префиксу (для callback-ов с параметрами)
	for key, h := range r.handlers {
		if strings.HasPrefix(command, key+":") {
			// Обновляем data в params для передачи параметров
			params.Data = command
			logger.Debug("🔄 Перенаправление по префиксу '%s' в %s", command, key)
			return r.executeHandler(h, command, params)
		}

		// Проверяем специальные случаи с префиксами в конце (payment_plan:)
		if strings.HasSuffix(key, ":") && strings.HasPrefix(command, key) {
			params.Data = command
			logger.Debug("🔄 Перенаправление по префиксу с двоеточием '%s' в %s", command, key)
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
		logger.Debug("🔄 Найдено совпадение без префикса / для command='%s'", command)
		return r.executeHandler(handler, command, params)
	}

	logger.Error("❌ Хэндлер для '%s' не найден", command)
	// Выводим все зарегистрированные обработчики для отладки
	r.debugRegisteredHandlers()

	return HandlerResult{},
		fmt.Errorf("хэндлер для '%s' не найден", command)
}

// executeHandler выполняет обработчик
func (r *routerImpl) executeHandler(handler Handler, command string, params HandlerParams) (HandlerResult, error) {
	logger.Debug("▶️ Вызов хэндлера: %s для: %s",
		handler.GetName(), command)

	result, err := handler.Execute(params)
	if err != nil {
		logger.Error("❌ Ошибка в хэндлере %s для %s: %v",
			handler.GetName(), command, err)
		return HandlerResult{}, err
	}

	// Приводим тип результата
	handlerResult, ok := result.(HandlerResult)
	if !ok {
		err := fmt.Errorf("неверный тип результата от хэндлера")
		logger.Error("❌ %s для %s: %v", handler.GetName(), command, err)
		return HandlerResult{}, err
	}

	logger.Debug("✅ Хэндлер %s для %s выполнен успешно",
		handler.GetName(), command)
	return handlerResult, nil
}

// debugRegisteredHandlers выводит отладочную информацию о зарегистрированных обработчиках
func (r *routerImpl) debugRegisteredHandlers() {
	logger.Debug("📋 Зарегистрированные обработчики (%d):", len(r.handlers))
	for key, handler := range r.handlers {
		logger.Debug("   • %s → %s (%s)", key, handler.GetName(), handler.GetType())
	}
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
