// application/services/orchestrator/layers/delivery.go
package layers

import (
	telegram_package "crypto-exchange-screener-bot/internal/delivery/telegram/package"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	events "crypto-exchange-screener-bot/internal/infrastructure/transport/event_bus"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
)

// DeliveryLayer слой доставки (Telegram)
type DeliveryLayer struct {
	*BaseLayer
	config          *config.Config
	coreLayer       *CoreLayer
	telegramPackage *telegram_package.TelegramDeliveryPackage
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

	// Создаем TelegramDeliveryPackage
	dl.telegramPackage = telegram_package.NewTelegramDeliveryPackage(
		telegram_package.TelegramDeliveryPackageDependencies{
			Config:      dl.config,
			CoreFactory: coreFactory,
			Exchange:    "BYBIT",
		},
	)

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
	if dl.telegramPackage != nil && dl.config.TelegramEnabled {
		if err := dl.telegramPackage.Start(); err != nil {
			dl.setError(err)
			return fmt.Errorf("не удалось запустить TelegramDeliveryPackage: %w", err)
		}
		logger.Info("🤖 Telegram бот запущен")
	} else if !dl.config.TelegramEnabled {
		logger.Info("⚠️ Telegram отключен в конфигурации, пропускаем запуск")
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
