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
	"crypto-exchange-screener-bot/internal/core/domain/users"
	"crypto-exchange-screener-bot/internal/delivery/telegram"
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
	telegramBot   *telegram.TelegramBot
	webhookServer *telegram.WebhookServer

	// НОВОЕ: Сервис базы данных
	databaseService *database.DatabaseService
	redisService    *redis.RedisService
	userService     *users.Service

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
// InitializeComponents инициализирует все компоненты
func (dm *DataManager) InitializeComponents(testMode bool) error {
	fmt.Printf("🔍 DataManager: RateLimitDelay = %v\n", dm.config.RateLimitDelay)

	if dm.config.RateLimitDelay > 0 {
		fmt.Println("⚠️  RateLimitingMiddleware активен для EventPriceUpdated")
		fmt.Printf("   Лимит: %v между событиями\n", dm.config.RateLimitDelay)
	}

	// 0. СОЗДАЕМ СЕРВИС БАЗЫ ДАННЫХ (первым)
	log.Println("🗄️  Creating database service...")
	dm.databaseService = database.NewDatabaseService(dm.config)

	// Пытаемся подключиться к базе данных
	if err := dm.databaseService.Start(); err != nil {
		log.Printf("⚠️  Failed to start database service: %v", err)
		log.Println("⚠️  Application will continue without database connection")
		// Не возвращаем ошибку, чтобы приложение могло работать без БД
	} else {
		log.Println("✅ Database service started successfully")
	}

	// 0.1 СОЗДАЕМ REDIS СЕРВИС (вторым)
	log.Println("🔴 Creating Redis service...")
	dm.redisService = redis.NewRedisService(dm.config)

	// Пытаемся подключиться к Redis
	if err := dm.redisService.Start(); err != nil {
		log.Printf("⚠️  Failed to start Redis service: %v", err)
		log.Println("⚠️  Application will continue without Redis connection")
		// Не возвращаем ошибку, чтобы приложение могло работать без Redis
	} else {
		log.Println("✅ Redis service started successfully")
	}

	// 🔐 СОЗДАЕМ СЕРВИС ПОЛЬЗОВАТЕЛЕЙ ДЛЯ АВТОРИЗАЦИИ (перед Telegram ботом)
	log.Println("👤 Создание сервиса пользователей для авторизации...")
	if dm.databaseService != nil && dm.redisService != nil {
		// Получаем соединение с БД
		db := dm.databaseService.GetDB()
		// Получаем Redis кэш
		redisCache := dm.redisService.GetCache()

		if db != nil && redisCache != nil {
			// Создаем конфигурацию
			userConfig := users.Config{
				DefaultMinGrowthThreshold: 2.0,
				DefaultMaxSignalsPerDay:   50,
				SessionTTL:                24 * time.Hour,
				MaxSessionsPerUser:        5,
			}

			// Создаем сервис пользователей
			var err error
			dm.userService, err = users.NewService(db, redisCache, nil, userConfig)
			if err != nil {
				log.Printf("⚠️  Не удалось создать сервис пользователей: %v", err)
			} else {
				log.Println("✅ Сервис пользователей создан для авторизации")
			}
		} else {
			log.Println("⚠️  Не удалось получить подключение к БД или Redis")
		}
	} else {
		log.Println("⚠️  DatabaseService или RedisService не создан, авторизация будет отключена")
	}

	// 1. Создаем EventBus
	eventBusConfig := events.EventBusConfig{
		BufferSize:    dm.config.EventBus.BufferSize,
		WorkerCount:   dm.config.EventBus.WorkerCount,
		EnableMetrics: dm.config.EventBus.EnableMetrics,
		EnableLogging: dm.config.EventBus.EnableLogging,
	}
	dm.eventBus = events.NewEventBus(eventBusConfig)

	// 2. Создаем хранилище
	storageConfig := &storage.StorageConfig{
		MaxHistoryPerSymbol: 10000,
		MaxSymbols:          1000,
		CleanupInterval:     5 * time.Minute,
		RetentionPeriod:     24 * time.Hour,
	}
	dm.storage = storage.NewInMemoryPriceStorage(storageConfig)

	// 3. Создаем API клиент
	apiClient := bybit.NewBybitClient(dm.config)

	// 4. Создаем PriceFetcher
	dm.priceFetcher = fetcher.NewPriceFetcher(apiClient, dm.storage, dm.eventBus)

	// 5. Создаем CompositeNotificationService через фабрику
	log.Println("📱 Создание CompositeNotificationService через фабрику...")
	notifierFactory := notifier.NewNotifierFactory(dm.eventBus)
	dm.notification = notifierFactory.CreateCompositeNotifier(dm.config)

	if dm.notification == nil {
		return fmt.Errorf("не удалось создать CompositeNotificationService")
	}
	log.Println("✅ CompositeNotificationService создан через фабрику")

	// 6. СОЗДАЕМ TELEGRAM БОТА С АВТОРИЗАЦИЕЙ (если userService создан)
	if dm.config.TelegramEnabled && dm.config.TelegramBotToken != "" {
		log.Println("🤖 Создание Telegram бота с авторизацией (Singleton)...")

		// Если userService создан, создаем бота с авторизацией
		if dm.userService != nil {
			dm.telegramBot = telegram.GetOrCreateBotWithAuth(dm.config, dm.userService)
			log.Println("✅ Telegram бот создан с авторизацией (Singleton)")
		} else {
			// Иначе создаем бота без авторизации
			dm.telegramBot = telegram.GetOrCreateBot(dm.config)
			log.Println("✅ Telegram бот создан без авторизации (Singleton)")
		}

		if dm.telegramBot != nil {
			dm.telegramBot.SetTestMode(testMode)

			// Отправляем приветственное сообщение только если не в тестовом режиме
			if !testMode {
				time.AfterFunc(2*time.Second, func() {
					if err := dm.telegramBot.SendWelcomeMessage(); err != nil {
						logger.Info("⚠️ Не удалось отправить приветственное сообщение: %v", err)
					}
				})
			} else {
				log.Println("🧪 Тестовый режим - приветственное сообщение отключено")
			}
		}
	}

	// 7. Создаем AnalysisEngine через фабрику, передавая уже созданного бота
	log.Println("🔧 Создание AnalysisEngine с передачей marketFetcher...")

	// 🔴 ИСПРАВЛЕНИЕ: Создаем фабрику с priceFetcher
	analysisFactory := engine.NewFactory(dm.priceFetcher)

	// Получаем TelegramNotifier из CompositeNotificationService
	var telegramNotifier *notification.TelegramNotifier // Изменен тип
	if dm.notification != nil {
		for _, notifier := range dm.notification.GetNotifiers() {
			if tn, ok := notifier.(*notification.TelegramNotifier); ok {
				telegramNotifier = tn // Теперь типы совместимы
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

	log.Printf("✅ AnalysisEngine создан с фабрикой")
	log.Printf("   PriceFetcher передан в фабрику: %v", dm.priceFetcher != nil)
	log.Printf("   TelegramNotifier: %v", telegramNotifier != nil)

	// 8. Создаем SignalPipeline
	dm.signalPipeline = pipeline.NewSignalPipeline(dm.eventBus)

	// 9. Регистрируем подписчиков (теперь только основные)
	log.Println("📋 Регистрация базовых подписчиков EventBus...")
	dm.registerBasicSubscribers()

	// 10. Создаем реестр сервисов
	dm.registry = NewServiceRegistry()

	// 11. Создаем менеджер жизненного цикла
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

	// 12. Настраиваем пайплайн
	dm.setupPipeline()

	// 13. Регистрируем сервисы
	if err := dm.registerServices(); err != nil {
		return err
	}

	// 14. Подписываем notification service на события
	dm.subscribeNotificationService()

	return nil
}

// registerBasicSubscribers регистрирует только базовых подписчиков
func (dm *DataManager) registerBasicSubscribers() {
	// Консольный логгер для ошибок и сигналов
	consoleSubscriber := events.NewConsoleLoggerSubscriber()
	dm.eventBus.Subscribe(types.EventSignalDetected, consoleSubscriber)
	dm.eventBus.Subscribe(types.EventPriceUpdated, consoleSubscriber)
	dm.eventBus.Subscribe(types.EventError, consoleSubscriber)

	// Регистрируем telegram.Notifier если Telegram бот доступен
	if dm.telegramBot != nil {
		telegramNotifier := telegram.NewNotifier(dm.config)
		telegramNotifier.SetTelegramBot(dm.telegramBot)

		dm.eventBus.Subscribe(types.EventSignalDetected, telegramNotifier)
		dm.eventBus.Subscribe(types.EventCounterSignalDetected, telegramNotifier)
		dm.eventBus.Subscribe(types.EventCounterNotificationRequest, telegramNotifier)

		log.Println("✅ Telegram Notifier зарегистрирован как подписчик EventBus")
	}

	log.Println("✅ Базовые подписчики зарегистрированы")
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
				// Преобразуем сигнал для нотификации
				if signal, ok := event.Data.(analysis.Signal); ok {
					trendSignal := adapters.AnalysisSignalToTrendSignal(signal)
					return dm.notification.Send(trendSignal)
				}
			}
			return nil
		},
	)

	dm.eventBus.Subscribe(types.EventSignalDetected, notificationSubscriber)
	log.Println("✅ Notification service подписан на события сигналов")
}

// setupPipeline настраивает этапы обработки сигналов
func (dm *DataManager) setupPipeline() {
	// Добавляем этапы валидации и обогащения
	dm.signalPipeline.AddStage(&pipeline.ValidationStage{})
	dm.signalPipeline.AddStage(&pipeline.EnrichmentStage{})
}

// registerServices регистрирует сервисы в реестре
func (dm *DataManager) registerServices() error {
	// Регистрируем все сервисы
	services := map[string]Service{
		"PriceStorage":        dm.newServiceAdapter("PriceStorage", dm.storage),
		"PriceFetcher":        dm.newServiceAdapter("PriceFetcher", dm.priceFetcher),
		"AnalysisEngine":      dm.newServiceAdapter("AnalysisEngine", dm.analysisEngine),
		"SignalPipeline":      dm.newServiceAdapter("SignalPipeline", dm.signalPipeline),
		"NotificationService": dm.newServiceAdapter("NotificationService", dm.notification),
		"EventBus":            dm.newServiceAdapter("EventBus", dm.eventBus),
	}

	// Регистрируем DatabaseService если он создан
	if dm.databaseService != nil {
		services["DatabaseService"] = dm.newServiceAdapter("DatabaseService", dm.databaseService)
	}

	// Регистрируем RedisService если он создан
	if dm.redisService != nil {
		services["RedisService"] = dm.newServiceAdapter("RedisService", dm.redisService)
	}

	if dm.telegramBot != nil {
		services["TelegramBot"] = dm.newServiceAdapter("TelegramBot", dm.telegramBot)
	}

	// ДОБАВИЛИ регистрацию WebhookServer
	if dm.webhookServer != nil {
		services["WebhookServer"] = dm.newServiceAdapter("WebhookServer", dm.webhookServer)
	}
	if dm.userService != nil {
		services["UserService"] = dm.newServiceAdapter("UserService", dm.userService)
	}
	for name, service := range services {
		if err := dm.registry.Register(name, service); err != nil {
			return fmt.Errorf("failed to register service %s: %w", name, err)
		}
	}

	return nil
}

// setupDependencies настраивает зависимости между сервисами
func (dm *DataManager) setupDependencies() {
	// AnalysisEngine зависит от PriceStorage и EventBus
	dm.lifecycle.AddDependency("AnalysisEngine", "PriceStorage")
	dm.lifecycle.AddDependency("AnalysisEngine", "EventBus")

	// SignalPipeline зависит от EventBus
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
}

// startBackgroundTasks запускает фоновые задачи
func (dm *DataManager) startBackgroundTasks() {
	// Задача обновления статистики системы
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

	// Задача автоматической очистки старых данных
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

	// Задача мониторинга здоровья
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

	// Получаем информацию о сервисах
	servicesInfo := dm.registry.GetAllInfo()

	// Получаем статистику хранилища
	storageStats := dm.storage.GetStats()

	// Получаем использование памяти
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Получаем статистику анализа
	var analysisStats interface{}
	if dm.analysisEngine != nil {
		analysisStats = dm.analysisEngine.GetStats()
	}

	// Получаем статистику EventBus
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
		// Публикуем событие в EventBus
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
func (dm *DataManager) GetWebhookServer() *telegram.WebhookServer {
	return dm.webhookServer
}

// GetTelegramBot возвращает Telegram бота
func (dm *DataManager) GetTelegramBot() *telegram.TelegramBot {
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
		return dm.redisService, dm.redisService != nil // НОВОЕ
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

	// Останавливаем фоновые задачи
	close(dm.stopChan)
	dm.wg.Wait()

	// Останавливаем сервисы через LifecycleManager
	errors := dm.lifecycle.StopAll()

	if len(errors) > 0 {
		for service, err := range errors {
			logger.Info("⚠️ Failed to stop %s: %v", service, err)
		}
	}

	// Останавливаем EventBus последним
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

	// Создаем подписчика для Telegram
	telegramSubscriber := events.NewTelegramNotifierSubscriber(dm.telegramBot)
	dm.eventBus.Subscribe(types.EventSignalDetected, telegramSubscriber)

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
		// Хранилище не требует запуска
		sa.state = StateRunning

	case fetcher.PriceFetcher:
		// Запускаем с интервалом из конфигурации
		updateInterval := time.Duration(10) * time.Second // дефолтное значение
		if err := s.Start(updateInterval); err != nil {
			sa.state = StateError
			return err
		}
		sa.state = StateRunning

	case *telegram.WebhookServer:
		if err := s.Start(); err != nil {
			sa.state = StateError
			return err
		}
		sa.state = StateRunning

	case *database.DatabaseService:
		// DatabaseService уже запущен при инициализации
		if s.State() == database.StateRunning {
			sa.state = StateRunning
		} else if s.State() == database.StateError {
			sa.state = StateError
			return fmt.Errorf("database service in error state")
		} else {
			// Пытаемся запустить
			if err := s.Start(); err != nil {
				sa.state = StateError
				return err
			}
			sa.state = StateRunning
		}

	case *redis.RedisService:
		// RedisService уже запущен при инициализации
		if s.State() == redis.StateRunning {
			sa.state = StateRunning
		} else if s.State() == redis.StateError {
			sa.state = StateError
			return fmt.Errorf("Redis service in error state")
		} else {
			// Пытаемся запустить
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
		// NotificationService не требует явного запуска
		sa.state = StateRunning

	case *events.EventBus:
		s.Start()
		sa.state = StateRunning

	case *telegram.TelegramBot:
		// Telegram бот запускается при создании
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

	case *telegram.WebhookServer:
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

	case *telegram.TelegramBot:
		// Telegram бот не требует явной остановки
	}

	sa.state = StateStopped
	return nil
}

func (sa *serviceAdapter) State() ServiceState {
	return sa.state
}

func (sa *serviceAdapter) HealthCheck() bool {
	// Простая проверка здоровья
	if sa.state != StateRunning {
		return false
	}

	switch s := sa.service.(type) {
	case *database.DatabaseService:
		return s.HealthCheck()
	case *redis.RedisService:
		return s.HealthCheck()
	case *engine.AnalysisEngine:
		// Для анализатора считаем, что он здоров если состояние Running
		return true
	case *fetcher.PriceFetcher:
		// Для PriceFetcher считаем, что он здоров если состояние Running
		return true
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

// Добавим метод для проверки инициализации
func (dm *DataManager) IsInitialized() bool {
	return dm.storage != nil && dm.eventBus != nil && dm.analysisEngine != nil
}

// Добавим метод для получения анализаторов
func (dm *DataManager) GetAnalyzers() []string {
	if dm.analysisEngine != nil {
		return dm.analysisEngine.GetAnalyzers()
	}
	return []string{}
}

// Добавим метод для ручного запуска анализа
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
