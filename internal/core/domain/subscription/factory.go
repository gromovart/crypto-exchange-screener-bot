// internal/core/domain/subscription/factory.go
package subscription

import (
	"crypto-exchange-screener-bot/internal/infrastructure/cache/redis"
	database "crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/database"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"sync"
)

// SubscriptionServiceFactory фабрика для создания SubscriptionService
type SubscriptionServiceFactory struct {
	config       Config
	database     *database.DatabaseService
	redisService *redis.RedisService
	notifier     NotificationService
	analytics    AnalyticsService
	mu           sync.RWMutex
	initialized  bool
}

// SubscriptionServiceDependencies зависимости для фабрики SubscriptionService
type SubscriptionServiceDependencies struct {
	Config       Config
	Database     *database.DatabaseService
	RedisService *redis.RedisService
	Notifier     NotificationService
	Analytics    AnalyticsService
}

// NewSubscriptionServiceFactory создает фабрику SubscriptionService
func NewSubscriptionServiceFactory(deps SubscriptionServiceDependencies) (*SubscriptionServiceFactory, error) {
	logger.Info("💎 Создание фабрики SubscriptionService...")

	if deps.Database == nil {
		return nil, fmt.Errorf("Database не может быть nil")
	}
	if deps.RedisService == nil {
		return nil, fmt.Errorf("RedisService не может быть nil")
	}
	if deps.Notifier == nil {
		logger.Warn("⚠️ Notifier не указан для SubscriptionService")
	}
	if deps.Analytics == nil {
		logger.Warn("⚠️ Analytics не указан для SubscriptionService")
	}

	factory := &SubscriptionServiceFactory{
		config:       deps.Config,
		database:     deps.Database,
		redisService: deps.RedisService,
		notifier:     deps.Notifier,
		analytics:    deps.Analytics,
		initialized:  true,
	}

	logger.Info("✅ Фабрика SubscriptionService создана")
	return factory, nil
}

// CreateSubscriptionService создает экземпляр SubscriptionService
func (f *SubscriptionServiceFactory) CreateSubscriptionService() (*Service, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.initialized {
		return nil, fmt.Errorf("фабрика SubscriptionService не инициализирована")
	}

	logger.Info("🔧 Создание SubscriptionService через фабрику...")

	db := f.database.GetDB()
	redisCache := f.redisService.GetCache()

	if db == nil {
		return nil, fmt.Errorf("подключение к БД не установлено")
	}
	if redisCache == nil {
		return nil, fmt.Errorf("подключение к Redis не установлено")
	}

	service, err := NewService(db, redisCache, f.notifier, f.analytics, f.config)
	if err != nil {
		return nil, fmt.Errorf("не удалось создать SubscriptionService: %w", err)
	}

	logger.Info("✅ SubscriptionService успешно создан через фабрику")
	return service, nil
}

// CreateSubscriptionServiceWithDefaults создает SubscriptionService с настройками по умолчанию
func (f *SubscriptionServiceFactory) CreateSubscriptionServiceWithDefaults() (*Service, error) {
	f.mu.Lock()
	f.config = Config{
		StripeSecretKey:  "",
		StripeWebhookKey: "",
		DefaultPlan:      "free",
		TrialPeriodDays:  7,
		GracePeriodDays:  3,
		AutoRenew:        true,
	}
	f.mu.Unlock()

	return f.CreateSubscriptionService()
}

// UpdateConfig обновляет конфигурацию фабрики
func (f *SubscriptionServiceFactory) UpdateConfig(newConfig Config) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.config = newConfig
}

// UpdateNotifier обновляет сервис уведомлений
func (f *SubscriptionServiceFactory) UpdateNotifier(notifier NotificationService) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.notifier = notifier
}

// UpdateAnalytics обновляет сервис аналитики
func (f *SubscriptionServiceFactory) UpdateAnalytics(analytics AnalyticsService) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.analytics = analytics
}

// GetConfig возвращает текущую конфигурацию
func (f *SubscriptionServiceFactory) GetConfig() Config {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.config
}

// Validate проверяет готовность фабрики к созданию сервиса
func (f *SubscriptionServiceFactory) Validate() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if !f.initialized {
		logger.Warn("⚠️ Фабрика SubscriptionService не инициализирована")
		return false
	}

	if f.database == nil || f.database.GetDB() == nil {
		logger.Warn("⚠️ DatabaseService не доступен для фабрики SubscriptionService")
		return false
	}

	if f.redisService == nil || f.redisService.GetCache() == nil {
		logger.Warn("⚠️ RedisService не доступен для фабрики SubscriptionService")
		return false
	}

	logger.Info("✅ Фабрика SubscriptionService валидирована")
	return true
}

// GetDependenciesInfo возвращает информацию о зависимостях
func (f *SubscriptionServiceFactory) GetDependenciesInfo() map[string]interface{} {
	f.mu.RLock()
	defer f.mu.RUnlock()

	info := map[string]interface{}{
		"initialized":      f.initialized,
		"database_ready":   f.database != nil && f.database.GetDB() != nil,
		"redis_ready":      f.redisService != nil && f.redisService.GetCache() != nil,
		"notifier_ready":   f.notifier != nil,
		"analytics_ready":  f.analytics != nil,
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
func (f *SubscriptionServiceFactory) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.database = nil
	f.redisService = nil
	f.notifier = nil
	f.analytics = nil
	f.initialized = false
	f.config = Config{}

	logger.Info("🔄 Фабрика SubscriptionService сброшена")
}

// SetDatabase устанавливает сервис базы данных
func (f *SubscriptionServiceFactory) SetDatabase(database *database.DatabaseService) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.database = database
}

// SetRedisService устанавливает сервис Redis
func (f *SubscriptionServiceFactory) SetRedisService(redisService *redis.RedisService) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.redisService = redisService
}

// SetNotifier устанавливает сервис уведомлений
func (f *SubscriptionServiceFactory) SetNotifier(notifier NotificationService) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.notifier = notifier
}

// SetAnalytics устанавливает сервис аналитики
func (f *SubscriptionServiceFactory) SetAnalytics(analytics AnalyticsService) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.analytics = analytics
}

// IsReady проверяет готовность фабрики
func (f *SubscriptionServiceFactory) IsReady() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return f.initialized &&
		f.database != nil &&
		f.database.GetDB() != nil &&
		f.redisService != nil &&
		f.redisService.GetCache() != nil
}

// CreatePlanManagementService создает сервис управления планами
func (f *SubscriptionServiceFactory) CreatePlanManagementService() (*Service, error) {
	// Это специализированный метод для создания сервиса с фокусом на управление планами
	logger.Info("📋 Создание сервиса управления планами...")

	service, err := f.CreateSubscriptionService()
	if err != nil {
		return nil, err
	}

	// Дополнительная инициализация для управления планами
	// (в будущем можно добавить специфическую логику)

	logger.Info("✅ Сервис управления планами создан")
	return service, nil
}

// CreateBillingService создает сервис биллинга
func (f *SubscriptionServiceFactory) CreateBillingService() (*Service, error) {
	// Это специализированный метод для создания сервиса с фокусом на биллинг
	logger.Info("💰 Создание сервиса биллинга...")

	// Клонируем конфиг с фокусом на биллинг
	billingConfig := f.config
	billingConfig.AutoRenew = true // Для биллинга всегда авто-продление

	// Временно заменяем конфиг
	f.mu.Lock()
	originalConfig := f.config
	f.config = billingConfig
	f.mu.Unlock()

	service, err := f.CreateSubscriptionService()

	// Восстанавливаем конфиг
	f.mu.Lock()
	f.config = originalConfig
	f.mu.Unlock()

	if err != nil {
		return nil, err
	}

	logger.Info("✅ Сервис биллинга создан")
	return service, nil
}
