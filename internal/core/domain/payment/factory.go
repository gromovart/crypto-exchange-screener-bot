// internal/core/domain/payment/factory.go
package payment

import (
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/http_client"
	event_bus "crypto-exchange-screener-bot/internal/infrastructure/transport/event_bus"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
)

// StarsServiceFactory фабрика для создания StarsService
type StarsServiceFactory struct {
	config      *Config
	userManager UserManager
	eventBus    *event_bus.EventBus
	logger      *logger.Logger
	starsClient *http_client.StarsClient
	botUsername string
	initialized bool
}

// Config конфигурация для StarsService
type Config struct {
	TelegramBotToken           string
	TelegramStarsProviderToken string
	TelegramBotUsername        string
}

// Dependencies зависимости для фабрики StarsService
type Dependencies struct {
	Config      *Config
	UserManager UserManager
	EventBus    *event_bus.EventBus
	Logger      *logger.Logger
}

// NewStarsServiceFactory создает новую фабрику StarsService
func NewStarsServiceFactory(deps Dependencies) (*StarsServiceFactory, error) {
	if deps.Config == nil {
		return nil, fmt.Errorf("конфигурация обязательна")
	}

	if deps.UserManager == nil {
		return nil, fmt.Errorf("UserManager обязателен")
	}

	if deps.EventBus == nil {
		return nil, fmt.Errorf("EventBus обязателен")
	}

	// Проверяем обязательные поля конфигурации
	if deps.Config.TelegramBotToken == "" {
		return nil, fmt.Errorf("TelegramBotToken обязателен")
	}

	if deps.Config.TelegramStarsProviderToken == "" {
		return nil, fmt.Errorf("TelegramStarsProviderToken обязателен")
	}

	if deps.Config.TelegramBotUsername == "" {
		return nil, fmt.Errorf("TelegramBotUsername обязателен")
	}

	factory := &StarsServiceFactory{
		config:      deps.Config,
		userManager: deps.UserManager,
		eventBus:    deps.EventBus,
		logger:      deps.Logger,
		botUsername: deps.Config.TelegramBotUsername,
	}

	// Создаем StarsClient
	baseURL := "https://api.telegram.org/bot" + deps.Config.TelegramBotToken + "/"
	factory.starsClient = http_client.NewStarsClient(baseURL, deps.Config.TelegramStarsProviderToken)

	factory.initialized = true

	deps.Logger.Info("✅ Фабрика StarsService создана")

	return factory, nil
}

// CreateStarsService создает новый StarsService
func (f *StarsServiceFactory) CreateStarsService() (*StarsService, error) {
	if !f.initialized {
		return nil, fmt.Errorf("фабрика не инициализирована")
	}

	if f.starsClient == nil {
		return nil, fmt.Errorf("StarsClient не создан")
	}

	service := NewStarsService(
		f.userManager,
		f.eventBus,
		f.logger,
		f.starsClient,
		f.botUsername,
	)

	f.logger.Info("✅ StarsService создан")

	return service, nil
}

// UpdateConfig обновляет конфигурацию
func (f *StarsServiceFactory) UpdateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("конфигурация не может быть nil")
	}

	f.config = config
	f.botUsername = config.TelegramBotUsername

	// Пересоздаем StarsClient с новым токеном
	baseURL := "https://api.telegram.org/bot" + config.TelegramBotToken + "/"
	f.starsClient = http_client.NewStarsClient(baseURL, config.TelegramStarsProviderToken)

	f.logger.Info("✅ Конфигурация StarsService обновлена")

	return nil
}

// GetConfig возвращает текущую конфигурацию
func (f *StarsServiceFactory) GetConfig() *Config {
	return f.config
}

// IsReady проверяет готовность фабрики
func (f *StarsServiceFactory) IsReady() bool {
	return f.initialized && f.starsClient != nil && f.userManager != nil
}

// Validate проверяет валидность фабрики
func (f *StarsServiceFactory) Validate() error {
	if !f.initialized {
		return fmt.Errorf("фабрика не инициализирована")
	}

	if f.starsClient == nil {
		return fmt.Errorf("StarsClient не создан")
	}

	if f.userManager == nil {
		return fmt.Errorf("UserManager не установлен")
	}

	if f.eventBus == nil {
		return fmt.Errorf("EventBus не установлен")
	}

	return nil
}

// GetDependenciesInfo возвращает информацию о зависимостях
func (f *StarsServiceFactory) GetDependenciesInfo() map[string]interface{} {
	return map[string]interface{}{
		"initialized":        f.initialized,
		"stars_client_ready": f.starsClient != nil,
		"user_manager_ready": f.userManager != nil,
		"event_bus_ready":    f.eventBus != nil,
		"bot_username":       f.botUsername,
		"has_provider_token": f.config != nil && f.config.TelegramStarsProviderToken != "",
	}
}

// SetUserManager устанавливает UserManager
func (f *StarsServiceFactory) SetUserManager(userManager UserManager) {
	f.userManager = userManager
}

// SetEventBus устанавливает EventBus
func (f *StarsServiceFactory) SetEventBus(eventBus *event_bus.EventBus) {
	f.eventBus = eventBus
}

// SetLogger устанавливает Logger
func (f *StarsServiceFactory) SetLogger(logger *logger.Logger) {
	f.logger = logger
}
