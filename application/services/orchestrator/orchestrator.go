// application/services/orchestrator/orchestrator.go
package orchestrator

import (
	"crypto-exchange-screener-bot/application/pipeline"
	"crypto-exchange-screener-bot/internal/adapters"
	fetcher "crypto-exchange-screener-bot/internal/adapters/market"
	"crypto-exchange-screener-bot/internal/adapters/notification"
	notifier "crypto-exchange-screener-bot/internal/adapters/notification"
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/core/domain/signals/engine"
	"crypto-exchange-screener-bot/internal/core/domain/subscription"
	subscriptiontypes "crypto-exchange-screener-bot/internal/core/domain/subscription"

	"crypto-exchange-screener-bot/internal/core/domain/users"
	telegrambot "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot" // ИЗМЕНЕНО
	telegramintegrations "crypto-exchange-screener-bot/internal/delivery/telegram/integrations"
	"crypto-exchange-screener-bot/internal/infrastructure/api/exchanges/bybit"
	redis "crypto-exchange-screener-bot/internal/infrastructure/cache/redis"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/in_memory_storage"
	database "crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/database"
	events "crypto-exchange-screener-bot/internal/infrastructure/transport/event_bus"
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"log"
	"runtime"
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
	telegramBot   *telegrambot.TelegramBot   // ИЗМЕНЕНО
	webhookServer *telegrambot.WebhookServer // ИЗМЕНЕНО

	// НОВОЕ: Сервис базы данных
	databaseService     *database.DatabaseService
	redisService        *redis.RedisService
	userService         *users.Service
	subscriptionService *subscription.Service

	// НОВОЕ: Telegram Package Service
	telegramPackageService telegramintegrations.TelegramPackageService

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

	// Настраиваем зависимости
	dm.setupDependencies()

	// Запускаем фоновые задачи
	dm.startBackgroundTasks()

	return dm, nil
}

// InitializeComponents инициализирует все компоненты
func (dm *DataManager) InitializeComponents(testMode bool) error {
	fmt.Printf("🔍 DataManager: RateLimitDelay = %v\n", dm.config.RateLimitDelay)

	if dm.config.RateLimitDelay > 0 {
		fmt.Println("⚠️  RateLimitingMiddleware активен для EventPriceUpdated")
		fmt.Printf("   Лимит: %v между событиями\n", dm.config.RateLimitDelay)
	}

	// ==================== БЛОК 1: ИНФРАСТРУКТУРА ====================

	// 1.1 База данных
	log.Println("🗄️  Creating database service...")
	dm.databaseService = database.NewDatabaseService(dm.config)
	if err := dm.databaseService.Start(); err != nil {
		log.Printf("⚠️  Failed to start database service: %v", err)
		log.Println("⚠️  Application will continue without database connection")
	} else {
		log.Println("✅ Database service started successfully")
	}

	// 1.2 Redis
	log.Println("🔴 Creating Redis service...")
	dm.redisService = redis.NewRedisService(dm.config)
	if err := dm.redisService.Start(); err != nil {
		log.Printf("⚠️  Failed to start Redis service: %v", err)
		log.Println("⚠️  Application will continue without Redis connection")
	} else {
		log.Println("✅ Redis service started successfully")
	}

	// 1.3 EventBus
	log.Println("🚌 Creating EventBus...")
	eventBusConfig := events.EventBusConfig{
		BufferSize:    dm.config.EventBus.BufferSize,
		WorkerCount:   dm.config.EventBus.WorkerCount,
		EnableMetrics: dm.config.EventBus.EnableMetrics,
		EnableLogging: dm.config.EventBus.EnableLogging,
	}
	dm.eventBus = events.NewEventBus(eventBusConfig)
	log.Println("✅ EventBus created")

	// ==================== БЛОК 2: ХРАНЕНИЕ И ПОЛУЧЕНИЕ ДАННЫХ ====================

	// 2.1 Хранилище цен
	log.Println("💾 Creating price storage...")
	storageConfig := &storage.StorageConfig{
		MaxHistoryPerSymbol: 10000,
		MaxSymbols:          1000,
		CleanupInterval:     5 * time.Minute,
		RetentionPeriod:     24 * time.Hour,
	}
	dm.storage = storage.NewInMemoryPriceStorage(storageConfig)
	log.Println("✅ Price storage created")

	// 2.2 API клиент
	log.Println("🌐 Creating API client...")
	apiClient := bybit.NewBybitClient(dm.config)

	// 2.3 Получение цен
	log.Println("📡 Creating PriceFetcher...")
	dm.priceFetcher = fetcher.NewPriceFetcher(apiClient, dm.storage, dm.eventBus)
	log.Println("✅ PriceFetcher created")

	// ==================== БЛОК 3: ПОЛЬЗОВАТЕЛИ И АВТОРИЗАЦИЯ ====================

	// 3.1 Сервис пользователей
	log.Println("👤 Creating user service...")
	if dm.databaseService != nil && dm.redisService != nil {
		db := dm.databaseService.GetDB()
		redisCache := dm.redisService.GetCache()

		if db != nil && redisCache != nil {
			userConfig := users.Config{
				DefaultMinGrowthThreshold: 2.0,
				DefaultMaxSignalsPerDay:   50,
				SessionTTL:                24 * time.Hour,
				MaxSessionsPerUser:        5,
			}

			var err error
			dm.userService, err = users.NewService(db, redisCache, nil, userConfig)
			if err != nil {
				log.Printf("⚠️  Не удалось создать сервис пользователей: %v", err)
			} else {
				log.Println("✅ User service created")
			}
		} else {
			log.Println("⚠️  Database or Redis connection not available")
		}
	} else {
		log.Println("⚠️  DatabaseService or RedisService not available")
	}

	// 3.2 Сервис подписок
	log.Println("💎 Creating subscription service...")
	if dm.databaseService != nil {
		db := dm.databaseService.GetDB()
		if db != nil && dm.redisService != nil {
			// Получаем кэш из redisService
			redisCache := dm.redisService.GetCache()
			if redisCache != nil {
				// Создаем конфигурацию подписок
				subscriptionConfig := subscriptiontypes.Config{
					StripeSecretKey:  "", // Пока не используется в конфигурации
					StripeWebhookKey: "", // Пока не используется в конфигурации
					DefaultPlan:      "free",
					TrialPeriodDays:  7,
					GracePeriodDays:  3,
					AutoRenew:        true,
				}

				// Создаем сервис подписок
				subService, err := subscriptiontypes.NewService(
					db,         // *sqlx.DB
					redisCache, // *redis.Cache
					nil,        // NotificationService (будет добавлено позже)
					nil,        // AnalyticsService (будет добавлено позже)
					subscriptionConfig,
				)

				if err != nil {
					log.Printf("⚠️  Не удалось создать сервис подписок: %v", err)
				} else {
					dm.subscriptionService = subService
					log.Println("✅ Subscription service created")
				}
			} else {
				log.Println("⚠️  Redis cache not available for subscription service")
			}
		} else {
			log.Println("⚠️  Database or Redis connection not available for subscription service")
		}
	}

	// ==================== БЛОК 4: TELEGRAM И УВЕДОМЛЕНИЯ ====================

	// 4.1 Telegram бот
	if dm.config.TelegramEnabled && dm.config.TelegramBotToken != "" {
		log.Println("🤖 Creating Telegram bot...")
		if dm.userService != nil {
			// ИЗМЕНЕНО: Используем новую функцию с зависимостями
			dm.telegramBot = telegrambot.GetOrCreateBotWithDeps(dm.config, &telegrambot.Dependencies{
				UserService: dm.userService,
			})
			log.Println("✅ Telegram bot created with auth (Singleton)")
		} else {
			dm.telegramBot = telegrambot.GetOrCreateBot(dm.config)
			log.Println("✅ Telegram bot created without auth (Singleton)")
		}

		if dm.telegramBot != nil {
			// TODO: Добавить метод SetTestMode если он существует
			if testMode {
				log.Println("🧪 Test mode - welcome messages disabled")
			} else {
				log.Println("✅ Bot ready, welcome message will be sent on /start command")
			}

			// Запускаем polling
			log.Println("🔄 Starting Telegram bot polling...")
			if err := dm.telegramBot.StartPolling(); err != nil {
				log.Printf("⚠️ Failed to start Telegram bot polling: %v", err)
			} else {
				log.Println("✅ Telegram bot polling started")
			}
		}
	}

	// 4.4 Telegram Package Service
	log.Println("📦 Creating Telegram package service...")
	if dm.config.TelegramEnabled && dm.userService != nil && dm.subscriptionService != nil && dm.eventBus != nil {
		telegramService, err := telegramintegrations.NewTelegramPackageServiceWithDefaults(
			dm.config,
			dm.userService,
			dm.subscriptionService,
			dm.eventBus,
		)

		if err != nil {
			log.Printf("⚠️  Failed to create Telegram package service: %v", err)
		} else {
			dm.telegramPackageService = telegramService
			log.Println("✅ Telegram package service created")
		}
	} else {
		log.Printf("⚠️  TelegramPackageService not created: TelegramEnabled=%v, userService=%v, subscriptionService=%v, eventBus=%v",
			dm.config.TelegramEnabled, dm.userService != nil, dm.subscriptionService != nil, dm.eventBus != nil)
	}

	// 4.5 Составной сервис уведомлений
	log.Println("📱 Creating CompositeNotificationService...")
	notifierFactory := notifier.NewNotifierFactory(dm.eventBus)
	dm.notification = notifierFactory.CreateCompositeNotifier(dm.config)
	if dm.notification == nil {
		return fmt.Errorf("failed to create CompositeNotificationService")
	}
	log.Println("✅ CompositeNotificationService created")

	// ==================== БЛОК 5: АНАЛИЗ И ОБРАБОТКА ====================

	// 5.1 Движок анализа
	log.Println("🔧 Creating AnalysisEngine...")
	analysisFactory := engine.NewFactory(dm.priceFetcher)

	var telegramNotifier *notification.TelegramNotifier
	if dm.notification != nil {
		for _, notifier := range dm.notification.GetNotifiers() {
			if tn, ok := notifier.(*notification.TelegramNotifier); ok {
				telegramNotifier = tn
				break
			}
		}
	}

	dm.analysisEngine = analysisFactory.NewAnalysisEngineFromConfig(
		dm.storage,
		dm.eventBus,
		dm.config,
		telegramNotifier,
	)
	log.Println("✅ AnalysisEngine created")

	// 5.2 Пайплайн сигналов
	log.Println("🔄 Creating SignalPipeline...")
	dm.signalPipeline = pipeline.NewSignalPipeline(dm.eventBus)
	log.Println("✅ SignalPipeline created")

	// ==================== БЛОК 6: РЕГИСТРАЦИЯ И НАСТРОЙКА ====================

	// 6.1 Регистрация подписчиков
	log.Println("📋 Registering EventBus subscribers...")
	dm.registerBasicSubscribers()

	// 6.2 Реестр сервисов
	log.Println("📝 Creating service registry...")
	dm.registry = NewServiceRegistry()

	// 6.3 Менеджер жизненного цикла
	log.Println("⚙️ Creating lifecycle manager...")
	coordinatorConfig := CoordinatorConfig{
		EnableEventLogging:  true,
		EventBufferSize:     1000,
		HealthCheckInterval: 30 * time.Second,
		RestartOnFailure:    true,
		MaxRestartAttempts:  3,
		RestartDelay:        5 * time.Second,
		EnableMetrics:       true,
		MetricsPort:         "9090",
	}
	dm.lifecycle = NewLifecycleManager(dm.registry, dm.eventBus, coordinatorConfig)
	log.Println("✅ Lifecycle manager created")

	// 6.4 Настройка пайплайна
	log.Println("🔗 Setting up pipeline...")
	dm.setupPipeline()

	// 6.5 Регистрация сервисов
	log.Println("🏷️ Registering services...")
	if err := dm.registerServices(); err != nil {
		return err
	}
	log.Println("✅ Services registered")

	// 6.6 Подписка notification service
	log.Println("✅ Subscribing notification service...")
	dm.subscribeNotificationService()

	log.Println("🎉 All components initialized successfully!")
	return nil
}

// registerBasicSubscribers регистрирует только базовых подписчиков
func (dm *DataManager) registerBasicSubscribers() {
	log.Println("📋 Starting subscriber registration...")

	// Консольный логгер для ошибок и сигналов
	consoleSubscriber := events.NewConsoleLoggerSubscriber()
	dm.eventBus.Subscribe(types.EventSignalDetected, consoleSubscriber)
	dm.eventBus.Subscribe(types.EventPriceUpdated, consoleSubscriber)
	dm.eventBus.Subscribe(types.EventError, consoleSubscriber)
	log.Println("✅ Console logger subscribed")

	// НОВОЕ: Telegram Package Service для обработки событий
	if dm.telegramPackageService != nil {
		// Создаем подписчика для событий счетчика
		counterSignalSubscriber := events.NewBaseSubscriber(
			"telegram_package_service_counter",
			[]types.EventType{types.EventCounterSignalDetected},
			func(event types.Event) error {
				return dm.telegramPackageService.HandleCounterSignal(event)
			},
		)
		dm.eventBus.Subscribe(types.EventCounterSignalDetected, counterSignalSubscriber)
		log.Println("✅ TelegramPackageService subscribed for EventCounterSignalDetected")
	}

	log.Println("🎯 Subscriber registration completed")
}

// subscribeNotificationService подписывает notification service на события сигналов
func (dm *DataManager) subscribeNotificationService() {
	if dm.notification == nil {
		return
	}

	notificationSubscriber := events.NewBaseSubscriber(
		"notification_service",
		[]types.EventType{types.EventSignalDetected},
		func(event types.Event) error {
			if dm.notification != nil && dm.notification.IsEnabled() {
				if signal, ok := event.Data.(analysis.Signal); ok {
					trendSignal := adapters.AnalysisSignalToTrendSignal(signal)
					return dm.notification.Send(trendSignal)
				}
			}
			return nil
		},
	)

	dm.eventBus.Subscribe(types.EventSignalDetected, notificationSubscriber)
	log.Println("✅ Notification service subscribed to signal events")
}

// setupPipeline настраивает этапы обработки сигналов
func (dm *DataManager) setupPipeline() {
	dm.signalPipeline.AddStage(&pipeline.ValidationStage{})
	dm.signalPipeline.AddStage(&pipeline.EnrichmentStage{})
	log.Println("✅ Pipeline stages configured")
}

// registerServices регистрирует сервисы в реестре
func (dm *DataManager) registerServices() error {
	services := map[string]Service{
		"PriceStorage":        dm.newServiceAdapter("PriceStorage", dm.storage),
		"PriceFetcher":        dm.newServiceAdapter("PriceFetcher", dm.priceFetcher),
		"AnalysisEngine":      dm.newServiceAdapter("AnalysisEngine", dm.analysisEngine),
		"SignalPipeline":      dm.newServiceAdapter("SignalPipeline", dm.signalPipeline),
		"NotificationService": dm.newServiceAdapter("NotificationService", dm.notification),
		"EventBus":            dm.newServiceAdapter("EventBus", dm.eventBus),
	}

	// Инфраструктура
	if dm.databaseService != nil {
		services["DatabaseService"] = dm.newServiceAdapter("DatabaseService", dm.databaseService)
	}
	if dm.redisService != nil {
		services["RedisService"] = dm.newServiceAdapter("RedisService", dm.redisService)
	}

	// Пользователи и подписки
	if dm.userService != nil {
		services["UserService"] = dm.newServiceAdapter("UserService", dm.userService)
	}
	if dm.subscriptionService != nil {
		services["SubscriptionService"] = dm.newServiceAdapter("SubscriptionService", dm.subscriptionService)
	}

	// Telegram
	if dm.telegramBot != nil {
		services["TelegramBot"] = dm.newServiceAdapter("TelegramBot", dm.telegramBot)
	}
	if dm.webhookServer != nil {
		services["WebhookServer"] = dm.newServiceAdapter("WebhookServer", dm.webhookServer)
	}

	// Telegram Package Service
	if dm.telegramPackageService != nil {
		services["TelegramPackageService"] = dm.newServiceAdapter("TelegramPackageService", dm.telegramPackageService)
	}

	for name, service := range services {
		if err := dm.registry.Register(name, service); err != nil {
			return fmt.Errorf("failed to register service %s: %w", name, err)
		}
		log.Printf("✅ Registered service: %s", name)
	}

	return nil
}

// setupDependencies настраивает зависимости между сервисами
func (dm *DataManager) setupDependencies() {
	// Анализ зависит от хранилища и EventBus
	dm.lifecycle.AddDependency("AnalysisEngine", "PriceStorage")
	dm.lifecycle.AddDependency("AnalysisEngine", "EventBus")

	// Пайплайн зависит от EventBus
	dm.lifecycle.AddDependency("SignalPipeline", "EventBus")

	// NotificationService зависит от EventBus
	dm.lifecycle.AddDependency("NotificationService", "EventBus")

	// TelegramBot зависит от EventBus
	if dm.telegramBot != nil {
		dm.lifecycle.AddDependency("TelegramBot", "EventBus")
	}

	// WebhookServer зависит от TelegramBot
	if dm.webhookServer != nil {
		dm.lifecycle.AddDependency("WebhookServer", "TelegramBot")
	}

	// TelegramPackageService зависит от EventBus
	if dm.telegramPackageService != nil {
		dm.lifecycle.AddDependency("TelegramPackageService", "EventBus")
	}
}

// startBackgroundTasks запускает фоновые задачи
func (dm *DataManager) startBackgroundTasks() {
	// Обновление статистики системы
	dm.wg.Add(1)
	go func() {
		defer dm.wg.Done()
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				dm.updateSystemStats()
			case <-dm.stopChan:
				return
			}
		}
	}()

	// Очистка старых данных
	dm.wg.Add(1)
	go func() {
		defer dm.wg.Done()
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if _, err := dm.storage.CleanOldData(24 * time.Hour); err != nil {
					logger.Info("⚠️ Failed to cleanup old data: %v", err)
				}
			case <-dm.stopChan:
				return
			}
		}
	}()

	// Мониторинг здоровья
	dm.wg.Add(1)
	go func() {
		defer dm.wg.Done()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				dm.checkHealth()
			case <-dm.stopChan:
				return
			}
		}
	}()
}

// updateSystemStats обновляет статистику системы
func (dm *DataManager) updateSystemStats() {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	servicesInfo := dm.registry.GetAllInfo()
	storageStats := dm.storage.GetStats()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	var analysisStats interface{}
	if dm.analysisEngine != nil {
		analysisStats = dm.analysisEngine.GetStats()
	}

	var eventBusStats interface{}
	if dm.eventBus != nil {
		eventBusStats = dm.eventBus.GetMetrics()
	}

	dm.systemStats = SystemStats{
		Services:      servicesInfo,
		StorageStats:  storageStats,
		AnalysisStats: analysisStats,
		EventBusStats: eventBusStats,
		Uptime:        time.Since(dm.startTime),
		TotalRequests: 0,
		MemoryUsageMB: float64(m.Alloc) / 1024 / 1024,
		CPUUsage:      0,
		ActiveSymbols: storageStats.TotalSymbols,
		LastUpdated:   time.Now(),
	}
}

// checkHealth проверяет здоровье системы
func (dm *DataManager) checkHealth() {
	health := dm.GetHealthStatus()
	if health.Status != "healthy" {
		dm.eventBus.Publish(types.Event{
			Type:   types.EventError,
			Source: "DataManager",
			Data: map[string]interface{}{
				"status":  health.Status,
				"message": "System health check failed",
			},
		})
		logger.Info("⚠️ System health check failed: %s", health.Status)
	}
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
func (dm *DataManager) GetWebhookServer() *telegrambot.WebhookServer { // ИЗМЕНЕНО
	return dm.webhookServer
}

// GetTelegramBot возвращает Telegram бота
func (dm *DataManager) GetTelegramBot() *telegrambot.TelegramBot { // ИЗМЕНЕНО
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

// RestartService перезапускает сервис
func (dm *DataManager) RestartService(name string) error {
	return dm.lifecycle.RestartService(name)
}

// IsRunning проверяет работает ли менеджер
func (dm *DataManager) IsRunning() bool {
	select {
	case <-dm.stopChan:
		return false
	default:
		return true
	}
}

// WaitForShutdown ожидает завершения работы
func (dm *DataManager) WaitForShutdown() {
	dm.wg.Wait()
}

// Cleanup очищает ресурсы
func (dm *DataManager) Cleanup() {
	dm.storage.Clear()
}

// Stop останавливает все сервисы
func (dm *DataManager) Stop() error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	log.Println("🛑 Stopping DataManager...")
	close(dm.stopChan)
	dm.wg.Wait()

	errors := dm.lifecycle.StopAll()
	if len(errors) > 0 {
		for service, err := range errors {
			logger.Info("⚠️ Failed to stop %s: %v", service, err)
		}
	}

	if dm.eventBus != nil {
		dm.eventBus.Stop()
	}

	log.Println("✅ DataManager stopped")
	return nil
}

// ==================== НОВЫЕ МЕТОДЫ ====================

// StartAllServices запускает все сервисы
func (dm *DataManager) StartAllServices() map[string]error {
	return dm.lifecycle.StartAll()
}

// StartService запускает конкретный сервис
func (dm *DataManager) StartService(name string) error {
	return dm.lifecycle.StartService(name)
}

// StopService останавливает конкретный сервис
func (dm *DataManager) StopService(name string) error {
	return dm.lifecycle.StopService(name)
}

// GetServicesInfo возвращает информацию о всех сервисах
func (dm *DataManager) GetServicesInfo() map[string]ServiceInfo {
	return dm.registry.GetAllInfo()
}

// GetStorageStats возвращает статистику хранилища
func (dm *DataManager) GetStorageStats() storage.StorageStats {
	return dm.storage.GetStats()
}

// GetAnalysisEngineStats возвращает статистику анализатора
func (dm *DataManager) GetAnalysisEngineStats() engine.EngineStats {
	if dm.analysisEngine != nil {
		return dm.analysisEngine.GetStats()
	}
	return engine.EngineStats{}
}

// RunAnalysis выполняет анализ всех символов
func (dm *DataManager) RunAnalysis() (map[string]*analysis.AnalysisResult, error) {
	if dm.analysisEngine == nil {
		return nil, fmt.Errorf("analysis engine not initialized")
	}
	return dm.analysisEngine.AnalyzeAll()
}

// GetAnalysisResults возвращает результаты анализа для символа
func (dm *DataManager) GetAnalysisResults(symbol string, periods []time.Duration) (*analysis.AnalysisResult, error) {
	if dm.analysisEngine == nil {
		return nil, fmt.Errorf("analysis engine not initialized")
	}
	return dm.analysisEngine.AnalyzeSymbol(symbol, periods)
}

// GetActiveAnalyzers возвращает список активных анализаторов
func (dm *DataManager) GetActiveAnalyzers() []string {
	if dm.analysisEngine != nil {
		return dm.analysisEngine.GetAnalyzers()
	}
	return []string{}
}

// AddConsoleSubscriber добавляет подписчика для вывода в консоль
func (dm *DataManager) AddConsoleSubscriber() {
	consoleSubscriber := events.NewConsoleLoggerSubscriber()
	dm.eventBus.Subscribe(types.EventSignalDetected, consoleSubscriber)
	dm.eventBus.Subscribe(types.EventPriceUpdated, consoleSubscriber)
	dm.eventBus.Subscribe(types.EventError, consoleSubscriber)
}

// AddTelegramSubscriber добавляет подписчика Telegram
func (dm *DataManager) AddTelegramSubscriber() error {
	if dm.telegramBot == nil {
		return fmt.Errorf("telegram bot not initialized")
	}
	// ИЗМЕНЕНО: Нужно переписать под новую архитектуру
	log.Println("⚠️  AddTelegramSubscriber needs to be reimplemented for new architecture")
	return nil
}

// ==================== Service Adapter ====================

// serviceAdapter адаптирует любой объект к интерфейсу Service
type serviceAdapter struct {
	name    string
	service interface{}
	state   ServiceState
}

func (sa *serviceAdapter) Name() string {
	return sa.name
}

func (sa *serviceAdapter) Start() error {
	sa.state = StateStarting

	switch s := sa.service.(type) {
	case storage.PriceStorage:
		sa.state = StateRunning

	case fetcher.PriceFetcher:
		updateInterval := time.Duration(10) * time.Second
		if err := s.Start(updateInterval); err != nil {
			sa.state = StateError
			return err
		}
		sa.state = StateRunning

	case *telegrambot.WebhookServer: // ИЗМЕНЕНО
		if err := s.Start(); err != nil {
			sa.state = StateError
			return err
		}
		sa.state = StateRunning

	case *database.DatabaseService:
		if s.State() == database.StateRunning {
			sa.state = StateRunning
		} else if s.State() == database.StateError {
			sa.state = StateError
			return fmt.Errorf("database service in error state")
		} else {
			if err := s.Start(); err != nil {
				sa.state = StateError
				return err
			}
			sa.state = StateRunning
		}

	case *redis.RedisService:
		if s.State() == redis.StateRunning {
			sa.state = StateRunning
		} else if s.State() == redis.StateError {
			sa.state = StateError
			return fmt.Errorf("Redis service in error state")
		} else {
			if err := s.Start(); err != nil {
				sa.state = StateError
				return err
			}
			sa.state = StateRunning
		}

	case *engine.AnalysisEngine:
		if err := s.Start(); err != nil {
			sa.state = StateError
			return err
		}
		sa.state = StateRunning

	case *pipeline.SignalPipeline:
		s.Start()
		sa.state = StateRunning

	case *notifier.CompositeNotificationService:
		sa.state = StateRunning

	case *events.EventBus:
		s.Start()
		sa.state = StateRunning

	case *telegrambot.TelegramBot: // ИЗМЕНЕНО
		sa.state = StateRunning

	case telegramintegrations.TelegramPackageService:
		if err := s.Start(); err != nil {
			sa.state = StateError
			return err
		}
		sa.state = StateRunning
	}

	return nil
}

func (sa *serviceAdapter) Stop() error {
	sa.state = StateStopping

	switch s := sa.service.(type) {
	case fetcher.PriceFetcher:
		s.Stop()
	case *engine.AnalysisEngine:
		s.Stop()
	case *events.EventBus:
		s.Stop()
	case *telegrambot.WebhookServer: // ИЗМЕНЕНО
		if err := s.Stop(); err != nil {
			return err
		}
	case *database.DatabaseService:
		if err := s.Stop(); err != nil {
			return err
		}
	case *redis.RedisService:
		if err := s.Stop(); err != nil {
			return err
		}
	case telegramintegrations.TelegramPackageService:
		if err := s.Stop(); err != nil {
			return err
		}
	}

	sa.state = StateStopped
	return nil
}

func (sa *serviceAdapter) State() ServiceState {
	return sa.state
}

func (sa *serviceAdapter) HealthCheck() bool {
	if sa.state != StateRunning {
		return false
	}

	switch s := sa.service.(type) {
	case *database.DatabaseService:
		return s.HealthCheck()
	case *redis.RedisService:
		return s.HealthCheck()
	default:
		return sa.state == StateRunning
	}
}

// newServiceAdapter создает адаптер сервиса
func (dm *DataManager) newServiceAdapter(name string, service interface{}) Service {
	return &serviceAdapter{
		name:    name,
		service: service,
		state:   StateStopped,
	}
}

// IsInitialized проверяет инициализацию
func (dm *DataManager) IsInitialized() bool {
	return dm.storage != nil && dm.eventBus != nil && dm.analysisEngine != nil
}

// GetAnalyzers возвращает список анализаторов
func (dm *DataManager) GetAnalyzers() []string {
	if dm.analysisEngine != nil {
		return dm.analysisEngine.GetAnalyzers()
	}
	return []string{}
}

// TriggerAnalysis запускает ручной анализ
func (dm *DataManager) TriggerAnalysis() {
	if dm.analysisEngine != nil {
		go func() {
			results, err := dm.analysisEngine.AnalyzeAll()
			if err != nil {
				logger.Info("Ошибка при ручном анализе: %v", err)
			} else {
				logger.Info("Ручной анализ завершен: %d символов обработано", len(results))
			}
		}()
	}
}
