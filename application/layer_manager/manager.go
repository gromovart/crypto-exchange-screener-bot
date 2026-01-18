// application/layer_manager/manager.go
package layer_manager

import (
	"crypto-exchange-screener-bot/application/layer_manager/layers"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"sync"
	"time"
)

// LayerManager менеджер слоев
type LayerManager struct {
	config        *config.Config
	layerRegistry *layers.LayerRegistry

	mu       sync.RWMutex
	stopChan chan struct{}
	wg       sync.WaitGroup

	startTime   time.Time
	initialized bool
	running     bool
}

// NewLayerManager создает новый менеджер слоев
func NewLayerManager(cfg *config.Config) *LayerManager {
	return &LayerManager{
		config:    cfg,
		stopChan:  make(chan struct{}),
		startTime: time.Now(),
	}
}

// Initialize инициализирует менеджер и создает слои
func (lm *LayerManager) Initialize() error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	if lm.initialized {
		return fmt.Errorf("LayerManager уже инициализирован")
	}

	logger.Info("🏗️  Инициализация LayerManager...")
	logger.Debug("Конфигурация: TelegramEnabled=%v, TestMode=%v",
		lm.config.TelegramEnabled, lm.config.MonitoringTestMode)

	// Создаем фабрику слоев
	logger.Debug("Создание LayerFactory...")
	factory := NewLayerFactory(lm.config)

	// Создаем слои
	logger.Debug("Создание слоев через фабрику...")
	layerRegistry, err := factory.CreateLayers()
	if err != nil {
		logger.Error("❌ Не удалось создать слои: %v", err)
		return fmt.Errorf("не удалось создать слои: %w", err)
	}

	lm.layerRegistry = layerRegistry
	lm.initialized = true

	logger.Info("✅ LayerManager инициализирован")
	logger.Debug("Зарегистрировано слоев: %d", layerRegistry.Count())

	// Логируем имена слоев
	layerNames := layerRegistry.Names()
	logger.Debug("Слои: %v", layerNames)

	return nil
}

// Start запускает все слои
func (lm *LayerManager) Start() error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	if lm.running {
		return fmt.Errorf("LayerManager уже запущен")
	}

	logger.Info("🚀 Запуск LayerManager и всех слоев...")

	// 1. Запускаем слой инфраструктуры первым
	infraLayer, exists := lm.layerRegistry.Get("InfrastructureLayer")
	if !exists {
		return fmt.Errorf("InfrastructureLayer не найден")
	}

	logger.Info("🏗️  Запуск InfrastructureLayer...")
	if err := infraLayer.Start(); err != nil {
		return fmt.Errorf("[manager.go]не удалось запустить InfrastructureLayer: %w", err)
	}

	// 2. Ждем готовности InfrastructureFactory
	logger.Info("⏳ Ожидание готовности InfrastructureFactory...")
	if !lm.waitForInfrastructureReady(30 * time.Second) {
		return fmt.Errorf("таймаут ожидания готовности InfrastructureFactory")
	}
	logger.Info("✅ InfrastructureFactory готова")

	// 3. Запускаем остальные слои через реестр (с учетом зависимостей)
	logger.Info("🚦 Запуск остальных слоев...")
	errors := lm.layerRegistry.StartAll()

	// Проверяем ошибки запуска
	if len(errors) > 0 {
		// Логируем ошибки, но не останавливаемся
		for layerName, err := range errors {
			logger.Warn("⚠️ Ошибка запуска слоя %s: %v", layerName, err)
		}
	}

	// 4. Проверяем здоровье всех слоев
	health := lm.layerRegistry.HealthCheck()
	healthyCount := 0
	for layerName, isHealthy := range health {
		if isHealthy {
			healthyCount++
		} else {
			logger.Warn("⚠️ Слой %s не здоров", layerName)
		}
	}

	logger.Info("📊 Статус слоев: %d/%d здоровы", healthyCount, len(health))

	lm.running = true
	lm.startTime = time.Now()
	logger.Info("✅ LayerManager запущен, все слои запущены")
	return nil
}

// waitForInfrastructureReady ожидает готовности InfrastructureFactory
func (lm *LayerManager) waitForInfrastructureReady(timeout time.Duration) bool {
	infraLayer, exists := lm.layerRegistry.Get("InfrastructureLayer")
	if !exists {
		return false
	}

	startTime := time.Now()
	checkInterval := 500 * time.Millisecond

	for {
		// Проверяем слой инфраструктуры
		if infraLayer.HealthCheck() {
			// Получаем фабрику инфраструктуры
			if infraInfra, ok := infraLayer.(*layers.InfrastructureLayer); ok {
				factory := infraInfra.GetInfrastructureFactory()
				if factory != nil && factory.IsReady() {
					return true
				}
			}
		}

		// Проверяем таймаут
		if time.Since(startTime) > timeout {
			logger.Warn("⏰ Таймаут ожидания готовности InfrastructureFactory")
			return false
		}

		// Ждем перед следующей проверкой
		time.Sleep(checkInterval)
	}
}

// Stop останавливает все слои
func (lm *LayerManager) Stop() error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	if !lm.running {
		return nil
	}

	logger.Info("🛑 Остановка LayerManager и всех слоев...")

	// Останавливаем фоновые задачи
	close(lm.stopChan)
	lm.wg.Wait()

	// Останавливаем слои
	errors := lm.layerRegistry.StopAll()
	if len(errors) > 0 {
		for layerName, err := range errors {
			logger.Warn("⚠️ Ошибка остановки слоя %s: %v", layerName, err)
		}
	}

	lm.running = false
	logger.Info("✅ LayerManager и все слои остановлены")
	return nil
}

// GetLayerRegistry возвращает реестр слоев
func (lm *LayerManager) GetLayerRegistry() *layers.LayerRegistry {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	return lm.layerRegistry
}

// GetComponent получает компонент из любого слоя
func (lm *LayerManager) GetComponent(name string) (interface{}, bool) {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	if lm.layerRegistry == nil {
		return nil, false
	}

	// FindComponent возвращает 3 значения, берем только первые два
	component, _, found := lm.layerRegistry.FindComponent(name)
	return component, found
}

// GetComponentWithLayer получает компонент и имя слоя
func (lm *LayerManager) GetComponentWithLayer(name string) (interface{}, string, bool) {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	if lm.layerRegistry == nil {
		return nil, "", false
	}

	return lm.layerRegistry.FindComponent(name)
}

// GetHealthStatus возвращает статус здоровья
func (lm *LayerManager) GetHealthStatus() map[string]interface{} {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	status := map[string]interface{}{
		"initialized": lm.initialized,
		"running":     lm.running,
		"uptime":      time.Since(lm.startTime).String(),
	}

	if lm.layerRegistry != nil {
		status["layers"] = lm.layerRegistry.GetStatus()
		status["health"] = lm.layerRegistry.HealthCheck()
	}

	return status
}

// startBackgroundTasks запускает фоновые задачи
func (lm *LayerManager) startBackgroundTasks() {
	logger.Info("🔄 Запуск фоновых задач LayerManager...")

	// Мониторинг здоровья
	lm.wg.Add(1)
	go func() {
		defer lm.wg.Done()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				lm.checkHealth()
			case <-lm.stopChan:
				return
			}
		}
	}()
}

// checkHealth проверяет здоровье системы
func (lm *LayerManager) checkHealth() {
	if lm.layerRegistry == nil {
		return
	}

	health := lm.layerRegistry.HealthCheck()
	unhealthy := []string{}

	for layerName, isHealthy := range health {
		if !isHealthy {
			unhealthy = append(unhealthy, layerName)
		}
	}

	if len(unhealthy) > 0 {
		logger.Warn("⚠️ Не здоровые слои: %v", unhealthy)
	}
}
