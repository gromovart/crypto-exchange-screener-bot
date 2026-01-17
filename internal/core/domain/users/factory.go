// internal/core/domain/users/factory.go
package users

import (
	"crypto-exchange-screener-bot/internal/infrastructure/cache/redis"
	database "crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/database"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"sync"
	"time"
)

// UserServiceFactory фабрика для создания UserService
type UserServiceFactory struct {
	config       Config
	database     *database.DatabaseService // ИЗМЕНЕНО
	redisService *redis.RedisService
	notifier     NotificationService
	mu           sync.RWMutex
	initialized  bool
}

// UserServiceDependencies зависимости для фабрики UserService
type UserServiceDependencies struct {
	Config       Config
	Database     *database.DatabaseService // ИЗМЕНЕНО
	RedisService *redis.RedisService
	Notifier     NotificationService
}

// NewUserServiceFactory создает фабрику UserService
func NewUserServiceFactory(deps UserServiceDependencies) (*UserServiceFactory, error) {
	logger.Info("👤 Создание фабрики UserService...")

	if deps.Database == nil {
		return nil, fmt.Errorf("Database не может быть nil")
	}
	if deps.RedisService == nil {
		return nil, fmt.Errorf("RedisService не может быть nil")
	}

	factory := &UserServiceFactory{
		config:       deps.Config,
		database:     deps.Database,
		redisService: deps.RedisService,
		notifier:     deps.Notifier,
		initialized:  true,
	}

	logger.Info("✅ Фабрика UserService создана")
	return factory, nil
}

// CreateUserService создает экземпляр UserService
func (f *UserServiceFactory) CreateUserService() (*Service, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.initialized {
		return nil, fmt.Errorf("фабрика UserService не инициализирована")
	}

	logger.Info("🔧 Создание UserService через фабрику...")

	db := f.database.GetDB()
	redisCache := f.redisService.GetCache()

	if db == nil {
		return nil, fmt.Errorf("подключение к БД не установлено")
	}
	if redisCache == nil {
		return nil, fmt.Errorf("подключение к Redis не установлено")
	}

	service, err := NewService(db, redisCache, f.notifier, f.config)
	if err != nil {
		return nil, fmt.Errorf("не удалось создать UserService: %w", err)
	}

	logger.Info("✅ UserService успешно создан через фабрику")
	return service, nil
}

// CreateUserServiceWithDefaults создает UserService с настройками по умолчанию
func (f *UserServiceFactory) CreateUserServiceWithDefaults() (*Service, error) {
	f.mu.Lock()
	f.config = Config{
		DefaultMinGrowthThreshold: 2.0,
		DefaultMaxSignalsPerDay:   50,
		SessionTTL:                24 * time.Hour,
		MaxSessionsPerUser:        5,
	}
	f.mu.Unlock()

	return f.CreateUserService()
}

// UpdateConfig обновляет конфигурацию фабрики
func (f *UserServiceFactory) UpdateConfig(newConfig Config) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.config = newConfig
}

// UpdateNotifier обновляет сервис уведомлений
func (f *UserServiceFactory) UpdateNotifier(notifier NotificationService) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.notifier = notifier
}

// GetConfig возвращает текущую конфигурацию
func (f *UserServiceFactory) GetConfig() Config {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.config
}

// Validate проверяет готовность фабрики к созданию сервиса
func (f *UserServiceFactory) Validate() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if !f.initialized {
		logger.Warn("⚠️ Фабрика UserService не инициализирована")
		return false
	}

	if f.database == nil || f.database.GetDB() == nil {
		logger.Warn("⚠️ DatabaseService не доступен для фабрики UserService")
		return false
	}

	if f.redisService == nil || f.redisService.GetCache() == nil {
		logger.Warn("⚠️ RedisService не доступен для фабрики UserService")
		return false
	}

	logger.Info("✅ Фабрика UserService валидирована")
	return true
}

// GetDependenciesInfo возвращает информацию о зависимостях
func (f *UserServiceFactory) GetDependenciesInfo() map[string]interface{} {
	f.mu.RLock()
	defer f.mu.RUnlock()

	info := map[string]interface{}{
		"initialized":      f.initialized,
		"database_ready":   f.database != nil && f.database.GetDB() != nil,
		"redis_ready":      f.redisService != nil && f.redisService.GetCache() != nil,
		"notifier_ready":   f.notifier != nil,
		"config_available": f.config != (Config{}),
	}

	if f.database != nil {
		info["database_state"] = f.database.State()
	}
	if f.redisService != nil {
		info["redis_state"] = f.redisService.State()
	}

	return info
}

// Reset сбрасывает фабрику
func (f *UserServiceFactory) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.database = nil
	f.redisService = nil
	f.notifier = nil
	f.initialized = false
	f.config = Config{}

	logger.Info("🔄 Фабрика UserService сброшена")
}

// SetDatabase устанавливает сервис базы данных
func (f *UserServiceFactory) SetDatabase(database *database.DatabaseService) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.database = database
}

// SetRedisService устанавливает сервис Redis
func (f *UserServiceFactory) SetRedisService(redisService *redis.RedisService) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.redisService = redisService
}

// SetNotifier устанавливает сервис уведомлений
func (f *UserServiceFactory) SetNotifier(notifier NotificationService) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.notifier = notifier
}

// IsReady проверяет готовность фабрики
func (f *UserServiceFactory) IsReady() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return f.initialized &&
		f.database != nil &&
		f.database.GetDB() != nil &&
		f.redisService != nil &&
		f.redisService.GetCache() != nil
}
