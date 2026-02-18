// internal/delivery/telegram/app/bot/handlers/factory.go
package handlers

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/router"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
	"crypto-exchange-screener-bot/pkg/logger"
)

// HandlerAdapter адаптер для преобразования Handler в router.Handler
type HandlerAdapter struct {
	handler Handler
}

// NewHandlerAdapter создает новый адаптер
func NewHandlerAdapter(handler Handler) *HandlerAdapter {
	return &HandlerAdapter{handler: handler}
}

// Execute выполняет обработку (адаптирует интерфейсы)
func (a *HandlerAdapter) Execute(params interface{}) (interface{}, error) {
	// Преобразуем params из router.HandlerParams в HandlerParams
	routerParams, ok := params.(router.HandlerParams)
	if !ok {
		return router.HandlerResult{}, fmt.Errorf("неверный тип параметров")
	}

	// Преобразуем User
	var user *models.User
	if routerParams.User != nil {
		user, ok = routerParams.User.(*models.User)
		if !ok {
			return router.HandlerResult{}, fmt.Errorf("неверный тип пользователя")
		}
	}

	// Вызываем оригинальный хэндлер
	result, err := a.handler.Execute(HandlerParams{
		User:     user,
		ChatID:   routerParams.ChatID,
		Text:     routerParams.Text,
		Data:     routerParams.Data,
		UpdateID: routerParams.UpdateID,
	})

	if err != nil {
		return router.HandlerResult{}, err
	}

	// Преобразуем результат
	return router.HandlerResult{
		Message:  result.Message,
		Keyboard: result.Keyboard,
		NextStep: result.NextStep,
		Metadata: result.Metadata,
	}, nil
}

// GetName возвращает имя хэндлера
func (a *HandlerAdapter) GetName() string {
	return a.handler.GetName()
}

// GetCommand возвращает команду/callback
func (a *HandlerAdapter) GetCommand() string {
	return a.handler.GetCommand()
}

// GetType возвращает тип хэндлера
func (a *HandlerAdapter) GetType() router.HandlerType {
	// Преобразуем наш HandlerType в router.HandlerType
	switch a.handler.GetType() {
	case TypeCommand:
		return router.TypeCommand
	case TypeCallback:
		return router.TypeCallback
	case TypeMessage:
		return router.TypeMessage
	default:
		return router.TypeCommand
	}
}

// HandlerFactory фабрика для создания и регистрации хэндлеров
type HandlerFactory struct {
	router          router.Router
	handlerCreators map[string]func() Handler
}

// NewHandlerFactory создает новую фабрику хэндлеров
func NewHandlerFactory() *HandlerFactory {
	return &HandlerFactory{
		router:          router.NewRouter(),
		handlerCreators: make(map[string]func() Handler),
	}
}

// RegisterHandlerCreator регистрирует создателя хэндлера
func (f *HandlerFactory) RegisterHandlerCreator(name string, creator func() Handler) {
	f.handlerCreators[name] = creator
	logger.Debug("✅ Зарегистрирован создатель хэндлера: %s", name)
}

// RegisterHandler регистрирует хэндлер через адаптер
func (f *HandlerFactory) RegisterHandler(handler Handler) {
	adapter := NewHandlerAdapter(handler)

	switch handler.GetType() {
	case TypeCommand:
		f.router.RegisterCommand(handler.GetCommand(), adapter)
		logger.Info("✅ Зарегистрирована команда: %s → %s", handler.GetCommand(), handler.GetName())

	case TypeCallback:
		f.router.RegisterCallback(handler.GetCommand(), adapter)
		logger.Info("✅ Зарегистрирован callback: %s → %s", handler.GetCommand(), handler.GetName())

	case TypeMessage:
		// Регистрируем как эвент
		f.router.RegisterEvent(handler.GetCommand(), adapter)
		logger.Info("✅ Зарегистрирован эвент: %s → %s", handler.GetCommand(), handler.GetName())

	default:
		logger.Warn("  ⚠️ Неизвестный тип хэндлера: %s", handler.GetName())
	}
}

// CreateAndRegisterHandler создает и регистрирует хэндлер по имени
func (f *HandlerFactory) CreateAndRegisterHandler(name string) {
	if creator, exists := f.handlerCreators[name]; exists {
		handler := creator()
		if handler != nil {
			f.RegisterHandler(handler)
			logger.Debug("  ✅ Хэндлер: %s → %s", handler.GetCommand(), name)
		}
	} else {
		logger.Error("  ❌ Создатель хэндлера не найден: %s", name)
	}
}

// RegisterAllHandlers создает и регистрирует все хэндлеры
func (f *HandlerFactory) RegisterAllHandlers() router.Router {
	logger.Info("🔧 Регистрация всех хэндлеров...")

	// Создаем и регистрируем все хэндлеры
	for name := range f.handlerCreators {
		f.CreateAndRegisterHandler(name)
	}

	logger.Debug("✅ Зарегистрировано хэндлеров: %d", len(f.router.GetCommands()))
	return f.router
}

// GetRouter возвращает роутер
func (f *HandlerFactory) GetRouter() router.Router {
	return f.router
}
