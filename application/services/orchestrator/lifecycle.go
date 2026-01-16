// application/services/orchestrator/lifecycle.go
package orchestrator

import (
	events "crypto-exchange-screener-bot/internal/infrastructure/transport/event_bus"
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
	"sync"
	"time"
)

// LifecycleManager управляет жизненным циклом сервисов
type LifecycleManager struct {
	mu              sync.RWMutex
	registry        *ServiceRegistry
	eventBus        *events.EventBus
	dependencyGraph *DependencyGraph
	restartAttempts map[string]int
	config          CoordinatorConfig
	startTime       time.Time
	healthTicker    *time.Ticker
	stopChan        chan struct{}
}

// NewLifecycleManager создает менеджер жизненного цикла
func NewLifecycleManager(registry *ServiceRegistry, eventBus *events.EventBus, config CoordinatorConfig) *LifecycleManager {
	lm := &LifecycleManager{
		registry:        registry,
		eventBus:        eventBus,
		dependencyGraph: &DependencyGraph{Services: make(map[string][]string)},
		restartAttempts: make(map[string]int),
		config:          config,
		startTime:       time.Now(),
		stopChan:        make(chan struct{}),
	}

	// Запускаем health check если интервал указан
	if lm.config.HealthCheckInterval > 0 {
		lm.healthTicker = time.NewTicker(lm.config.HealthCheckInterval)
		go lm.healthCheckLoop()
	}

	return lm
}

// StartService запускает сервис
func (lm *LifecycleManager) StartService(name string) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	service, exists := lm.registry.Get(name)
	if !exists {
		return ErrServiceNotFound
	}

	// Проверяем зависимости
	if deps, ok := lm.dependencyGraph.Services[name]; ok {
		for _, dep := range deps {
			if depInfo, exists := lm.registry.GetInfo(dep); exists {
				if depInfo.State != StateRunning {
					return ManagerError{"зависимость " + dep + " не запущена"}
				}
			}
		}
	}

	// Обновляем состояние
	lm.registry.UpdateInfo(name, ServiceInfo{
		State:     StateStarting,
		StartedAt: time.Now(),
	})

	// Публикуем событие через eventBus если он есть
	lm.publishEvent(types.Event{
		Type:   types.EventServiceStarted,
		Source: name,
		Data: map[string]interface{}{
			"сервис":   name,
			"действие": "запуск",
		},
		Timestamp: time.Now(),
	})

	// Запускаем сервис
	err := service.Start() // ИЗМЕНЕНО: напрямую вызываем service.Start()
	if err != nil {
		lm.registry.UpdateInfo(name, ServiceInfo{
			State: StateError,
			Error: err.Error(),
		})

		lm.publishEvent(types.Event{
			Type:   types.EventError,
			Source: name,
			Data: map[string]interface{}{
				"сервис":   name,
				"ошибка":   err.Error(),
				"действие": "ошибка_запуска",
			},
			Timestamp: time.Now(),
		})

		// Пытаемся перезапустить если включено
		if lm.config.RestartOnFailure {
			lm.scheduleRestart(name)
		}

		return err
	}

	// Обновляем состояние
	lm.registry.UpdateInfo(name, ServiceInfo{
		State:     StateRunning,
		StartedAt: time.Now(),
	})

	// Сбрасываем счетчик перезапусков
	delete(lm.restartAttempts, name)

	lm.publishEvent(types.Event{
		Type:   types.EventServiceStarted,
		Source: name,
		Data: map[string]interface{}{
			"сервис":   name,
			"действие": "запущен",
			"статус":   "работает",
		},
		Timestamp: time.Now(),
	})

	return nil
}

// StopService останавливает сервис
func (lm *LifecycleManager) StopService(name string) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	service, exists := lm.registry.Get(name)
	if !exists {
		return ErrServiceNotFound
	}

	// Обновляем состояние
	lm.registry.UpdateInfo(name, ServiceInfo{
		State:     StateStopping,
		StoppedAt: time.Now(),
	})

	lm.publishEvent(types.Event{
		Type:   types.EventServiceStopped,
		Source: name,
		Data: map[string]interface{}{
			"сервис":   name,
			"действие": "остановка",
		},
		Timestamp: time.Now(),
	})

	// Останавливаем сервис
	err := service.Stop()
	if err != nil {
		lm.registry.UpdateInfo(name, ServiceInfo{
			State: StateError,
			Error: err.Error(),
		})

		lm.publishEvent(types.Event{
			Type:   types.EventServiceError,
			Source: name,
			Data: map[string]interface{}{
				"сервис":   name,
				"действие": "ошибка_остановки",
				"ошибка":   err.Error(),
			},
			Timestamp: time.Now(),
		})

		return err
	}

	// Обновляем состояние
	lm.registry.UpdateInfo(name, ServiceInfo{
		State:     StateStopped,
		StoppedAt: time.Now(),
	})

	lm.publishEvent(types.Event{
		Type:   types.EventServiceStopped,
		Source: name,
		Data: map[string]interface{}{
			"сервис":   name,
			"действие": "остановлен",
			"статус":   "остановлен",
			"время":    time.Now(),
		},
		Timestamp: time.Now(),
	})

	return nil
}

// RestartService перезапускает сервис
func (lm *LifecycleManager) RestartService(name string) error {
	// Останавливаем
	if err := lm.StopService(name); err != nil {
		return err
	}

	// Ждем немного
	time.Sleep(lm.config.RestartDelay)

	// Запускаем
	return lm.StartService(name)
}

// AddDependency добавляет зависимость
func (lm *LifecycleManager) AddDependency(service, dependency string) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	if _, exists := lm.dependencyGraph.Services[service]; !exists {
		lm.dependencyGraph.Services[service] = make([]string, 0)
	}

	// Проверяем нет ли уже такой зависимости
	for _, dep := range lm.dependencyGraph.Services[service] {
		if dep == dependency {
			return
		}
	}

	lm.dependencyGraph.Services[service] = append(lm.dependencyGraph.Services[service], dependency)
}

// GetDependencies возвращает зависимости сервиса
func (lm *LifecycleManager) GetDependencies(service string) []string {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	if deps, exists := lm.dependencyGraph.Services[service]; exists {
		return deps
	}

	return []string{}
}

// publishEvent публикует событие через eventBus или логирует
func (lm *LifecycleManager) publishEvent(event types.Event) {
	if lm.eventBus != nil {
		lm.eventBus.Publish(event)
	} else {
		logger.Info("[СОБЫТИЕ] %s: %s - %v", event.Type, event.Source, event.Data)
	}
}

// Stop останавливает менеджер
func (lm *LifecycleManager) Stop() {
	close(lm.stopChan)
}

// GetUptime возвращает время работы
func (lm *LifecycleManager) GetUptime() time.Duration {
	return time.Since(lm.startTime)
}
