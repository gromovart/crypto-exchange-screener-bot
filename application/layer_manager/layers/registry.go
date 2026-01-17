// application/services/orchestrator/layers/registry.go
package layers

import (
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"sync"
)

// LayerRegistry реестр слоев
type LayerRegistry struct {
	mu     sync.RWMutex
	layers map[string]Layer
}

// NewLayerRegistry создает новый реестр слоев
func NewLayerRegistry() *LayerRegistry {
	return &LayerRegistry{
		layers: make(map[string]Layer),
	}
}

// Register регистрирует слой
func (lr *LayerRegistry) Register(layer Layer) error {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	name := layer.Name()
	if _, exists := lr.layers[name]; exists {
		return fmt.Errorf("слой уже зарегистрирован: %s", name)
	}

	lr.layers[name] = layer
	logger.Info("✅ Зарегистрирован слой: %s", name)
	return nil
}

// Get возвращает слой по имени
func (lr *LayerRegistry) Get(name string) (Layer, bool) {
	lr.mu.RLock()
	defer lr.mu.RUnlock()

	layer, exists := lr.layers[name]
	return layer, exists
}

// GetAll возвращает все слои
func (lr *LayerRegistry) GetAll() map[string]Layer {
	lr.mu.RLock()
	defer lr.mu.RUnlock()

	result := make(map[string]Layer)
	for k, v := range lr.layers {
		result[k] = v
	}
	return result
}

// Count возвращает количество слоев
func (lr *LayerRegistry) Count() int {
	lr.mu.RLock()
	defer lr.mu.RUnlock()
	return len(lr.layers)
}

// Names возвращает имена всех слоев
func (lr *LayerRegistry) Names() []string {
	lr.mu.RLock()
	defer lr.mu.RUnlock()

	names := make([]string, 0, len(lr.layers))
	for name := range lr.layers {
		names = append(names, name)
	}
	return names
}

// Remove удаляет слой
func (lr *LayerRegistry) Remove(name string) {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	delete(lr.layers, name)
	logger.Info("🗑️  Удален слой: %s", name)
}

// InitializeAll инициализирует все слои
func (lr *LayerRegistry) InitializeAll() map[string]error {
	lr.mu.RLock()
	defer lr.mu.RUnlock()

	errors := make(map[string]error)
	for name, layer := range lr.layers {
		if err := layer.Initialize(); err != nil {
			errors[name] = err
			logger.Warn("⚠️ Ошибка инициализации слоя %s: %v", name, err)
		} else {
			logger.Info("✅ Слой инициализирован: %s", name)
		}
	}
	return errors
}

// StartAll запускает все слои
func (lr *LayerRegistry) StartAll() map[string]error {
	lr.mu.RLock()
	defer lr.mu.RUnlock()

	errors := make(map[string]error)
	for name, layer := range lr.layers {
		if !layer.IsInitialized() {
			errors[name] = fmt.Errorf("слой не инициализирован")
			continue
		}

		if err := layer.Start(); err != nil {
			errors[name] = err
			logger.Warn("⚠️ Ошибка запуска слоя %s: %v", name, err)
		} else {
			logger.Info("🚀 Слой запущен: %s", name)
		}
	}
	return errors
}

// StopAll останавливает все слои
func (lr *LayerRegistry) StopAll() map[string]error {
	lr.mu.RLock()
	defer lr.mu.RUnlock()

	errors := make(map[string]error)
	for name, layer := range lr.layers {
		if err := layer.Stop(); err != nil {
			errors[name] = err
			logger.Warn("⚠️ Ошибка остановки слоя %s: %v", name, err)
		} else {
			logger.Info("🛑 Слой остановлен: %s", name)
		}
	}
	return errors
}

// ResetAll сбрасывает все слои
func (lr *LayerRegistry) ResetAll() map[string]error {
	lr.mu.RLock()
	defer lr.mu.RUnlock()

	errors := make(map[string]error)
	for name, layer := range lr.layers {
		if err := layer.Reset(); err != nil {
			errors[name] = err
			logger.Warn("⚠️ Ошибка сброса слоя %s: %v", name, err)
		} else {
			logger.Info("🔄 Слой сброшен: %s", name)
		}
	}
	return errors
}

// HealthCheck проверяет здоровье всех слоев
func (lr *LayerRegistry) HealthCheck() map[string]bool {
	lr.mu.RLock()
	defer lr.mu.RUnlock()

	health := make(map[string]bool)
	for name, layer := range lr.layers {
		health[name] = layer.HealthCheck()
	}
	return health
}

// GetStatus возвращает статус всех слоев
func (lr *LayerRegistry) GetStatus() map[string]LayerStatus {
	lr.mu.RLock()
	defer lr.mu.RUnlock()

	status := make(map[string]LayerStatus)
	for name, layer := range lr.layers {
		status[name] = layer.GetStatus()
	}
	return status
}

// ValidateDependencies проверяет зависимости всех слоев
func (lr *LayerRegistry) ValidateDependencies() error {
	lr.mu.RLock()
	defer lr.mu.RUnlock()

	// Собираем все слои для проверки зависимостей
	allLayers := make(map[string]Layer)
	for name, layer := range lr.layers {
		allLayers[name] = layer
	}

	// Проверяем зависимости каждого слоя
	for name, layer := range lr.layers {
		if err := layer.SetDependencies(allLayers); err != nil {
			return fmt.Errorf("ошибка зависимостей для слоя %s: %w", name, err)
		}
	}

	return nil
}

// FindComponent ищет компонент во всех слоях
func (lr *LayerRegistry) FindComponent(componentName string) (interface{}, string, bool) {
	lr.mu.RLock()
	defer lr.mu.RUnlock()

	for layerName, layer := range lr.layers {
		if component, exists := layer.GetComponent(componentName); exists {
			return component, layerName, true
		}
	}

	return nil, "", false
}

// Clear очищает реестр
func (lr *LayerRegistry) Clear() {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	lr.layers = make(map[string]Layer)
	logger.Info("🧹 Реестр слоев очищен")
}
