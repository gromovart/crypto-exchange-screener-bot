// internal/core/package/package.go
package core_factory

import (
	"crypto-exchange-screener-bot/internal/core/domain/payment"
	"crypto-exchange-screener-bot/internal/core/domain/subscription"
	"crypto-exchange-screener-bot/internal/core/domain/users"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/http_client"
	infrastructure_factory "crypto-exchange-screener-bot/internal/infrastructure/package"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"sync"
	"time"
)

// CoreServiceFactory главная фабрика сервисов ядра приложения
type CoreServiceFactory struct {
	config                *Config
	infrastructureFactory *infrastructure_factory.InfrastructureFactory
	userFactory           *users.UserServiceFactory
	subscriptionFactory   *subscription.SubscriptionServiceFactory
	mu                    sync.RWMutex
	initialized           bool
}

// Config конфигурация фабрики ядра
type Config struct {
	UserConfig         users.Config
	SubscriptionConfig subscription.Config
	PaymentsConfig     payment.Config
}

// CoreServiceDependencies зависимости для фабрики ядра
type CoreServiceDependencies struct {
	InfrastructureFactory *infrastructure_factory.InfrastructureFactory
	UserNotifier          users.NotificationService
	Analytics             subscription.AnalyticsService
	Config                *Config
}

// NewCoreServiceFactory создает главную фабрику сервисов ядра
func NewCoreServiceFactory(deps CoreServiceDependencies) (*CoreServiceFactory, error) {
	logger.Info("🏗️  Создание главной фабрики сервисов ядра...")

	if deps.InfrastructureFactory == nil {
		return nil, fmt.Errorf("InfrastructureFactory не может быть nil")
	}

	if !deps.InfrastructureFactory.IsReady() {
		return nil, fmt.Errorf("InfrastructureFactory не готова")
	}

	// Создаем конфигурацию по умолчанию если не предоставлена
	if deps.Config == nil {
		deps.Config = &Config{
			UserConfig: users.Config{
				UserDefaults: struct {
					MinGrowthThreshold float64
					MinFallThreshold   float64
					Language           string
					Timezone           string
				}{
					MinGrowthThreshold: 2.0,
					MinFallThreshold:   2.0,
					Language:           "ru",
					Timezone:           "Europe/Moscow",
				},
				DefaultMaxSignalsPerDay: 50,
				SessionTTL:              24 * time.Hour,
				MaxSessionsPerUser:      5,
			},
			SubscriptionConfig: subscription.Config{
				DefaultPlan:     "free",
				TrialPeriodDays: 7,
				GracePeriodDays: 3,
				AutoRenew:       true,
			},
			PaymentsConfig: payment.Config{
				TelegramBotToken:           "",
				TelegramStarsProviderToken: "",
				TelegramBotUsername:        "",
			},
		}
	}

	// Получаем сервисы из инфраструктурной фабрики
	databaseService, err := deps.InfrastructureFactory.CreateDatabaseService()
	if err != nil {
		logger.Warn("⚠️ DatabaseService недоступен: %v", err)
	}

	redisService, err := deps.InfrastructureFactory.CreateRedisService()
	if err != nil {
		logger.Warn("⚠️ RedisService недоступен: %v", err)
	}

	planRepo, err := deps.InfrastructureFactory.GetPlanRepository()
	if err != nil {
		logger.Warn("⚠️ Не удалось получить PlanRepository: %v", err)
	}

	// ⭐ Получаем PaymentRepository из инфраструктурной фабрики
	paymentRepo, err := deps.InfrastructureFactory.GetPaymentRepository()
	if err != nil {
		logger.Warn("⚠️ Не удалось получить PaymentRepository: %v", err)
	}

	// Создаем фабрику UserService
	userFactory, err := users.NewUserServiceFactory(users.UserServiceDependencies{
		Config:       deps.Config.UserConfig,
		Database:     databaseService,
		RedisService: redisService,
		Notifier:     deps.UserNotifier,
	})
	if err != nil {
		return nil, fmt.Errorf("не удалось создать фабрику UserService: %w", err)
	}

	// Создаем фабрику SubscriptionService с PaymentRepository
	subscriptionDeps := subscription.Dependencies{
		Config:            deps.Config.SubscriptionConfig,
		PlanRepo:          planRepo,
		Cache:             redisService.GetCache(),
		Analytics:         deps.Analytics,
		PaymentRepo:       paymentRepo,     // ⭐ Передаем PaymentRepository
		ValidatorInterval: 1 * time.Minute, // Интервал для валидатора
	}

	subscriptionFactory, err := subscription.NewSubscriptionServiceFactory(subscriptionDeps)
	if err != nil {
		return nil, fmt.Errorf("не удалось создать фабрику SubscriptionService: %w", err)
	}

	factory := &CoreServiceFactory{
		config:                deps.Config,
		infrastructureFactory: deps.InfrastructureFactory,
		userFactory:           userFactory,
		subscriptionFactory:   subscriptionFactory,
		initialized:           true,
	}

	logger.Info("✅ Главная фабрика сервисов ядра создана")
	return factory, nil
}

// Initialize инициализирует фабрику и создает все сервисы
func (f *CoreServiceFactory) Initialize() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.initialized {
		return fmt.Errorf("фабрика ядра не инициализирована")
	}

	logger.Info("🔧 Инициализация фабрики ядра и создание сервисов...")

	// Проверяем зависимости
	if !f.validateDependencies() {
		return fmt.Errorf("зависимости не готовы")
	}

	f.initialized = true
	logger.Info("✅ Фабрика ядра инициализирована")
	return nil
}

// CreateUserService создает UserService
func (f *CoreServiceFactory) CreateUserService() (*users.Service, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if !f.initialized {
		return nil, fmt.Errorf("фабрика ядра не инициализирована")
	}

	if f.userFactory == nil {
		return nil, fmt.Errorf("фабрика UserService не создана")
	}

	// Получаем актуальные сервисы из инфраструктурной фабрики
	databaseService, err := f.infrastructureFactory.CreateDatabaseService()
	if err != nil {
		return nil, fmt.Errorf("не удалось получить DatabaseService: %w", err)
	}

	redisService, err := f.infrastructureFactory.CreateRedisService()
	if err != nil {
		return nil, fmt.Errorf("не удалось получить RedisService: %w", err)
	}

	// Обновляем зависимости фабрики UserService
	f.userFactory.SetDatabase(databaseService)
	f.userFactory.SetRedisService(redisService)

	return f.userFactory.CreateUserService()
}

// CreatePaymentService создает PaymentCoreService (StarsService)
func (f *CoreServiceFactory) CreatePaymentService() (*payment.PaymentService, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if !f.initialized {
		return nil, fmt.Errorf("фабрика ядра не инициализирована")
	}

	// Проверяем наличие конфигурации платежей
	if f.config.PaymentsConfig.TelegramBotToken == "" {
		return nil, fmt.Errorf("TelegramBotToken не настроен в PaymentsConfig")
	}

	// Получаем InfrastructureFactory
	infraFactory := f.GetInfrastructureFactory()
	if infraFactory == nil {
		return nil, fmt.Errorf("InfrastructureFactory не доступна")
	}

	// Получаем EventBus
	eventBus, err := infraFactory.GetEventBus()
	if err != nil {
		logger.Warn("⚠️ EventBus недоступен: %v", err)
	}

	// Получаем UserService
	userService, err := f.userFactory.CreateUserService()
	if err != nil {
		return nil, fmt.Errorf("не удалось создать UserService: %w", err)
	}

	// Создаем StarsClient
	baseURL := "https://api.telegram.org/bot" + f.config.PaymentsConfig.TelegramBotToken + "/"
	starsClient := http_client.NewStarsClient(baseURL, f.config.PaymentsConfig.TelegramStarsProviderToken)

	// Создаем StarsService
	starsService := payment.NewStarsService(
		userService,
		eventBus,
		logger.GetLogger(),
		starsClient,
		f.config.PaymentsConfig.TelegramBotUsername,
	)

	// Получаем PaymentRepository
	paymentRepo, err := infraFactory.GetPaymentRepository()
	if err != nil {
		logger.Error("❌ PaymentRepository недоступен: %v", err)
		return nil, fmt.Errorf("PaymentRepository не доступен: %w", err)
	}

	// Получаем InvoiceRepository
	invoiceRepo, err := infraFactory.GetInvoiceRepository()
	if err != nil {
		logger.Error("❌ InvoiceRepository недоступен: %v", err)
		return nil, fmt.Errorf("InvoiceRepository не доступен: %w", err)
	}

	// Создаем PaymentService через фабрику
	paymentServiceFactory, err := payment.NewPaymentServiceFactory(payment.PaymentServiceDependencies{
		StarsService: starsService,
		PaymentRepo:  paymentRepo,
		InvoiceRepo:  invoiceRepo,
		Logger:       logger.GetLogger(),
	})
	if err != nil {
		return nil, fmt.Errorf("не удалось создать фабрику PaymentService: %w", err)
	}

	paymentService, err := paymentServiceFactory.CreatePaymentService()
	if err != nil {
		return nil, fmt.Errorf("не удалось создать PaymentService: %w", err)
	}

	logger.Info("✅ PaymentService создан")
	return paymentService, nil
}

// CreateSubscriptionService создает SubscriptionService (автоматически запускает валидатор)
func (f *CoreServiceFactory) CreateSubscriptionService() (*subscription.Service, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if !f.initialized {
		return nil, fmt.Errorf("фабрика ядра не инициализирована")
	}

	if f.subscriptionFactory == nil {
		return nil, fmt.Errorf("фабрика SubscriptionService не создана")
	}

	// Получаем актуальные сервисы из инфраструктурной фабрики
	databaseService, err := f.infrastructureFactory.CreateDatabaseService()
	if err != nil {
		return nil, fmt.Errorf("не удалось получить DatabaseService: %w", err)
	}

	redisService, err := f.infrastructureFactory.CreateRedisService()
	if err != nil {
		return nil, fmt.Errorf("не удалось получить RedisService: %w", err)
	}

	// Обновляем зависимости фабрики SubscriptionService
	f.subscriptionFactory.SetDatabase(databaseService.GetDB())
	f.subscriptionFactory.SetRedisService(redisService)

	// Создаем сервис (валидатор запустится автоматически)
	return f.subscriptionFactory.CreateSubscriptionService(databaseService.GetDB())
}

// CreateAllServices создает все сервисы ядра
func (f *CoreServiceFactory) CreateAllServices() (map[string]interface{}, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.initialized {
		return nil, fmt.Errorf("фабрика ядра не инициализирована")
	}

	logger.Info("🏭 Создание всех сервисов ядра...")

	// Получаем актуальные сервисы из инфраструктурной фабрики
	databaseService, err := f.infrastructureFactory.CreateDatabaseService()
	if err != nil {
		return nil, fmt.Errorf("не удалось получить DatabaseService: %w", err)
	}

	redisService, err := f.infrastructureFactory.CreateRedisService()
	if err != nil {
		return nil, fmt.Errorf("не удалось получить RedisService: %w", err)
	}

	services := make(map[string]interface{})

	// Обновляем зависимости фабрик
	f.userFactory.SetDatabase(databaseService)
	f.userFactory.SetRedisService(redisService)
	f.subscriptionFactory.SetDatabase(databaseService.GetDB())
	f.subscriptionFactory.SetRedisService(redisService)

	// Создаем UserService
	userService, err := f.userFactory.CreateUserService()
	if err != nil {
		return nil, fmt.Errorf("не удалось создать UserService: %w", err)
	}
	services["UserService"] = userService
	logger.Info("✅ UserService создан")

	// Создаем SubscriptionService (с валидатором)
	subscriptionService, err := f.subscriptionFactory.CreateSubscriptionService(databaseService.GetDB())
	if err != nil {
		logger.Warn("⚠️ Не удалось создать SubscriptionService: %v", err)
		services["SubscriptionService"] = nil
	} else {
		services["SubscriptionService"] = subscriptionService
		logger.Info("✅ SubscriptionService создан (валидатор запущен)")
	}

	logger.Info("✅ Все сервисы ядра созданы")
	return services, nil
}

// UpdateConfig обновляет конфигурацию
func (f *CoreServiceFactory) UpdateConfig(newConfig *Config) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if newConfig != nil {
		f.config = newConfig

		// Обновляем конфигурации вложенных фабрик
		if f.userFactory != nil {
			f.userFactory.UpdateConfig(newConfig.UserConfig)
		}
		if f.subscriptionFactory != nil {
			f.subscriptionFactory.UpdateConfig(newConfig.SubscriptionConfig)
		}
	}
}

// GetInfrastructureFactory возвращает инфраструктурную фабрику
func (f *CoreServiceFactory) GetInfrastructureFactory() *infrastructure_factory.InfrastructureFactory {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.infrastructureFactory
}

// GetUserFactory возвращает фабрику UserService
func (f *CoreServiceFactory) GetUserFactory() *users.UserServiceFactory {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.userFactory
}

// GetSubscriptionFactory возвращает фабрику SubscriptionService
func (f *CoreServiceFactory) GetSubscriptionFactory() *subscription.SubscriptionServiceFactory {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.subscriptionFactory
}

// GetConfig возвращает текущую конфигурацию
func (f *CoreServiceFactory) GetConfig() *Config {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.config
}

// Validate проверяет готовность фабрики
func (f *CoreServiceFactory) Validate() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if !f.initialized {
		logger.Warn("⚠️ Главная фабрика ядра не инициализирована")
		return false
	}

	if !f.validateDependencies() {
		return false
	}

	if f.userFactory == nil {
		logger.Warn("⚠️ Фабрика UserService не создана")
		return false
	}

	if !f.userFactory.Validate() {
		logger.Warn("⚠️ Фабрика UserService не валидна")
		return false
	}

	if f.subscriptionFactory != nil && !f.subscriptionFactory.Validate() {
		logger.Warn("⚠️ Фабрика SubscriptionService не валидна")
		return false
	}

	logger.Info("✅ Главная фабрика ядра валидирована")
	return true
}

// validateDependencies проверяет зависимости
func (f *CoreServiceFactory) validateDependencies() bool {
	if f.infrastructureFactory == nil {
		logger.Warn("⚠️ InfrastructureFactory не доступна для фабрики ядра")
		return false
	}

	if !f.infrastructureFactory.IsReady() {
		logger.Warn("⚠️ InfrastructureFactory не готова")
		return false
	}

	return true
}

// GetHealthStatus возвращает статус здоровья фабрики
func (f *CoreServiceFactory) GetHealthStatus() map[string]interface{} {
	f.mu.RLock()
	defer f.mu.RUnlock()

	status := map[string]interface{}{
		"initialized":          f.initialized,
		"infrastructure_ready": f.infrastructureFactory != nil && f.infrastructureFactory.IsReady(),
		"user_factory":         f.userFactory != nil,
		"subscription_factory": f.subscriptionFactory != nil,
		"config_available":     f.config != nil,
	}

	// Проверяем доступность сервисов инфраструктуры
	if f.infrastructureFactory != nil {
		infraStatus := f.infrastructureFactory.GetHealthStatus()
		status["infrastructure_status"] = infraStatus

		// Проверяем ключевые компоненты
		status["database_ready"] = infraStatus["database_service_ready"] == true
		status["redis_ready"] = infraStatus["redis_service_ready"] == true
	}

	// Добавляем информацию о фабриках
	if f.userFactory != nil {
		status["user_factory_info"] = f.userFactory.GetDependenciesInfo()
	}
	if f.subscriptionFactory != nil {
		status["subscription_factory_info"] = f.subscriptionFactory.GetDependenciesInfo()
	}

	return status
}

// Reset сбрасывает фабрику
func (f *CoreServiceFactory) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.infrastructureFactory = nil
	f.userFactory = nil
	f.subscriptionFactory = nil
	f.initialized = false
	f.config = nil

	logger.Info("🔄 Главная фабрика ядра сброшена")
}

// IsReady проверяет готовность фабрики к созданию сервисов
func (f *CoreServiceFactory) IsReady() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Базовые проверки
	if !f.initialized {
		logger.Debug("⚠️ Фабрика ядра не инициализирована")
		return false
	}

	if f.infrastructureFactory == nil {
		logger.Warn("⚠️ InfrastructureFactory не доступна для фабрики ядра")
		return false
	}

	// Проверяем что инфраструктурная фабрика хотя бы создана
	if !f.infrastructureFactory.IsReady() {
		logger.Debug("⚠️ InfrastructureFactory не готова")
		return false
	}

	// Проверяем вложенные фабрики
	if f.userFactory == nil {
		logger.Debug("⚠️ Фабрика UserService не создана")
		return false
	}

	if !f.userFactory.IsReady() {
		logger.Debug("⚠️ Фабрика UserService не готова")
		return false
	}

	if f.subscriptionFactory == nil {
		logger.Debug("⚠️ Фабрика SubscriptionService не создана")
		return false
	}

	if !f.subscriptionFactory.Validate() { // ⭐ Используем Validate вместо IsReady
		logger.Debug("⚠️ Фабрика SubscriptionService не готова")
		return false
	}

	logger.Debug("✅ CoreServiceFactory готова к созданию сервисов")
	return true
}

// UpdateDependencies обновляет зависимости фабрики
func (f *CoreServiceFactory) UpdateDependencies(deps CoreServiceDependencies) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if deps.InfrastructureFactory == nil {
		return fmt.Errorf("InfrastructureFactory не может быть nil")
	}

	f.infrastructureFactory = deps.InfrastructureFactory

	// Обновляем конфигурацию если предоставлена
	if deps.Config != nil {
		f.config = deps.Config
	}

	// Обновляем зависимости вложенных фабрик если они существуют
	if f.userFactory != nil {
		databaseService, err := deps.InfrastructureFactory.CreateDatabaseService()
		if err == nil && databaseService != nil {
			f.userFactory.SetDatabase(databaseService)
		}

		redisService, err := deps.InfrastructureFactory.CreateRedisService()
		if err == nil && redisService != nil {
			f.userFactory.SetRedisService(redisService)
		}

		if deps.UserNotifier != nil {
			f.userFactory.SetNotifier(deps.UserNotifier)
		}
	}

	if f.subscriptionFactory != nil {
		databaseService, err := deps.InfrastructureFactory.CreateDatabaseService()
		if err == nil && databaseService != nil {
			f.subscriptionFactory.SetDatabase(databaseService.GetDB())
		}

		redisService, err := deps.InfrastructureFactory.CreateRedisService()
		if err == nil && redisService != nil {
			f.subscriptionFactory.SetRedisService(redisService)
		}

		if deps.Analytics != nil {
			f.subscriptionFactory.SetAnalytics(deps.Analytics)
		}

		// Обновляем PaymentRepository
		paymentRepo, err := deps.InfrastructureFactory.GetPaymentRepository()
		if err == nil && paymentRepo != nil {
			f.subscriptionFactory.SetPaymentRepository(paymentRepo)
		}
	}

	logger.Info("✅ Зависимости фабрики ядра обновлены")
	return nil
}
