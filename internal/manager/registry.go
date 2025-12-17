package manager

import (
	"sync"
	"time"
)

// ServiceRegistry реестр сервисов
type ServiceRegistry struct {
	mu       sync.RWMutex
	services map[string]Service
	info     map[string]ServiceInfo
}

// NewServiceRegistry создает новый реестр сервисов
func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		services: make(map[string]Service),
		info:     make(map[string]ServiceInfo),
	}
}

// Register регистрирует сервис
func (sr *ServiceRegistry) Register(name string, service Service) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	if _, exists := sr.services[name]; exists {
		return ErrServiceAlreadyExists
	}

	sr.services[name] = service
	sr.info[name] = ServiceInfo{
		Name:  name,
		State: StateStopped,
	}

	return nil
}

// Get возвращает сервис по имени
func (sr *ServiceRegistry) Get(name string) (Service, bool) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	service, exists := sr.services[name]
	return service, exists
}

// GetAll возвращает все сервисы
func (sr *ServiceRegistry) GetAll() map[string]Service {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	result := make(map[string]Service)
	for k, v := range sr.services {
		result[k] = v
	}
	return result
}

// UpdateInfo обновляет информацию о сервисе
func (sr *ServiceRegistry) UpdateInfo(name string, info ServiceInfo) {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	if current, exists := sr.info[name]; exists {
		current.State = info.State
		if info.StartedAt.IsZero() {
			info.StartedAt = current.StartedAt
		}
		if info.StoppedAt.IsZero() {
			info.StoppedAt = current.StoppedAt
		}
		if info.Error != "" {
			current.Error = info.Error
		}
		if info.Restarts > 0 {
			current.Restarts = info.Restarts
		}
		sr.info[name] = current
	}
}

// GetInfo возвращает информацию о сервисе
func (sr *ServiceRegistry) GetInfo(name string) (ServiceInfo, bool) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	info, exists := sr.info[name]
	return info, exists
}

// GetAllInfo возвращает информацию о всех сервисах
func (sr *ServiceRegistry) GetAllInfo() map[string]ServiceInfo {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	result := make(map[string]ServiceInfo)
	for k, v := range sr.info {
		result[k] = v
	}
	return result
}

// Remove удаляет сервис
func (sr *ServiceRegistry) Remove(name string) {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	delete(sr.services, name)
	delete(sr.info, name)
}

// Count возвращает количество сервисов
func (sr *ServiceRegistry) Count() int {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	return len(sr.services)
}

// ServiceNames возвращает имена всех сервисов
func (sr *ServiceRegistry) ServiceNames() []string {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	names := make([]string, 0, len(sr.services))
	for name := range sr.services {
		names = append(names, name)
	}
	return names
}

// StartAll запускает все сервисы
func (sr *ServiceRegistry) StartAll() map[string]error {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	errors := make(map[string]error)
	for name, service := range sr.services {
		if err := service.Start(); err != nil {
			errors[name] = err
			sr.updateServiceState(name, StateError, err.Error())
		} else {
			sr.updateServiceState(name, StateRunning, "")
		}
	}
	return errors
}

// StopAll останавливает все сервисы
func (sr *ServiceRegistry) StopAll() map[string]error {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	errors := make(map[string]error)
	for name, service := range sr.services {
		if err := service.Stop(); err != nil {
			errors[name] = err
		} else {
			sr.updateServiceState(name, StateStopped, "")
		}
	}
	return errors
}

// updateServiceState обновляет состояние сервиса
func (sr *ServiceRegistry) updateServiceState(name string, state ServiceState, errorMsg string) {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	if info, exists := sr.info[name]; exists {
		info.State = state
		info.Error = errorMsg

		switch state {
		case StateRunning:
			info.StartedAt = time.Now()
		case StateStopped, StateError:
			info.StoppedAt = time.Now()
		}

		sr.info[name] = info
	}
}

// CheckHealth проверяет здоровье всех сервисов
func (sr *ServiceRegistry) CheckHealth() map[string]bool {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	health := make(map[string]bool)
	for name, service := range sr.services {
		health[name] = service.HealthCheck()

		// Обновляем состояние если сервис умер
		if !health[name] {
			sr.updateServiceState(name, StateError, "health check failed")
		}
	}
	return health
}

// Ошибки реестра
var (
	ErrServiceAlreadyExists = ManagerError{"service already exists"}
	ErrServiceNotFound      = ManagerError{"service not found"}
)

// ManagerError ошибка менеджера
type ManagerError struct {
	Message string
}

func (e ManagerError) Error() string {
	return e.Message
}
