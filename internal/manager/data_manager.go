package manager

import (
	"crypto-exchange-screener-bot/internal/config"
	"crypto-exchange-screener-bot/internal/monitor"
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

	// Хранилище данных
	storage storage.PriceStorage

	// Мониторы
	priceMonitor  *monitor.PriceMonitor
	growthMonitor *monitor.GrowthMonitor

	// Координация
	coordinator  *EventCoordinator
	storageCoord *StorageCoordinator
	lifecycle    *LifecycleManager
	registry     *ServiceRegistry

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
	// 1. Создаем хранилище (отвечает за данные)
	storageConfig := &storage.StorageConfig{
		MaxHistoryPerSymbol: 10000,
		MaxSymbols:          1000,
		CleanupInterval:     5 * time.Minute,
		RetentionPeriod:     24 * time.Hour,
	}

	dm.storage = storage.NewInMemoryPriceStorage(storageConfig)

	// 2. Создаем мониторы (отвечают за получение данных)
	dm.priceMonitor = monitor.NewPriceMonitor(dm.config, dm.storage)
	dm.growthMonitor = monitor.NewGrowthMonitor(dm.config, dm.storage)

	// 3. Создаем координатор событий
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

	dm.coordinator = NewEventCoordinator(coordinatorConfig)

	// 4. Создаем координатор хранилища
	dm.storageCoord = NewStorageCoordinator(dm.storage, dm.coordinator)

	// 5. Создаем реестр сервисов
	dm.registry = NewServiceRegistry()

	// 6. Создаем менеджер жизненного цикла
	dm.lifecycle = NewLifecycleManager(dm.registry, dm.coordinator, coordinatorConfig)

	// 7. Telegram бот
	if dm.config.TelegramEnabled && dm.config.TelegramAPIKey != "" {
		dm.telegramBot = telegram.NewTelegramBot(dm.config)
	}

	// 8. Регистрируем сервисы
	if err := dm.registerServices(); err != nil {
		return err
	}

	return nil
}

// registerServices регистрирует сервисы в реестре
func (dm *DataManager) registerServices() error {
	// Регистрируем все сервисы
	services := map[string]Service{
		"PriceStorage":     dm.newServiceAdapter("PriceStorage", dm.storage),
		"PriceMonitor":     dm.newServiceAdapter("PriceMonitor", dm.priceMonitor),
		"GrowthMonitor":    dm.newServiceAdapter("GrowthMonitor", dm.growthMonitor),
		"EventCoordinator": dm.newServiceAdapter("EventCoordinator", dm.coordinator),
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
	// PriceMonitor зависит от PriceStorage
	dm.lifecycle.AddDependency("PriceMonitor", "PriceStorage")

	// GrowthMonitor зависит от PriceStorage
	dm.lifecycle.AddDependency("GrowthMonitor", "PriceStorage")

	// TelegramBot зависит от GrowthMonitor
	if dm.telegramBot != nil {
		dm.lifecycle.AddDependency("TelegramBot", "GrowthMonitor")
	}
}

// Start запускает все сервисы
func (dm *DataManager) Start() error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	log.Println("🚀 Starting DataManager...")

	// Запускаем сервисы в правильном порядке
	errors := dm.lifecycle.StartAll()

	if len(errors) > 0 {
		for service, err := range errors {
			log.Printf("❌ Failed to start %s: %v", service, err)
		}
		return fmt.Errorf("failed to start some services")
	}

	// Запускаем мониторы
	updateInterval := time.Duration(dm.config.UpdateInterval) * time.Second
	dm.priceMonitor.StartMonitoring(updateInterval)
	dm.growthMonitor.Start()

	log.Println("✅ DataManager started successfully")
	return nil
}

// Stop останавливает все сервисы
func (dm *DataManager) Stop() error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	log.Println("🛑 Stopping DataManager...")

	// Останавливаем фоновые задачи
	close(dm.stopChan)
	dm.wg.Wait()

	// Останавливаем мониторы
	dm.priceMonitor.StopMonitoring()
	dm.growthMonitor.Stop()

	// Останавливаем сервисы
	errors := dm.lifecycle.StopAll()

	if len(errors) > 0 {
		for service, err := range errors {
			log.Printf("⚠️ Failed to stop %s: %v", service, err)
		}
	}

	// Останавливаем менеджер жизненного цикла
	dm.lifecycle.Stop()

	log.Println("✅ DataManager stopped")
	return nil
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

	// Получаем статистику роста
	var growthStats map[string]interface{}
	if dm.growthMonitor != nil {
		growthStats = dm.growthMonitor.GetGrowthStats()
	}

	dm.systemStats = SystemStats{
		Services:      servicesInfo,
		StorageStats:  storageStats,
		Uptime:        time.Since(dm.startTime),
		TotalRequests: 0, // Можно отслеживать в будущем
		MemoryUsageMB: float64(m.Alloc) / 1024 / 1024,
		CPUUsage:      0, // Нужны дополнительные метрики
		ActiveSymbols: storageStats.TotalSymbols,
		GrowthStats:   growthStats,
		LastUpdated:   time.Now(),
	}
}

// checkHealth проверяет здоровье системы
func (dm *DataManager) checkHealth() {
	health := dm.GetHealthStatus()

	if health.Status != "healthy" {
		dm.coordinator.PublishEvent(Event{
			Type:      EventHealthCheck,
			Service:   "DataManager",
			Message:   fmt.Sprintf("System health check failed: %s", health.Status),
			Timestamp: time.Now(),
			Severity:  "warning",
		})

		// Можно добавить автоматическое восстановление
		// dm.attemptRecovery()
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

// GetPriceMonitor возвращает монитор цен
func (dm *DataManager) GetPriceMonitor() *monitor.PriceMonitor {
	return dm.priceMonitor
}

// GetGrowthMonitor возвращает монитор роста
func (dm *DataManager) GetGrowthMonitor() *monitor.GrowthMonitor {
	return dm.growthMonitor
}

// GetTelegramBot возвращает Telegram бота
func (dm *DataManager) GetTelegramBot() *telegram.TelegramBot {
	return dm.telegramBot
}

// GetEventCoordinator возвращает координатор событий
func (dm *DataManager) GetEventCoordinator() *EventCoordinator {
	return dm.coordinator
}

// GetService возвращает сервис по имени
func (dm *DataManager) GetService(name string) (interface{}, bool) {
	switch name {
	case "PriceStorage":
		return dm.storage, true
	case "PriceMonitor":
		return dm.priceMonitor, true
	case "GrowthMonitor":
		return dm.growthMonitor, true
	case "TelegramBot":
		return dm.telegramBot, dm.telegramBot != nil
	case "EventCoordinator":
		return dm.coordinator, true
	default:
		return nil, false
	}
}

// PublishEvent публикует событие
func (dm *DataManager) PublishEvent(event Event) {
	dm.coordinator.PublishEvent(event)
}

// Subscribe подписывает на события
func (dm *DataManager) Subscribe(subscriber DataSubscriber) {
	dm.coordinator.Subscribe(subscriber)
}

// Unsubscribe отписывает от событий
func (dm *DataManager) Unsubscribe(subscriber DataSubscriber) {
	dm.coordinator.Unsubscribe(subscriber)
}

// GetRecentEvents возвращает последние события
func (dm *DataManager) GetRecentEvents(limit int) []Event {
	return dm.coordinator.GetEvents(limit)
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
	dm.coordinator.ClearBuffer()
}

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
	sa.state = StateRunning

	// В зависимости от типа сервиса вызываем соответствующий метод
	switch s := sa.service.(type) {
	case storage.PriceStorage:
		// Хранилище не требует запуска
	case *monitor.PriceMonitor:
		// PriceMonitor запускается отдельно через StartMonitoring
	case *monitor.GrowthMonitor:
		s.Start()
	case *EventCoordinator:
		// Координатор запускается автоматически
	case *telegram.TelegramBot:
		// TelegramBot не требует запуска
	}

	return nil
}

func (sa *serviceAdapter) Stop() error {
	sa.state = StateStopping

	switch s := sa.service.(type) {
	case *monitor.GrowthMonitor:
		s.Stop()
	case *EventCoordinator:
		s.Stop()
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
