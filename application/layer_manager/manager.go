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

	// 1. Получаем слой инфраструктуры
	infraLayer, exists := lm.layerRegistry.Get("InfrastructureLayer")
	if !exists {
		return fmt.Errorf("InfrastructureLayer не найден")
	}

	// 2. Запускаем InfrastructureLayer если еще не запущен
	if !infraLayer.IsRunning() {
		logger.Info("🏗️  Запуск InfrastructureLayer...")
		if err := infraLayer.Start(); err != nil {
			return fmt.Errorf("[manager.go]не удалось запустить InfrastructureLayer: %w", err)
		}
	} else {
		logger.Info("✅ InfrastructureLayer уже запущен, пропускаем")
	}

	// 3. Ждем готовности InfrastructureFactory (для уже запущенных слоев фабрика должна быть готова)
	if infraLayer.IsRunning() {
		logger.Info("✅ InfrastructureFactory уже готова, пропускаем ожидание")
	} else {
		logger.Info("⏳ Ожидание готовности InfrastructureFactory...")
		if !lm.waitForInfrastructureReady(30 * time.Second) {
			return fmt.Errorf("таймаут ожидания готовности InfrastructureFactory")
		}
		logger.Info("✅ InfrastructureFactory готова")
	}

	// 4. Запускаем остальные слои через реестр (с учетом зависимостей)
	logger.Info("🚦 Запуск остальных слоев...")
	errors := lm.layerRegistry.StartAll()

	// Проверяем ошибки запуска
	if len(errors) > 0 {
		// Логируем ошибки, но не останавливаемся
		for layerName, err := range errors {
			logger.Warn("⚠️ Ошибка запуска слоя %s: %v", layerName, err)
		}
	}

	// 5. Проверяем здоровье всех слоев
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
				if factory != nil && factory.IsReady() && factory.IsRunning() {
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

// GetDetailedStatus возвращает детализированный статус системы
func (lm *LayerManager) GetDetailedStatus() map[string]interface{} {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	status := map[string]interface{}{
		"initialized": lm.initialized,
		"running":     lm.running,
		"uptime":      time.Since(lm.startTime).String(),
		"start_time":  lm.startTime.Format(time.RFC3339),
		"environment": lm.config.Environment,
		"version":     lm.config.Version,
	}

	if lm.layerRegistry == nil {
		return status
	}

	// Получаем статус всех слоев
	layersStatus := make(map[string]interface{})
	healthCheck := make(map[string]bool)

	for name, layer := range lm.layerRegistry.GetAll() {
		layerStatus := layer.GetStatus()
		layerInfo := map[string]interface{}{
			"state":        layerStatus.State,
			"is_healthy":   layerStatus.IsHealthy,
			"initialized":  layerStatus.Initialized,
			"running":      layerStatus.Running,
			"uptime":       layerStatus.Uptime.String(),
			"last_error":   layerStatus.LastError,
			"dependencies": layerStatus.Dependencies,
			"components":   layerStatus.Components,
		}

		// Добавляем дополнительную информацию для DeliveryLayer
		if name == "DeliveryLayer" {
			// Получаем TelegramDeliveryPackage если он есть
			if component, exists := layer.GetComponent("TelegramDeliveryPackage"); exists {
				// Пробуем получить статус через интерфейс
				if pkg, ok := component.(interface{ GetHealthStatus() map[string]interface{} }); ok {
					if pkgStatus := pkg.GetHealthStatus(); pkgStatus != nil {
						layerInfo["telegram_package"] = pkgStatus

						// Извлекаем информацию о транспорте
						if transportType, ok := pkgStatus["transport_type"].(string); ok {
							layerInfo["telegram_transport"] = transportType
						}
						if transportStatus, ok := pkgStatus["transport_status"].(string); ok {
							layerInfo["telegram_transport_status"] = transportStatus
						}
						if telegramMode, ok := pkgStatus["telegram_mode"].(string); ok {
							layerInfo["telegram_mode"] = telegramMode
						}
					}
				}
			}

			// Пробуем получить транспорт напрямую
			if component, exists := layer.GetComponent("TelegramBot"); exists {
				if bot, ok := component.(interface{ IsPolling() bool }); ok {
					layerInfo["telegram_polling"] = bot.IsPolling()
				}
			}
		}

		// Добавляем информацию для CoreLayer
		if name == "CoreLayer" {
			// Пробуем получить AnalysisEngine
			if component, exists := layer.GetComponent("AnalysisEngine"); exists {
				if engine, ok := component.(interface{ IsRunning() bool }); ok {
					layerInfo["analysis_engine_running"] = engine.IsRunning()
				}
			}

			// Пробуем получить BybitPriceFetcher
			if component, exists := layer.GetComponent("BybitPriceFetcher"); exists {
				if fetcher, ok := component.(interface{ IsRunning() bool }); ok {
					layerInfo["price_fetcher_running"] = fetcher.IsRunning()
				}
			}
		}

		layersStatus[name] = layerInfo
		healthCheck[name] = layerStatus.IsHealthy
	}

	status["layers"] = layersStatus
	status["health"] = healthCheck

	// Подсчитываем статистику
	totalLayers := len(layersStatus)
	healthyLayers := 0
	runningLayers := 0

	for _, layerInfo := range layersStatus {
		if info, ok := layerInfo.(map[string]interface{}); ok {
			if isHealthy, ok := info["is_healthy"].(bool); ok && isHealthy {
				healthyLayers++
			}
			if isRunning, ok := info["running"].(bool); ok && isRunning {
				runningLayers++
			}
		}
	}

	status["total_layers"] = totalLayers
	status["healthy_layers"] = healthyLayers
	status["running_layers"] = runningLayers
	status["health_percentage"] = float64(healthyLayers) / float64(totalLayers) * 100

	return status
}

// GetTransportInfo возвращает информацию о транспорте Telegram
func (lm *LayerManager) GetTransportInfo() map[string]interface{} {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	info := map[string]interface{}{
		"telegram_enabled": false,
		"mode":             "unknown",
		"status":           "not_initialized",
	}

	if lm.layerRegistry == nil {
		return info
	}

	// Получаем DeliveryLayer
	deliveryLayer, exists := lm.layerRegistry.Get("DeliveryLayer")
	if !exists {
		return info
	}

	// Получаем TelegramDeliveryPackage
	component, exists := deliveryLayer.GetComponent("TelegramDeliveryPackage")
	if !exists {
		return info
	}

	// Пробуем получить статус через интерфейс
	if pkg, ok := component.(interface{ GetHealthStatus() map[string]interface{} }); ok {
		if pkgStatus := pkg.GetHealthStatus(); pkgStatus != nil {
			info["telegram_enabled"] = true

			// Извлекаем информацию о транспорте
			if transportType, ok := pkgStatus["transport_type"].(string); ok {
				info["mode"] = transportType
			}
			if transportStatus, ok := pkgStatus["transport_status"].(string); ok {
				info["status"] = transportStatus
			}
			if transportName, ok := pkgStatus["transport_name"].(string); ok {
				info["transport_name"] = transportName
			}
			if telegramMode, ok := pkgStatus["telegram_mode"].(string); ok {
				info["config_mode"] = telegramMode
			}
			if botStatus, ok := pkgStatus["bot_status"].(string); ok {
				info["bot_status"] = botStatus
			}

			// Добавляем дополнительную информацию
			info["services_count"] = pkgStatus["services_count"]
			info["controllers_count"] = pkgStatus["controllers_count"]
			info["initialized"] = pkgStatus["initialized"]
			info["is_running"] = pkgStatus["is_running"]
		}
	}

	return info
}

// GetTelegramStatus возвращает статус Telegram компонентов
func (lm *LayerManager) GetTelegramStatus() map[string]interface{} {
	transportInfo := lm.GetTransportInfo()

	status := map[string]interface{}{
		"transport": transportInfo,
		"config": map[string]interface{}{
			"telegram_enabled": lm.config.Telegram.Enabled,
			"telegram_mode":    lm.config.TelegramMode,
			"webhook_url":      lm.config.GetWebhookURL(),
			"polling_timeout":  lm.config.Polling.Timeout,
			"polling_retry":    lm.config.Polling.RetryInterval,
		},
	}

	// Добавляем информацию о вебхуке если используется webhook режим
	if lm.config.IsWebhookMode() {
		status["webhook"] = map[string]interface{}{
			"domain":       lm.config.Webhook.Domain,
			"port":         lm.config.Webhook.Port,
			"path":         lm.config.Webhook.Path,
			"use_tls":      lm.config.Webhook.UseTLS,
			"has_tls_cert": lm.config.Webhook.TLSCertPath != "",
			"has_tls_key":  lm.config.Webhook.TLSKeyPath != "",
		}
	}

	return status
}
