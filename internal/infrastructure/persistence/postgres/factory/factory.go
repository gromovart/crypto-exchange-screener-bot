// internal/infrastructure/persistence/postgres/factory/factory.go
package postgres_factory

import (
	"crypto-exchange-screener-bot/internal/infrastructure/cache/redis"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/database"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/repository/activity"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/repository/api_key"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/repository/invoice"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/repository/payment"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/repository/plan"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/repository/session"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/repository/subscription"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/repository/users"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"sync"
)

// RepositoryFactory фабрика для создания репозиториев PostgreSQL
type RepositoryFactory struct {
	db                     *database.DatabaseService
	cache                  *redis.Cache
	encryptionKey          string
	userRepository         users.UserRepository
	activityRepository     activity.ActivityRepository
	apiKeyRepository       api_key.APIKeyRepository
	sessionRepository      session.SessionRepository
	subscriptionRepository subscription.SubscriptionRepository
	planRepository         plan.PlanRepository
	invoiceRepository      invoice.InvoiceRepository
	paymentRepository      payment.PaymentRepository
	mu                     sync.RWMutex
	initialized            bool
}

// RepositoryDependencies зависимости для фабрики репозиториев
type RepositoryDependencies struct {
	DatabaseService *database.DatabaseService
	Cache           *redis.Cache
	EncryptionKey   string
}

// NewRepositoryFactory создает новую фабрику репозиториев
func NewRepositoryFactory(deps RepositoryDependencies) (*RepositoryFactory, error) {
	logger.Info("🏗️  Создание фабрики репозиториев PostgreSQL...")

	if deps.DatabaseService == nil {
		return nil, fmt.Errorf("DatabaseService не может быть nil")
	}

	factory := &RepositoryFactory{
		db:            deps.DatabaseService,
		cache:         deps.Cache,
		encryptionKey: deps.EncryptionKey,
		initialized:   true,
	}

	logger.Info("✅ Фабрика репозиториев PostgreSQL создана")
	return factory, nil
}

// Initialize инициализирует все репозитории
func (rf *RepositoryFactory) Initialize() error {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if !rf.initialized {
		return fmt.Errorf("фабрика репозиториев не инициализирована")
	}

	logger.Info("🔧 Инициализация репозиториев PostgreSQL...")

	// Инициализируем все репозитории
	// Ленивая инициализация происходит при первом запросе

	logger.Info("✅ Репозитории PostgreSQL готовы к созданию")
	return nil
}

// CreateUserRepository создает или возвращает репозиторий пользователей
func (rf *RepositoryFactory) CreateUserRepository() (users.UserRepository, error) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if !rf.initialized {
		return nil, fmt.Errorf("фабрика репозиториев не инициализирована")
	}

	if rf.userRepository == nil {
		db := rf.db.GetDB()
		if db == nil {
			return nil, fmt.Errorf("соединение с базой данных не установлено")
		}

		if rf.cache == nil {
			return nil, fmt.Errorf("кэш Redis не инициализирован")
		}

		rf.userRepository = users.NewUserRepository(db, rf.cache)
		logger.Info("✅ UserRepository создан")
	}

	return rf.userRepository, nil
}

// CreateActivityRepository создает или возвращает репозиторий активности
func (rf *RepositoryFactory) CreateActivityRepository() (activity.ActivityRepository, error) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if !rf.initialized {
		return nil, fmt.Errorf("фабрика репозиториев не инициализирована")
	}

	if rf.activityRepository == nil {
		db := rf.db.GetDB()
		if db == nil {
			return nil, fmt.Errorf("соединение с базой данных не установлено")
		}

		if rf.cache == nil {
			return nil, fmt.Errorf("кэш Redis не инициализирован")
		}

		rf.activityRepository = activity.NewActivityRepository(db, rf.cache)
		logger.Info("✅ ActivityRepository создан")
	}

	return rf.activityRepository, nil
}

// CreateAPIKeyRepository создает или возвращает репозиторий API ключей
func (rf *RepositoryFactory) CreateAPIKeyRepository() (api_key.APIKeyRepository, error) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if !rf.initialized {
		return nil, fmt.Errorf("фабрика репозиториев не инициализирована")
	}

	if rf.apiKeyRepository == nil {
		db := rf.db.GetDB()
		if db == nil {
			return nil, fmt.Errorf("соединение с базой данных не установлено")
		}

		if rf.encryptionKey == "" {
			return nil, fmt.Errorf("ключ шифрования не установлен")
		}

		rf.apiKeyRepository = api_key.NewAPIKeyRepository(db, rf.encryptionKey)
		logger.Info("✅ APIKeyRepository создан")
	}

	return rf.apiKeyRepository, nil
}

// CreateSessionRepository создает или возвращает репозиторий сессий
func (rf *RepositoryFactory) CreateSessionRepository() (session.SessionRepository, error) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if !rf.initialized {
		return nil, fmt.Errorf("фабрика репозиториев не инициализирована")
	}

	if rf.sessionRepository == nil {
		db := rf.db.GetDB()
		if db == nil {
			return nil, fmt.Errorf("соединение с базой данных не установлено")
		}

		if rf.cache == nil {
			return nil, fmt.Errorf("кэш Redis не инициализирован")
		}

		rf.sessionRepository = session.NewSessionRepository(db, rf.cache)
		logger.Info("✅ SessionRepository создан")
	}

	return rf.sessionRepository, nil
}

// CreateSubscriptionRepository создает или возвращает репозиторий подписок
func (rf *RepositoryFactory) CreateSubscriptionRepository() (subscription.SubscriptionRepository, error) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if !rf.initialized {
		return nil, fmt.Errorf("фабрика репозиториев не инициализирована")
	}

	if rf.subscriptionRepository == nil {
		db := rf.db.GetDB()
		if db == nil {
			return nil, fmt.Errorf("соединение с базой данных не установлено")
		}

		if rf.cache == nil {
			return nil, fmt.Errorf("кэш Redis не инициализирован")
		}

		rf.subscriptionRepository = subscription.NewSubscriptionRepository(db)
		logger.Info("✅ SubscriptionRepository создан")
	}

	return rf.subscriptionRepository, nil
}

// CreateInvoiceRepository создает или возвращает репозиторий счетов
func (rf *RepositoryFactory) CreateInvoiceRepository() (invoice.InvoiceRepository, error) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if !rf.initialized {
		return nil, fmt.Errorf("фабрика репозиториев не инициализирована")
	}

	if rf.invoiceRepository == nil {
		db := rf.db.GetDB()
		if db == nil {
			return nil, fmt.Errorf("соединение с базой данных не установлено")
		}

		if rf.cache == nil {
			return nil, fmt.Errorf("кэш Redis не инициализирован")
		}

		rf.invoiceRepository = invoice.NewInvoiceRepository(db)
		logger.Info("✅ SubscriptionRepository создан")
	}

	return rf.invoiceRepository, nil
}

// CreatePaymentRepository создает или возвращает репозиторий планов
func (rf *RepositoryFactory) CreatePaymentRepository() (payment.PaymentRepository, error) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if !rf.initialized {
		return nil, fmt.Errorf("фабрика репозиториев не инициализирована")
	}

	if rf.paymentRepository == nil {
		db := rf.db.GetDB()
		if db == nil {
			return nil, fmt.Errorf("соединение с базой данных не установлено")
		}

		if rf.cache == nil {
			return nil, fmt.Errorf("кэш Redis не инициализирован")
		}

		rf.paymentRepository = payment.NewPaymentRepository(db)
		logger.Info("✅ SubscriptionRepository создан")
	}

	return rf.paymentRepository, nil
}

// CreatePlanRepository создает или возвращает репозиторий планов
func (rf *RepositoryFactory) CreatePlanRepository() (plan.PlanRepository, error) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if !rf.initialized {
		return nil, fmt.Errorf("фабрика репозиториев не инициализирована")
	}

	if rf.planRepository == nil {
		db := rf.db.GetDB()
		if db == nil {
			return nil, fmt.Errorf("соединение с базой данных не установлено")
		}

		if rf.cache == nil {
			return nil, fmt.Errorf("кэш Redis не инициализирован")
		}

		rf.planRepository = plan.NewPlanRepository(db)
		logger.Info("✅ SubscriptionRepository создан")
	}

	return rf.planRepository, nil
}

// GetAllRepositories создает и возвращает все репозитории
func (rf *RepositoryFactory) GetAllRepositories() (map[string]interface{}, error) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if !rf.initialized {
		return nil, fmt.Errorf("фабрика репозиториев не инициализирована")
	}

	logger.Info("🏭 Создание всех репозиториев PostgreSQL...")

	repositories := make(map[string]interface{})
	var err error

	// Создаем все репозитории
	repositories["UserRepository"], err = rf.CreateUserRepository()
	if err != nil {
		logger.Warn("⚠️ Не удалось создать UserRepository: %v", err)
	}

	repositories["ActivityRepository"], err = rf.CreateActivityRepository()
	if err != nil {
		logger.Warn("⚠️ Не удалось создать ActivityRepository: %v", err)
	}

	repositories["APIKeyRepository"], err = rf.CreateAPIKeyRepository()
	if err != nil {
		logger.Warn("⚠️ Не удалось создать APIKeyRepository: %v", err)
	}

	repositories["SessionRepository"], err = rf.CreateSessionRepository()
	if err != nil {
		logger.Warn("⚠️ Не удалось создать SessionRepository: %v", err)
	}

	repositories["SubscriptionRepository"], err = rf.CreateSubscriptionRepository()
	if err != nil {
		logger.Warn("⚠️ Не удалось создать SubscriptionRepository: %v", err)
	}

	repositories["InvoiceRepository"], err = rf.CreateInvoiceRepository()
	if err != nil {
		logger.Warn("⚠️ Не удалось создать InvoiceRepository: %v", err)
	}
	repositories["CreatePaymentRepository"], err = rf.CreatePaymentRepository()
	if err != nil {
		logger.Warn("⚠️ Не удалось создать CreatePaymentRepository: %v", err)
	}

	repositories["CreatePlanRepository"], err = rf.CreatePlanRepository()
	if err != nil {
		logger.Warn("⚠️ Не удалось создать CreatePlanRepository: %v", err)
	}

	logger.Info("✅ Все репозитории PostgreSQL созданы")
	return repositories, nil
}

// Validate проверяет готовность фабрики
func (rf *RepositoryFactory) Validate() bool {
	rf.mu.RLock()
	defer rf.mu.RUnlock()

	if !rf.initialized {
		logger.Warn("⚠️ Фабрика репозиториев не инициализирована")
		return false
	}

	if rf.db == nil {
		logger.Warn("⚠️ DatabaseService не установлен")
		return false
	}

	return true
}

// GetHealthStatus возвращает статус здоровья фабрики репозиториев
func (rf *RepositoryFactory) GetHealthStatus() map[string]interface{} {
	rf.mu.RLock()
	defer rf.mu.RUnlock()

	status := map[string]interface{}{
		"initialized":                   rf.initialized,
		"database_service_ready":        rf.db != nil,
		"cache_ready":                   rf.cache != nil,
		"encryption_key_set":            rf.encryptionKey != "",
		"user_repository_ready":         rf.userRepository != nil,
		"activity_repository_ready":     rf.activityRepository != nil,
		"api_key_repository_ready":      rf.apiKeyRepository != nil,
		"session_repository_ready":      rf.sessionRepository != nil,
		"subscription_repository_ready": rf.subscriptionRepository != nil,
	}

	// Добавляем статус базы данных если она доступна
	if rf.db != nil {
		status["database_healthy"] = rf.db.HealthCheck()
		status["database_state"] = rf.db.State()
	}

	return status
}

// IsReady проверяет готовность фабрики
func (rf *RepositoryFactory) IsReady() bool {
	rf.mu.RLock()
	defer rf.mu.RUnlock()

	return rf.initialized && rf.db != nil
}

// GetDatabaseService возвращает сервис базы данных
func (rf *RepositoryFactory) GetDatabaseService() *database.DatabaseService {
	rf.mu.RLock()
	defer rf.mu.RUnlock()
	return rf.db
}

// GetCache возвращает кэш Redis
func (rf *RepositoryFactory) GetCache() *redis.Cache {
	rf.mu.RLock()
	defer rf.mu.RUnlock()
	return rf.cache
}

// Reset сбрасывает фабрику (очищает все репозитории)
func (rf *RepositoryFactory) Reset() {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	rf.userRepository = nil
	rf.activityRepository = nil
	rf.apiKeyRepository = nil
	rf.sessionRepository = nil
	rf.subscriptionRepository = nil
	rf.initialized = false

	logger.Info("🔄 Фабрика репозиториев сброшена")
}
