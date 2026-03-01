// application/services/orchestrator/layers/delivery.go
package layers

import (
	"context"
	"fmt"

	max_package "crypto-exchange-screener-bot/internal/delivery/max"
	max_bot "crypto-exchange-screener-bot/internal/delivery/max/bot"
	telegram_package "crypto-exchange-screener-bot/internal/delivery/telegram/package"
	notifySvc "crypto-exchange-screener-bot/internal/delivery/telegram/services/notifications_toggle"
	signalSvc "crypto-exchange-screener-bot/internal/delivery/telegram/services/signal_settings"
	redis_service "crypto-exchange-screener-bot/internal/infrastructure/cache/redis"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	events "crypto-exchange-screener-bot/internal/infrastructure/transport/event_bus"
	"crypto-exchange-screener-bot/pkg/logger"
)

// DeliveryLayer слой доставки (Telegram + MAX)
type DeliveryLayer struct {
	*BaseLayer
	config          *config.Config
	coreLayer       *CoreLayer
	telegramPackage *telegram_package.TelegramDeliveryPackage
	maxPackage      *max_package.Package
	maxBot          *max_bot.Bot
	maxBotCancel    context.CancelFunc
	initialized     bool
}

// NewDeliveryLayer создает слой доставки
func NewDeliveryLayer(cfg *config.Config, coreLayer *CoreLayer) *DeliveryLayer {
	layer := &DeliveryLayer{
		BaseLayer: NewBaseLayer("DeliveryLayer", []string{"CoreLayer"}),
		config:    cfg,
		coreLayer: coreLayer,
	}
	return layer
}

// SetDependencies устанавливает зависимости
func (dl *DeliveryLayer) SetDependencies(deps map[string]Layer) error {
	// Получаем слой ядра из зависимостей
	coreLayer, exists := deps["CoreLayer"]
	if !exists {
		return fmt.Errorf("зависимость CoreLayer не найдена")
	}

	// Приводим к правильному типу
	core, ok := coreLayer.(*CoreLayer)
	if !ok {
		return fmt.Errorf("неверный тип CoreLayer")
	}

	dl.coreLayer = core
	return nil
}

// Initialize инициализирует слой доставки
func (dl *DeliveryLayer) Initialize() error {
	if dl.initialized {
		return fmt.Errorf("слой доставки уже инициализирован")
	}

	// Проверяем зависимости
	if dl.coreLayer == nil {
		return fmt.Errorf("CoreLayer не установлен")
	}

	if !dl.coreLayer.IsInitialized() {
		return fmt.Errorf("CoreLayer не инициализирован")
	}

	dl.updateState(StateInitializing)
	logger.Info("📦 Инициализация слоя доставки...")

	// Получаем фабрику ядра
	coreFactory := dl.coreLayer.GetCoreFactory()
	if coreFactory == nil {
		return fmt.Errorf("фабрика ядра не создана")
	}

	// Получаем Redis клиент для очереди (опционально)
	var redisClient *redis_service.RedisService
	if redisComp, exists := dl.coreLayer.infraLayer.GetComponent("RedisService"); exists {
		if lc, ok := redisComp.(*LazyComponent); ok {
			if val, err := lc.Get(); err == nil {
				redisClient, _ = val.(*redis_service.RedisService)
			}
		}
	}

	// Создаем TelegramDeliveryPackage
	deps := telegram_package.TelegramDeliveryPackageDependencies{
		Config:      dl.config,
		CoreFactory: coreFactory,
		Exchange:    "BYBIT",
	}
	if redisClient != nil && redisClient.IsRunning() {
		deps.RedisClient = redisClient.GetClient()
		logger.Info("🔗 DeliveryLayer: Redis клиент передан в TelegramDeliveryPackage")
	} else {
		logger.Info("ℹ️  DeliveryLayer: Redis недоступен, очередь отключена")
	}

	dl.telegramPackage = telegram_package.NewTelegramDeliveryPackage(deps)

	// Получаем EventBus из InfrastructureLayer
	// Для этого нужно получить доступ к InfrastructureLayer через CoreLayer
	if dl.coreLayer.infraLayer == nil {
		return fmt.Errorf("InfrastructureLayer не доступен")
	}

	// Получаем EventBus из InfrastructureLayer
	eventBusComp, exists := dl.coreLayer.infraLayer.GetComponent("EventBus")
	if !exists {
		return fmt.Errorf("EventBus не найден в InfrastructureLayer")
	}

	// Приводим к правильному типу
	eventBus, ok := eventBusComp.(*events.EventBus)
	if !ok {
		// Если это LazyComponent, получаем его значение
		if lc, ok := eventBusComp.(*LazyComponent); ok {
			eventBusInterface, err := lc.Get()
			if err != nil {
				return fmt.Errorf("не удалось получить EventBus: %w", err)
			}
			eventBus, ok = eventBusInterface.(*events.EventBus)
			if !ok {
				return fmt.Errorf("неверный тип EventBus после получения из LazyComponent")
			}
		} else {
			return fmt.Errorf("неверный тип компонента EventBus")
		}
	}

	// Инициализируем TelegramDeliveryPackage с EventBus
	if err := dl.telegramPackage.Initialize(eventBus); err != nil {
		dl.setError(err)
		return fmt.Errorf("не удалось инициализировать TelegramDeliveryPackage: %w", err)
	}

	// Инициализируем MAX Package если включён
	if dl.config.MAX.Enabled {
		dl.maxPackage = max_package.NewPackage(dl.config.MAX.BotToken, dl.config.MAX.ChatID)

		if err := dl.maxPackage.Initialize(eventBus); err != nil {
			logger.Warn("⚠️ Не удалось инициализировать MAX Package: %v", err)
			dl.maxPackage = nil
		} else {
			logger.Info("📲 MAX Package инициализирован")

			// Создаём interactive-бота с UserService
			if userSvc, err := coreFactory.CreateUserService(); err == nil && userSvc != nil {
				deps := max_bot.Dependencies{
					UserService:   userSvc,
					NotifyService: notifySvc.NewServiceWithDependencies(userSvc),
					SignalService: signalSvc.NewServiceWithDependencies(userSvc),
				}
				dl.maxBot = max_bot.NewBot(dl.maxPackage.GetClient(), deps)
				logger.Info("🤖 MAX Bot создан")
			} else {
				logger.Warn("⚠️ MAX: не удалось получить UserService (%v) — interactive-бот не запустится", err)
			}
		}
	} else {
		logger.Info("ℹ️  MAX отключён в конфигурации, пропускаем")
	}

	// Регистрируем компоненты
	dl.registerDeliveryComponents()

	dl.initialized = true
	dl.updateState(StateInitialized)
	logger.Info("✅ Слой доставки инициализирован")
	return nil
}

// InitializeWithEventBus инициализирует слой доставки с EventBus
func (dl *DeliveryLayer) InitializeWithEventBus(eventBus interface{}) error {
	if !dl.initialized {
		return fmt.Errorf("слой доставки не инициализирован")
	}

	if dl.telegramPackage == nil {
		return fmt.Errorf("TelegramDeliveryPackage не создан")
	}

	logger.Info("🔌 Инициализация слоя доставки с EventBus...")

	// Приводим EventBus к правильному типу
	eventBusTyped, ok := eventBus.(*events.EventBus)
	if !ok {
		return fmt.Errorf("неверный тип EventBus: ожидается *events.EventBus")
	}

	if err := dl.telegramPackage.Initialize(eventBusTyped); err != nil {
		dl.setError(err)
		return fmt.Errorf("не удалось инициализировать TelegramDeliveryPackage: %w", err)
	}

	logger.Info("✅ Слой доставки инициализирован с EventBus")
	return nil
}

// Start запускает слой доставки
func (dl *DeliveryLayer) Start() error {
	if !dl.initialized {
		return fmt.Errorf("слой доставки не инициализирован")
	}

	if dl.IsRunning() {
		return fmt.Errorf("слой доставки уже запущен")
	}

	dl.updateState(StateStarting)
	logger.Info("🚀 Запуск слоя доставки...")

	// Запускаем TelegramDeliveryPackage если он создан
	if dl.telegramPackage != nil && dl.config.Telegram.Enabled {
		if err := dl.telegramPackage.Start(); err != nil {
			dl.setError(err)
			return fmt.Errorf("не удалось запустить TelegramDeliveryPackage: %w", err)
		}
		logger.Info("🤖 Telegram бот запущен")
	} else if !dl.config.Telegram.Enabled {
		logger.Info("⚠️ Telegram отключен в конфигурации, пропускаем запуск")
	}

	// Запускаем MAX Package если инициализирован
	if dl.maxPackage != nil {
		if err := dl.maxPackage.Start(); err != nil {
			logger.Warn("⚠️ Не удалось запустить MAX Package: %v", err)
		} else {
			logger.Info("📲 MAX Package запущен")
		}
	}

	// Запускаем MAX Bot (polling) если создан
	if dl.maxBot != nil {
		ctx, cancel := context.WithCancel(context.Background())
		dl.maxBotCancel = cancel
		go dl.maxBot.Start(ctx)
		logger.Info("🤖 MAX Bot запущен (polling)")
	}

	dl.running = true
	dl.updateState(StateRunning)
	logger.Info("✅ Слой доставки запущен")
	return nil
}

// Stop останавливает слой доставки
func (dl *DeliveryLayer) Stop() error {
	if !dl.IsRunning() {
		return nil
	}

	dl.updateState(StateStopping)
	logger.Info("🛑 Остановка слоя доставки...")

	// Останавливаем TelegramDeliveryPackage
	if dl.telegramPackage != nil {
		if err := dl.telegramPackage.Stop(); err != nil {
			logger.Warn("⚠️ Ошибка остановки TelegramDeliveryPackage: %v", err)
		}
		logger.Info("🤖 Telegram бот остановлен")
	}

	// Останавливаем MAX Bot
	if dl.maxBotCancel != nil {
		dl.maxBotCancel()
		dl.maxBotCancel = nil
		logger.Info("🤖 MAX Bot остановлен")
	}

	// Останавливаем MAX Package
	if dl.maxPackage != nil {
		dl.maxPackage.Stop()
		logger.Info("📲 MAX Package остановлен")
	}

	dl.running = false
	dl.updateState(StateStopped)
	logger.Info("✅ Слой доставки остановлен")
	return nil
}

// Reset сбрасывает слой доставки
func (dl *DeliveryLayer) Reset() error {
	logger.Info("🔄 Сброс слоя доставки...")

	// Останавливаем если запущен
	if dl.IsRunning() {
		dl.Stop()
	}

	// Сбрасываем Telegram пакет
	if dl.telegramPackage != nil {
		dl.telegramPackage.Reset()
		dl.telegramPackage.UnsubscribeControllers()
	}

	// Сбрасываем базовый слой
	dl.BaseLayer.Reset()

	// Сбрасываем MAX Bot
	if dl.maxBotCancel != nil {
		dl.maxBotCancel()
		dl.maxBotCancel = nil
	}
	dl.maxBot = nil

	// Сбрасываем MAX Package
	if dl.maxPackage != nil {
		dl.maxPackage.Stop()
		dl.maxPackage = nil
	}

	dl.telegramPackage = nil
	dl.initialized = false
	logger.Info("✅ Слой доставки сброшен")
	return nil
}

// IsInitialized проверяет инициализацию
func (dl *DeliveryLayer) IsInitialized() bool {
	return dl.initialized
}

// GetTelegramPackage возвращает TelegramDeliveryPackage
func (dl *DeliveryLayer) GetTelegramPackage() *telegram_package.TelegramDeliveryPackage {
	dl.mu.RLock()
	defer dl.mu.RUnlock()
	return dl.telegramPackage
}

// registerDeliveryComponents регистрирует компоненты доставки
func (dl *DeliveryLayer) registerDeliveryComponents() {
	if dl.telegramPackage == nil {
		return
	}

	// Регистрируем компоненты доставки
	components := map[string]string{
		"TelegramDeliveryPackage": "пакет доставки Telegram",
		"TelegramBot":             "Telegram бот",
	}

	for name, description := range components {
		dl.registerComponent(name, &LazyComponent{
			name:        name,
			description: description,
			getter:      dl.getDeliveryComponent(name),
		})
		logger.Debug("📱 Зарегистрирован компонент доставки: %s (%s)", name, description)
	}
}

// getDeliveryComponent возвращает геттер для компонента доставки
func (dl *DeliveryLayer) getDeliveryComponent(name string) func() (interface{}, error) {
	return func() (interface{}, error) {
		if dl.telegramPackage == nil {
			return nil, fmt.Errorf("TelegramDeliveryPackage не создан")
		}

		switch name {
		case "TelegramDeliveryPackage":
			return dl.telegramPackage, nil
		case "TelegramBot":
			return dl.telegramPackage.GetBot(), nil
		default:
			return nil, fmt.Errorf("неизвестный компонент доставки: %s", name)
		}
	}
}

// GetTelegramBot возвращает Telegram бота (ленивое создание)
func (dl *DeliveryLayer) GetTelegramBot() (interface{}, error) {
	comp, exists := dl.GetComponent("TelegramBot")
	if !exists {
		return nil, fmt.Errorf("TelegramBot не зарегистрирован")
	}

	lc, ok := comp.(*LazyComponent)
	if !ok {
		return nil, fmt.Errorf("неверный тип компонента TelegramBot")
	}

	return lc.Get()
}
