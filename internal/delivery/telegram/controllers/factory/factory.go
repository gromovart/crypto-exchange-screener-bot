// /internal/delivery/telegram/controllers/factory/factory.go
package controllers_factory

import (
	counterctrl "crypto-exchange-screener-bot/internal/delivery/telegram/controllers/counter"
	"crypto-exchange-screener-bot/internal/delivery/telegram/services/counter"
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
)

// ControllerFactory фабрика контроллеров для EventBus
type ControllerFactory struct {
	counterService counter.Service
}

// ControllerDependencies зависимости для фабрики контроллеров
type ControllerDependencies struct {
	CounterService counter.Service
}

// NewControllerFactory создает фабрику контроллеров
func NewControllerFactory(deps ControllerDependencies) *ControllerFactory {
	logger.Info("🎛️  Создание фабрики контроллеров...")

	return &ControllerFactory{
		counterService: deps.CounterService,
	}
}

// CreateCounterController создает CounterController
func (f *ControllerFactory) CreateCounterController() types.EventSubscriber {
	return counterctrl.NewController(f.counterService)
}

// GetAllControllers создает все контроллеры
func (f *ControllerFactory) GetAllControllers() map[string]types.EventSubscriber {
	controllers := make(map[string]types.EventSubscriber)

	if f.counterService != nil {
		controllers["CounterController"] = f.CreateCounterController()
	}

	logger.Info("✅ ControllerFactory создала %d контроллеров", len(controllers))
	return controllers
}

// Validate проверяет фабрику контроллеров
func (f *ControllerFactory) Validate() bool {
	if f.counterService == nil {
		logger.Warn("⚠️ ControllerFactory: CounterService не доступен")
		return false
	}

	logger.Info("✅ ControllerFactory валидирована")
	return true
}
