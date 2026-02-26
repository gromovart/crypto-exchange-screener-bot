// application/services/orchestrator/layers/registry.go
package layers

import (
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"sync"
	"time"
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

// InitializeAll инициализирует все слои с учетом зависимостей
func (lr *LayerRegistry) InitializeAll() map[string]error {
	lr.mu.RLock()
	allLayers := lr.getAllLayersCopy()
	lr.mu.RUnlock()

	// Сортируем слои по зависимостям (сначала без зависимостей)
	sortedLayers := lr.sortLayersByDependencies(allLayers)

	errors := make(map[string]error)

	// Инициализируем слои в правильном порядке
	for _, layer := range sortedLayers {
		name := layer.Name()

		// Проверяем зависимости перед инициализацией
		deps := layer.GetDependencies()
		if len(deps) > 0 {
			ready, notReady := layer.AreDependenciesReady(allLayers)
			if !ready {
				errors[name] = fmt.Errorf("зависимости не готовы: %v", notReady)
				logger.Warn("⚠️ Пропускаем инициализацию слоя %s: зависимости не готовы", name)
				continue
			}
		}

		if err := layer.Initialize(); err != nil {
			errors[name] = err
			logger.Warn("⚠️ Ошибка инициализации слоя %s: %v", name, err)
		} else {
			logger.Info("✅ Слой инициализирован: %s", name)
		}
	}
	return errors
}

// StartAll запускает все слои с учетом зависимостей
func (lr *LayerRegistry) StartAll() map[string]error {
	lr.mu.RLock()
	allLayers := lr.getAllLayersCopy()
	lr.mu.RUnlock()

	// Сортируем слои по зависимостям
	sortedLayers := lr.sortLayersByDependencies(allLayers)

	errors := make(map[string]error)

	// Запускаем слои в правильном порядке
	for _, layer := range sortedLayers {
		name := layer.Name()

		// Пропускаем уже запущенные слои (например, InfrastructureLayer
		// запускается явно в LayerManager.Start() до вызова StartAll())
		if layer.IsRunning() {
			logger.Info("✅ Слой %s уже запущен, пропускаем", name)
			continue
		}

		// Проверяем что слой инициализирован
		if !layer.IsInitialized() {
			errors[name] = fmt.Errorf("слой не инициализирован")
			logger.Warn("⚠️ Пропускаем запуск слоя %s: не инициализирован", name)
			continue
		}

		// Проверяем зависимости
		deps := layer.GetDependencies()
		if len(deps) > 0 {
			// Ожидаем готовности зависимостей
			timeout := 30 * time.Second
			if err := layer.WaitForDependencies(allLayers, timeout); err != nil {
				errors[name] = err
				logger.Warn("⚠️ Ошибка ожидания зависимостей для слоя %s: %v", name, err)
				continue
			}
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
	allLayers := lr.getAllLayersCopy()
	lr.mu.RUnlock()

	// Проверяем зависимости каждого слоя
	for name, layer := range allLayers {
		if err := layer.SetDependencies(allLayers); err != nil {
			return fmt.Errorf("ошибка зависимостей для слоя %s: %w", name, err)
		}
	}

	// Проверяем циклические зависимости
	if err := lr.checkForCyclicDependencies(allLayers); err != nil {
		return fmt.Errorf("обнаружены циклические зависимости: %w", err)
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

// sortLayersByDependencies сортирует слои по зависимостям (топологическая сортировка)
func (lr *LayerRegistry) sortLayersByDependencies(layers map[string]Layer) []Layer {
	// Создаем граф зависимостей
	graph := make(map[string][]string)
	inDegree := make(map[string]int)
	layerMap := make(map[string]Layer)

	for name, layer := range layers {
		layerMap[name] = layer
		graph[name] = layer.GetDependencies()
		inDegree[name] = 0
	}

	// Вычисляем входящие степени
	for _, deps := range graph {
		for _, dep := range deps {
			inDegree[dep]++
		}
	}

	// Алгоритм Кана (топологическая сортировка)
	var queue []string
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}

	var sorted []Layer
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]

		sorted = append(sorted, layerMap[name])

		for _, dep := range graph[name] {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
	}

	// Если остались вершины с ненулевой степенью - есть цикл
	for name, degree := range inDegree {
		if degree > 0 {
			logger.Warn("⚠️ Обнаружен возможный цикл в зависимостях для слоя: %s", name)
		}
	}

	return sorted
}

// checkForCyclicDependencies проверяет наличие циклических зависимостей
func (lr *LayerRegistry) checkForCyclicDependencies(layers map[string]Layer) error {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var dfs func(string) bool
	dfs = func(name string) bool {
		if recStack[name] {
			return true // Найден цикл
		}
		if visited[name] {
			return false
		}

		visited[name] = true
		recStack[name] = true

		layer, exists := layers[name]
		if !exists {
			recStack[name] = false
			return false
		}

		for _, dep := range layer.GetDependencies() {
			if dfs(dep) {
				return true
			}
		}

		recStack[name] = false
		return false
	}

	for name := range layers {
		if dfs(name) {
			return fmt.Errorf("циклическая зависимость обнаружена для слоя: %s", name)
		}
	}

	return nil
}

// getAllLayersCopy возвращает копию всех слоев
func (lr *LayerRegistry) getAllLayersCopy() map[string]Layer {
	result := make(map[string]Layer)
	for k, v := range lr.layers {
		result[k] = v
	}
	return result
}

// GetStartOrder возвращает порядок запуска слоев
func (lr *LayerRegistry) GetStartOrder() []string {
	lr.mu.RLock()
	allLayers := lr.getAllLayersCopy()
	lr.mu.RUnlock()

	sortedLayers := lr.sortLayersByDependencies(allLayers)

	order := make([]string, 0, len(sortedLayers))
	for _, layer := range sortedLayers {
		order = append(order, layer.Name())
	}

	return order
}

// WaitForLayer ожидает готовности конкретного слоя
func (lr *LayerRegistry) WaitForLayer(layerName string, timeout time.Duration) error {
	lr.mu.RLock()
	layer, exists := lr.layers[layerName]
	lr.mu.RUnlock()

	if !exists {
		return fmt.Errorf("слой не найден: %s", layerName)
	}

	startTime := time.Now()
	checkInterval := 100 * time.Millisecond

	for {
		if layer.IsInitialized() && layer.IsRunning() {
			return nil
		}

		if time.Since(startTime) > timeout {
			return fmt.Errorf("таймаут ожидания слоя %s (%v)", layerName, timeout)
		}

		time.Sleep(checkInterval)
	}
}
