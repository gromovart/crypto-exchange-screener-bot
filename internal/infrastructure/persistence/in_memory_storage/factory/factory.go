// internal/infrastructure/persistence/in_memory_storage/factory/factory.go
package storage_factory

import (
	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/in_memory_storage"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"sync"
	"time"
)

// StorageFactory фабрика для создания in-memory хранилищ
type StorageFactory struct {
	defaultStorage storage.PriceStorage
	customStorages map[string]storage.PriceStorage
	config         *StorageFactoryConfig
	mu             sync.RWMutex
	initialized    bool
	cleanupRunning bool
	stopCleanup    chan struct{}
	cleanupWg      sync.WaitGroup
}

// StorageFactoryConfig конфигурация фабрики хранилищ
type StorageFactoryConfig struct {
	// Конфигурация хранилища по умолчанию
	DefaultStorageConfig *storage.StorageConfig

	// Настройки фабрики
	EnableCleanupRoutine bool
	CleanupInterval      time.Duration
	MaxCustomStorages    int
}

// StorageDependencies зависимости для фабрики хранилищ
type StorageDependencies struct {
	Config *StorageFactoryConfig
}

// NewStorageFactory создает новую фабрику хранилищ
func NewStorageFactory(deps StorageDependencies) (*StorageFactory, error) {
	logger.Info("🏗️  Создание фабрики in-memory хранилищ...")

	// Используем конфигурацию по умолчанию если не предоставлена
	config := deps.Config
	if config == nil {
		config = &StorageFactoryConfig{
			DefaultStorageConfig: &storage.StorageConfig{
				MaxHistoryPerSymbol: 10000,
				MaxSymbols:          1000,
				CleanupInterval:     5 * time.Minute,
				RetentionPeriod:     24 * time.Hour,
				EnableCompression:   false,
				EnablePersistence:   false,
				PersistencePath:     "",
			},
			EnableCleanupRoutine: true,
			CleanupInterval:      1 * time.Minute,
			MaxCustomStorages:    10,
		}
	}

	// Проверяем конфигурацию по умолчанию
	if config.DefaultStorageConfig == nil {
		return nil, fmt.Errorf("конфигурация хранилища по умолчанию не может быть nil")
	}

	factory := &StorageFactory{
		customStorages: make(map[string]storage.PriceStorage),
		config:         config,
		initialized:    false,
		cleanupRunning: false,
		stopCleanup:    make(chan struct{}),
	}

	logger.Info("✅ Фабрика in-memory хранилищ создана")
	return factory, nil
}

// Initialize инициализирует фабрику и создает хранилище по умолчанию
func (sf *StorageFactory) Initialize() error {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	if sf.initialized {
		return fmt.Errorf("фабрика хранилищ уже инициализирована")
	}

	logger.Info("🔧 Инициализация фабрики in-memory хранилищ...")

	// Создаем хранилище по умолчанию
	sf.defaultStorage = storage.NewInMemoryPriceStorage(sf.config.DefaultStorageConfig)

	sf.initialized = true

	logger.Info("✅ Фабрика in-memory хранилищ инициализирована")
	logger.Info("   • Хранилище по умолчанию создано")
	logger.Info("   • Макс. символов: %d", sf.config.DefaultStorageConfig.MaxSymbols)
	logger.Info("   • Макс. история: %d на символ", sf.config.DefaultStorageConfig.MaxHistoryPerSymbol)
	logger.Info("   • Очистка включена: %v", sf.config.EnableCleanupRoutine)

	return nil
}

// Start запускает фоновые задачи фабрики
func (sf *StorageFactory) Start() error {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	if !sf.initialized {
		return fmt.Errorf("фабрика хранилищ не инициализирована")
	}

	if sf.cleanupRunning {
		return fmt.Errorf("фабрика хранилищ уже запущена")
	}

	logger.Info("🚀 Запуск фоновых задач фабрики хранилищ...")

	// Запускаем рутину очистки если включено
	if sf.config.EnableCleanupRoutine {
		sf.cleanupRunning = true
		sf.cleanupWg.Add(1)
		go sf.startCleanupRoutine()
		logger.Info("   • Фоновая очистка запущена")
	}

	logger.Info("✅ Фоновые задачи фабрики хранилищ запущены")
	return nil
}

// Stop останавливает фоновые задачи фабрики
func (sf *StorageFactory) Stop() error {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	if !sf.cleanupRunning {
		return nil
	}

	logger.Info("🛑 Остановка фоновых задач фабрики хранилищ...")

	// Останавливаем рутину очистки
	if sf.cleanupRunning && sf.config.EnableCleanupRoutine {
		close(sf.stopCleanup)
		sf.cleanupWg.Wait()
		sf.cleanupRunning = false
		sf.stopCleanup = make(chan struct{}) // Создаем новый канал для возможного перезапуска
		logger.Info("   • Фоновая очистка остановлена")
	}

	logger.Info("✅ Фоновые задачи фабрики хранилищ остановлены")
	return nil
}

// CreateDefaultStorage создает или возвращает хранилище по умолчанию
func (sf *StorageFactory) CreateDefaultStorage() (storage.PriceStorage, error) {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	if !sf.initialized {
		return nil, fmt.Errorf("фабрика хранилищ не инициализирована")
	}

	if sf.defaultStorage == nil {
		sf.defaultStorage = storage.NewInMemoryPriceStorage(sf.config.DefaultStorageConfig)
		logger.Info("✅ Хранилище по умолчанию создано")
	}

	return sf.defaultStorage, nil
}

// CreateCustomStorage создает кастомное хранилище с указанным ID
func (sf *StorageFactory) CreateCustomStorage(storageID string, config *storage.StorageConfig) (storage.PriceStorage, error) {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	if !sf.initialized {
		return nil, fmt.Errorf("фабрика хранилищ не инициализирована")
	}

	// Проверяем лимит кастомных хранилищ
	if len(sf.customStorages) >= sf.config.MaxCustomStorages {
		return nil, fmt.Errorf("достигнут лимит кастомных хранилищ: %d", sf.config.MaxCustomStorages)
	}

	// Проверяем уникальность ID
	if _, exists := sf.customStorages[storageID]; exists {
		return nil, fmt.Errorf("хранилище с ID '%s' уже существует", storageID)
	}

	// Используем конфигурацию по умолчанию если не предоставлена
	if config == nil {
		config = &storage.StorageConfig{
			MaxHistoryPerSymbol: sf.config.DefaultStorageConfig.MaxHistoryPerSymbol,
			MaxSymbols:          sf.config.DefaultStorageConfig.MaxSymbols,
			CleanupInterval:     sf.config.DefaultStorageConfig.CleanupInterval,
			RetentionPeriod:     sf.config.DefaultStorageConfig.RetentionPeriod,
			EnableCompression:   sf.config.DefaultStorageConfig.EnableCompression,
			EnablePersistence:   sf.config.DefaultStorageConfig.EnablePersistence,
			PersistencePath:     sf.config.DefaultStorageConfig.PersistencePath,
		}
	}

	// Создаем хранилище
	customStorage := storage.NewInMemoryPriceStorage(config)
	sf.customStorages[storageID] = customStorage

	logger.Info("✅ Кастомное хранилище создано: %s", storageID)
	logger.Info("   • ID: %s", storageID)
	logger.Info("   • Макс. символов: %d", config.MaxSymbols)
	logger.Info("   • Макс. история: %d на символ", config.MaxHistoryPerSymbol)

	return customStorage, nil
}

// GetStorage возвращает хранилище по ID
func (sf *StorageFactory) GetStorage(storageID string) (storage.PriceStorage, bool) {
	sf.mu.RLock()
	defer sf.mu.RUnlock()

	if storageID == "default" || storageID == "" {
		return sf.defaultStorage, sf.defaultStorage != nil
	}

	storage, exists := sf.customStorages[storageID]
	return storage, exists
}

// RemoveStorage удаляет кастомное хранилище
func (sf *StorageFactory) RemoveStorage(storageID string) error {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	if !sf.initialized {
		return fmt.Errorf("фабрика хранилищ не инициализирована")
	}

	if storageID == "default" || storageID == "" {
		return fmt.Errorf("нельзя удалить хранилище по умолчанию")
	}

	storage, exists := sf.customStorages[storageID]
	if !exists {
		return fmt.Errorf("хранилище с ID '%s' не найдено", storageID)
	}

	// Очищаем хранилище перед удалением
	if err := storage.Clear(); err != nil {
		logger.Warn("⚠️ Не удалось очистить хранилище %s: %v", storageID, err)
	}

	delete(sf.customStorages, storageID)
	logger.Info("✅ Хранилище удалено: %s", storageID)

	return nil
}

// GetAllStorages возвращает все хранилища
func (sf *StorageFactory) GetAllStorages() map[string]storage.PriceStorage {
	sf.mu.RLock()
	defer sf.mu.RUnlock()

	allStorages := make(map[string]storage.PriceStorage)

	// Добавляем хранилище по умолчанию
	if sf.defaultStorage != nil {
		allStorages["default"] = sf.defaultStorage
	}

	// Добавляем кастомные хранилища
	for id, storage := range sf.customStorages {
		allStorages[id] = storage
	}

	return allStorages
}

// CleanupAllStorages очищает все хранилища
func (sf *StorageFactory) CleanupAllStorages() (map[string]int, error) {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	if !sf.initialized {
		return nil, fmt.Errorf("фабрика хранилищ не инициализирована")
	}

	logger.Info("🧹 Очистка всех хранилищ...")

	results := make(map[string]int)

	// Очищаем хранилище по умолчанию
	if sf.defaultStorage != nil {
		if err := sf.defaultStorage.Clear(); err != nil {
			logger.Warn("⚠️ Не удалось очистить хранилище по умолчанию: %v", err)
			results["default"] = -1
		} else {
			stats := sf.defaultStorage.GetStats()
			results["default"] = int(stats.TotalDataPoints)
			logger.Info("   ✅ Хранилище по умолчанию очищено")
		}
	}

	// Очищаем кастомные хранилища
	for id, storage := range sf.customStorages {
		if err := storage.Clear(); err != nil {
			logger.Warn("⚠️ Не удалось очистить хранилище %s: %v", id, err)
			results[id] = -1
		} else {
			stats := storage.GetStats()
			results[id] = int(stats.TotalDataPoints)
			logger.Info("   ✅ Хранилище %s очищено", id)
		}
	}

	logger.Info("✅ Все хранилища очищены")
	return results, nil
}

// GetHealthStatus возвращает статус здоровья фабрики
func (sf *StorageFactory) GetHealthStatus() map[string]interface{} {
	sf.mu.RLock()
	defer sf.mu.RUnlock()

	status := map[string]interface{}{
		"initialized":           sf.initialized,
		"cleanup_running":       sf.cleanupRunning,
		"default_storage_ready": sf.defaultStorage != nil,
		"custom_storages_count": len(sf.customStorages),
		"max_custom_storages":   sf.config.MaxCustomStorages,
		"cleanup_enabled":       sf.config.EnableCleanupRoutine,
	}

	// Добавляем статистику хранилища по умолчанию
	if sf.defaultStorage != nil {
		stats := sf.defaultStorage.GetStats()
		status["default_storage_stats"] = map[string]interface{}{
			"total_symbols":      stats.TotalSymbols,
			"total_data_points":  stats.TotalDataPoints,
			"memory_usage_bytes": stats.MemoryUsageBytes,
			"oldest_timestamp":   stats.OldestTimestamp,
			"newest_timestamp":   stats.NewestTimestamp,
		}
	}

	// Добавляем список кастомных хранилищ
	customStorageIDs := make([]string, 0, len(sf.customStorages))
	for id := range sf.customStorages {
		customStorageIDs = append(customStorageIDs, id)
	}
	status["custom_storage_ids"] = customStorageIDs

	return status
}

// Validate проверяет готовность фабрики
func (sf *StorageFactory) Validate() bool {
	sf.mu.RLock()
	defer sf.mu.RUnlock()

	if !sf.initialized {
		logger.Warn("⚠️ Фабрика хранилищ не инициализирована")
		return false
	}

	if sf.config == nil {
		logger.Warn("⚠️ Конфигурация фабрики не установлена")
		return false
	}

	return true
}

// IsReady проверяет готовность фабрики
func (sf *StorageFactory) IsReady() bool {
	sf.mu.RLock()
	defer sf.mu.RUnlock()

	return sf.initialized && sf.config != nil
}

// IsRunning проверяет запущены ли фоновые задачи
func (sf *StorageFactory) IsRunning() bool {
	sf.mu.RLock()
	defer sf.mu.RUnlock()
	return sf.cleanupRunning
}

// Reset сбрасывает фабрику
func (sf *StorageFactory) Reset() {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	// Останавливаем фоновые задачи если они запущены
	if sf.cleanupRunning {
		close(sf.stopCleanup)
		sf.cleanupWg.Wait()
		sf.cleanupRunning = false
		sf.stopCleanup = make(chan struct{})
	}

	// Очищаем все хранилища
	if sf.defaultStorage != nil {
		sf.defaultStorage.Clear()
		sf.defaultStorage = nil
	}

	for id, storage := range sf.customStorages {
		storage.Clear()
		delete(sf.customStorages, id)
	}

	sf.customStorages = make(map[string]storage.PriceStorage)
	sf.initialized = false

	logger.Info("🔄 Фабрика хранилищ сброшена")
}

// startCleanupRoutine запускает рутину очистки старых данных
func (sf *StorageFactory) startCleanupRoutine() {
	defer sf.cleanupWg.Done()

	if !sf.config.EnableCleanupRoutine {
		return
	}

	ticker := time.NewTicker(sf.config.CleanupInterval)
	defer ticker.Stop()

	logger.Info("🔄 Запуск рутины очистки хранилищ (интервал: %v)", sf.config.CleanupInterval)

	for {
		select {
		case <-ticker.C:
			sf.cleanupOldData()
		case <-sf.stopCleanup:
			logger.Info("🛑 Рутина очистки хранилищ остановлена")
			return
		}
	}
}

// cleanupOldData очищает старые данные во всех хранилищах
func (sf *StorageFactory) cleanupOldData() {
	sf.mu.RLock()
	defer sf.mu.RUnlock()

	if !sf.initialized {
		return
	}

	logger.Debug("🧹 Автоматическая очистка старых данных...")

	// Очищаем хранилище по умолчанию
	if sf.defaultStorage != nil {
		config := sf.config.DefaultStorageConfig
		if removed, err := sf.defaultStorage.CleanOldData(config.RetentionPeriod); err != nil {
			logger.Warn("⚠️ Не удалось очистить хранилище по умолчанию: %v", err)
		} else if removed > 0 {
			logger.Debug("   ✅ Хранилище по умолчанию: удалено %d старых записей", removed)
		}
	}

	// Очищаем кастомные хранилища
	for id, storage := range sf.customStorages {
		// Для кастомных используем настройки из фабрики
		config := sf.config.DefaultStorageConfig
		if removed, err := storage.CleanOldData(config.RetentionPeriod); err != nil {
			logger.Warn("⚠️ Не удалось очистить хранилище %s: %v", id, err)
		} else if removed > 0 {
			logger.Debug("   ✅ Хранилище %s: удалено %d старых записей", id, removed)
		}
	}
}

// GetConfig возвращает конфигурацию фабрики
func (sf *StorageFactory) GetConfig() *StorageFactoryConfig {
	sf.mu.RLock()
	defer sf.mu.RUnlock()
	return sf.config
}

// UpdateConfig обновляет конфигурацию фабрики
func (sf *StorageFactory) UpdateConfig(newConfig *StorageFactoryConfig) error {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	if !sf.initialized {
		return fmt.Errorf("фабрика хранилищ не инициализирована")
	}

	if newConfig == nil {
		return fmt.Errorf("новая конфигурация не может быть nil")
	}

	sf.config = newConfig
	logger.Info("🔄 Конфигурация фабрики хранилищ обновлена")

	return nil
}
