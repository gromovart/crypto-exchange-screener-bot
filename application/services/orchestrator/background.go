// application/services/orchestrator/background.go
package orchestrator

import (
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
	"runtime"
	"time"
)

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
					logger.Info("⚠️ Не удалось очистить старые данные: %v", err)
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
				"message": "Проверка здоровья системы не пройдена",
			},
		})
		logger.Info("⚠️ Проверка здоровья системы не пройдена: %s", health.Status)
	}
}
