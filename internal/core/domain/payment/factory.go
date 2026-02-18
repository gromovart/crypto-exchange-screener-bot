// internal/core/domain/payment/factory.go
package payment

import (
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/http_client"
	invoice_repo "crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/repository/invoice"
	payment_repo "crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/repository/payment"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
)

// StarsServiceFactory фабрика для создания StarsService
type StarsServiceFactory struct {
	config         *Config
	userManager    UserManager
	eventPublisher EventPublisher
	logger         *logger.Logger
	starsClient    *http_client.StarsClient
	botUsername    string
	initialized    bool
}

// Config конфигурация для StarsService
type Config struct {
	TelegramBotToken           string
	TelegramStarsProviderToken string
	TelegramBotUsername        string
}

// Dependencies зависимости для фабрики StarsService
type Dependencies struct {
	Config         *Config
	UserManager    UserManager
	EventPublisher EventPublisher
	Logger         *logger.Logger
}

// NewStarsServiceFactory создает новую фабрику StarsService
func NewStarsServiceFactory(deps Dependencies) (*StarsServiceFactory, error) {
	if deps.Config == nil {
		return nil, fmt.Errorf("конфигурация обязательна")
	}

	if deps.UserManager == nil {
		return nil, fmt.Errorf("UserManager обязателен")
	}

	if deps.EventPublisher == nil {
		return nil, fmt.Errorf("EventPublisher обязателен")
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
		config:         deps.Config,
		userManager:    deps.UserManager,
		eventPublisher: deps.EventPublisher,
		logger:         deps.Logger,
		botUsername:    deps.Config.TelegramBotUsername,
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
		f.eventPublisher,
		f.logger,
		f.starsClient,
		f.botUsername,
	)

	f.logger.Info("✅ StarsService создан")

	return service, nil
}

// ==================== НОВАЯ ФАБРИКА ДЛЯ PAYMENTSERVICE ====================

// PaymentServiceFactory фабрика для создания PaymentService
type PaymentServiceFactory struct {
	starsService *StarsService
	paymentRepo  payment_repo.PaymentRepository
	invoiceRepo  invoice_repo.InvoiceRepository
	logger       *logger.Logger
	initialized  bool
}

// PaymentServiceDependencies зависимости для фабрики PaymentService
type PaymentServiceDependencies struct {
	StarsService *StarsService
	PaymentRepo  payment_repo.PaymentRepository
	InvoiceRepo  invoice_repo.InvoiceRepository
	Logger       *logger.Logger
}

// NewPaymentServiceFactory создает новую фабрику PaymentService
func NewPaymentServiceFactory(deps PaymentServiceDependencies) (*PaymentServiceFactory, error) {
	if deps.StarsService == nil {
		return nil, fmt.Errorf("StarsService обязателен")
	}

	if deps.PaymentRepo == nil {
		return nil, fmt.Errorf("PaymentRepo обязателен")
	}

	if deps.InvoiceRepo == nil {
		return nil, fmt.Errorf("InvoiceRepo обязателен")
	}

	if deps.Logger == nil {
		return nil, fmt.Errorf("Logger обязателен")
	}

	factory := &PaymentServiceFactory{
		starsService: deps.StarsService,
		paymentRepo:  deps.PaymentRepo,
		invoiceRepo:  deps.InvoiceRepo,
		logger:       deps.Logger,
		initialized:  true,
	}

	deps.Logger.Info("✅ Фабрика PaymentService создана")
	return factory, nil
}

// CreatePaymentService создает новый PaymentService
func (f *PaymentServiceFactory) CreatePaymentService() (*PaymentService, error) {
	if !f.initialized {
		return nil, fmt.Errorf("фабрика не инициализирована")
	}

	service := NewPaymentService(
		f.starsService,
		f.paymentRepo,
		f.invoiceRepo, // ⭐ Передаем invoiceRepo
		f.logger,
	)

	f.logger.Info("✅ PaymentService создан")
	return service, nil
}

// IsReady проверяет готовность фабрики
func (f *PaymentServiceFactory) IsReady() bool {
	return f.initialized && f.starsService != nil && f.paymentRepo != nil && f.invoiceRepo != nil
}

// Validate проверяет валидность фабрики
func (f *PaymentServiceFactory) Validate() error {
	if !f.initialized {
		return fmt.Errorf("фабрика не инициализирована")
	}

	if f.starsService == nil {
		return fmt.Errorf("StarsService не установлен")
	}

	if f.paymentRepo == nil {
		return fmt.Errorf("PaymentRepo не установлен")
	}

	if f.invoiceRepo == nil {
		return fmt.Errorf("InvoiceRepo не установлен")
	}

	return nil
}

// GetDependenciesInfo возвращает информацию о зависимостях
func (f *PaymentServiceFactory) GetDependenciesInfo() map[string]interface{} {
	return map[string]interface{}{
		"initialized":   f.initialized,
		"stars_service": f.starsService != nil,
		"payment_repo":  f.paymentRepo != nil,
		"invoice_repo":  f.invoiceRepo != nil,
	}
}

// ==================== СТАРЫЕ МЕТОДЫ (ОСТАВЛЯЕМ) ====================

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

	if f.eventPublisher == nil {
		return fmt.Errorf("EventPublisher не установлен")
	}

	return nil
}

// GetDependenciesInfo возвращает информацию о зависимостях
func (f *StarsServiceFactory) GetDependenciesInfo() map[string]interface{} {
	return map[string]interface{}{
		"initialized":           f.initialized,
		"stars_client_ready":    f.starsClient != nil,
		"user_manager_ready":    f.userManager != nil,
		"event_publisher_ready": f.eventPublisher != nil,
		"bot_username":          f.botUsername,
		"has_provider_token":    f.config != nil && f.config.TelegramStarsProviderToken != "",
	}
}

// SetUserManager устанавливает UserManager
func (f *StarsServiceFactory) SetUserManager(userManager UserManager) {
	f.userManager = userManager
}

// SetEventPublisher устанавливает EventPublisher
func (f *StarsServiceFactory) SetEventPublisher(eventPublisher EventPublisher) {
	f.eventPublisher = eventPublisher
}

// SetLogger устанавливает Logger
func (f *StarsServiceFactory) SetLogger(logger *logger.Logger) {
	f.logger = logger
}
