// internal/core/domain/subscription/factory.go
package subscription

import (
	"crypto-exchange-screener-bot/internal/infrastructure/cache/redis"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/repository/plan"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// SubscriptionServiceFactory фабрика для создания SubscriptionService
type SubscriptionServiceFactory struct {
	planRepo          plan.PlanRepository
	cache             *redis.Cache
	analytics         AnalyticsService
	paymentRepo       PaymentRepository // ⭐ Добавляем репозиторий платежей
	config            Config
	database          *sqlx.DB
	redis             interface{}   // Для совместимости
	validatorInterval time.Duration // Интервал для валидатора
}

// SubscriptionServiceDependencies зависимости для фабрики SubscriptionService
type Dependencies struct {
	PlanRepo          plan.PlanRepository
	Cache             *redis.Cache
	Analytics         AnalyticsService
	PaymentRepo       PaymentRepository // ⭐ Добавляем
	Config            Config
	ValidatorInterval time.Duration // Интервал для валидатора (по умолчанию 10 мин)
}

// NewSubscriptionServiceFactory создает фабрику SubscriptionService
func NewSubscriptionServiceFactory(deps Dependencies) (*SubscriptionServiceFactory, error) {
	logger.Info("💎 Создание фабрики SubscriptionService...")

	if deps.PlanRepo == nil {
		return nil, fmt.Errorf("PlanRepo не может быть nil")
	}
	if deps.Cache == nil {
		return nil, fmt.Errorf("Cache не может быть nil")
	}

	// Устанавливаем интервал по умолчанию
	validatorInterval := deps.ValidatorInterval
	if validatorInterval == 0 {
		validatorInterval = 10 * time.Minute
	}

	factory := &SubscriptionServiceFactory{
		planRepo:          deps.PlanRepo,
		cache:             deps.Cache,
		analytics:         deps.Analytics,
		paymentRepo:       deps.PaymentRepo, // ⭐ Сохраняем
		config:            deps.Config,
		validatorInterval: validatorInterval,
	}

	logger.Info("✅ Фабрика SubscriptionService создана")
	return factory, nil
}

// CreateSubscriptionService создает экземпляр SubscriptionService
func (f *SubscriptionServiceFactory) CreateSubscriptionService(db *sqlx.DB) (*Service, error) {
	logger.Info("🔧 Создание SubscriptionService через фабрику...")

	if db == nil {
		return nil, fmt.Errorf("подключение к БД не установлено")
	}

	// Создаем сервис
	service, err := NewService(
		db,
		f.planRepo,
		f.cache,
		f.analytics,
		f.config,
	)
	if err != nil {
		return nil, fmt.Errorf("не удалось создать SubscriptionService: %w", err)
	}

	// ⭐ Если есть репозиторий платежей, запускаем валидатор
	if f.paymentRepo != nil {
		logger.Info("🔄 Запуск валидатора подписок с интервалом %v", f.validatorInterval)
		service.StartSubscriptionValidator(f.validatorInterval, f.paymentRepo)
	} else {
		logger.Warn("⚠️ PaymentRepository не предоставлен, валидатор подписок не запущен")
	}

	logger.Info("✅ SubscriptionService успешно создан через фабрику")
	return service, nil
}

// CreateSubscriptionServiceWithDefaults создает SubscriptionService с настройками по умолчанию
func (f *SubscriptionServiceFactory) CreateSubscriptionServiceWithDefaults(db *sqlx.DB) (*Service, error) {
	f.config = Config{
		DefaultPlan:     "free",
		TrialPeriodDays: 1,
		GracePeriodDays: 3,
		AutoRenew:       true,
	}

	return f.CreateSubscriptionService(db)
}

// SetDatabase устанавливает подключение к БД
func (f *SubscriptionServiceFactory) SetDatabase(db *sqlx.DB) {
	f.database = db
	logger.Debug("✅ Установлено подключение к БД для фабрики SubscriptionService")
}

// SetRedisService устанавливает Redis сервис (для совместимости)
func (f *SubscriptionServiceFactory) SetRedisService(redis interface{}) {
	f.redis = redis
	logger.Debug("✅ Установлен RedisService для фабрики SubscriptionService")
}

// SetAnalytics устанавливает сервис аналитики
func (f *SubscriptionServiceFactory) SetAnalytics(analytics AnalyticsService) {
	f.analytics = analytics
	logger.Debug("✅ Установлен AnalyticsService для фабрики SubscriptionService")
}

// SetPaymentRepository устанавливает репозиторий платежей
func (f *SubscriptionServiceFactory) SetPaymentRepository(paymentRepo PaymentRepository) {
	f.paymentRepo = paymentRepo
	logger.Debug("✅ Установлен PaymentRepository для фабрики SubscriptionService")
}

// SetValidatorInterval устанавливает интервал валидатора
func (f *SubscriptionServiceFactory) SetValidatorInterval(interval time.Duration) {
	f.validatorInterval = interval
	logger.Debug("✅ Установлен интервал валидатора: %v", interval)
}

// UpdateConfig обновляет конфигурацию
func (f *SubscriptionServiceFactory) UpdateConfig(config Config) {
	f.config = config
	logger.Debug("✅ Конфигурация фабрики SubscriptionService обновлена")
}

// Validate проверяет готовность фабрики
func (f *SubscriptionServiceFactory) Validate() bool {
	if f.planRepo == nil {
		logger.Warn("⚠️ PlanRepo не установлен в фабрике SubscriptionService")
		return false
	}
	if f.cache == nil {
		logger.Warn("⚠️ Cache не установлен в фабрике SubscriptionService")
		return false
	}
	return true
}

// GetDependenciesInfo возвращает информацию о зависимостях
func (f *SubscriptionServiceFactory) GetDependenciesInfo() map[string]interface{} {
	info := map[string]interface{}{
		"plan_repo_set":      f.planRepo != nil,
		"cache_set":          f.cache != nil,
		"analytics_set":      f.analytics != nil,
		"payment_repo_set":   f.paymentRepo != nil,
		"database_set":       f.database != nil,
		"redis_set":          f.redis != nil,
		"validator_interval": f.validatorInterval.String(),
		"config":             f.config,
	}
	return info
}

// GetPlanRepo возвращает репозиторий планов
func (f *SubscriptionServiceFactory) GetPlanRepo() plan.PlanRepository {
	return f.planRepo
}

// GetCache возвращает кэш
func (f *SubscriptionServiceFactory) GetCache() *redis.Cache {
	return f.cache
}

// GetConfig возвращает конфигурацию
func (f *SubscriptionServiceFactory) GetConfig() Config {
	return f.config
}

// GetPaymentRepo возвращает репозиторий платежей
func (f *SubscriptionServiceFactory) GetPaymentRepo() PaymentRepository {
	return f.paymentRepo
}
