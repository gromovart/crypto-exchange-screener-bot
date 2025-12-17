package manager

import (
	"sync"
	"time"
)

// LifecycleManager управляет жизненным циклом сервисов
type LifecycleManager struct {
	mu              sync.RWMutex
	registry        *ServiceRegistry
	coordinator     *EventCoordinator
	dependencyGraph *DependencyGraph
	restartAttempts map[string]int
	config          CoordinatorConfig
	startTime       time.Time
	healthTicker    *time.Ticker
	stopChan        chan struct{}
}

// NewLifecycleManager создает менеджер жизненного цикла
func NewLifecycleManager(registry *ServiceRegistry, coordinator *EventCoordinator, config CoordinatorConfig) *LifecycleManager {
	lm := &LifecycleManager{
		registry:        registry,
		coordinator:     coordinator,
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
					return ManagerError{"dependency " + dep + " is not running"}
				}
			}
		}
	}

	// Обновляем состояние
	lm.registry.UpdateInfo(name, ServiceInfo{
		State:     StateStarting,
		StartedAt: time.Now(),
	})

	// Публикуем событие
	lm.coordinator.PublishEvent(Event{
		Type:      EventServiceStarted,
		Service:   name,
		Message:   "Starting service",
		Timestamp: time.Now(),
		Severity:  "info",
	})

	// Запускаем сервис
	err := service.Start()
	if err != nil {
		lm.registry.UpdateInfo(name, ServiceInfo{
			State: StateError,
			Error: err.Error(),
		})

		lm.coordinator.PublishEvent(Event{
			Type:      EventServiceError,
			Service:   name,
			Message:   "Failed to start: " + err.Error(),
			Timestamp: time.Now(),
			Severity:  "error",
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

	lm.coordinator.PublishEvent(Event{
		Type:      EventServiceStarted,
		Service:   name,
		Message:   "Service started successfully",
		Timestamp: time.Now(),
		Severity:  "info",
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

	lm.coordinator.PublishEvent(Event{
		Type:      EventServiceStopped,
		Service:   name,
		Message:   "Stopping service",
		Timestamp: time.Now(),
		Severity:  "info",
	})

	// Останавливаем сервис
	err := service.Stop()
	if err != nil {
		lm.registry.UpdateInfo(name, ServiceInfo{
			State: StateError,
			Error: err.Error(),
		})

		lm.coordinator.PublishEvent(Event{
			Type:      EventServiceError,
			Service:   name,
			Message:   "Failed to stop: " + err.Error(),
			Timestamp: time.Now(),
			Severity:  "error",
		})

		return err
	}

	// Обновляем состояние
	lm.registry.UpdateInfo(name, ServiceInfo{
		State:     StateStopped,
		StoppedAt: time.Now(),
	})

	lm.coordinator.PublishEvent(Event{
		Type:      EventServiceStopped,
		Service:   name,
		Message:   "Service stopped successfully",
		Timestamp: time.Now(),
		Severity:  "info",
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

// StartAll запускает все сервисы в правильном порядке
func (lm *LifecycleManager) StartAll() map[string]error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	// Получаем порядок запуска на основе зависимостей
	order := lm.getStartOrder()

	errors := make(map[string]error)

	for _, serviceName := range order {
		service, exists := lm.registry.Get(serviceName)
		if !exists {
			continue
		}

		if err := service.Start(); err != nil {
			errors[serviceName] = err
			lm.registry.UpdateInfo(serviceName, ServiceInfo{
				State: StateError,
				Error: err.Error(),
			})

			// Пытаемся перезапустить если включено
			if lm.config.RestartOnFailure {
				lm.scheduleRestart(serviceName)
			}
		} else {
			lm.registry.UpdateInfo(serviceName, ServiceInfo{
				State:     StateRunning,
				StartedAt: time.Now(),
			})
		}
	}

	return errors
}

// StopAll останавливает все сервисы в обратном порядке
func (lm *LifecycleManager) StopAll() map[string]error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	// Получаем порядок остановки (обратный порядку запуска)
	order := lm.getStopOrder()

	errors := make(map[string]error)

	for _, serviceName := range order {
		service, exists := lm.registry.Get(serviceName)
		if !exists {
			continue
		}

		if err := service.Stop(); err != nil {
			errors[serviceName] = err
			lm.registry.UpdateInfo(serviceName, ServiceInfo{
				State: StateError,
				Error: err.Error(),
			})
		} else {
			lm.registry.UpdateInfo(serviceName, ServiceInfo{
				State:     StateStopped,
				StoppedAt: time.Now(),
			})
		}
	}

	return errors
}

// getStartOrder возвращает порядок запуска на основе зависимостей
func (lm *LifecycleManager) getStartOrder() []string {
	// Простая топологическая сортировка
	visited := make(map[string]bool)
	temp := make(map[string]bool)
	order := make([]string, 0)

	var visit func(string)
	visit = func(service string) {
		if temp[service] {
			// Циклическая зависимость
			return
		}

		if !visited[service] {
			temp[service] = true

			// Сначала посещаем зависимости
			if deps, exists := lm.dependencyGraph.Services[service]; exists {
				for _, dep := range deps {
					visit(dep)
				}
			}

			temp[service] = false
			visited[service] = true
			order = append(order, service)
		}
	}

	// Запускаем для всех сервисов
	for service := range lm.registry.GetAll() {
		if !visited[service] {
			visit(service)
		}
	}

	return order
}

// getStopOrder возвращает порядок остановки
func (lm *LifecycleManager) getStopOrder() []string {
	// Обратный порядок запуска
	startOrder := lm.getStartOrder()
	stopOrder := make([]string, len(startOrder))

	for i, service := range startOrder {
		stopOrder[len(startOrder)-1-i] = service
	}

	return stopOrder
}

// scheduleRestart планирует перезапуск сервиса
func (lm *LifecycleManager) scheduleRestart(name string) {
	attempts := lm.restartAttempts[name]
	attempts++
	lm.restartAttempts[name] = attempts

	// Проверяем максимальное количество попыток
	if attempts > lm.config.MaxRestartAttempts && lm.config.MaxRestartAttempts > 0 {
		lm.coordinator.PublishEvent(Event{
			Type:      EventServiceError,
			Service:   name,
			Message:   "Max restart attempts exceeded, giving up",
			Timestamp: time.Now(),
			Severity:  "error",
		})
		return
	}

	// Запланировать перезапуск
	go func() {
		time.Sleep(lm.config.RestartDelay)

		lm.coordinator.PublishEvent(Event{
			Type:      EventServiceStarted,
			Service:   name,
			Message:   "Attempting restart",
			Timestamp: time.Now(),
			Severity:  "info",
		})

		if err := lm.StartService(name); err != nil {
			lm.coordinator.PublishEvent(Event{
				Type:      EventServiceError,
				Service:   name,
				Message:   "Restart failed: " + err.Error(),
				Timestamp: time.Now(),
				Severity:  "error",
			})
		}
	}()
}

// healthCheckLoop цикл проверки здоровья
func (lm *LifecycleManager) healthCheckLoop() {
	for {
		select {
		case <-lm.healthTicker.C:
			lm.performHealthCheck()
		case <-lm.stopChan:
			lm.healthTicker.Stop()
			return
		}
	}
}

// performHealthCheck выполняет проверку здоровья
func (lm *LifecycleManager) performHealthCheck() {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	health := lm.registry.CheckHealth()

	// Публикуем событие проверки здоровья
	lm.coordinator.PublishEvent(Event{
		Type:      EventHealthCheck,
		Service:   "LifecycleManager",
		Message:   "Health check performed",
		Data:      health,
		Timestamp: time.Now(),
		Severity:  "info",
	})

	// Перезапускаем неудачные сервисы если включено
	if lm.config.RestartOnFailure {
		for service, healthy := range health {
			if !healthy {
				lm.scheduleRestart(service)
			}
		}
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
