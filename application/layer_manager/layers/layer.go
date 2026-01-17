// application/services/orchestrator/layers/layer.go
package layers

import (
	"sync"
	"time"
)

// LayerState состояние слоя
type LayerState string

const (
	StateCreated      LayerState = "created"
	StateInitializing LayerState = "initializing"
	StateInitialized  LayerState = "initialized"
	StateStarting     LayerState = "starting"
	StateRunning      LayerState = "running"
	StateStopping     LayerState = "stopping"
	StateStopped      LayerState = "stopped"
	StateError        LayerState = "error"
)

// LayerStatus статус слоя
type LayerStatus struct {
	Name         string
	State        LayerState
	IsHealthy    bool
	Initialized  bool
	Running      bool
	Uptime       time.Duration
	StartTime    time.Time
	LastError    string
	Dependencies []string
	Components   []string
}

// Layer интерфейс для слоя приложения
type Layer interface {
	// Основные методы
	Name() string
	Initialize() error
	Start() error
	Stop() error
	Reset() error

	// Состояние и мониторинг
	GetStatus() LayerStatus
	HealthCheck() bool
	IsInitialized() bool
	IsRunning() bool
	GetUptime() time.Duration

	// Компоненты
	GetComponents() map[string]interface{}
	GetComponent(name string) (interface{}, bool)
	HasComponent(name string) bool

	// Зависимости
	SetDependencies(deps map[string]Layer) error
	GetDependencies() []string
	ValidateDependencies() error

	// Конфигурация
	UpdateConfig(config interface{}) error
	GetConfig() interface{}
}

// BaseLayer базовая реализация Layer
type BaseLayer struct {
	mu           sync.RWMutex
	name         string
	state        LayerState
	isHealthy    bool
	initialized  bool
	running      bool
	startTime    time.Time
	lastError    string
	dependencies []string
	components   map[string]interface{}
	config       interface{}
}

// NewBaseLayer создает базовый слой
func NewBaseLayer(name string, deps []string) *BaseLayer {
	return &BaseLayer{
		name:         name,
		state:        StateCreated,
		isHealthy:    true,
		initialized:  false,
		running:      false,
		dependencies: deps,
		components:   make(map[string]interface{}),
	}
}

// Name возвращает имя слоя
func (bl *BaseLayer) Name() string {
	bl.mu.RLock()
	defer bl.mu.RUnlock()
	return bl.name
}

// GetStatus возвращает статус слоя
func (bl *BaseLayer) GetStatus() LayerStatus {
	bl.mu.RLock()
	defer bl.mu.RUnlock()

	var uptime time.Duration
	if !bl.startTime.IsZero() && bl.running {
		uptime = time.Since(bl.startTime)
	}

	componentNames := make([]string, 0, len(bl.components))
	for name := range bl.components {
		componentNames = append(componentNames, name)
	}

	return LayerStatus{
		Name:         bl.name,
		State:        bl.state,
		IsHealthy:    bl.isHealthy,
		Initialized:  bl.initialized,
		Running:      bl.running,
		Uptime:       uptime,
		StartTime:    bl.startTime,
		LastError:    bl.lastError,
		Dependencies: bl.dependencies,
		Components:   componentNames,
	}
}

// HealthCheck проверяет здоровье слоя
func (bl *BaseLayer) HealthCheck() bool {
	bl.mu.RLock()
	defer bl.mu.RUnlock()
	return bl.isHealthy && bl.state != StateError
}

// IsInitialized проверяет инициализацию
func (bl *BaseLayer) IsInitialized() bool {
	bl.mu.RLock()
	defer bl.mu.RUnlock()
	return bl.initialized
}

// IsRunning проверяет работает ли слой
func (bl *BaseLayer) IsRunning() bool {
	bl.mu.RLock()
	defer bl.mu.RUnlock()
	return bl.running
}

// GetUptime возвращает время работы
func (bl *BaseLayer) GetUptime() time.Duration {
	bl.mu.RLock()
	defer bl.mu.RUnlock()

	if bl.startTime.IsZero() || !bl.running {
		return 0
	}
	return time.Since(bl.startTime)
}

// GetComponents возвращает все компоненты
func (bl *BaseLayer) GetComponents() map[string]interface{} {
	bl.mu.RLock()
	defer bl.mu.RUnlock()

	result := make(map[string]interface{})
	for k, v := range bl.components {
		result[k] = v
	}
	return result
}

// GetComponent возвращает компонент по имени
func (bl *BaseLayer) GetComponent(name string) (interface{}, bool) {
	bl.mu.RLock()
	defer bl.mu.RUnlock()
	comp, exists := bl.components[name]
	return comp, exists
}

// HasComponent проверяет наличие компонента
func (bl *BaseLayer) HasComponent(name string) bool {
	bl.mu.RLock()
	defer bl.mu.RUnlock()
	_, exists := bl.components[name]
	return exists
}

// SetDependencies устанавливает зависимости
func (bl *BaseLayer) SetDependencies(deps map[string]Layer) error {
	bl.mu.Lock()
	defer bl.mu.Unlock()

	// Проверяем что все зависимости доступны
	for _, depName := range bl.dependencies {
		if _, exists := deps[depName]; !exists {
			return &LayerError{
				LayerName: bl.name,
				Message:   "зависимость не найдена: " + depName,
			}
		}
	}

	// Здесь можно добавить логику проверки зависимостей
	return nil
}

// GetDependencies возвращает зависимости
func (bl *BaseLayer) GetDependencies() []string {
	bl.mu.RLock()
	defer bl.mu.RUnlock()
	return bl.dependencies
}

// ValidateDependencies проверяет зависимости
func (bl *BaseLayer) ValidateDependencies() error {
	// Базовая реализация - просто проверяет что зависимости указаны
	if len(bl.dependencies) == 0 {
		return nil
	}
	return &LayerError{
		LayerName: bl.name,
		Message:   "зависимости требуют реализации в конкретном слое",
	}
}

// UpdateConfig обновляет конфигурацию
func (bl *BaseLayer) UpdateConfig(config interface{}) error {
	bl.mu.Lock()
	defer bl.mu.Unlock()
	bl.config = config
	return nil
}

// GetConfig возвращает конфигурацию
func (bl *BaseLayer) GetConfig() interface{} {
	bl.mu.RLock()
	defer bl.mu.RUnlock()
	return bl.config
}

// Initialize базовая реализация инициализации
func (bl *BaseLayer) Initialize() error {
	bl.mu.Lock()
	defer bl.mu.Unlock()

	if bl.initialized {
		return &LayerError{
			LayerName: bl.name,
			Message:   "слой уже инициализирован",
		}
	}

	bl.state = StateInitializing
	bl.isHealthy = true

	// В производных классах нужно переопределить эту логику
	bl.initialized = true
	bl.state = StateInitialized

	return nil
}

// Start базовая реализация запуска
func (bl *BaseLayer) Start() error {
	bl.mu.Lock()
	defer bl.mu.Unlock()

	if !bl.initialized {
		return &LayerError{
			LayerName: bl.name,
			Message:   "слой не инициализирован",
		}
	}

	if bl.running {
		return &LayerError{
			LayerName: bl.name,
			Message:   "слой уже запущен",
		}
	}

	bl.state = StateStarting
	bl.startTime = time.Now()

	// В производных классах нужно переопределить эту логику
	bl.running = true
	bl.state = StateRunning

	return nil
}

// Stop базовая реализация остановки
func (bl *BaseLayer) Stop() error {
	bl.mu.Lock()
	defer bl.mu.Unlock()

	if !bl.running {
		return nil
	}

	bl.state = StateStopping

	// В производных классах нужно переопределить эту логику
	bl.running = false
	bl.state = StateStopped

	return nil
}

// Reset базовая реализация сброса
func (bl *BaseLayer) Reset() error {
	bl.mu.Lock()
	defer bl.mu.Unlock()

	// Останавливаем если запущен
	if bl.running {
		bl.state = StateStopping
		bl.running = false
	}

	// Сбрасываем состояние
	bl.initialized = false
	bl.isHealthy = true
	bl.state = StateCreated
	bl.startTime = time.Time{}
	bl.lastError = ""

	// Очищаем компоненты
	bl.components = make(map[string]interface{})

	return nil
}

// registerComponent регистрирует компонент в слое
func (bl *BaseLayer) registerComponent(name string, component interface{}) {
	bl.mu.Lock()
	defer bl.mu.Unlock()
	bl.components[name] = component
}

// unregisterComponent удаляет компонент из слоя
func (bl *BaseLayer) unregisterComponent(name string) {
	bl.mu.Lock()
	defer bl.mu.Unlock()
	delete(bl.components, name)
}

// updateState обновляет состояние слоя
func (bl *BaseLayer) updateState(state LayerState) {
	bl.mu.Lock()
	defer bl.mu.Unlock()
	bl.state = state
}

// setError устанавливает ошибку слоя
func (bl *BaseLayer) setError(err error) {
	bl.mu.Lock()
	defer bl.mu.Unlock()
	bl.state = StateError
	bl.isHealthy = false
	if err != nil {
		bl.lastError = err.Error()
	}
}

// clearError очищает ошибку слоя
func (bl *BaseLayer) clearError() {
	bl.mu.Lock()
	defer bl.mu.Unlock()
	if bl.state == StateError {
		bl.state = StateStopped
	}
	bl.isHealthy = true
	bl.lastError = ""
}

// LayerError ошибка слоя
type LayerError struct {
	LayerName string
	Message   string
}

func (e *LayerError) Error() string {
	return "слой " + e.LayerName + ": " + e.Message
}

// IsLayerError проверяет является ли ошибка LayerError
func IsLayerError(err error) bool {
	_, ok := err.(*LayerError)
	return ok
}
