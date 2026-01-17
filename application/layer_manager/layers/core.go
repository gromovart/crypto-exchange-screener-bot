// application/services/orchestrator/layers/core.go
package layers

import (
	"crypto-exchange-screener-bot/internal/core/domain/subscription"
	"crypto-exchange-screener-bot/internal/core/domain/users"
	core_factory "crypto-exchange-screener-bot/internal/core/package"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"time"
)

// CoreLayer слой ядра (бизнес-логика)
type CoreLayer struct {
	*BaseLayer
	config      *config.Config
	infraLayer  *InfrastructureLayer
	coreFactory *core_factory.CoreServiceFactory
	initialized bool
}

// NewCoreLayer создает слой ядра
func NewCoreLayer(cfg *config.Config, infraLayer *InfrastructureLayer) *CoreLayer {
	layer := &CoreLayer{
		BaseLayer:  NewBaseLayer("CoreLayer", []string{"InfrastructureLayer"}),
		config:     cfg,
		infraLayer: infraLayer,
	}
	return layer
}

// SetDependencies устанавливает зависимости
func (cl *CoreLayer) SetDependencies(deps map[string]Layer) error {
	// Получаем слой инфраструктуры из зависимостей
	infraLayer, exists := deps["InfrastructureLayer"]
	if !exists {
		return fmt.Errorf("зависимость InfrastructureLayer не найдена")
	}

	// Приводим к правильному типу
	infra, ok := infraLayer.(*InfrastructureLayer)
	if !ok {
		return fmt.Errorf("неверный тип InfrastructureLayer")
	}

	cl.infraLayer = infra
	return nil
}

// Initialize инициализирует слой ядра
func (cl *CoreLayer) Initialize() error {
	if cl.initialized {
		return fmt.Errorf("слой ядра уже инициализирован")
	}

	// Проверяем зависимости
	if cl.infraLayer == nil {
		return fmt.Errorf("InfrastructureLayer не установлен")
	}

	if !cl.infraLayer.IsInitialized() {
		return fmt.Errorf("InfrastructureLayer не инициализирован")
	}

	cl.updateState(StateInitializing)
	logger.Info("🧠 Инициализация слоя ядра...")

	// Получаем фабрику инфраструктуры
	infraFactory := cl.infraLayer.GetInfrastructureFactory()
	if infraFactory == nil {
		return fmt.Errorf("фабрика инфраструктуры не создана")
	}

	// Создаем конфигурацию для фабрики ядра со значениями по умолчанию
	coreConfig := &core_factory.Config{
		UserConfig: users.Config{
			DefaultMinGrowthThreshold: 2.0, // значение по умолчанию
			DefaultMaxSignalsPerDay:   50,
			SessionTTL:                24 * time.Hour,
			MaxSessionsPerUser:        5,
		},
		SubscriptionConfig: subscription.Config{
			StripeSecretKey:  "",
			StripeWebhookKey: "",
			DefaultPlan:      "free",
			TrialPeriodDays:  7,
			GracePeriodDays:  3,
			AutoRenew:        true,
		},
	}

	// Создаем фабрику ядра
	deps := core_factory.CoreServiceDependencies{
		InfrastructureFactory: infraFactory,
		Config:                coreConfig,
		UserNotifier:          nil,
		SubNotifier:           nil,
		Analytics:             nil,
	}

	var err error
	cl.coreFactory, err = core_factory.NewCoreServiceFactory(deps)
	if err != nil {
		cl.setError(err)
		return fmt.Errorf("не удалось создать фабрику ядра: %w", err)
	}

	// Инициализируем фабрику ядра
	if err := cl.coreFactory.Initialize(); err != nil {
		cl.setError(err)
		return fmt.Errorf("не удалось инициализировать фабрику ядра: %w", err)
	}

	// Регистрируем компоненты
	cl.registerCoreComponents()

	cl.initialized = true
	cl.updateState(StateInitialized)
	logger.Info("✅ Слой ядра инициализирован")
	return nil
}

// Start запускает слой ядра
func (cl *CoreLayer) Start() error {
	if !cl.initialized {
		return fmt.Errorf("слой ядра не инициализирован")
	}

	if cl.IsRunning() {
		return fmt.Errorf("слой ядра уже запущен")
	}

	cl.updateState(StateStarting)
	logger.Info("🚀 Запуск слоя ядра...")

	// Фабрика ядра не требует отдельного запуска,
	// так как сервисы создаются лениво

	cl.running = true
	cl.updateState(StateRunning)
	logger.Info("✅ Слой ядра запущен")
	return nil
}

// Stop останавливает слой ядра
func (cl *CoreLayer) Stop() error {
	if !cl.IsRunning() {
		return nil
	}

	cl.updateState(StateStopping)
	logger.Info("🛑 Остановка слоя ядра...")

	// Останавливаем фабрику ядра если нужно
	// (в текущей реализации нет метода Stop у CoreServiceFactory)

	cl.running = false
	cl.updateState(StateStopped)
	logger.Info("✅ Слой ядра остановлен")
	return nil
}

// Reset сбрасывает слой ядра
func (cl *CoreLayer) Reset() error {
	logger.Info("🔄 Сброс слоя ядра...")

	// Останавливаем если запущен
	if cl.IsRunning() {
		cl.Stop()
	}

	// Сбрасываем фабрику
	if cl.coreFactory != nil {
		cl.coreFactory.Reset()
	}

	// Сбрасываем базовый слой
	cl.BaseLayer.Reset()

	cl.coreFactory = nil
	cl.initialized = false
	logger.Info("✅ Слой ядра сброшен")
	return nil
}

// IsInitialized проверяет инициализацию
func (cl *CoreLayer) IsInitialized() bool {
	return cl.initialized
}

// GetCoreFactory возвращает фабрику ядра
func (cl *CoreLayer) GetCoreFactory() *core_factory.CoreServiceFactory {
	cl.mu.RLock()
	defer cl.mu.RUnlock()
	return cl.coreFactory
}

// registerCoreComponents регистрирует компоненты ядра
func (cl *CoreLayer) registerCoreComponents() {
	if cl.coreFactory == nil {
		return
	}

	// Регистрируем компоненты ядра
	components := map[string]string{
		"UserService":         "сервис пользователей",
		"SubscriptionService": "сервис подписок",
	}

	for name, description := range components {
		cl.registerComponent(name, &LazyComponent{
			name:        name,
			description: description,
			getter:      cl.getCoreComponent(name),
		})
		logger.Debug("🧩 Зарегистрирован компонент ядра: %s (%s)", name, description)
	}
}

// getCoreComponent возвращает геттер для компонента ядра
func (cl *CoreLayer) getCoreComponent(name string) func() (interface{}, error) {
	return func() (interface{}, error) {
		if cl.coreFactory == nil {
			return nil, fmt.Errorf("фабрика ядра не создана")
		}

		switch name {
		case "UserService":
			return cl.coreFactory.CreateUserService()
		case "SubscriptionService":
			return cl.coreFactory.CreateSubscriptionService()
		default:
			return nil, fmt.Errorf("неизвестный компонент ядра: %s", name)
		}
	}
}

// GetUserService возвращает UserService (ленивое создание)
func (cl *CoreLayer) GetUserService() (interface{}, error) {
	comp, exists := cl.GetComponent("UserService")
	if !exists {
		return nil, fmt.Errorf("UserService не зарегистрирован")
	}

	lc, ok := comp.(*LazyComponent)
	if !ok {
		return nil, fmt.Errorf("неверный тип компонента UserService")
	}

	return lc.Get()
}

// GetSubscriptionService возвращает SubscriptionService (ленивое создание)
func (cl *CoreLayer) GetSubscriptionService() (interface{}, error) {
	comp, exists := cl.GetComponent("SubscriptionService")
	if !exists {
		return nil, fmt.Errorf("SubscriptionService не зарегистрирован")
	}

	lc, ok := comp.(*LazyComponent)
	if !ok {
		return nil, fmt.Errorf("неверный тип компонента SubscriptionService")
	}

	return lc.Get()
}
