// internal/delivery/telegram/services/factory/factory.go
package services_factory

import (
	"crypto-exchange-screener-bot/internal/core/domain/subscription"
	"crypto-exchange-screener-bot/internal/core/domain/users"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/buttons"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/formatters"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/message_sender"
	"crypto-exchange-screener-bot/internal/delivery/telegram/services/counter"
	"crypto-exchange-screener-bot/internal/delivery/telegram/services/notifications_toggle"
	"crypto-exchange-screener-bot/internal/delivery/telegram/services/profile"
	"crypto-exchange-screener-bot/internal/delivery/telegram/services/signal_settings"
	"crypto-exchange-screener-bot/pkg/logger"
)

// ServiceFactory фабрика сервисов уровня пакета Telegram
type ServiceFactory struct {
	userService         *users.Service
	subscriptionService *subscription.Service
	messageSender       message_sender.MessageSender
	buttonBuilder       *buttons.ButtonBuilder
	formatterProvider   *formatters.FormatterProvider
}

// ServiceDependencies зависимости для фабрики сервисов
type ServiceDependencies struct {
	UserService         *users.Service
	SubscriptionService *subscription.Service
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

// Validate проверяет доступность зависимостей
func (f *ServiceFactory) Validate() bool {
	if f.userService == nil {
		logger.Warn("⚠️ ServiceFactory: UserService не доступен")
		return false
	}

	logger.Info("✅ ServiceFactory валидирована")
	return true
}
