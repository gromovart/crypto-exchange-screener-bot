// application/services/orchestrator/methods.go
package orchestrator

import (
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/core/domain/signals/engine"
	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/in_memory_storage"
	events "crypto-exchange-screener-bot/internal/infrastructure/transport/event_bus"
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"time"
)

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

	logger.Info("🛑 Остановка DataManager...")
	close(dm.stopChan)
	dm.wg.Wait()

	errors := dm.lifecycle.StopAll()
	if len(errors) > 0 {
		for service, err := range errors {
			logger.Info("⚠️ Не удалось остановить %s: %v", service, err)
		}
	}

	if dm.eventBus != nil {
		dm.eventBus.Stop()
	}

	logger.Info("✅ DataManager остановлен")
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
		return nil, fmt.Errorf("анализатор не инициализирован")
	}
	return dm.analysisEngine.AnalyzeAll()
}

// GetAnalysisResults возвращает результаты анализа для символа
func (dm *DataManager) GetAnalysisResults(symbol string, periods []time.Duration) (*analysis.AnalysisResult, error) {
	if dm.analysisEngine == nil {
		return nil, fmt.Errorf("анализатор не инициализирован")
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
	logger.Info("✅ Консольный подписчик добавлен")
}

// AddTelegramSubscriber добавляет подписчика Telegram
func (dm *DataManager) AddTelegramSubscriber() error {
	if dm.telegramBot == nil {
		return fmt.Errorf("telegram бот не инициализирован")
	}
	// Нужно переписать под новую архитектуру
	logger.Info("⚠️ AddTelegramSubscriber нуждается в переписывании для новой архитектуры")
	return nil
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
