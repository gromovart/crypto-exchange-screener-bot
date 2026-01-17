// internal/delivery/telegram/package/package.go
package telegram_package

import (
	"fmt"
	"sync"

	"crypto-exchange-screener-bot/internal/core/domain/subscription"
	"crypto-exchange-screener-bot/internal/core/domain/users"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot"
	components_factory "crypto-exchange-screener-bot/internal/delivery/telegram/components/factory"
	controllers_factory "crypto-exchange-screener-bot/internal/delivery/telegram/controllers/factory"
	"crypto-exchange-screener-bot/internal/delivery/telegram/services/counter"
	services_factory "crypto-exchange-screener-bot/internal/delivery/telegram/services/factory"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	events "crypto-exchange-screener-bot/internal/infrastructure/transport/event_bus"
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
)

// TelegramDeliveryPackage основной пакет доставки через Telegram
type TelegramDeliveryPackage struct {
	mu                  sync.RWMutex
	config              *config.Config
	userService         *users.Service
	subscriptionService *subscription.Service
	eventBus            *events.EventBus

	// Подфабрики
	componentFactory  *components_factory.ComponentFactory
	serviceFactory    *services_factory.ServiceFactory
	controllerFactory *controllers_factory.ControllerFactory

	// Созданные компоненты
	components  components_factory.ComponentSet
	services    map[string]interface{}
	controllers map[string]types.EventSubscriber // ИЗМЕНЕНО

	// Telegram бот
	bot         *bot.TelegramBot
	isRunning   bool
	initialized bool
}

// TelegramDeliveryPackageDependencies зависимости для создания пакета
type TelegramDeliveryPackageDependencies struct {
	Config              *config.Config
	UserService         *users.Service
	SubscriptionService *subscription.Service
	Exchange            string
}

// NewTelegramDeliveryPackage создает новый пакет доставки Telegram
func NewTelegramDeliveryPackage(deps TelegramDeliveryPackageDependencies) *TelegramDeliveryPackage {
	logger.Info("📦 Создание TelegramDeliveryPackage...")

	if deps.Exchange == "" {
		deps.Exchange = "BYBIT"
	}

	return &TelegramDeliveryPackage{
		config:              deps.Config,
		userService:         deps.UserService,
		subscriptionService: deps.SubscriptionService,
		services:            make(map[string]interface{}),
		controllers:         make(map[string]types.EventSubscriber), // ИЗМЕНЕНО
	}
}

// Initialize инициализирует весь пакет
func (p *TelegramDeliveryPackage) Initialize(eventBus *events.EventBus) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.initialized {
		logger.Warn("⚠️ TelegramDeliveryPackage уже инициализирован")
		return nil
	}

	if eventBus == nil {
		return fmt.Errorf("EventBus не может быть nil")
	}

	p.eventBus = eventBus

	logger.Info("🔧 Инициализация TelegramDeliveryPackage...")

	// 1. Создаем ComponentFactory
	if err := p.createComponentFactory(); err != nil {
		return fmt.Errorf("ошибка создания ComponentFactory: %w", err)
	}

	// 2. Создаем ServiceFactory
	if err := p.createServiceFactory(); err != nil {
		return fmt.Errorf("ошибка создания ServiceFactory: %w", err)
	}

	// 3. Создаем сервисы
	if err := p.createServices(); err != nil {
		return fmt.Errorf("ошибка создания сервисов: %w", err)
	}

	// 4. Создаем ControllerFactory
	if err := p.createControllerFactory(); err != nil {
		return fmt.Errorf("ошибка создания ControllerFactory: %w", err)
	}

	// 5. Создаем контроллеры
	if err := p.createControllers(); err != nil {
		return fmt.Errorf("ошибка создания контроллеров: %w", err)
	}

	// 6. Создаем Telegram бота
	if err := p.createBot(); err != nil {
		return fmt.Errorf("ошибка создания бота: %w", err)
	}

	// 7. Подписываем контроллеры на EventBus
	p.subscribeControllersToEventBus()

	p.initialized = true
	logger.Info("✅ TelegramDeliveryPackage инициализирован")

	// 8. Автозапуск бота если Telegram включен
	if p.config.TelegramEnabled && p.config.TelegramBotToken != "" && p.bot != nil {
		go func() {
			if err := p.Start(); err != nil {
				logger.Error("❌ Ошибка автозапуска Telegram бота: %v", err)
			}
		}()
	}

	return nil
}

// createComponentFactory создает фабрику компонентов
func (p *TelegramDeliveryPackage) createComponentFactory() error {
	logger.Debug("🛠️  Создание ComponentFactory...")

	p.componentFactory = components_factory.NewComponentFactory(
		components_factory.ComponentDependencies{
			Config:   p.config,
			Exchange: "BYBIT",
		},
	)

	if !p.componentFactory.Validate() {
		return fmt.Errorf("ComponentFactory не валидна")
	}

	p.components = p.componentFactory.CreateAllComponents()
	logger.Info("✅ ComponentFactory создана")
	return nil
}

// createServiceFactory создает фабрику сервисов
func (p *TelegramDeliveryPackage) createServiceFactory() error {
	logger.Debug("🏭 Создание ServiceFactory...")

	// Компоненты уже имеют правильные типы (определены в ComponentSet)
	p.serviceFactory = services_factory.NewServiceFactory(
		services_factory.ServiceDependencies{
			UserService:         p.userService,
			SubscriptionService: p.subscriptionService,
			MessageSender:       p.components.MessageSender,
			ButtonBuilder:       p.components.ButtonBuilder,
			FormatterProvider:   p.components.FormatterProvider,
		},
	)

	if !p.serviceFactory.Validate() {
		return fmt.Errorf("ServiceFactory не валидна")
	}

	logger.Info("✅ ServiceFactory создана")
	return nil
}

// createServices создает все сервисы
func (p *TelegramDeliveryPackage) createServices() error {
	logger.Debug("🔧 Создание сервисов...")

	p.services["ProfileService"] = p.serviceFactory.CreateProfileService()
	p.services["CounterService"] = p.serviceFactory.CreateCounterService()
	p.services["NotificationToggleService"] = p.serviceFactory.CreateNotificationToggleService()
	p.services["SignalSettingsService"] = p.serviceFactory.CreateSignalSettingsService()

	// Проверяем что сервисы созданы
	for name, service := range p.services {
		if service == nil {
			return fmt.Errorf("сервис %s не создан", name)
		}
	}

	logger.Info("✅ Создано %d сервисов", len(p.services))
	return nil
}

// createControllerFactory создает фабрику контроллеров
func (p *TelegramDeliveryPackage) createControllerFactory() error {
	logger.Debug("🎛️  Создание ControllerFactory...")

	// Получаем CounterService
	counterService, ok := p.services["CounterService"].(counter.Service)
	if !ok {
		return fmt.Errorf("невозможно привести CounterService")
	}

	p.controllerFactory = controllers_factory.NewControllerFactory(
		controllers_factory.ControllerDependencies{
			CounterService: counterService,
		},
	)

	if !p.controllerFactory.Validate() {
		return fmt.Errorf("ControllerFactory не валидна")
	}

	logger.Info("✅ ControllerFactory создана")
	return nil
}

// createControllers создает все контроллеры
func (p *TelegramDeliveryPackage) createControllers() error {
	logger.Debug("🎮 Создание контроллеров...")

	p.controllers = p.controllerFactory.GetAllControllers()

	if len(p.controllers) == 0 {
		return fmt.Errorf("не создано ни одного контроллера")
	}

	logger.Info("✅ Создано %d контроллеров", len(p.controllers))
	return nil
}

// createBot создает Telegram бота
func (p *TelegramDeliveryPackage) createBot() error {
	logger.Debug("🤖 Создание Telegram бота...")

	if !p.config.TelegramEnabled {
		logger.Warn("⚠️ Telegram отключен в конфигурации")
		return nil
	}

	if p.config.TelegramBotToken == "" {
		logger.Warn("⚠️ Токен Telegram бота не указан")
		return nil
	}

	// Зависимости для бота
	deps := &bot.Dependencies{
		UserService: p.userService,
	}

	p.bot = bot.NewTelegramBot(p.config, deps)

	logger.Info("✅ Telegram бот создан")
	return nil
}

// subscribeControllersToEventBus подписывает контроллеры на события
func (p *TelegramDeliveryPackage) subscribeControllersToEventBus() {
	if p.eventBus == nil {
		logger.Warn("⚠️ EventBus не установлен, пропускаю подписку контроллеров")
		return
	}

	subscribedCount := 0
	for name, ctrl := range p.controllers {
		for _, eventType := range ctrl.GetSubscribedEvents() {
			p.eventBus.Subscribe(eventType, ctrl)
			subscribedCount++
			logger.Debug("✅ Контроллер %s подписан на %s", name, eventType)
		}
	}

	logger.Info("🎛️  Подписано %d контроллеров на EventBus", subscribedCount)
}

// Start запускает Telegram бота
func (p *TelegramDeliveryPackage) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isRunning {
		return fmt.Errorf("TelegramDeliveryPackage уже запущен")
	}

	if p.bot == nil {
		return fmt.Errorf("Telegram бот не инициализирован")
	}

	logger.Info("🚀 Запуск Telegram бота...")

	// Проверяем методы бота
	if botWithPolling, ok := interface{}(p.bot).(interface{ StartPolling() error }); ok {
		if err := botWithPolling.StartPolling(); err != nil {
			return fmt.Errorf("ошибка запуска бота: %w", err)
		}
	} else {
		// Пробуем общий метод Start если есть
		if botWithStart, ok := interface{}(p.bot).(interface{ Start() error }); ok {
			if err := botWithStart.Start(); err != nil {
				return fmt.Errorf("ошибка запуска бота: %w", err)
			}
		} else {
			return fmt.Errorf("бот не поддерживает методы запуска")
		}
	}

	p.isRunning = true
	logger.Info("✅ Telegram бот запущен")
	return nil
}

// Stop останавливает Telegram бота
func (p *TelegramDeliveryPackage) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isRunning {
		return nil
	}

	logger.Info("🛑 Остановка Telegram бота...")

	if p.bot != nil {
		// Проверяем методы бота
		if botWithPolling, ok := interface{}(p.bot).(interface{ StopPolling() error }); ok {
			if err := botWithPolling.StopPolling(); err != nil {
				logger.Warn("⚠️ Ошибка остановки бота: %v", err)
			}
		} else if botWithStop, ok := interface{}(p.bot).(interface{ Stop() error }); ok {
			if err := botWithStop.Stop(); err != nil {
				logger.Warn("⚠️ Ошибка остановки бота: %v", err)
			}
		}
	}

	p.isRunning = false
	logger.Info("✅ Telegram бот остановлен")
	return nil
}

// GetHealthStatus возвращает статус здоровья пакета
func (p *TelegramDeliveryPackage) GetHealthStatus() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	status := map[string]interface{}{
		"initialized":       p.initialized,
		"is_running":        p.isRunning,
		"bot_status":        "stopped",
		"services_count":    len(p.services),
		"controllers_count": len(p.controllers),
		"event_bus_linked":  p.eventBus != nil,
	}

	if p.bot != nil {
		status["bot_status"] = "created"
		if p.isRunning {
			status["bot_status"] = "running"
		}
	}

	return status
}

// GetService возвращает сервис по имени
func (p *TelegramDeliveryPackage) GetService(name string) interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.services[name]
}

// GetAllServices возвращает все сервисы
func (p *TelegramDeliveryPackage) GetAllServices() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make(map[string]interface{})
	for k, v := range p.services {
		result[k] = v
	}
	return result
}

// GetController возвращает контроллер по имени
func (p *TelegramDeliveryPackage) GetController(name string) types.EventSubscriber {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.controllers[name]
}

// GetAllControllers возвращает все контроллеры
func (p *TelegramDeliveryPackage) GetAllControllers() map[string]types.EventSubscriber {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make(map[string]types.EventSubscriber)
	for k, v := range p.controllers {
		result[k] = v
	}
	return result
}

// GetBot возвращает Telegram бота
func (p *TelegramDeliveryPackage) GetBot() *bot.TelegramBot {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.bot
}

// IsInitialized проверяет инициализацию пакета
func (p *TelegramDeliveryPackage) IsInitialized() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.initialized
}

// IsRunning проверяет работает ли пакет
func (p *TelegramDeliveryPackage) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.isRunning
}

// UnsubscribeControllers отписывает контроллеры от EventBus
func (p *TelegramDeliveryPackage) UnsubscribeControllers() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.eventBus = nil
	logger.Info("🛑 Контроллеры отписаны от EventBus")
}
