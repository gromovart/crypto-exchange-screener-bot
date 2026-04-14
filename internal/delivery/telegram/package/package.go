// internal/delivery/telegram/package/package.go
package telegram_package

import (
	"fmt"
	"sync"

	"crypto-exchange-screener-bot/internal/core/domain/payment"
	"crypto-exchange-screener-bot/internal/core/domain/subscription"
	"crypto-exchange-screener-bot/internal/core/domain/users"
	core_factory "crypto-exchange-screener-bot/internal/core/package"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/message_sender"
	components_factory "crypto-exchange-screener-bot/internal/delivery/telegram/components/factory"
	controllers_factory "crypto-exchange-screener-bot/internal/delivery/telegram/controllers/factory"
	"crypto-exchange-screener-bot/internal/delivery/telegram/queue"
	"crypto-exchange-screener-bot/internal/delivery/telegram/services/counter"
	services_factory "crypto-exchange-screener-bot/internal/delivery/telegram/services/factory"
	watchlist_service "crypto-exchange-screener-bot/internal/delivery/telegram/services/watchlist"

	trading_session "crypto-exchange-screener-bot/internal/delivery/telegram/services/trading_session"
	"crypto-exchange-screener-bot/internal/delivery/telegram/transport"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	events "crypto-exchange-screener-bot/internal/infrastructure/transport/event_bus"
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"

	goredis "github.com/go-redis/redis/v8"
)

// TelegramDeliveryPackage основной пакет доставки через Telegram
type TelegramDeliveryPackage struct {
	mu          sync.RWMutex
	config      *config.Config
	coreFactory *core_factory.CoreServiceFactory
	eventBus    *events.EventBus
	redisClient *goredis.Client // опционально, для очереди

	// Созданные сервисы ядра (ленивое создание)
	userService         *users.Service
	subscriptionService *subscription.Service
	paymentService      *payment.PaymentService // ⭐ Новый сервис платежей

	// Сервис торговых сессий (создаётся один раз, разделяется между ботом и CounterService)
	tradingSessionService trading_session.Service

	// Подфабрики
	componentFactory  *components_factory.ComponentFactory
	serviceFactory    *services_factory.ServiceFactory
	controllerFactory *controllers_factory.ControllerFactory

	// Созданные компоненты
	components  components_factory.ComponentSet
	services    map[string]interface{}
	controllers map[string]types.EventSubscriber

	// Queue worker (nil если Redis недоступен)
	worker *queue.Worker

	// Watchlist service (опционально)
	watchlistService watchlist_service.Service

	// Telegram бот и транспорт
	bot         *bot.TelegramBot
	transport   transport.TelegramTransport
	isRunning   bool
	initialized bool
}

// TelegramDeliveryPackageDependencies зависимости для создания пакета
type TelegramDeliveryPackageDependencies struct {
	Config           *config.Config
	CoreFactory      *core_factory.CoreServiceFactory
	Exchange         string
	RedisClient      *goredis.Client           // опционально, для Redis-очереди
	WatchlistService watchlist_service.Service // опционально, для вотчлиста
}

// NewTelegramDeliveryPackage создает новый пакет доставки Telegram
func NewTelegramDeliveryPackage(deps TelegramDeliveryPackageDependencies) *TelegramDeliveryPackage {
	logger.Info("📦 Создание TelegramDeliveryPackage...")

	if deps.Exchange == "" {
		deps.Exchange = "BYBIT"
	}

	return &TelegramDeliveryPackage{
		config:           deps.Config,
		coreFactory:      deps.CoreFactory,
		redisClient:      deps.RedisClient,
		watchlistService: deps.WatchlistService,
		services:         make(map[string]interface{}),
		controllers:      make(map[string]types.EventSubscriber),
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

	// 1. Создаем сервисы ядра через фабрику (лениво)
	if err := p.createCoreServices(); err != nil {
		return fmt.Errorf("ошибка создания сервисов ядра: %w", err)
	}

	// 2. Создаем ComponentFactory
	if err := p.createComponentFactory(); err != nil {
		return fmt.Errorf("ошибка создания ComponentFactory: %w", err)
	}

	// 3. Создаем ServiceFactory
	if err := p.createServiceFactory(); err != nil {
		return fmt.Errorf("ошибка создания ServiceFactory: %w", err)
	}

	// 4. Создаем сервисы Telegram
	if err := p.createServices(); err != nil {
		return fmt.Errorf("ошибка создания сервисов: %w", err)
	}

	// 5. Создаем ControllerFactory
	if err := p.createControllerFactory(); err != nil {
		return fmt.Errorf("ошибка создания ControllerFactory: %w", err)
	}

	// 6. Создаем контроллеры
	if err := p.createControllers(); err != nil {
		return fmt.Errorf("ошибка создания контроллеров: %w", err)
	}

	// 7. Создаем Telegram бота и транспорт
	if err := p.createBotAndTransport(); err != nil {
		return fmt.Errorf("ошибка создания бота и транспорта: %w", err)
	}

	// 8. Подписываем контроллеры на EventBus
	p.subscribeControllersToEventBus()

	p.initialized = true
	logger.Info("✅ TelegramDeliveryPackage инициализирован")

	return nil
}

// createCoreServices создает сервисы ядра через CoreServiceFactory
func (p *TelegramDeliveryPackage) createCoreServices() error {
	logger.Debug("🏗️  Проверка готовности CoreServiceFactory...")

	if p.coreFactory == nil {
		return fmt.Errorf("CoreServiceFactory не установлена")
	}

	if !p.coreFactory.IsReady() {
		return fmt.Errorf("CoreServiceFactory не готова")
	}

	// НЕ создаем сервисы сейчас - только проверяем готовность
	// Сервисы будут созданы лениво при первом обращении
	logger.Info("✅ CoreServiceFactory готова (ленивое создание сервисов)")
	return nil
}

// createComponentFactory создает фабрику компонентов
func (p *TelegramDeliveryPackage) createComponentFactory() error {
	logger.Debug("🛠️  Создание ComponentFactory...")

	p.componentFactory = components_factory.NewComponentFactory(
		components_factory.ComponentDependencies{
			Config:      p.config,
			Exchange:    "BYBIT",
			RedisClient: p.redisClient,
		},
	)

	if !p.componentFactory.Validate() {
		return fmt.Errorf("ComponentFactory не валидна")
	}

	p.components = p.componentFactory.CreateAllComponents()
	p.worker = p.components.Worker
	if p.worker != nil {
		logger.Info("✅ Queue worker создан")
	} else {
		logger.Info("ℹ️  Queue worker не создан (Redis недоступен)")
	}
	logger.Info("✅ ComponentFactory создана")
	return nil
}

// createServiceFactory создает фабрику сервисов Telegram
func (p *TelegramDeliveryPackage) createServiceFactory() error {
	logger.Debug("🏭 Создание ServiceFactory...")

	// Получаем UserService из CoreFactory
	userService, err := p.getUserService()
	if err != nil {
		return fmt.Errorf("не удалось получить UserService: %w", err)
	}

	// Получаем SubscriptionService
	subscriptionService, err := p.getSubscriptionService()
	if err != nil {
		logger.Warn("⚠️ SubscriptionService не доступен: %v", err)
	}

	// ⭐ Получаем PaymentService (новый сервис ядра)
	paymentSvc, err := p.getPaymentService()
	if err != nil {
		logger.Warn("⚠️ PaymentService не доступен: %v", err)
		paymentSvc = nil
	}

	// ⭐ Проверяем что все зависимости есть
	if paymentSvc == nil {
		logger.Error("❌ PaymentService не создан, будет использоваться заглушка")
	}

	// Создаём сервис торговых сессий один раз — используется и ботом, и CounterService
	ms := message_sender.NewMessageSender(p.config)
	p.tradingSessionService = trading_session.NewService(userService, ms)
	logger.Info("✅ TradingSessionService создан")

	var signalPublisher counter.SignalPublisher
	if p.redisClient != nil {
		signalPublisher = counter.NewRedisSignalPublisher(p.redisClient)
		logger.Info("📡 SignalPublisher: Redis Pub/Sub активирован (канал: %s)", counter.ScreenerSignalChannel)
	} else {
		logger.Info("ℹ️  SignalPublisher: Redis недоступен, публикация сигналов отключена")
	}

	p.serviceFactory = services_factory.NewServiceFactory(
		services_factory.ServiceDependencies{
			UserService:           userService,
			SubscriptionService:   subscriptionService,
			PaymentCoreService:    paymentSvc,
			MessageSender:         p.components.MessageSender,
			ButtonBuilder:         p.components.ButtonBuilder,
			FormatterProvider:     p.components.FormatterProvider,
			TradingSessionService: p.tradingSessionService,
			SignalPublisher:       signalPublisher,
		},
	)

	if !p.serviceFactory.Validate() {
		return fmt.Errorf("ServiceFactory не валидна")
	}

	logger.Info("✅ ServiceFactory создана")
	return nil
}

// getUserService получает UserService из CoreFactory
func (p *TelegramDeliveryPackage) getUserService() (*users.Service, error) {
	if p.userService != nil {
		return p.userService, nil
	}

	if p.coreFactory == nil {
		return nil, fmt.Errorf("CoreServiceFactory не установлена")
	}

	// Создаем UserService через фабрику
	userService, err := p.coreFactory.CreateUserService()
	if err != nil {
		return nil, fmt.Errorf("не удалось создать UserService: %w", err)
	}

	p.userService = userService
	logger.Info("✅ UserService создан и сохранен в пакете")
	return p.userService, nil
}

// getSubscriptionService получает SubscriptionService из CoreFactory
func (p *TelegramDeliveryPackage) getSubscriptionService() (*subscription.Service, error) {
	if p.subscriptionService != nil {
		return p.subscriptionService, nil
	}

	if p.coreFactory == nil {
		return nil, fmt.Errorf("CoreServiceFactory не установлена")
	}

	// Создаем SubscriptionService через фабрику
	subscriptionService, err := p.coreFactory.CreateSubscriptionService()
	if err != nil {
		logger.Error("❌ Не удалось создать SubscriptionService: %v", err)
		return nil, fmt.Errorf("не удалось создать SubscriptionService: %w", err)
	}

	p.subscriptionService = subscriptionService
	logger.Info("✅ SubscriptionService создан и сохранен в пакете")
	return p.subscriptionService, nil
}

// ⭐ НОВЫЙ МЕТОД: getPaymentService получает PaymentService из CoreFactory
func (p *TelegramDeliveryPackage) getPaymentService() (*payment.PaymentService, error) {
	if p.paymentService != nil {
		return p.paymentService, nil
	}

	if p.coreFactory == nil {
		return nil, fmt.Errorf("CoreServiceFactory не установлена")
	}

	// Создаем PaymentService через фабрику
	logger.Info("🔍 Создание PaymentService через CoreFactory...")
	paymentSvc, err := p.coreFactory.CreatePaymentService()
	if err != nil {
		logger.Error("❌ Не удалось создать PaymentService: %v", err)
		return nil, fmt.Errorf("не удалось создать PaymentService: %w", err)
	}

	if paymentSvc == nil {
		logger.Error("❌ CreatePaymentService вернул nil")
		return nil, fmt.Errorf("CreatePaymentService вернул nil")
	}

	p.paymentService = paymentSvc
	logger.Info("✅ PaymentService создан и сохранен в пакете")
	return p.paymentService, nil
}

// createServices создает все сервисы Telegram
func (p *TelegramDeliveryPackage) createServices() error {
	logger.Debug("🔧 Создание сервисов Telegram...")

	p.services["ProfileService"] = p.serviceFactory.CreateProfileService()
	p.services["CounterService"] = p.serviceFactory.CreateCounterService()
	p.services["NotificationToggleService"] = p.serviceFactory.CreateNotificationToggleService()
	p.services["SignalSettingsService"] = p.serviceFactory.CreateSignalSettingsService()

	// Создаем PaymentService
	p.services["PaymentService"] = p.serviceFactory.CreatePaymentService()

	// Проверяем что сервисы созданы
	for name, service := range p.services {
		if service == nil {
			logger.Warn("⚠️ Сервис %s не создан", name)
		}
	}

	logger.Info("✅ Создано %d сервисов Telegram", len(p.services))
	return nil
}

// createControllerFactory создает фабрику контроллеров
func (p *TelegramDeliveryPackage) createControllerFactory() error {
	logger.Debug("🎛️  Создание ControllerFactory...")

	// Получаем CounterService
	counterService, ok := p.services["CounterService"].(counter.Service)
	if !ok {
		return fmt.Errorf("невозможно привести CounterService к правильному типу")
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

// createBotAndTransport создает Telegram бота и транспорт
func (p *TelegramDeliveryPackage) createBotAndTransport() error {
	logger.Debug("🤖 Создание Telegram бота и транспорта...")

	if !p.config.Telegram.Enabled {
		logger.Warn("⚠️ Telegram отключен в конфигурации")
		return nil
	}

	if p.config.TelegramBotToken == "" {
		logger.Warn("⚠️ Токен Telegram бота не указан")
		return nil
	}

	// Проверяем что ServiceFactory создана
	if p.serviceFactory == nil {
		return fmt.Errorf("ServiceFactory не создана")
	}

	// Создаем зависимости для бота
	deps := &bot.Dependencies{
		ServiceFactory:   p.serviceFactory,
		WatchlistService: p.watchlistService,
	}

	// Создаем бота
	p.bot = bot.NewTelegramBot(p.config, deps)

	// Создаем транспорт на основе конфигурации
	transportFactory := transport.NewTransportFactory(p.config, p.bot)
	transport, err := transportFactory.CreateTransport()
	if err != nil {
		return fmt.Errorf("ошибка создания транспорта: %w", err)
	}

	p.transport = transport

	logger.Info("✅ Telegram бот создан (режим: %s)", p.config.TelegramMode)
	logger.Info("✅ Транспорт создан: %s", p.transport.Name())
	return nil
}

// subscribeControllersToEventBus подписывает контроллеры на события
func (p *TelegramDeliveryPackage) subscribeControllersToEventBus() int {
	if p.eventBus == nil {
		logger.Warn("⚠️ EventBus не установлен, пропускаю подписку контроллеров")
		return 0
	}

	subscribedCount := 0
	for name, ctrl := range p.controllers {
		for _, eventType := range ctrl.GetSubscribedEvents() {
			p.eventBus.Subscribe(eventType, ctrl)
			subscribedCount++
			logger.Debug("✅ Контроллер %s подписан на %s", name, eventType)
		}
	}

	if subscribedCount > 0 {
		logger.Info("🎛️  Подписано %d контроллеров на EventBus", subscribedCount)
	}

	return subscribedCount
}

// Start запускает Telegram бота через транспорт
func (p *TelegramDeliveryPackage) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isRunning {
		return fmt.Errorf("TelegramDeliveryPackage уже запущен")
	}

	if p.transport == nil {
		return fmt.Errorf("транспорт не инициализирован")
	}

	logger.Info("🚀 Запуск Telegram бота (режим: %s, транспорт: %s)...",
		p.config.TelegramMode, p.transport.Name())

	// В вебхук-режиме нужно убедиться что контроллеры подписаны на EventBus
	if p.config.IsWebhookMode() {
		logger.Info("🔗 Вебхук-режим: проверка подписки контроллеров на EventBus...")

		if p.eventBus == nil {
			logger.Error("❌ EventBus не установлен в вебхук-режиме - уведомления о сигналах не будут работать!")
		} else {
			// Переподписываем контроллеры на EventBus
			subscribedCount := p.subscribeControllersToEventBus()
			if subscribedCount == 0 {
				logger.Warn("⚠️ Ни один контроллер не был подписан в вебхук-режиме")
			} else {
				logger.Info("✅ %d контроллеров подписано на EventBus (вебхук)", subscribedCount)
			}
		}
	} else {
		// В polling режиме подписки уже выполнены при инициализации
		logger.Debug("🔗 Polling-режим: подписки контроллеров уже установлены при инициализации")
	}

	// Запускаем через транспорт
	if err := p.transport.Start(); err != nil {
		return fmt.Errorf("ошибка запуска транспорта %s: %w", p.transport.Name(), err)
	}

	// Запускаем queue worker если доступен
	if p.worker != nil {
		p.worker.Start()
	}

	p.isRunning = true
	logger.Info("✅ Telegram бот запущен через %s", p.transport.Name())

	// Логируем информацию о контроллерах
	p.logControllerInfo()

	return nil
}

// logControllerInfo логирует информацию о контроллерах
func (p *TelegramDeliveryPackage) logControllerInfo() {
	if len(p.controllers) == 0 {
		logger.Warn("⚠️ TelegramDeliveryPackage: контроллеры не созданы")
		return
	}

	logger.Info("🎛️  Информация о контроллерах Telegram:")
	for name, ctrl := range p.controllers {
		events := ctrl.GetSubscribedEvents()
		if len(events) > 0 {
			logger.Info("   • %s: подписан на %v", name, events)

			// Проверяем подписку на EventBus
			for _, eventType := range events {
				if p.eventBus != nil {
					subscriberCount := p.eventBus.GetSubscriberCount(eventType)
					logger.Debug("     - %s: %d подписчиков в EventBus", eventType, subscriberCount)
				}
			}
		} else {
			logger.Info("   • %s: нет подписок на события", name)
		}
	}
}

// Stop останавливает Telegram бота через транспорт
func (p *TelegramDeliveryPackage) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isRunning {
		return nil
	}

	logger.Info("🛑 Остановка Telegram бота (транспорт: %s)...", p.transport.Name())

	if p.transport != nil {
		if err := p.transport.Stop(); err != nil {
			logger.Warn("⚠️ Ошибка остановки транспорта %s: %v", p.transport.Name(), err)
		}
	}

	// Останавливаем queue worker
	if p.worker != nil {
		p.worker.Stop()
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
		"initialized":          p.initialized,
		"is_running":           p.isRunning,
		"bot_status":           "stopped",
		"transport_status":     "none",
		"transport_type":       "none",
		"services_count":       len(p.services),
		"controllers_count":    len(p.controllers),
		"event_bus_linked":     p.eventBus != nil,
		"core_factory_ready":   p.coreFactory != nil && p.coreFactory.IsReady(),
		"user_service":         p.userService != nil,
		"subscription_service": p.subscriptionService != nil,
		"payment_service":      p.services["PaymentService"] != nil,
		"telegram_mode":        p.config.TelegramMode,
	}

	if p.bot != nil {
		status["bot_status"] = "created"
		if p.isRunning {
			status["bot_status"] = "running"
		}
	}

	if p.transport != nil {
		status["transport_status"] = "stopped"
		if p.transport.IsRunning() {
			status["transport_status"] = "running"
		}
		status["transport_type"] = string(p.transport.Type())
		status["transport_name"] = p.transport.Name()
	}

	return status
}

// GetService возвращает сервис по имени
func (p *TelegramDeliveryPackage) GetService(name string) interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.services[name]
}

// GetAllServices возвращает все сервисы Telegram
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

// GetTransport возвращает транспорт
func (p *TelegramDeliveryPackage) GetTransport() transport.TelegramTransport {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.transport
}

// GetCoreFactory возвращает фабрику ядра
func (p *TelegramDeliveryPackage) GetCoreFactory() *core_factory.CoreServiceFactory {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.coreFactory
}

// UpdateCoreFactory обновляет фабрику ядра
func (p *TelegramDeliveryPackage) UpdateCoreFactory(newFactory *core_factory.CoreServiceFactory) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if newFactory == nil {
		return fmt.Errorf("новая фабрика не может быть nil")
	}

	p.coreFactory = newFactory

	// Сбрасываем созданные сервисы ядра, чтобы пересоздать с новой фабрикой
	p.userService = nil
	p.subscriptionService = nil
	p.paymentService = nil

	logger.Info("✅ Фабрика ядра обновлена")
	return nil
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

// Reset сбрасывает состояние пакета
func (p *TelegramDeliveryPackage) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Останавливаем транспорт если работает
	if p.transport != nil && p.transport.IsRunning() {
		p.transport.Stop()
	}

	p.services = make(map[string]interface{})
	p.controllers = make(map[string]types.EventSubscriber)
	p.bot = nil
	p.transport = nil
	p.worker = nil
	p.isRunning = false
	p.initialized = false
	p.userService = nil
	p.subscriptionService = nil
	p.paymentService = nil

	logger.Info("🔄 TelegramDeliveryPackage сброшен")
}
