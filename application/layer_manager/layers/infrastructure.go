// application/services/orchestrator/layers/infrastructure.go
package layers

import (
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	infrastructure_factory "crypto-exchange-screener-bot/internal/infrastructure/package"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"time"
)

// InfrastructureLayer слой инфраструктуры
type InfrastructureLayer struct {
	*BaseLayer
	config       *config.Config
	infraFactory *infrastructure_factory.InfrastructureFactory
}

// NewInfrastructureLayer создает слой инфраструктуры
func NewInfrastructureLayer(cfg *config.Config) *InfrastructureLayer {
	layer := &InfrastructureLayer{
		BaseLayer: NewBaseLayer("InfrastructureLayer", nil),
		config:    cfg,
	}
	return layer
}

// Initialize инициализирует слой инфраструктуры
func (il *InfrastructureLayer) Initialize() error {
	if il.IsInitialized() {
		return fmt.Errorf("слой инфраструктуры уже инициализирован")
	}

	il.updateState(StateInitializing)
	logger.Info("🏗️  Инициализация слоя инфраструктуры...")

	// Создаем фабрику инфраструктуры
	deps := infrastructure_factory.InfrastructureDependencies{
		Config: il.config,
	}

	var err error
	il.infraFactory, err = infrastructure_factory.NewInfrastructureFactory(deps)
	if err != nil {
		il.setError(err)
		return fmt.Errorf("не удалось создать фабрику инфраструктуры: %w", err)
	}

	// Инициализируем фабрику
	if err := il.infraFactory.Initialize(); err != nil {
		il.setError(err)
		return fmt.Errorf("не удалось инициализировать фабрику инфраструктуры: %w", err)
	}

	// ✅ ГАРАНТИЯ: ждем пока фабрика станет готовой
	if !il.waitForFactoryReady(10 * time.Second) {
		il.setError(fmt.Errorf("таймаут ожидания готовности фабрики инфраструктуры"))
		return fmt.Errorf("фабрика инфраструктуры не стала готовой в течение 10 секунд")
	}

	// Регистрируем компоненты
	il.registerInfrastructureComponents()

	il.initialized = true
	il.updateState(StateInitialized)
	logger.Info("✅ Слой инфраструктуры инициализирован (фабрика готова)")
	return nil
}

// waitForFactoryReady ожидает готовности фабрики инфраструктуры
func (il *InfrastructureLayer) waitForFactoryReady(timeout time.Duration) bool {
	if il.infraFactory == nil {
		return false
	}

	startTime := time.Now()
	checkInterval := 100 * time.Millisecond

	for {
		if il.infraFactory.IsReady() {
			logger.Info("   ✅ Фабрика инфраструктуры готова")
			return true
		}

		if time.Since(startTime) > timeout {
			logger.Warn("   ⚠️ Таймаут ожидания готовности фабрики инфраструктуры")
			return false
		}

		time.Sleep(checkInterval)
	}
}

// Start запускает слой инфраструктуры
func (il *InfrastructureLayer) Start() error {
	if !il.IsInitialized() {
		return fmt.Errorf("слой инфраструктуры не инициализирован")
	}

	if il.IsRunning() {
		return fmt.Errorf("слой инфраструктуры уже запущен")
	}

	il.updateState(StateStarting)
	logger.Info("🚀 Запуск слоя инфраструктуры...")

	// Запускаем фабрику инфраструктуры
	if il.infraFactory != nil {
		if err := il.infraFactory.Start(); err != nil {
			il.setError(err)
			logger.Error("❌ InfrastructureLayer: ОШИБКА в infraFactory.Start(): %v", err)
			return fmt.Errorf("не удалось запустить фабрику инфраструктуры: %w", err)
		}
		logger.Info("   • Фабрика инфраструктуры запущена")
	} else {
		logger.Error("❌ InfrastructureLayer: infraFactory равен nil!")
		return fmt.Errorf("фабрика инфраструктуры не создана")
	}

	il.running = true
	il.startTime = il.startTime // Используем время из BaseLayer
	il.updateState(StateRunning)
	logger.Info("✅ Слой инфраструктуры запущен")
	return nil
}

// Stop останавливает слой инфраструктуры
func (il *InfrastructureLayer) Stop() error {
	if !il.IsRunning() {
		return nil
	}

	il.updateState(StateStopping)
	logger.Info("🛑 Остановка слоя инфраструктуры...")

	// Останавливаем фабрику инфраструктуры
	if il.infraFactory != nil {
		if err := il.infraFactory.Stop(); err != nil {
			logger.Warn("⚠️ Ошибка остановки фабрики инфраструктуры: %v", err)
		}
		logger.Info("   • Фабрика инфраструктуры остановлена")
	}

	il.running = false
	il.updateState(StateStopped)
	logger.Info("✅ Слой инфраструктуры остановлен")
	return nil
}

// Reset сбрасывает слой инфраструктуры
func (il *InfrastructureLayer) Reset() error {
	logger.Info("🔄 Сброс слоя инфраструктуры...")

	// Останавливаем если запущен
	if il.IsRunning() {
		il.Stop()
	}

	// Сбрасываем фабрику
	if il.infraFactory != nil {
		il.infraFactory.Reset()
	}

	// Сбрасываем базовый слой
	il.BaseLayer.Reset()

	il.infraFactory = nil
	logger.Info("✅ Слой инфраструктуры сброшен")
	return nil
}

// GetInfrastructureFactory возвращает фабрику инфраструктуры
func (il *InfrastructureLayer) GetInfrastructureFactory() *infrastructure_factory.InfrastructureFactory {
	il.mu.RLock()
	defer il.mu.RUnlock()
	return il.infraFactory
}

// registerInfrastructureComponents регистрирует компоненты инфраструктуры
func (il *InfrastructureLayer) registerInfrastructureComponents() {
	if il.infraFactory == nil {
		return
	}

	// Регистрируем основные компоненты
	components := map[string]string{
		"DatabaseService": "сервис базы данных",
		"RedisService":    "сервис Redis",
		"EventBus":        "шина событий",
		"APIClient":       "API клиент",
		"StorageFactory":  "фабрика хранилищ",
	}

	for name, description := range components {
		il.registerComponent(name, &LazyComponent{
			name:        name,
			description: description,
			getter:      il.getInfrastructureComponent(name),
		})
		logger.Debug("📦 Зарегистрирован компонент инфраструктуры: %s (%s)", name, description)
	}
}

// getInfrastructureComponent возвращает геттер для компонента инфраструктуры
func (il *InfrastructureLayer) getInfrastructureComponent(name string) func() (interface{}, error) {
	return func() (interface{}, error) {
		if il.infraFactory == nil {
			return nil, fmt.Errorf("фабрика инфраструктуры не создана")
		}

		switch name {
		case "DatabaseService":
			return il.infraFactory.CreateDatabaseService()
		case "RedisService":
			return il.infraFactory.CreateRedisService()
		case "EventBus":
			return il.infraFactory.CreateEventBus()
		case "APIClient":
			return il.infraFactory.CreateAPIClient()
		case "StorageFactory":
			return il.infraFactory.CreateStorageFactory()
		default:
			return nil, fmt.Errorf("неизвестный компонент инфраструктуры: %s", name)
		}
	}
}

// LazyComponent ленивый компонент (создается при первом обращении)
type LazyComponent struct {
	name        string
	description string
	getter      func() (interface{}, error)
	cache       interface{}
	cached      bool
}

// Get возвращает компонент (лениво создает при первом вызове)
func (lc *LazyComponent) Get() (interface{}, error) {
	if lc.cached {
		return lc.cache, nil
	}

	component, err := lc.getter()
	if err != nil {
		return nil, err
	}

	lc.cache = component
	lc.cached = true
	return component, nil
}

// Name возвращает имя компонента
func (lc *LazyComponent) Name() string {
	return lc.name
}

// Description возвращает описание компонента
func (lc *LazyComponent) Description() string {
	return lc.description
}
