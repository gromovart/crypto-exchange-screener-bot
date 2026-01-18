// application/layer_manager/factory.go
package layer_manager

import (
	"crypto-exchange-screener-bot/application/layer_manager/layers"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"time"
)

// LayerFactory фабрика для создания слоев
type LayerFactory struct {
	config *config.Config
}

// NewLayerFactory создает фабрику слоев
func NewLayerFactory(cfg *config.Config) *LayerFactory {
	return &LayerFactory{
		config: cfg,
	}
}

// CreateLayers создает все слои
func (lf *LayerFactory) CreateLayers() (*layers.LayerRegistry, error) {
	logger.Info("🏗️  Создание слоев через LayerFactory...")

	// 1. Создаем реестр слоев
	registry := layers.NewLayerRegistry()

	// 2. Создаем слой инфраструктуры
	logger.Debug("Создание InfrastructureLayer...")
	infraLayer := layers.NewInfrastructureLayer(lf.config)
	if err := registry.Register(infraLayer); err != nil {
		return nil, fmt.Errorf("не удалось зарегистрировать InfrastructureLayer: %w", err)
	}

	// 3. Инициализируем слой инфраструктуры
	logger.Debug("Инициализация InfrastructureLayer...")
	if err := infraLayer.Initialize(); err != nil {
		return nil, fmt.Errorf("не удалось инициализировать InfrastructureLayer: %w", err)
	}
	logger.Info("✅ InfrastructureLayer инициализирован")

	// 4. ЗАПУСКАЕМ ОБЯЗАТЕЛЬНЫЕ СЕРВИСЫ ДЛЯ CoreLayer
	// Проблема: CoreLayer требует работающих PostgreSQL и Redis для создания UserService
	// Решение: Запускаем эти сервисы до инициализации CoreLayer
	logger.Debug("Запуск обязательных сервисов для CoreLayer...")
	if err := lf.startEssentialServices(infraLayer); err != nil {
		return nil, fmt.Errorf("не удалось запустить обязательные сервисы: %w", err)
	}

	// 5. Ждем готовности InfrastructureFactory (инициализация)
	logger.Debug("Ожидание готовности InfrastructureFactory (инициализация)...")
	if !lf.waitForInfrastructureInitialized(infraLayer, 30*time.Second) {
		return nil, fmt.Errorf("таймаут ожидания инициализации InfrastructureFactory")
	}
	logger.Info("✅ InfrastructureFactory инициализирована")

	// 6. Создаем слой ядра (зависит от инфраструктуры)
	logger.Debug("Создание CoreLayer...")
	coreLayer := layers.NewCoreLayer(lf.config, infraLayer)
	if err := registry.Register(coreLayer); err != nil {
		return nil, fmt.Errorf("не удалось зарегистрировать CoreLayer: %w", err)
	}

	// 7. Инициализируем слой ядра
	logger.Debug("Инициализация CoreLayer...")
	if err := coreLayer.Initialize(); err != nil {
		return nil, fmt.Errorf("не удалось инициализировать CoreLayer: %w", err)
	}
	logger.Info("✅ CoreLayer инициализирован")

	// 8. Создаем слой доставки (зависит от ядра)
	logger.Debug("Создание DeliveryLayer...")
	deliveryLayer := layers.NewDeliveryLayer(lf.config, coreLayer)
	if err := registry.Register(deliveryLayer); err != nil {
		return nil, fmt.Errorf("не удалось зарегистрировать DeliveryLayer: %w", err)
	}

	// 9. Инициализируем слой доставки
	logger.Debug("Инициализация DeliveryLayer...")
	if err := deliveryLayer.Initialize(); err != nil {
		return nil, fmt.Errorf("не удалось инициализировать DeliveryLayer: %w", err)
	}
	logger.Info("✅ DeliveryLayer инициализирован")

	// 10. Валидация зависимостей
	logger.Debug("Валидация зависимостей слоев...")
	if err := registry.ValidateDependencies(); err != nil {
		return nil, fmt.Errorf("ошибка зависимостей слоев: %w", err)
	}
	logger.Info("✅ Зависимости слоев валидированы")

	logger.Info("✅ Все слои созданы и инициализированы")
	return registry, nil
}

// startEssentialServices запускает обязательные сервисы для CoreLayer
// CoreLayer требует работающих PostgreSQL и Redis для создания UserService
func (lf *LayerFactory) startEssentialServices(infraLayer *layers.InfrastructureLayer) error {
	if infraLayer == nil {
		return fmt.Errorf("InfrastructureLayer не установлен")
	}

	// Получаем фабрику инфраструктуры
	factory := infraLayer.GetInfrastructureFactory()
	if factory == nil {
		return fmt.Errorf("фабрика инфраструктуры не создана")
	}

	// Получаем конфигурацию
	config := factory.GetConfig()
	if config == nil {
		return fmt.Errorf("конфигурация не доступна")
	}

	// 1. ЗАПУСКАЕМ DATABASESERVICE (ОБЯЗАТЕЛЬНО)
	// UserService не может работать без PostgreSQL
	logger.Debug("Запуск DatabaseService для CoreLayer...")
	dbService, err := factory.CreateDatabaseService()
	if err != nil {
		return fmt.Errorf("не удалось создать DatabaseService: %w", err)
	}

	if dbService == nil {
		return fmt.Errorf("DatabaseService не создан")
	}

	if !dbService.IsRunning() {
		return fmt.Errorf("DatabaseService не запущен")
	}
	logger.Info("✅ DatabaseService запущен для CoreLayer")

	// 2. ЗАПУСКАЕМ REDISSERVICE (ОБЯЗАТЕЛЬНО, ЕСЛИ ВКЛЮЧЕН В КОНФИГУРАЦИИ)
	// UserService использует Redis для кэширования сессий и данных
	if config.Redis.Enabled {
		logger.Debug("Запуск RedisService для CoreLayer...")
		redisService, err := factory.CreateRedisService()
		if err != nil {
			// Redis может быть не критичным, но логируем предупреждение
			logger.Warn("⚠️ Не удалось создать RedisService: %v", err)
			logger.Warn("⚠️ UserService будет работать без кэширования (ограниченный функционал)")
			// Не падаем, если Redis не доступен
		} else if redisService != nil {
			if !redisService.IsRunning() {
				logger.Warn("⚠️ RedisService не запущен, UserService будет работать без кэширования")
			} else {
				logger.Info("✅ RedisService запущен для CoreLayer")
			}
		}
	} else {
		logger.Info("ℹ️ Redis отключен в конфигурации, UserService будет работать без кэширования")
	}

	return nil
}

// waitForInfrastructureInitialized ожидает инициализации инфраструктуры (без запуска)
func (lf *LayerFactory) waitForInfrastructureInitialized(infraLayer *layers.InfrastructureLayer, timeout time.Duration) bool {
	startTime := time.Now()
	checkInterval := 500 * time.Millisecond

	for {
		// Проверяем что слой инициализирован
		if infraLayer.IsInitialized() {
			// Получаем фабрику инфраструктуры
			factory := infraLayer.GetInfrastructureFactory()
			if factory != nil && factory.IsReady() {
				return true
			}
		}

		// Проверяем таймаут
		if time.Since(startTime) > timeout {
			logger.Warn("⏰ Таймаут ожидания инициализации InfrastructureFactory")
			return false
		}

		// Ждем перед следующей проверкой
		time.Sleep(checkInterval)
	}
}
