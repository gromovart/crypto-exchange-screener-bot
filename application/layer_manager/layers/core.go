// application/layer_manager/layers/core.go
package layers

import (
	"crypto-exchange-screener-bot/internal/core/domain/fetchers" // НОВЫЙ импорт
	"crypto-exchange-screener-bot/internal/core/domain/subscription"
	"crypto-exchange-screener-bot/internal/core/domain/users"
	core_factory "crypto-exchange-screener-bot/internal/core/package"
	"crypto-exchange-screener-bot/internal/infrastructure/config" // НОВЫЙ импорт
	in_memory_storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/in_memory_storage/factory"
	events "crypto-exchange-screener-bot/internal/infrastructure/transport/event_bus" // НОВЫЙ импорт
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"time"
)

// CoreLayer слой ядра (бизнес-логика)
type CoreLayer struct {
	*BaseLayer
	config            *config.Config
	infraLayer        *InfrastructureLayer
	coreFactory       *core_factory.CoreServiceFactory
	initialized       bool
	bybitPriceFetcher *fetchers.BybitPriceFetcher    // НОВОЕ: правильное название
	fetcherFactory    *fetchers.MarketFetcherFactory // НОВОЕ: фабрика фетчеров
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

	// СОЗДАЕМ UserService СРАЗУ (не лениво) для ServiceFactory
	userService, err := cl.coreFactory.CreateUserService()
	if err != nil {
		cl.setError(err)
		return fmt.Errorf("не удалось создать UserService: %w", err)
	}

	// Сохраняем UserService для быстрого доступа
	cl.registerComponent("UserService", userService)
	logger.Info("✅ UserService создан и зарегистрирован")

	// НОВОЕ: Создаем фабрику фетчеров
	cl.fetcherFactory = fetchers.NewMarketFetcherFactory(cl.config)
	logger.Info("🏭 Фабрика MarketFetcher создана")

	// Регистрируем остальные компоненты
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

	// НОВОЕ: Запускаем BybitPriceFetcher если включен в конфигурации
	if cl.config.TelegramEnabled && cl.infraLayer != nil {
		cl.startBybitPriceFetcher()
	}

	// Фабрика ядра не требует отдельного запуска,
	// так как сервисы создаются лениво

	cl.running = true
	cl.updateState(StateRunning)
	logger.Info("✅ Слой ядра запущен")
	return nil
}

// НОВЫЙ МЕТОД: запуск BybitPriceFetcher
func (cl *CoreLayer) startBybitPriceFetcher() {
	logger.Info("🔄 CoreLayer: инициализация BybitPriceFetcher...")

	// Получаем EventBus из инфраструктуры
	eventBusComp, exists := cl.infraLayer.GetComponent("EventBus")
	if !exists {
		logger.Warn("⚠️ CoreLayer: EventBus не найден в инфраструктуре")
		return
	}

	// Получаем EventBus из LazyComponent
	eventBusInterface, err := cl.getComponentValue(eventBusComp)
	if err != nil {
		logger.Warn("⚠️ CoreLayer: не удалось получить EventBus: %v", err)
		return
	}

	eventBus, ok := eventBusInterface.(*events.EventBus)
	if !ok {
		logger.Warn("⚠️ CoreLayer: неверный тип EventBus")
		return
	}

	// Получаем StorageFactory
	storageFactoryComp, exists := cl.infraLayer.GetComponent("StorageFactory")
	if !exists {
		logger.Warn("⚠️ CoreLayer: StorageFactory не найден")
		return
	}

	// Получаем StorageFactory из LazyComponent
	storageInterface, err := cl.getComponentValue(storageFactoryComp)
	if err != nil {
		logger.Warn("⚠️ CoreLayer: не удалось получить StorageFactory: %v", err)
		return
	}

	storageFactory, ok := storageInterface.(*in_memory_storage.StorageFactory)
	if !ok {
		logger.Warn("⚠️ CoreLayer: неверный тип StorageFactory")
		return
	}

	// Создаем хранилище цен
	priceStorage, err := storageFactory.CreateDefaultStorage()
	if err != nil {
		logger.Error("❌ CoreLayer: ошибка создания хранилища цен: %v", err)
		return
	}

	// Создаем фетчер
	fetcher, err := cl.fetcherFactory.CreateBybitFetcher(priceStorage, eventBus)
	if err != nil {
		logger.Error("❌ CoreLayer: ошибка создания BybitPriceFetcher: %v", err)
		return
	}

	// Сохраняем фетчер
	cl.bybitPriceFetcher = fetcher

	// Регистрируем компонент
	cl.registerComponent("BybitPriceFetcher", fetcher)
	logger.Info("✅ BybitPriceFetcher создан и зарегистрирован")

	// Запускаем фетчер с интервалом из конфигурации
	interval := time.Duration(cl.config.UpdateInterval) * time.Second
	if interval == 0 {
		interval = 10 * time.Second // дефолтное значение
		logger.Info("ℹ️  Используется дефолтный интервал для BybitPriceFetcher: %v", interval)
	}

	if err := fetcher.Start(interval); err != nil {
		logger.Error("❌ CoreLayer: ошибка запуска BybitPriceFetcher: %v", err)
		cl.setError(err)
	} else {
		logger.Info("🚀 BybitPriceFetcher запущен с интервалом %v", interval)
	}
}

// Stop останавливает слой ядра
func (cl *CoreLayer) Stop() error {
	if !cl.IsRunning() {
		return nil
	}

	cl.updateState(StateStopping)
	logger.Info("🛑 Остановка слоя ядра...")

	// НОВОЕ: Останавливаем BybitPriceFetcher если запущен
	if cl.bybitPriceFetcher != nil && cl.bybitPriceFetcher.IsRunning() {
		if err := cl.bybitPriceFetcher.Stop(); err != nil {
			logger.Warn("⚠️ Ошибка остановки BybitPriceFetcher: %v", err)
		} else {
			logger.Info("🛑 BybitPriceFetcher остановлен")
		}
	}

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

	// НОВОЕ: Сбрасываем фетчер
	if cl.bybitPriceFetcher != nil {
		cl.bybitPriceFetcher = nil
	}
	if cl.fetcherFactory != nil {
		cl.fetcherFactory = nil
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

// НОВЫЙ МЕТОД: получает значение компонента из LazyComponent
func (cl *CoreLayer) getComponentValue(component interface{}) (interface{}, error) {
	if lc, ok := component.(*LazyComponent); ok {
		return lc.Get()
	}
	return nil, fmt.Errorf("компонент не является LazyComponent")
}

// GetBybitPriceFetcher возвращает BybitPriceFetcher (НОВОЕ)
func (cl *CoreLayer) GetBybitPriceFetcher() *fetchers.BybitPriceFetcher {
	cl.mu.RLock()
	defer cl.mu.RUnlock()
	return cl.bybitPriceFetcher
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
			// Если UserService уже создан, возвращаем его
			if userService, exists := cl.GetComponent("UserService"); exists {
				return userService, nil
			}
			// Иначе создаем через фабрику
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
