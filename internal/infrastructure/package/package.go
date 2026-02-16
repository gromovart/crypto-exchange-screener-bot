// internal/infrastructure/package/package.go
package infrastructure_factory

import (
	"crypto-exchange-screener-bot/internal/infrastructure/api/exchanges/bybit"
	"crypto-exchange-screener-bot/internal/infrastructure/cache/redis"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	database "crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/database"
	postgres_factory "crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/factory"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/repository/activity"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/repository/api_key"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/repository/invoice"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/repository/payment"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/repository/plan"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/repository/session"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/repository/subscription"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/repository/users"
	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	redis_storage_factory "crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage/factory"
	events "crypto-exchange-screener-bot/internal/infrastructure/transport/event_bus"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"sync"
	"time"
)

// Алиасы для совместимости
type PriceStorage = redis_storage_factory.PriceStorage
type StorageFactory = redis_storage_factory.StorageFactory

// InfrastructureFactory главная фабрика инфраструктурных компонентов
type InfrastructureFactory struct {
	config            *config.Config
	databaseService   *database.DatabaseService
	redisService      *redis.RedisService
	redisCache        *redis.Cache
	eventBus          *events.EventBus
	apiClient         *bybit.BybitClient
	repositoryFactory *postgres_factory.RepositoryFactory
	storageFactory    *StorageFactory
	mu                sync.RWMutex
	initialized       bool
	running           bool
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
		initialized: false,
		running:     false,
	}

	logger.Info("✅ Главная фабрика инфраструктуры создана")
	return factory, nil
}

// Initialize инициализирует все инфраструктурные компоненты
func (f *InfrastructureFactory) Initialize() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.initialized {
		return fmt.Errorf("фабрика инфраструктуры уже инициализирована")
	}

	logger.Info("🔧 Инициализация инфраструктурных компонентов...")

	// 1. Создаем сервис базы данных
	if f.config.Database.Enabled {
		f.databaseService = database.NewDatabaseService(f.config)
		logger.Info("✅ DatabaseService создан (не запущен)")
	}

	// 2. Создаем Redis сервис
	if f.config.Redis.Enabled {
		f.redisService = redis.NewRedisService(f.config)
		logger.Info("✅ RedisService создан (не запущен)")
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

	// 5. Создаем фабрику хранилищ (redis_storage_factory)
	storageFactoryConfig := &redis_storage_factory.StorageFactoryConfig{
		DefaultStorageConfig: &storage.StorageConfig{
			MaxHistoryPerSymbol: 10000,
			MaxSymbols:          1000,
			CleanupInterval:     5 * 60 * time.Second,
			RetentionPeriod:     24 * 60 * 60 * time.Second,
		},
		EnableCleanupRoutine: true,
		CleanupInterval:      60 * time.Second,
		MaxCustomStorages:    10,
	}

	// Передаем RedisService как RedisClient
	var redisClient interface{} = nil
	if f.config.Redis.Enabled && f.redisService != nil {
		redisClient = f.redisService
		logger.Debug("🔧 RedisService передан в StorageFactory")
	} else if f.config.Redis.Enabled {
		logger.Warn("⚠️ RedisService не создан, но Redis включен в конфигурации")
	}

	storageFactory, err := redis_storage_factory.NewStorageFactory(redis_storage_factory.StorageDependencies{
		Config:      storageFactoryConfig,
		RedisClient: redisClient,
	})
	if err != nil {
		logger.Warn("⚠️ Не удалось создать Redis StorageFactory: %v", err)
	} else {
		f.storageFactory = storageFactory
		if err := f.storageFactory.Initialize(); err != nil {
			logger.Warn("⚠️ Не удалось инициализировать Redis StorageFactory: %v", err)
		} else {
			logger.Info("✅ Redis StorageFactory инициализирована")
		}
	}

	// 5. Создаем фабрику репозиториев
	repositoryFactory, err := postgres_factory.NewRepositoryFactory(postgres_factory.RepositoryDependencies{
		Cache:           f.redisCache,
		DatabaseService: f.databaseService,
		EncryptionKey:   "",
	})

	if err != nil {
		logger.Warn("⚠️ Не удалось создать RepositoryFactory: %v", err)
	} else {
		f.repositoryFactory = repositoryFactory
		if err := f.repositoryFactory.Initialize(); err != nil {
			logger.Warn("⚠️ Не удалось инициализировать RepositoryFactory: %v", err)
		} else {
			logger.Info("✅ RepositoryFactory инициализирована ")
		}

	}

	f.initialized = true
	logger.Info("✅ Все инфраструктурные компоненты инициализированы")
	return nil
}

// Start запускает все инфраструктурные компоненты
func (f *InfrastructureFactory) Start() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.initialized {
		return fmt.Errorf("фабрика инфраструктуры не инициализирована")
	}

	if f.running {
		return fmt.Errorf("фабрика инфраструктуры уже запущена")
	}

	logger.Info("🚀 Запуск инфраструктурных компонентов...")

	// Запускаем компоненты
	errors := []error{}

	// 1. Запускаем DatabaseService (ЕСЛИ ЕЩЕ НЕ ЗАПУЩЕН)
	if f.config.Database.Enabled && f.databaseService != nil {
		if err := f.startDatabaseService(); err != nil {
			errors = append(errors, err)
		}
	}

	// 2. Запускаем RedisService (ЕСЛИ ЕЩЕ НЕ ЗАПУЩЕН)
	if f.config.Redis.Enabled && f.redisService != nil {
		if err := f.startRedisService(); err != nil {
			errors = append(errors, err)
		}
	}

	// 3. Запускаем EventBus (если еще не запущен)
	if err := f.startEventBus(); err != nil {
		errors = append(errors, err)
	}

	// 4. Запускаем StorageFactory (если еще не запущена)
	if err := f.startStorageFactory(); err != nil {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		// ДОБАВИТЬ: Подробное логирование ошибок
		logger.Error("❌ InfrastructureFactory.Start(): обнаружены ошибки:")
		for i, err := range errors {
			logger.Error("   %d. %v", i+1, err)
		}
		return fmt.Errorf("ошибки при запуске: %v", errors)
	}

	f.running = true
	logger.Info("✅ Все инфраструктурные компоненты запущены")
	return nil
}

// startDatabaseService запускает DatabaseService если еще не запущен
func (f *InfrastructureFactory) startDatabaseService() error {
	if f.databaseService == nil {
		return fmt.Errorf("DatabaseService не создан")
	}

	if !f.databaseService.IsRunning() {
		if err := f.databaseService.Start(); err != nil {
			logger.Warn("⚠️ Не удалось запустить DatabaseService: %v", err)
			return fmt.Errorf("ошибка запуска DatabaseService: %w", err)
		}
		logger.Info("✅ DatabaseService запущен")
	} else {
		logger.Info("✅ DatabaseService уже запущен, пропускаем")
	}
	return nil
}

// startRedisService запускает RedisService если еще не запущен
func (f *InfrastructureFactory) startRedisService() error {
	if f.redisService == nil {
		return fmt.Errorf("RedisService не создан")
	}

	if !f.redisService.IsRunning() {
		if err := f.redisService.Start(); err != nil {
			logger.Warn("⚠️ Не удалось запустить RedisService: %v", err)
			return fmt.Errorf("ошибка запуска RedisService: %w", err)
		}
		logger.Info("✅ RedisService запущен")
		// Создаем кэш после успешного запуска сервиса
		f.redisCache = f.redisService.GetCache()
		if f.redisCache != nil {
			logger.Info("✅ Redis кэш создан")
		}
	} else {
		logger.Info("✅ RedisService уже запущен, пропускаем")
	}
	return nil
}

// startEventBus запускает EventBus если еще не запущен
func (f *InfrastructureFactory) startEventBus() error {
	if f.eventBus == nil {
		return fmt.Errorf("EventBus не создан")
	}

	if !f.eventBus.IsRunning() {
		f.eventBus.Start()
		logger.Info("✅ EventBus запущен")
	} else {
		logger.Info("✅ EventBus уже запущен, пропускаем")
	}
	return nil
}

// startStorageFactory запускает StorageFactory если еще не запущена
func (f *InfrastructureFactory) startStorageFactory() error {
	if f.storageFactory == nil {
		return nil // StorageFactory может быть nil, это не ошибка
	}

	if !f.storageFactory.IsRunning() {
		if err := f.storageFactory.Start(); err != nil {
			logger.Warn("⚠️ Не удалось запустить StorageFactory: %v", err)
			logger.Warn("⚠️ Детали ошибки: %+v", err)
			return fmt.Errorf("ошибка запуска StorageFactory: %w", err)
		}
		logger.Info("✅ StorageFactory запущена")
	} else {
		logger.Info("✅ StorageFactory уже запущена, пропускаем")
	}
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
		logger.Info("✅ DatabaseService создан")
	}

	// ЗАПУСКАЕМ ДАЖЕ ЕСЛИ ФАБРИКА НЕ ЗАПУЩЕНА - ВАЖНО ДЛЯ CoreLayer
	// Проблема: CoreLayer требует работающей БД для создания UserService
	// Решение: Запускаем DatabaseService независимо от состояния фабрики
	if !f.databaseService.IsRunning() {
		if err := f.databaseService.Start(); err != nil {
			return nil, fmt.Errorf("не удалось запустить DatabaseService: %w", err)
		}
		logger.Info("✅ DatabaseService запущен")
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
		logger.Info("✅ RedisService создан")
	}

	// ЗАПУСКАЕМ ДАЖЕ ЕСЛИ ФАБРИКА НЕ ЗАПУЩЕНА - ВАЖНО ДЛЯ CoreLayer
	// Проблема: UserService требует Redis для кэширования
	// Решение: Запускаем RedisService независимо от состояния фабрики
	if !f.redisService.IsRunning() {
		if err := f.redisService.Start(); err != nil {
			return nil, fmt.Errorf("не удалось запустить RedisService: %w", err)
		}
		logger.Info("✅ RedisService запущен")
		// Создаем кэш после запуска
		f.redisCache = f.redisService.GetCache()
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
		return nil, fmt.Errorf("фабрика инфраструктуры не инициализирован")
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

	// Запускаем если фабрика уже запущена
	if f.running && !f.eventBus.IsRunning() {
		f.eventBus.Start()
		logger.Info("✅ EventBus запущен")
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
func (f *InfrastructureFactory) CreateStorageFactory() (*StorageFactory, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.initialized {
		return nil, fmt.Errorf("фабрика инфраструктуры не инициализирована")
	}

	if f.storageFactory == nil {
		storageFactoryConfig := &redis_storage_factory.StorageFactoryConfig{
			DefaultStorageConfig: &storage.StorageConfig{
				MaxHistoryPerSymbol: 10000,
				MaxSymbols:          1000,
				CleanupInterval:     5 * 60,
				RetentionPeriod:     24 * 60 * 60,
			},
			EnableCleanupRoutine: true,
			CleanupInterval:      60,
			MaxCustomStorages:    10,
		}

		var err error
		f.storageFactory, err = redis_storage_factory.NewStorageFactory(redis_storage_factory.StorageDependencies{
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

	// Запускаем если фабрика уже запущена
	if f.running && f.storageFactory != nil && !f.storageFactory.IsRunning() {
		if err := f.storageFactory.Start(); err != nil {
			return nil, fmt.Errorf("не удалось запустить StorageFactory: %w", err)
		}
		logger.Info("✅ StorageFactory запущена")
	}

	return f.storageFactory, nil
}

// GetDefaultStorage создает или возвращает хранилище по умолчанию через фабрику
func (f *InfrastructureFactory) GetDefaultStorage() (PriceStorage, error) {
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
func (f *InfrastructureFactory) GetAllStorages() (map[string]PriceStorage, error) {
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
		"running":                  f.running,
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

	if !f.running {
		return nil
	}

	logger.Info("🛑 Остановка инфраструктурных компонентов...")

	errors := []error{}

	// Останавливаем DatabaseService
	if f.databaseService != nil && f.databaseService.IsRunning() {
		if err := f.databaseService.Stop(); err != nil {
			errors = append(errors, fmt.Errorf("ошибка остановки DatabaseService: %w", err))
		} else {
			logger.Info("✅ DatabaseService остановлен")
		}
	}

	// Останавливаем RedisService
	if f.redisService != nil && f.redisService.IsRunning() {
		if err := f.redisService.Stop(); err != nil {
			errors = append(errors, fmt.Errorf("ошибка остановки RedisService: %w", err))
		} else {
			logger.Info("✅ RedisService остановлен")
		}
	}

	// Останавливаем EventBus
	if f.eventBus != nil && f.eventBus.IsRunning() {
		f.eventBus.Stop()
		logger.Info("✅ EventBus остановлен")
	}

	// Останавливаем StorageFactory
	if f.storageFactory != nil && f.storageFactory.IsRunning() {
		if err := f.storageFactory.Stop(); err != nil {
			errors = append(errors, fmt.Errorf("ошибка остановки StorageFactory: %w", err))
		} else {
			logger.Info("✅ StorageFactory остановлена")
		}
	}

	// Сбрасываем фабрики
	if f.repositoryFactory != nil {
		f.repositoryFactory.Reset()
		logger.Info("✅ RepositoryFactory сброшена")
	}

	f.running = false

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

	// Останавливаем если запущена
	if f.running {
		f.Stop()
	}

	f.databaseService = nil
	f.redisService = nil
	f.redisCache = nil
	f.eventBus = nil
	f.apiClient = nil
	f.repositoryFactory = nil
	f.storageFactory = nil
	f.initialized = false
	f.running = false

	logger.Info("🔄 Фабрика инфраструктуры сброшена")
}

// IsReady проверяет готовность фабрики
func (f *InfrastructureFactory) IsReady() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return f.initialized && f.config != nil
}

// IsRunning проверяет запущена ли фабрика
func (f *InfrastructureFactory) IsRunning() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.running
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

// GetPlanRepository получает репозиторий планов
func (f *InfrastructureFactory) GetPlanRepository() (plan.PlanRepository, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.initialized {
		return nil, fmt.Errorf("фабрика инфраструктуры не инициализирована")
	}

	if f.repositoryFactory == nil {
		return nil, fmt.Errorf("RepositoryFactory не создана")
	}

	// ⭐ Создаем PlanRepository если его нет
	if !f.repositoryFactory.HasRepository("PlanRepository") {
		logger.Info("🔄 Создание PlanRepository...")
		if _, err := f.repositoryFactory.CreatePlanRepository(); err != nil {
			return nil, fmt.Errorf("не удалось создать PlanRepository: %w", err)
		}
		logger.Info("✅ PlanRepository создан")
	}

	repo, err := f.repositoryFactory.GetRepository("PlanRepository")
	if err != nil {
		return nil, err
	}

	planRepo, ok := repo.(plan.PlanRepository)
	if !ok {
		return nil, fmt.Errorf("неверный тип репозитория: ожидается PlanRepository")
	}

	return planRepo, nil
}

// GetUserRepository получает репозиторий пользователей
func (f *InfrastructureFactory) GetUserRepository() (users.UserRepository, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if !f.initialized {
		return nil, fmt.Errorf("фабрика инфраструктуры не инициализирована")
	}

	if f.repositoryFactory == nil {
		return nil, fmt.Errorf("RepositoryFactory не создана")
	}

	repo, err := f.repositoryFactory.GetRepository("UserRepository")
	if err != nil {
		return nil, err
	}

	userRepo, ok := repo.(users.UserRepository)
	if !ok {
		return nil, fmt.Errorf("неверный тип репозитория: ожидается UserRepository")
	}

	return userRepo, nil
}

// GetSubscriptionRepository получает репозиторий подписок
func (f *InfrastructureFactory) GetSubscriptionRepository() (subscription.SubscriptionRepository, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if !f.initialized {
		return nil, fmt.Errorf("фабрика инфраструктуры не инициализирована")
	}

	if f.repositoryFactory == nil {
		return nil, fmt.Errorf("RepositoryFactory не создана")
	}

	repo, err := f.repositoryFactory.GetRepository("SubscriptionRepository")
	if err != nil {
		return nil, err
	}

	subRepo, ok := repo.(subscription.SubscriptionRepository)
	if !ok {
		return nil, fmt.Errorf("неверный тип репозитория: ожидается SubscriptionRepository")
	}

	return subRepo, nil
}

// GetPaymentRepository получает репозиторий платежей
func (f *InfrastructureFactory) GetPaymentRepository() (payment.PaymentRepository, error) {
	f.mu.Lock() // ⚠️ Lock вместо RLock для записи
	defer f.mu.Unlock()

	if !f.initialized {
		return nil, fmt.Errorf("фабрика инфраструктуры не инициализирована")
	}

	if f.repositoryFactory == nil {
		return nil, fmt.Errorf("RepositoryFactory не создана")
	}

	// ⭐ Создаем PaymentRepository если его нет
	if !f.repositoryFactory.HasRepository("PaymentRepository") {
		logger.Info("🔄 Создание PaymentRepository...")
		if _, err := f.repositoryFactory.CreatePaymentRepository(); err != nil {
			logger.Error("❌ Не удалось создать PaymentRepository: %v", err)
			return nil, fmt.Errorf("не удалось создать PaymentRepository: %w", err)
		}
		logger.Info("✅ PaymentRepository создан")
	}

	repo, err := f.repositoryFactory.GetRepository("PaymentRepository")
	if err != nil {
		return nil, err
	}

	paymentRepo, ok := repo.(payment.PaymentRepository)
	if !ok {
		return nil, fmt.Errorf("неверный тип репозитория: ожидается PaymentRepository")
	}

	return paymentRepo, nil
}

// GetInvoiceRepository получает репозиторий инвойсов
func (f *InfrastructureFactory) GetInvoiceRepository() (invoice.InvoiceRepository, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.initialized {
		return nil, fmt.Errorf("фабрика инфраструктуры не инициализирована")
	}

	if f.repositoryFactory == nil {
		return nil, fmt.Errorf("RepositoryFactory не создана")
	}

	// Создаем InvoiceRepository если его нет
	if !f.repositoryFactory.HasRepository("InvoiceRepository") {
		logger.Info("🔄 Создание InvoiceRepository...")
		if _, err := f.repositoryFactory.CreateInvoiceRepository(); err != nil {
			logger.Error("❌ Не удалось создать InvoiceRepository: %v", err)
			return nil, fmt.Errorf("не удалось создать InvoiceRepository: %w", err)
		}
		logger.Info("✅ InvoiceRepository создан")
	}

	repo, err := f.repositoryFactory.GetRepository("InvoiceRepository")
	if err != nil {
		return nil, err
	}

	invoiceRepo, ok := repo.(invoice.InvoiceRepository)
	if !ok {
		return nil, fmt.Errorf("неверный тип репозитория: ожидается InvoiceRepository")
	}

	return invoiceRepo, nil
}

// GetSessionRepository получает репозиторий сессий
func (f *InfrastructureFactory) GetSessionRepository() (session.SessionRepository, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if !f.initialized {
		return nil, fmt.Errorf("фабрика инфраструктуры не инициализирована")
	}

	if f.repositoryFactory == nil {
		return nil, fmt.Errorf("RepositoryFactory не создана")
	}

	repo, err := f.repositoryFactory.GetRepository("SessionRepository")
	if err != nil {
		return nil, err
	}

	sessionRepo, ok := repo.(session.SessionRepository)
	if !ok {
		return nil, fmt.Errorf("неверный тип репозитория: ожидается SessionRepository")
	}

	return sessionRepo, nil
}

// GetActivityRepository получает репозиторий активности
func (f *InfrastructureFactory) GetActivityRepository() (activity.ActivityRepository, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if !f.initialized {
		return nil, fmt.Errorf("фабрика инфраструктуры не инициализирована")
	}

	if f.repositoryFactory == nil {
		return nil, fmt.Errorf("RepositoryFactory не создана")
	}

	repo, err := f.repositoryFactory.GetRepository("ActivityRepository")
	if err != nil {
		return nil, err
	}

	activityRepo, ok := repo.(activity.ActivityRepository)
	if !ok {
		return nil, fmt.Errorf("неверный тип репозитория: ожидается ActivityRepository")
	}

	return activityRepo, nil
}

// GetAPIKeyRepository получает репозиторий API ключей
func (f *InfrastructureFactory) GetAPIKeyRepository() (api_key.APIKeyRepository, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if !f.initialized {
		return nil, fmt.Errorf("фабрика инфраструктуры не инициализирована")
	}

	if f.repositoryFactory == nil {
		return nil, fmt.Errorf("RepositoryFactory не создана")
	}

	repo, err := f.repositoryFactory.GetRepository("APIKeyRepository")
	if err != nil {
		return nil, err
	}

	apiKeyRepo, ok := repo.(api_key.APIKeyRepository)
	if !ok {
		return nil, fmt.Errorf("неверный тип репозитория: ожидается APIKeyRepository")
	}

	return apiKeyRepo, nil
}

// GetEventBus возвращает EventBus
func (f *InfrastructureFactory) GetEventBus() (*events.EventBus, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if !f.initialized {
		return nil, fmt.Errorf("фабрика инфраструктуры не инициализирована")
	}

	if f.eventBus == nil {
		return nil, fmt.Errorf("EventBus не создан")
	}

	return f.eventBus, nil
}
