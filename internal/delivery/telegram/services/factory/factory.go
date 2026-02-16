// internal/delivery/telegram/services/factory/factory.go
package services_factory

import (
	"crypto-exchange-screener-bot/internal/core/domain/payment"
	"crypto-exchange-screener-bot/internal/core/domain/subscription"
	"crypto-exchange-screener-bot/internal/core/domain/users"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/buttons"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/formatters"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/message_sender"
	"crypto-exchange-screener-bot/internal/delivery/telegram/services/counter"
	"crypto-exchange-screener-bot/internal/delivery/telegram/services/notifications_toggle"
	payment_service "crypto-exchange-screener-bot/internal/delivery/telegram/services/payment"
	"crypto-exchange-screener-bot/internal/delivery/telegram/services/profile"
	"crypto-exchange-screener-bot/internal/delivery/telegram/services/signal_settings"
	"crypto-exchange-screener-bot/pkg/logger"
)

// ServiceFactory фабрика сервисов уровня пакета Telegram
type ServiceFactory struct {
	userService         *users.Service
	subscriptionService *subscription.Service
	paymentCoreService  *payment.PaymentService // ⭐ Изменено с StarsService на PaymentService
	messageSender       message_sender.MessageSender
	buttonBuilder       *buttons.ButtonBuilder
	formatterProvider   *formatters.FormatterProvider
}

// ServiceDependencies зависимости для фабрики сервисов
type ServiceDependencies struct {
	UserService         *users.Service
	SubscriptionService *subscription.Service
	PaymentCoreService  *payment.PaymentService // ⭐ Изменено
	MessageSender       message_sender.MessageSender
	ButtonBuilder       *buttons.ButtonBuilder
	FormatterProvider   *formatters.FormatterProvider
}

// NewServiceFactory создает фабрику сервисов
func NewServiceFactory(deps ServiceDependencies) *ServiceFactory {
	logger.Info("🏭 Создание фабрики сервисов Telegram-пакета...")

	return &ServiceFactory{
		userService:         deps.UserService,
		subscriptionService: deps.SubscriptionService,
		paymentCoreService:  deps.PaymentCoreService,
		messageSender:       deps.MessageSender,
		buttonBuilder:       deps.ButtonBuilder,
		formatterProvider:   deps.FormatterProvider,
	}
}

// CreateProfileService создает ProfileService
func (f *ServiceFactory) CreateProfileService() profile.Service {
	return profile.NewService(f.userService, f.subscriptionService)
}

// CreateCounterService создает CounterService
func (f *ServiceFactory) CreateCounterService() counter.Service {
	return counter.NewService(
		f.userService,
		f.formatterProvider,
		f.messageSender,
		f.buttonBuilder,
	)
}

// CreateNotificationToggleService создает NotificationToggleService
func (f *ServiceFactory) CreateNotificationToggleService() notifications_toggle.Service {
	return notifications_toggle.NewService(f.userService)
}

// CreateSignalSettingsService создает SignalSettingsService
func (f *ServiceFactory) CreateSignalSettingsService() signal_settings.Service {
	return signal_settings.NewService(f.userService)
}

// CreatePaymentService создает PaymentService
func (f *ServiceFactory) CreatePaymentService() payment_service.Service {
	if f.paymentCoreService == nil {
		logger.Warn("⚠️ PaymentCoreService не доступен, создается заглушка")
		return f.createPaymentServiceStub()
	}

	// Создаем зависимости для payment service
	deps := payment_service.Dependencies{
		PaymentService:      f.paymentCoreService, // ⭐ Передаем PaymentService
		SubscriptionService: f.subscriptionService,
		UserService:         f.userService,
	}

	// Используем NewServiceWithDependencies
	return payment_service.NewServiceWithDependencies(deps)
}

// createPaymentServiceStub создает заглушку для PaymentService
func (f *ServiceFactory) createPaymentServiceStub() payment_service.Service {
	return &paymentServiceStub{}
}

// paymentServiceStub заглушка для PaymentService
type paymentServiceStub struct{}

func (p *paymentServiceStub) Exec(params payment_service.PaymentParams) (payment_service.PaymentResult, error) {
	logger.Warn("🔄 PaymentService заглушка: %s для пользователя %d", params.Action, params.UserID)

	return payment_service.PaymentResult{
		Success: false,
		Message: "Payment service не инициализирован. Необходимо настроить зависимости в application layer.",
	}, nil
}

// Validate проверяет доступность зависимостей
func (f *ServiceFactory) Validate() bool {
	if f.userService == nil {
		logger.Warn("⚠️ ServiceFactory: UserService не доступен")
		return false
	}

	logger.Info("✅ ServiceFactory валидирована")
	return true
}

// GetUserService возвращает UserService (геттер для приватного поля)
func (f *ServiceFactory) GetUserService() *users.Service {
	return f.userService
}
