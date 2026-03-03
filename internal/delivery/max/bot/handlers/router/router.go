// internal/delivery/max/bot/handlers/router/router.go
package router

import (
	"fmt"
	"strings"

	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/pkg/logger"
)

// Router — маршрутизатор команд и callback-ов
type Router interface {
	RegisterCommand(command string, handler handlers.Handler)
	RegisterCallback(callback string, handler handlers.Handler)
	RegisterEvent(name string, handler handlers.Handler)
	Handle(command string, params handlers.HandlerParams) (handlers.HandlerResult, error)
	GetHandler(command string) (handlers.Handler, bool)
}

type routerImpl struct {
	handlers map[string]handlers.Handler
}

// NewRouter создаёт новый роутер
func NewRouter() Router {
	return &routerImpl{
		handlers: make(map[string]handlers.Handler),
	}
}

func (r *routerImpl) RegisterCommand(command string, handler handlers.Handler) {
	if !strings.HasPrefix(command, "/") {
		command = "/" + command
	}
	r.handlers[command] = handler
	logger.Info("✅ MAX Router: команда %s → %s", command, handler.GetName())
}

func (r *routerImpl) RegisterCallback(callback string, handler handlers.Handler) {
	r.handlers[callback] = handler
	logger.Info("✅ MAX Router: callback %s → %s", callback, handler.GetName())
}

func (r *routerImpl) RegisterEvent(name string, handler handlers.Handler) {
	r.handlers[name] = handler
	logger.Info("✅ MAX Router: event %s → %s", name, handler.GetName())
}

// Handle маршрутизирует команду / callback к нужному хэндлеру
func (r *routerImpl) Handle(command string, params handlers.HandlerParams) (handlers.HandlerResult, error) {
	logger.Debug("🚀 MAX Router.Handle: command='%s'", command)

	// 1. Паттерны с * (wildcard)
	for key, h := range r.handlers {
		if strings.HasSuffix(key, "*") {
			prefix := strings.TrimSuffix(key, "*")
			if strings.HasPrefix(command, prefix) {
				return r.exec(h, command, params)
			}
		}
	}

	// 2. Команда с аргументами (/start ref123)
	if strings.HasPrefix(command, "/") && strings.Contains(command, " ") {
		parts := strings.SplitN(command, " ", 2)
		if h, ok := r.handlers[parts[0]]; ok {
			params.Data = parts[1]
			return r.exec(h, command, params)
		}
	}

	// 3. Точное совпадение
	if h, ok := r.handlers[command]; ok {
		return r.exec(h, command, params)
	}

	// 4. Параметризованный callback с : (period_5m → period_select)
	if strings.HasPrefix(command, "period_") {
		if h, ok := r.handlers["period_select"]; ok {
			params.Data = command
			return r.exec(h, command, params)
		}
	}

	// 5. Поиск по префиксу с : (signal_set_growth_threshold:1.0 → signal_set_growth_threshold)
	if strings.Contains(command, ":") {
		prefix := strings.SplitN(command, ":", 2)[0]
		if h, ok := r.handlers[prefix]; ok {
			params.Data = command
			return r.exec(h, command, params)
		}
		// fallback: with_params
		if h, ok := r.handlers["with_params"]; ok {
			params.Data = command
			return r.exec(h, command, params)
		}
	}

	// 6. Без префикса /
	if strings.HasPrefix(command, "/") {
		if h, ok := r.handlers[command[1:]]; ok {
			return r.exec(h, command, params)
		}
	} else {
		if h, ok := r.handlers["/"+command]; ok {
			return r.exec(h, command, params)
		}
	}

	logger.Error("❌ MAX Router: хэндлер для '%s' не найден", command)
	return handlers.HandlerResult{}, fmt.Errorf("MAX: хэндлер для '%s' не найден", command)
}

func (r *routerImpl) exec(h handlers.Handler, command string, params handlers.HandlerParams) (handlers.HandlerResult, error) {
	logger.Debug("▶️ MAX exec: %s для %s", h.GetName(), command)
	result, err := h.Execute(params)
	if err != nil {
		logger.Error("❌ MAX %s ошибка для %s: %v", h.GetName(), command, err)
	}
	return result, err
}

func (r *routerImpl) GetHandler(command string) (handlers.Handler, bool) {
	h, ok := r.handlers[command]
	return h, ok
}
