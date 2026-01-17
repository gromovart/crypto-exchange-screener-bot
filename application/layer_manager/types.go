// application/layer_manager/types.go
package layer_manager

import (
	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/in_memory_storage"
	"time"
)

// ServiceState состояние сервиса
type ServiceState string

const (
	StateStopped  ServiceState = "stopped"
	StateStarting ServiceState = "starting"
	StateRunning  ServiceState = "running"
	StateStopping ServiceState = "stopping"
	StateError    ServiceState = "error"
)

// ServiceInfo информация о сервиса
type ServiceInfo struct {
	Name      string       `json:"name"`
	State     ServiceState `json:"state"`
	StartedAt time.Time    `json:"started_at,omitempty"`
	StoppedAt time.Time    `json:"stopped_at,omitempty"`
	Error     string       `json:"error,omitempty"`
	Restarts  int          `json:"restarts"`
}

// SystemStats статистика системы
type SystemStats struct {
	Services      map[string]ServiceInfo `json:"services"`
	StorageStats  storage.StorageStats   `json:"storage_stats"`
	AnalysisStats interface{}            `json:"analysis_stats,omitempty"`
	EventBusStats interface{}            `json:"event_bus_stats,omitempty"`
	Uptime        time.Duration          `json:"uptime"`
	TotalRequests int64                  `json:"total_requests"`
	MemoryUsageMB float64                `json:"memory_usage_mb"`
	CPUUsage      float64                `json:"cpu_usage"`
	ActiveSymbols int                    `json:"active_symbols"`
	LastError     string                 `json:"last_error,omitempty"`
	LastUpdated   time.Time              `json:"last_updated"`
}

// EventType тип события
type EventType string

const (
	// События сервисов
	EventServiceStarted EventType = "service_started"
	EventServiceStopped EventType = "service_stopped"
	EventServiceError   EventType = "service_error"
	EventHealthCheck    EventType = "health_check"

	// События данных
	EventPriceUpdated   EventType = "price_updated"
	EventSignalDetected EventType = "signal_detected"
	EventSymbolAdded    EventType = "symbol_added"
	EventSymbolRemoved  EventType = "symbol_removed"

	// События интеграций
	EventTelegramSent  EventType = "telegram_sent"
	EventConfigChanged EventType = "config_changed"
)

// Event событие системы
type Event struct {
	Type      EventType   `json:"type"`
	Service   string      `json:"service,omitempty"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	Severity  string      `json:"severity"` // info, warning, error
}

// Service интерфейс сервиса
type Service interface {
	Name() string
	Start() error
	Stop() error
	State() ServiceState
	HealthCheck() bool
}

// DataSubscriber подписчик на данные
type DataSubscriber interface {
	OnPriceUpdate(symbol string, price, volume float64, timestamp time.Time)
	OnSymbolAdded(symbol string)
	OnSymbolRemoved(symbol string)
	OnEvent(event Event)
}

// CoordinatorConfig конфигурация координатора
type CoordinatorConfig struct {
	EnableEventLogging  bool          `json:"enable_event_logging"`
	EventBufferSize     int           `json:"event_buffer_size"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	RestartOnFailure    bool          `json:"restart_on_failure"`
	MaxRestartAttempts  int           `json:"max_restart_attempts"`
	RestartDelay        time.Duration `json:"restart_delay"`
	EnableMetrics       bool          `json:"enable_metrics"`
	MetricsPort         string        `json:"metrics_port"`
}

// DependencyGraph граф зависимостей
type DependencyGraph struct {
	Services map[string][]string `json:"services"` // service -> dependencies
}

// HealthStatus статус здоровья
type HealthStatus struct {
	Status    string            `json:"status"` // healthy, degraded, unhealthy
	Services  map[string]string `json:"services"`
	Timestamp time.Time         `json:"timestamp"`
}
