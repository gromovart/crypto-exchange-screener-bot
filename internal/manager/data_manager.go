// internal/manager/data_manager.go (исправленная версия)
package manager

import (
	"crypto-exchange-screener-bot/internal/adapters"
	"crypto-exchange-screener-bot/internal/analysis"
	"crypto-exchange-screener-bot/internal/analysis/engine"
	"crypto-exchange-screener-bot/internal/api/bybit"
	"crypto-exchange-screener-bot/internal/config"
	"crypto-exchange-screener-bot/internal/events"
	"crypto-exchange-screener-bot/internal/fetcher"
	"crypto-exchange-screener-bot/internal/notifier"
	"crypto-exchange-screener-bot/internal/pipeline"
	"crypto-exchange-screener-bot/internal/storage"
	"crypto-exchange-screener-bot/internal/telegram"
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
	telegramBot *telegram.TelegramBot

	// Управление
	mu       sync.RWMutex
	stopChan chan struct{}
	wg       sync.WaitGroup

	// Статистика
	startTime   time.Time
	systemStats SystemStats
}

// NewDataManager создает новый менеджер данных
func NewDataManager(cfg *config.Config) (*DataManager, error) {
	dm := &DataManager{
		config:    cfg,
		stopChan:  make(chan struct{}),
		startTime: time.Now(),
		systemStats: SystemStats{
			Services:    make(map[string]ServiceInfo),
			LastUpdated: time.Now(),
		},
	}

	// Инициализируем компоненты
	if err := dm.initializeComponents(); err != nil {
		return nil, err
	}

	// Настраиваем зависимости
	dm.setupDependencies()

	// Запускаем фоновые задачи
	dm.startBackgroundTasks()

	return dm, nil
}

// initializeComponents инициализирует все компоненты
func (dm *DataManager) initializeComponents() error {
	fmt.Printf("🔍 DataManager: RateLimitDelay = %v\n", dm.config.RateLimitDelay)

	// Если RateLimitDelay > 0, то RateLimitingMiddleware добавляется
	if dm.config.RateLimitDelay > 0 {
		fmt.Println("⚠️  RateLimitingMiddleware активен для EventPriceUpdated")
		fmt.Printf("   Лимит: %v между событиями\n", dm.config.RateLimitDelay)
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

	// 5. Создаем AnalysisEngine через фабрику
	analysisFactory := &engine.Factory{}
	dm.analysisEngine = analysisFactory.NewAnalysisEngineFromConfig(
		dm.storage,
		dm.eventBus,
		dm.config,
	)

	// 6. Создаем SignalPipeline
	dm.signalPipeline = pipeline.NewSignalPipeline(dm.eventBus)

	// 7. Создаем CompositeNotificationService
	dm.notification = notifier.NewCompositeNotificationService()

	// 8. Создаем Telegram бота если включен
	if dm.config.TelegramEnabled && dm.config.TelegramAPIKey != "" {
		var err error
		dm.telegramBot = telegram.NewTelegramBot(dm.config)
		if err != nil {
			log.Printf("⚠️ Не удалось создать Telegram бота: %v", err)
		}
	}

	// 9. Создаем реестр сервисов
	dm.registry = NewServiceRegistry()

	// 10. Создаем менеджер жизненного цикла
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

	// 11. Настраиваем нотификаторы
	dm.setupNotifiers()

	// 12. Регистрируем сервисы
	if err := dm.registerServices(); err != nil {
		return err
	}

	// 13. Настраиваем пайплайн
	dm.setupPipeline()

	return nil
}

// setupNotifiers настраивает нотификаторы
func (dm *DataManager) setupNotifiers() {
	if dm.notification == nil {
		return
	}

	// Добавляем консольный нотификатор
	consoleNotifier := notifier.NewConsoleNotifier(dm.config.MessageFormat == "compact")
	dm.notification.AddNotifier(consoleNotifier)

	// Добавляем Telegram нотификатор если включен
	if dm.config.TelegramEnabled && dm.config.TelegramAPIKey != "" && dm.telegramBot != nil {
		telegramNotifier := notifier.NewTelegramNotifier(dm.config)
		if telegramNotifier != nil {
			dm.notification.AddNotifier(telegramNotifier)
		}
	}

	// Подписываем CompositeNotificationService на события сигналов
	notificationSubscriber := events.NewBaseSubscriber(
		"notification_service",
		[]events.EventType{events.EventSignalDetected},
		func(event events.Event) error {
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

	dm.eventBus.Subscribe(events.EventSignalDetected, notificationSubscriber)

	log.Printf("✅ Нотификаторы настроены")
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

	if dm.telegramBot != nil {
		services["TelegramBot"] = dm.newServiceAdapter("TelegramBot", dm.telegramBot)
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
					log.Printf("⚠️ Failed to cleanup old data: %v", err)
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
		dm.eventBus.Publish(events.Event{
			Type:   events.EventError,
			Source: "DataManager",
			Data: map[string]interface{}{
				"status":  health.Status,
				"message": "System health check failed",
			},
		})

		log.Printf("⚠️ System health check failed: %s", health.Status)
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

// GetTelegramBot возвращает Telegram бота
func (dm *DataManager) GetTelegramBot() *telegram.TelegramBot {
	return dm.telegramBot
}

// GetPriceFetcher возвращает PriceFetcher
func (dm *DataManager) GetPriceFetcher() fetcher.PriceFetcher {
	return dm.priceFetcher
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
	default:
		return nil, false
	}
}

// PublishEvent публикует событие
func (dm *DataManager) PublishEvent(event events.Event) {
	dm.eventBus.Publish(event)
}

// Subscribe подписывает на события
func (dm *DataManager) Subscribe(eventType events.EventType, subscriber events.Subscriber) {
	dm.eventBus.Subscribe(eventType, subscriber)
}

// Unsubscribe отписывает от событий
func (dm *DataManager) Unsubscribe(eventType events.EventType, subscriber events.Subscriber) {
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
			log.Printf("⚠️ Failed to stop %s: %v", service, err)
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
	if dm.analysisEngine == nil {
		return []string{}
	}
	return dm.analysisEngine.GetAnalyzers()
}

// AddConsoleSubscriber добавляет подписчика для вывода в консоль
func (dm *DataManager) AddConsoleSubscriber() {
	consoleSubscriber := events.NewConsoleLoggerSubscriber()
	dm.eventBus.Subscribe(events.EventSignalDetected, consoleSubscriber)
	dm.eventBus.Subscribe(events.EventPriceUpdated, consoleSubscriber)
	dm.eventBus.Subscribe(events.EventError, consoleSubscriber)
}

// AddTelegramSubscriber добавляет подписчика Telegram
func (dm *DataManager) AddTelegramSubscriber() error {
	if dm.telegramBot == nil {
		return fmt.Errorf("telegram bot not initialized")
	}

	// Создаем подписчика для Telegram
	telegramSubscriber := events.NewTelegramNotifierSubscriber(dm.telegramBot)
	dm.eventBus.Subscribe(events.EventSignalDetected, telegramSubscriber)

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
	return sa.state == StateRunning
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
				log.Printf("Ошибка при ручном анализе: %v", err)
			} else {
				log.Printf("Ручной анализ завершен: %d символов обработано", len(results))
			}
		}()
	}
}
