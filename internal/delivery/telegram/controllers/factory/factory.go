// internal/delivery/telegram/controllers/factory/factory.go
package controllers_factory

import (
	counterctrl "crypto-exchange-screener-bot/internal/delivery/telegram/controllers/counter"
	paymentctrl "crypto-exchange-screener-bot/internal/delivery/telegram/controllers/payment" // ⭐ ДОБАВЛЕНО
	"crypto-exchange-screener-bot/internal/delivery/telegram/services/counter"
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
)

// ControllerFactory фабрика контроллеров для EventBus
type ControllerFactory struct {
	counterService counter.Service
	// Добавляем другие сервисы по мере необходимости
}

// ControllerDependencies зависимости для фабрики контроллеров
type ControllerDependencies struct {
	CounterService counter.Service
	// Здесь можно добавить другие зависимости позже
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

// ⭐ НОВЫЙ МЕТОД: CreatePaymentController создает PaymentController
func (f *ControllerFactory) CreatePaymentController() types.EventSubscriber {
	return paymentctrl.NewController()
}

// GetAllControllers создает все контроллеры
func (f *ControllerFactory) GetAllControllers() map[string]types.EventSubscriber {
	controllers := make(map[string]types.EventSubscriber)

	if f.counterService != nil {
		controllers["CounterController"] = f.CreateCounterController()
	}

	// ⭐ Добавляем PaymentController (не требует зависимостей)
	controllers["PaymentController"] = f.CreatePaymentController()

	logger.Info("✅ ControllerFactory создала %d контроллеров", len(controllers))
	return controllers
}

// Validate проверяет фабрику контроллеров
func (f *ControllerFactory) Validate() bool {
	if f.counterService == nil {
		logger.Warn("⚠️ ControllerFactory: CounterService не доступен")
		// Не возвращаем false, так как PaymentController работает без сервиса
	}

	logger.Info("✅ ControllerFactory валидирована")
	return true
}
