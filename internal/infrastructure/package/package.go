// internal/infrastructure/package/package.go
package infrastructure_factory

import (
	"crypto-exchange-screener-bot/internal/infrastructure/api/exchanges/bybit"
	"crypto-exchange-screener-bot/internal/infrastructure/cache/redis"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/in_memory_storage"
	storage_factory "crypto-exchange-screener-bot/internal/infrastructure/persistence/in_memory_storage/factory"
	database "crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/database"
	postgres_factory "crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/factory"
	events "crypto-exchange-screener-bot/internal/infrastructure/transport/event_bus"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"sync"
)

// InfrastructureFactory главная фабрика инфраструктурных компонентов
type InfrastructureFactory struct {
	config            *config.Config
	databaseService   *database.DatabaseService
	redisService      *redis.RedisService
	redisCache        *redis.Cache
	eventBus          *events.EventBus
	apiClient         *bybit.BybitClient
	repositoryFactory *postgres_factory.RepositoryFactory
	storageFactory    *storage_factory.StorageFactory
	mu                sync.RWMutex
	initialized       bool
}

// InfrastructureDependencies зависимости для фабрики инфраструктуры
type InfrastructureDependencies struct {
	Config *config.Config
}

// NewInfrastructureFactory создает главную фабрику инфраструктуры
func NewInfrastructureFactory(deps InfrastructureDependencies) (*InfrastructureFactory, error) {
	logger.Info("🏗️  Создание главной фабрики инфраструктуры...")

	if deps.Config == nil {
		return nil, fmt.Errorf("конфигурация не может быть nil")
	}

	factory := &InfrastructureFactory{
		config:      deps.Config,
		initialized: true,
	}

	logger.Info("✅ Главная фабрика инфраструктуры создана")
	return factory, nil
}

// Initialize инициализирует все инфраструктурные компоненты
func (f *InfrastructureFactory) Initialize() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.initialized {
		return fmt.Errorf("фабрика инфраструктуры не инициализирована")
	}

	logger.Info("🔧 Инициализация инфраструктурных компонентов...")

	// 1. Создаем сервис базы данных
	if f.config.Database.Enabled {
		f.databaseService = database.NewDatabaseService(f.config)
		if err := f.databaseService.Start(); err != nil {
			logger.Warn("⚠️ Не удалось запустить DatabaseService: %v", err)
		} else {
			logger.Info("✅ DatabaseService инициализирован")
		}
	}

	// 2. Создаем Redis сервис и кэш
	if f.config.Redis.Enabled {
		f.redisService = redis.NewRedisService(f.config)
		if err := f.redisService.Start(); err != nil {
			logger.Warn("⚠️ Не удалось запустить RedisService: %v", err)
		} else {
			logger.Info("✅ RedisService инициализирован")
			// Создаем кэш после успешного запуска сервиса
			f.redisCache = f.redisService.GetCache()
			if f.redisCache != nil {
				logger.Info("✅ Redis кэш создан")
			}
		}
	}

	// 3. Создаем EventBus
	eventBusConfig := events.EventBusConfig{
		BufferSize:    f.config.EventBus.BufferSize,
		WorkerCount:   f.config.EventBus.WorkerCount,
		EnableMetrics: f.config.EventBus.EnableMetrics,
		EnableLogging: f.config.EventBus.EnableLogging,
	}
	f.eventBus = events.NewEventBus(eventBusConfig)
	logger.Info("✅ EventBus создан")

	// 4. Создаем API клиент
	if f.config.Exchange == "BYBIT" || f.config.Exchange == "BYBIT futures" {
		f.apiClient = bybit.NewBybitClient(f.config)
		logger.Info("✅ Bybit API клиент создан")
	}

	// 5. Создаем фабрику хранилищ
	storageFactoryConfig := &storage_factory.StorageFactoryConfig{
		DefaultStorageConfig: &storage.StorageConfig{
			MaxHistoryPerSymbol: 10000,
			MaxSymbols:          1000,
			CleanupInterval:     5 * 60,       // 5 минут в секундах
			RetentionPeriod:     24 * 60 * 60, // 24 часа в секундах
		},
		EnableCleanupRoutine: true,
		CleanupInterval:      60, // 1 минута в секундах
		MaxCustomStorages:    10,
	}
	storageFactory, err := storage_factory.NewStorageFactory(storage_factory.StorageDependencies{
		Config: storageFactoryConfig,
	})
	if err != nil {
		logger.Warn("⚠️ Не удалось создать StorageFactory: %v", err)
	} else {
		f.storageFactory = storageFactory
		if err := f.storageFactory.Initialize(); err != nil {
			logger.Warn("⚠️ Не удалось инициализировать StorageFactory: %v", err)
		} else {
			logger.Info("✅ StorageFactory инициализирована")
		}
	}

	logger.Info("✅ Все инфраструктурные компоненты инициализированы")
	return nil
}

// CreateDatabaseService создает или возвращает DatabaseService
func (f *InfrastructureFactory) CreateDatabaseService() (*database.DatabaseService, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.initialized {
		return nil, fmt.Errorf("фабрика инфраструктуры не инициализирована")
	}

	if f.databaseService == nil {
		if !f.config.Database.Enabled {
			return nil, fmt.Errorf("PostgreSQL отключен в конфигурации")
		}

		f.databaseService = database.NewDatabaseService(f.config)
		if err := f.databaseService.Start(); err != nil {
			return nil, fmt.Errorf("не удалось создать DatabaseService: %w", err)
		}
		logger.Info("✅ DatabaseService создан")
	}

	return f.databaseService, nil
}

// CreateRedisService создает или возвращает RedisService
func (f *InfrastructureFactory) CreateRedisService() (*redis.RedisService, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.initialized {
		return nil, fmt.Errorf("фабрика инфраструктуры не инициализирована")
	}

	if f.redisService == nil {
		if !f.config.Redis.Enabled {
			return nil, fmt.Errorf("Redis отключен в конфигурации")
		}

		f.redisService = redis.NewRedisService(f.config)
		if err := f.redisService.Start(); err != nil {
			return nil, fmt.Errorf("не удалось создать RedisService: %w", err)
		}
		logger.Info("✅ RedisService создан")
	}

	return f.redisService, nil
}

// CreateRedisCache создает или возвращает Redis Cache
func (f *InfrastructureFactory) CreateRedisCache() (*redis.Cache, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.initialized {
		return nil, fmt.Errorf("фабрика инфраструктуры не инициализирована")
	}

	if f.redisCache == nil {
		// Сначала создаем RedisService если нужно
		redisService, err := f.CreateRedisService()
		if err != nil {
			return nil, fmt.Errorf("не удалось создать RedisService для кэша: %w", err)
		}

		f.redisCache = redisService.GetCache()
		if f.redisCache == nil {
			return nil, fmt.Errorf("не удалось получить Redis кэш")
		}
		logger.Info("✅ Redis кэш создан")
	}

	return f.redisCache, nil
}

// CreateEventBus создает или возвращает EventBus
func (f *InfrastructureFactory) CreateEventBus() (*events.EventBus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.initialized {
		return nil, fmt.Errorf("фабрика инфраструктуры не инициализирована")
	}

	if f.eventBus == nil {
		eventBusConfig := events.EventBusConfig{
			BufferSize:    f.config.EventBus.BufferSize,
			WorkerCount:   f.config.EventBus.WorkerCount,
			EnableMetrics: f.config.EventBus.EnableMetrics,
			EnableLogging: f.config.EventBus.EnableLogging,
		}
		f.eventBus = events.NewEventBus(eventBusConfig)
		logger.Info("✅ EventBus создан")
	}

	return f.eventBus, nil
}

// CreateAPIClient создает или возвращает API клиент
func (f *InfrastructureFactory) CreateAPIClient() (*bybit.BybitClient, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.initialized {
		return nil, fmt.Errorf("фабрика инфраструктуры не инициализирована")
	}

	if f.apiClient == nil {
		if f.config.Exchange == "BYBIT" || f.config.Exchange == "BYBIT futures" {
			f.apiClient = bybit.NewBybitClient(f.config)
			logger.Info("✅ Bybit API клиент создан")
		} else {
			return nil, fmt.Errorf("неподдерживаемая биржа: %s", f.config.Exchange)
		}
	}

	return f.apiClient, nil
}

// CreateRepositoryFactory создает или возвращает фабрику репозиториев
func (f *InfrastructureFactory) CreateRepositoryFactory() (*postgres_factory.RepositoryFactory, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.initialized {
		return nil, fmt.Errorf("фабрика инфраструктуры не инициализирована")
	}

	if f.repositoryFactory == nil {
		if !f.config.Database.Enabled {
			return nil, fmt.Errorf("PostgreSQL отключен в конфигурации")
		}

		// Создаем DatabaseService если нужно
		databaseService, err := f.CreateDatabaseService()
		if err != nil {
			return nil, fmt.Errorf("не удалось создать DatabaseService: %w", err)
		}

		// Создаем Redis Cache если нужно (для большинства репозиториев)
		redisCache, err := f.CreateRedisCache()
		if err != nil {
			logger.Warn("⚠️ Redis кэш недоступен, репозитории будут работать без кэша")
			// APIKeyRepository не требует кэша, но требует encryptionKey
		}

		// Получаем ключ шифрования из конфигурации
		// TODO: Добавить поле encryptionKey в конфигурацию
		encryptionKey := "default-encryption-key" // Временное значение

		f.repositoryFactory, err = postgres_factory.NewRepositoryFactory(postgres_factory.RepositoryDependencies{
			DatabaseService: databaseService,
			Cache:           redisCache,
			EncryptionKey:   encryptionKey,
		})
		if err != nil {
			return nil, fmt.Errorf("не удалось создать RepositoryFactory: %w", err)
		}

		if err := f.repositoryFactory.Initialize(); err != nil {
			return nil, fmt.Errorf("не удалось инициализировать RepositoryFactory: %w", err)
		}

		logger.Info("✅ RepositoryFactory создана")
	}

	return f.repositoryFactory, nil
}

// CreateStorageFactory создает или возвращает фабрику хранилищ
func (f *InfrastructureFactory) CreateStorageFactory() (*storage_factory.StorageFactory, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.initialized {
		return nil, fmt.Errorf("фабрика инфраструктуры не инициализирована")
	}

	if f.storageFactory == nil {
		storageFactoryConfig := &storage_factory.StorageFactoryConfig{
			DefaultStorageConfig: &storage.StorageConfig{
				MaxHistoryPerSymbol: 10000,
				MaxSymbols:          1000,
				CleanupInterval:     5 * 60,       // 5 минут в секундах
				RetentionPeriod:     24 * 60 * 60, // 24 часа в секундах
			},
			EnableCleanupRoutine: true,
			CleanupInterval:      60, // 1 минута в секундах
			MaxCustomStorages:    10,
		}

		var err error
		f.storageFactory, err = storage_factory.NewStorageFactory(storage_factory.StorageDependencies{
			Config: storageFactoryConfig,
		})
		if err != nil {
			return nil, fmt.Errorf("не удалось создать StorageFactory: %w", err)
		}

		if err := f.storageFactory.Initialize(); err != nil {
			return nil, fmt.Errorf("не удалось инициализировать StorageFactory: %w", err)
		}

		logger.Info("✅ StorageFactory создана")
	}

	return f.storageFactory, nil
}

// GetDefaultStorage создает или возвращает хранилище по умолчанию через фабрику
func (f *InfrastructureFactory) GetDefaultStorage() (storage.PriceStorage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.initialized {
		return nil, fmt.Errorf("фабрика инфраструктуры не инициализирована")
	}

	// Создаем StorageFactory если нужно
	storageFactory, err := f.CreateStorageFactory()
	if err != nil {
		return nil, fmt.Errorf("не удалось создать StorageFactory: %w", err)
	}

	// Получаем хранилище по умолчанию через фабрику
	return storageFactory.CreateDefaultStorage()
}

// GetAllComponents создает все инфраструктурные компоненты
func (f *InfrastructureFactory) GetAllComponents() (map[string]interface{}, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.initialized {
		return nil, fmt.Errorf("фабрика инфраструктуры не инициализирована")
	}

	logger.Info("🏭 Создание всех инфраструктурных компонентов...")

	components := make(map[string]interface{})

	// DatabaseService
	if f.config.Database.Enabled {
		dbService, err := f.CreateDatabaseService()
		if err != nil {
			logger.Warn("⚠️ Не удалось создать DatabaseService: %v", err)
		} else {
			components["DatabaseService"] = dbService
		}
	}

	// RedisService
	if f.config.Redis.Enabled {
		redisService, err := f.CreateRedisService()
		if err != nil {
			logger.Warn("⚠️ Не удалось создать RedisService: %v", err)
		} else {
			components["RedisService"] = redisService
		}
	}

	// Redis Cache
	if f.config.Redis.Enabled {
		redisCache, err := f.CreateRedisCache()
		if err != nil {
			logger.Warn("⚠️ Не удалось создать Redis Cache: %v", err)
		} else {
			components["RedisCache"] = redisCache
		}
	}

	// EventBus
	eventBus, err := f.CreateEventBus()
	if err != nil {
		logger.Warn("⚠️ Не удалось создать EventBus: %v", err)
	} else {
		components["EventBus"] = eventBus
	}

	// APIClient
	apiClient, err := f.CreateAPIClient()
	if err != nil {
		logger.Warn("⚠️ Не удалось создать APIClient: %v", err)
	} else {
		components["APIClient"] = apiClient
	}

	// RepositoryFactory (только если включена БД)
	if f.config.Database.Enabled {
		repoFactory, err := f.CreateRepositoryFactory()
		if err != nil {
			logger.Warn("⚠️ Не удалось создать RepositoryFactory: %v", err)
		} else {
			components["RepositoryFactory"] = repoFactory
		}
	}

	// StorageFactory
	storageFactory, err := f.CreateStorageFactory()
	if err != nil {
		logger.Warn("⚠️ Не удалось создать StorageFactory: %v", err)
	} else {
		components["StorageFactory"] = storageFactory
	}

	logger.Info("✅ Все инфраструктурные компоненты созданы")
	return components, nil
}

// GetAllRepositories создает все репозитории через RepositoryFactory
func (f *InfrastructureFactory) GetAllRepositories() (map[string]interface{}, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.initialized {
		return nil, fmt.Errorf("фабрика инфраструктуры не инициализирована")
	}

	if !f.config.Database.Enabled {
		return nil, fmt.Errorf("PostgreSQL отключен в конфигурации")
	}

	// Создаем RepositoryFactory если нужно
	repoFactory, err := f.CreateRepositoryFactory()
	if err != nil {
		return nil, fmt.Errorf("не удалось создать RepositoryFactory: %w", err)
	}

	// Получаем все репозитории через фабрику
	return repoFactory.GetAllRepositories()
}

// GetAllStorages создает все хранилища через StorageFactory
func (f *InfrastructureFactory) GetAllStorages() (map[string]storage.PriceStorage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.initialized {
		return nil, fmt.Errorf("фабрика инфраструктуры не инициализирована")
	}

	// Создаем StorageFactory если нужно
	storageFactory, err := f.CreateStorageFactory()
	if err != nil {
		return nil, fmt.Errorf("не удалось создать StorageFactory: %w", err)
	}

	// Получаем все хранилища через фабрику
	return storageFactory.GetAllStorages(), nil
}

// Validate проверяет готовность фабрики
func (f *InfrastructureFactory) Validate() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if !f.initialized {
		logger.Warn("⚠️ Фабрика инфраструктуры не инициализирована")
		return false
	}

	if f.config == nil {
		logger.Warn("⚠️ Конфигурация не установлена")
		return false
	}

	logger.Info("✅ Фабрика инфраструктуры валидирована")
	return true
}

// GetHealthStatus возвращает статус здоровья инфраструктуры
func (f *InfrastructureFactory) GetHealthStatus() map[string]interface{} {
	f.mu.RLock()
	defer f.mu.RUnlock()

	status := map[string]interface{}{
		"initialized":              f.initialized,
		"config_available":         f.config != nil,
		"database_service_ready":   f.databaseService != nil,
		"redis_service_ready":      f.redisService != nil,
		"redis_cache_ready":        f.redisCache != nil,
		"event_bus_ready":          f.eventBus != nil,
		"api_client_ready":         f.apiClient != nil,
		"repository_factory_ready": f.repositoryFactory != nil,
		"storage_factory_ready":    f.storageFactory != nil,
	}

	// Добавляем статусы сервисов если они созданы
	if f.databaseService != nil {
		status["database_state"] = f.databaseService.State()
		status["database_healthy"] = f.databaseService.HealthCheck()
	}
	if f.redisService != nil {
		status["redis_state"] = f.redisService.State()
		status["redis_healthy"] = f.redisService.HealthCheck()
	}
	if f.eventBus != nil {
		status["event_bus_healthy"] = f.eventBus.HealthCheck()
	}
	if f.repositoryFactory != nil {
		status["repository_factory_healthy"] = f.repositoryFactory.Validate()
	}
	if f.storageFactory != nil {
		status["storage_factory_healthy"] = f.storageFactory.Validate()
	}

	return status
}

// Stop останавливает все инфраструктурные компоненты
func (f *InfrastructureFactory) Stop() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	logger.Info("🛑 Остановка инфраструктурных компонентов...")

	errors := []error{}

	// Останавливаем DatabaseService
	if f.databaseService != nil {
		if err := f.databaseService.Stop(); err != nil {
			errors = append(errors, fmt.Errorf("ошибка остановки DatabaseService: %w", err))
		} else {
			logger.Info("✅ DatabaseService остановлен")
		}
	}

	// Останавливаем RedisService
	if f.redisService != nil {
		if err := f.redisService.Stop(); err != nil {
			errors = append(errors, fmt.Errorf("ошибка остановки RedisService: %w", err))
		} else {
			logger.Info("✅ RedisService остановлен")
		}
	}

	// Останавливаем EventBus
	if f.eventBus != nil {
		f.eventBus.Stop()
		logger.Info("✅ EventBus остановлен")
	}

	// Сбрасываем фабрики
	if f.repositoryFactory != nil {
		f.repositoryFactory.Reset()
		logger.Info("✅ RepositoryFactory сброшена")
	}

	if f.storageFactory != nil {
		f.storageFactory.Reset()
		logger.Info("✅ StorageFactory сброшена")
	}

	f.initialized = false

	if len(errors) > 0 {
		return fmt.Errorf("ошибки при остановке: %v", errors)
	}

	logger.Info("✅ Все инфраструктурные компоненты остановлены")
	return nil
}

// Reset сбрасывает фабрику
func (f *InfrastructureFactory) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.databaseService = nil
	f.redisService = nil
	f.redisCache = nil
	f.eventBus = nil
	f.apiClient = nil
	f.repositoryFactory = nil
	f.storageFactory = nil
	f.initialized = false

	logger.Info("🔄 Фабрика инфраструктуры сброшена")
}

// IsReady проверяет готовность фабрики
func (f *InfrastructureFactory) IsReady() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return f.initialized && f.config != nil
}

// GetConfig возвращает конфигурацию
func (f *InfrastructureFactory) GetConfig() *config.Config {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.config
}

// UpdateConfig обновляет конфигурацию
func (f *InfrastructureFactory) UpdateConfig(newConfig *config.Config) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.config = newConfig
}
