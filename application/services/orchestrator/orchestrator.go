// application/services/orchestrator/orchestrator.go
package orchestrator

import (
	"crypto-exchange-screener-bot/application/pipeline"
	fetcher "crypto-exchange-screener-bot/internal/adapters/market"
	notifier "crypto-exchange-screener-bot/internal/adapters/notification"
	"crypto-exchange-screener-bot/internal/core/domain/candle"
	"crypto-exchange-screener-bot/internal/core/domain/signals/engine"
	"crypto-exchange-screener-bot/internal/core/domain/subscription"

	"crypto-exchange-screener-bot/internal/core/domain/users"
	telegrambot "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot"
	telegramintegrations "crypto-exchange-screener-bot/internal/delivery/telegram/integrations"
	redis "crypto-exchange-screener-bot/internal/infrastructure/cache/redis"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/in_memory_storage"
	database "crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/database"
	events "crypto-exchange-screener-bot/internal/infrastructure/transport/event_bus"
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
	"sync"
	"time"
)

// DataManager главный менеджер данных
type DataManager struct {
	config *config.Config

	// Компоненты цепочки
	priceFetcher   fetcher.PriceFetcher
	storage        storage.PriceStorage
	analysisEngine *engine.AnalysisEngine
	signalPipeline *pipeline.SignalPipeline
	notification   *notifier.CompositeNotificationService

	// EventBus и координация
	eventBus  *events.EventBus
	lifecycle *LifecycleManager
	registry  *ServiceRegistry

	// Дополнительные сервисы
	telegramBot   *telegrambot.TelegramBot
	webhookServer *telegrambot.WebhookServer

	// Сервис базы данных
	databaseService     *database.DatabaseService
	redisService        *redis.RedisService
	userService         *users.Service
	subscriptionService *subscription.Service

	// Telegram Package Service
	telegramPackageService telegramintegrations.TelegramPackageService

	// Свечная система
	candleSystem *candle.CandleSystem

	// Управление
	mu       sync.RWMutex
	stopChan chan struct{}
	wg       sync.WaitGroup

	// Статистика
	startTime   time.Time
	systemStats SystemStats
}

// Старая версия для обратной совместимости
func NewDataManagerDefault(cfg *config.Config) (*DataManager, error) {
	return NewDataManager(cfg, false)
}

// NewDataManager создает новый менеджер данных
func NewDataManager(cfg *config.Config, testMode bool) (*DataManager, error) {
	dm := &DataManager{
		config:    cfg,
		stopChan:  make(chan struct{}),
		startTime: time.Now(),
		systemStats: SystemStats{
			Services:    make(map[string]ServiceInfo),
			LastUpdated: time.Now(),
		},
	}

	// Инициализируем компоненты с тестовым режимом
	if err := dm.InitializeComponents(testMode); err != nil {
		return nil, err
	}

	// Инициализируем сервисы, зависящие от запущенных БД/Redis
	dm.initPostStartServices()

	// Настраиваем зависимости
	dm.setupDependencies()

	// Запускаем фоновые задачи
	dm.startBackgroundTasks()

	logger.Info("🚀 DataManager успешно создан")
	return dm, nil
}

// InitializeComponents инициализирует все компоненты
func (dm *DataManager) InitializeComponents(testMode bool) error {
	logger.Warn("🔍 DataManager: RateLimitDelay = %v\n", dm.config.RateLimitDelay)

	if dm.config.RateLimitDelay > 0 {
		logger.Warn("⚠️  RateLimitingMiddleware активен для EventPriceUpdated")
		logger.Warn("   Лимит: %v между событиями\n", dm.config.RateLimitDelay)
	}

	// Инициализация инфраструктуры
	if err := dm.initInfrastructure(testMode); err != nil {
		return err
	}

	// Инициализация хранения и получения данных
	candleSystem, err := dm.initStorageAndFetchers()
	if err != nil {
		return err
	}
	dm.candleSystem = candleSystem

	// Инициализация пользователей и авторизации
	if err := dm.initUsersAndAuth(); err != nil {
		return err
	}

	// Инициализация Telegram и уведомлений
	if err := dm.initTelegramAndNotifications(testMode); err != nil {
		return err
	}

	// Инициализация анализа и обработки
	if err := dm.initAnalysisAndProcessing(); err != nil {
		return err
	}

	// Регистрация и настройка
	if err := dm.initRegistrationAndSetup(); err != nil {
		return err
	}

	// Подписка notification service
	dm.subscribeNotificationService()

	logger.Info("🎉 Все компоненты успешно инициализированы!")
	return nil
}

// GetSystemStats возвращает статистику системы
func (dm *DataManager) GetSystemStats() SystemStats {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return dm.systemStats
}

// GetHealthStatus возвращает статус здоровья системы
func (dm *DataManager) GetHealthStatus() HealthStatus {
	servicesInfo := dm.registry.GetAllInfo()
	serviceStatus := make(map[string]string)
	allHealthy := true

	for name, info := range servicesInfo {
		status := "healthy"
		if info.State != StateRunning {
			status = "unhealthy"
			allHealthy = false
		}
		serviceStatus[name] = status
	}

	overallStatus := "healthy"
	if !allHealthy {
		overallStatus = "degraded"
	}

	return HealthStatus{
		Status:    overallStatus,
		Services:  serviceStatus,
		Timestamp: time.Now(),
	}
}

// GetStorage возвращает хранилище
func (dm *DataManager) GetStorage() storage.PriceStorage {
	return dm.storage
}

// GetAnalysisEngine возвращает движок анализа
func (dm *DataManager) GetAnalysisEngine() *engine.AnalysisEngine {
	return dm.analysisEngine
}

// GetEventBus возвращает EventBus
func (dm *DataManager) GetEventBus() *events.EventBus {
	return dm.eventBus
}

// GetWebhookServer возвращает Webhook сервер
func (dm *DataManager) GetWebhookServer() *telegrambot.WebhookServer {
	return dm.webhookServer
}

// GetTelegramBot возвращает Telegram бота
func (dm *DataManager) GetTelegramBot() *telegrambot.TelegramBot {
	return dm.telegramBot
}

// GetPriceFetcher возвращает PriceFetcher
func (dm *DataManager) GetPriceFetcher() fetcher.PriceFetcher {
	return dm.priceFetcher
}

// GetDatabaseService возвращает сервис базы данных
func (dm *DataManager) GetDatabaseService() *database.DatabaseService {
	return dm.databaseService
}

// GetRedisService возвращает Redis сервис
func (dm *DataManager) GetRedisService() *redis.RedisService {
	return dm.redisService
}

// GetUserService возвращает сервис пользователей
func (dm *DataManager) GetUserService() *users.Service {
	return dm.userService
}

// GetSubscriptionService возвращает сервис подписок
func (dm *DataManager) GetSubscriptionService() *subscription.Service {
	return dm.subscriptionService
}

// GetTelegramPackageService возвращает Telegram package service
func (dm *DataManager) GetTelegramPackageService() telegramintegrations.TelegramPackageService {
	return dm.telegramPackageService
}

// GetService возвращает сервис по имени
func (dm *DataManager) GetService(name string) (interface{}, bool) {
	switch name {
	case "PriceStorage":
		return dm.storage, true
	case "PriceFetcher":
		return dm.priceFetcher, true
	case "AnalysisEngine":
		return dm.analysisEngine, true
	case "EventBus":
		return dm.eventBus, true
	case "TelegramBot":
		return dm.telegramBot, dm.telegramBot != nil
	case "DatabaseService":
		return dm.databaseService, dm.databaseService != nil
	case "RedisService":
		return dm.redisService, dm.redisService != nil
	case "UserService":
		return dm.userService, dm.userService != nil
	case "SubscriptionService":
		return dm.subscriptionService, dm.subscriptionService != nil
	case "TelegramPackageService":
		return dm.telegramPackageService, dm.telegramPackageService != nil
	default:
		return nil, false
	}
}

// PublishEvent публикует событие
func (dm *DataManager) PublishEvent(event types.Event) {
	dm.eventBus.Publish(event)
}

// Subscribe подписывает на события
func (dm *DataManager) Subscribe(eventType types.EventType, subscriber types.EventSubscriber) {
	dm.eventBus.Subscribe(eventType, subscriber)
}

// Unsubscribe отписывает от событий
func (dm *DataManager) Unsubscribe(eventType types.EventType, subscriber types.EventSubscriber) {
	dm.eventBus.Unsubscribe(eventType, subscriber)
}

// GetManagedService возвращает управляемый сервис по имени
func (dm *DataManager) GetManagedService(name string) (Service, bool) {
	return dm.registry.Get(name)
}
