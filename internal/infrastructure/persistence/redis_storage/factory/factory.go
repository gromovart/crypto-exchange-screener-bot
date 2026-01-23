// internal/infrastructure/persistence/redis_storage/factory/factory.go
package redis_storage_factory

import (
	redis_service "crypto-exchange-screener-bot/internal/infrastructure/cache/redis"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage/price_storage"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"sync"
	"time"
)

// PriceStorage алиас для интерфейса
type PriceStorage = redis_storage.PriceStorageInterface

// StorageFactoryConfig конфигурация фабрики хранилищ
type StorageFactoryConfig struct {
	DefaultStorageConfig *redis_storage.StorageConfig
	EnableCleanupRoutine bool
	CleanupInterval      time.Duration
	MaxCustomStorages    int
}

// StorageDependencies зависимости для фабрики хранилищ
type StorageDependencies struct {
	Config      *StorageFactoryConfig
	RedisClient interface{} // Redis клиент или сервис
}

// StorageFactory фабрика для создания хранилищ цен
type StorageFactory struct {
	mu                    sync.RWMutex
	config                *StorageFactoryConfig
	redisClient           interface{} // Redis клиент/сервис
	defaultStorage        PriceStorage
	customStorages        map[string]PriceStorage
	cleanupRoutineRunning bool
	stopCleanupChan       chan struct{}
}

// NewStorageFactory создает новую фабрику хранилищ
func NewStorageFactory(deps StorageDependencies) (*StorageFactory, error) {
	if deps.Config == nil {
		return nil, fmt.Errorf("конфигурация не может быть nil")
	}

	return &StorageFactory{
		config:          deps.Config,
		redisClient:     deps.RedisClient,
		customStorages:  make(map[string]PriceStorage),
		stopCleanupChan: make(chan struct{}),
	}, nil
}

// Initialize инициализирует фабрику хранилищ
func (sf *StorageFactory) Initialize() error {
	logger.Info("🏭 Инициализация Redis StorageFactory...")
	return nil
}

// SetRedisClient устанавливает Redis клиент
func (sf *StorageFactory) SetRedisClient(client interface{}) {
	sf.mu.Lock()
	defer sf.mu.Unlock()
	sf.redisClient = client
}

// CreateDefaultStorage создает хранилище по умолчанию
func (sf *StorageFactory) CreateDefaultStorage() (PriceStorage, error) {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	if sf.defaultStorage == nil {
		if sf.redisClient == nil {
			return nil, fmt.Errorf("Redis клиент не установлен")
		}

		// Проверяем тип redisClient
		var redisService *redis_service.RedisService
		switch client := sf.redisClient.(type) {
		case *redis_service.RedisService:
			redisService = client
		default:
			return nil, fmt.Errorf("неподдерживаемый тип Redis клиента: %T", client)
		}

		// Создаем RedisStorage
		storageConfig := sf.config.DefaultStorageConfig
		if storageConfig == nil {
			storageConfig = &redis_storage.StorageConfig{
				MaxHistoryPerSymbol: 10000,
				MaxSymbols:          1000,
				CleanupInterval:     5 * time.Minute,
				RetentionPeriod:     48 * time.Hour,
			}
		}

		// Создаем структуру PriceStorage (используем упрощенный конструктор)
		priceStorage := price_storage.NewPriceStorageSimple(redisService, storageConfig)

		// Инициализируем хранилище
		if err := priceStorage.Initialize(); err != nil {
			return nil, fmt.Errorf("ошибка инициализации Redis хранилища: %w", err)
		}

		// Присваиваем интерфейсу
		sf.defaultStorage = priceStorage

		logger.Info("✅ Создано Redis хранилище по умолчанию")
	}

	return sf.defaultStorage, nil
}

// Start запускает фабрику
func (sf *StorageFactory) Start() error {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	// Создаем хранилище по умолчанию если еще не создано
	if sf.defaultStorage == nil {
		// Временная переменная для хранения ошибки
		var err error

		// Создаем хранилище с временной разблокировкой
		sf.mu.Unlock()
		sf.defaultStorage, err = sf.createDefaultStorageUnsafe()
		sf.mu.Lock()

		if err != nil {
			return fmt.Errorf("не удалось создать хранилище: %w", err)
		}
	}

	// Запускаем очистку если настроено
	if sf.config.EnableCleanupRoutine {
		sf.startCleanupRoutine()
	}

	logger.Info("🚀 Redis StorageFactory запущена")
	return nil
}

// createDefaultStorageUnsafe создает хранилище без блокировки (для внутреннего использования)
func (sf *StorageFactory) createDefaultStorageUnsafe() (PriceStorage, error) {
	if sf.redisClient == nil {
		return nil, fmt.Errorf("Redis клиент не установлен")
	}

	// Проверяем тип redisClient
	var redisService *redis_service.RedisService
	switch client := sf.redisClient.(type) {
	case *redis_service.RedisService:
		redisService = client
	default:
		return nil, fmt.Errorf("неподдерживаемый тип Redis клиента: %T", client)
	}

	// Создаем RedisStorage
	storageConfig := sf.config.DefaultStorageConfig
	if storageConfig == nil {
		storageConfig = &redis_storage.StorageConfig{
			MaxHistoryPerSymbol: 10000,
			MaxSymbols:          1000,
			CleanupInterval:     5 * time.Minute,
			RetentionPeriod:     48 * time.Hour,
		}
	}

	// Создаем структуру PriceStorage (используем упрощенный конструктор)
	priceStorage := price_storage.NewPriceStorageSimple(redisService, storageConfig)

	// Инициализируем хранилище
	if err := priceStorage.Initialize(); err != nil {
		return nil, fmt.Errorf("ошибка инициализации Redis хранилища: %w", err)
	}

	logger.Info("✅ Создано Redis хранилище по умолчанию")
	return priceStorage, nil
}

// Stop останавливает фабрику
func (sf *StorageFactory) Stop() error {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	// Останавливаем очистку
	if sf.cleanupRoutineRunning {
		close(sf.stopCleanupChan)
		sf.cleanupRoutineRunning = false
	}

	logger.Info("🛑 Redis StorageFactory остановлена")
	return nil
}

// startCleanupRoutine запускает рутину очистки
func (sf *StorageFactory) startCleanupRoutine() {
	if sf.cleanupRoutineRunning {
		return
	}

	sf.cleanupRoutineRunning = true
	go sf.cleanupRoutine()

	logger.Info("🧹 Запущена очистка Redis хранилища")
}

// cleanupRoutine рутина очистки
func (sf *StorageFactory) cleanupRoutine() {
	ticker := time.NewTicker(sf.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sf.cleanupOldData()
		case <-sf.stopCleanupChan:
			return
		}
	}
}

// cleanupOldData очищает старые данные
func (sf *StorageFactory) cleanupOldData() {
	sf.mu.RLock()
	defer sf.mu.RUnlock()

	// Очищаем хранилище по умолчанию
	if sf.defaultStorage != nil {
		if removed, err := sf.defaultStorage.CleanOldData(24 * time.Hour); err == nil && removed > 0 {
			logger.Debug("🧹 Очищено %d старых записей из хранилища по умолчанию", removed)
		}
	}

	// Очищаем кастомные хранилища
	for name, storage := range sf.customStorages {
		if removed, err := storage.CleanOldData(24 * time.Hour); err == nil && removed > 0 {
			logger.Debug("🧹 Очищено %d старых записей из хранилища %s", removed, name)
		}
	}
}

// GetAllStorages возвращает все хранилища
func (sf *StorageFactory) GetAllStorages() map[string]PriceStorage {
	sf.mu.RLock()
	defer sf.mu.RUnlock()

	storages := make(map[string]PriceStorage)

	if sf.defaultStorage != nil {
		storages["default"] = sf.defaultStorage
	}

	for name, storage := range sf.customStorages {
		storages[name] = storage
	}

	return storages
}

// Validate проверяет валидность фабрики
func (sf *StorageFactory) Validate() bool {
	sf.mu.RLock()
	defer sf.mu.RUnlock()
	return sf.defaultStorage != nil
}

// IsRunning проверяет запущена ли фабрика
func (sf *StorageFactory) IsRunning() bool {
	sf.mu.RLock()
	defer sf.mu.RUnlock()
	return sf.cleanupRoutineRunning
}

// Reset сбрасывает фабрику
func (sf *StorageFactory) Reset() {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	// Останавливаем если запущена
	if sf.cleanupRoutineRunning {
		sf.Stop()
	}

	sf.defaultStorage = nil
	sf.customStorages = make(map[string]PriceStorage)
	sf.redisClient = nil
}
