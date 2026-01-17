// application/layer_manager/background.go
package layer_manager

import (
	"crypto-exchange-screener-bot/application/layer_manager/layers"
	"crypto-exchange-screener-bot/pkg/logger"
	"runtime"
	"sync"
	"time"
)

// BackgroundManager управляет фоновыми задачами LayerManager
type BackgroundManager struct {
	layerRegistry *layers.LayerRegistry
	stopChan      chan struct{}
	wg            sync.WaitGroup
	mu            sync.RWMutex
}

// NewBackgroundManager создает менеджер фоновых задач
func NewBackgroundManager(layerRegistry *layers.LayerRegistry) *BackgroundManager {
	return &BackgroundManager{
		layerRegistry: layerRegistry,
		stopChan:      make(chan struct{}),
	}
}

// Start запускает фоновые задачи
func (bm *BackgroundManager) Start() {
	logger.Info("🔄 Запуск фоновых задач для LayerManager...")

	// Мониторинг здоровья слоев
	bm.wg.Add(1)
	go func() {
		defer bm.wg.Done()
		bm.healthMonitoringLoop()
	}()

	// Обновление статистики
	bm.wg.Add(1)
	go func() {
		defer bm.wg.Done()
		bm.statsUpdateLoop()
	}()

	logger.Info("✅ Фоновые задачи запущены")
}

// Stop останавливает фоновые задачи
func (bm *BackgroundManager) Stop() {
	logger.Info("🛑 Остановка фоновых задач...")
	close(bm.stopChan)
	bm.wg.Wait()
	logger.Info("✅ Фоновые задачи остановлены")
}

// healthMonitoringLoop цикл мониторинга здоровья
func (bm *BackgroundManager) healthMonitoringLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			bm.checkLayerHealth()
		case <-bm.stopChan:
			return
		}
	}
}

// statsUpdateLoop цикл обновления статистики
func (bm *BackgroundManager) statsUpdateLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			bm.updateSystemStats()
		case <-bm.stopChan:
			return
		}
	}
}

// checkLayerHealth проверяет здоровье слоев
func (bm *BackgroundManager) checkLayerHealth() {
	if bm.layerRegistry == nil {
		return
	}

	health := bm.layerRegistry.HealthCheck()
	unhealthyLayers := []string{}

	for layerName, isHealthy := range health {
		if !isHealthy {
			unhealthyLayers = append(unhealthyLayers, layerName)
		}
	}

	if len(unhealthyLayers) > 0 {
		logger.Warn("⚠️ Не здоровые слои: %v", unhealthyLayers)
	}
}

// updateSystemStats обновляет статистику системы
func (bm *BackgroundManager) updateSystemStats() {
	if bm.layerRegistry == nil {
		return
	}

	// Получаем статус всех слоев
	layerStatus := bm.layerRegistry.GetStatus()

	// Статистика памяти
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Собираем статистику
	totalComponents := 0
	healthyLayers := 0
	totalLayers := len(layerStatus)

	for _, status := range layerStatus {
		totalComponents += len(status.Components)
		if status.IsHealthy {
			healthyLayers++
		}
	}

	logger.Debug("📊 Статистика системы: слои=%d/%d здоровы, компоненты=%d, память=%.2f MB",
		healthyLayers, totalLayers, totalComponents, float64(m.Alloc)/1024/1024)
}
