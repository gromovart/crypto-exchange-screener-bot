// application/layer_manager/layers/core.go
package layers

import (
	"crypto-exchange-screener-bot/internal/core/domain/candle"
	"crypto-exchange-screener-bot/internal/core/domain/fetchers"
	"crypto-exchange-screener-bot/internal/core/domain/payment"
	engine "crypto-exchange-screener-bot/internal/core/domain/signals/engine"
	"crypto-exchange-screener-bot/internal/core/domain/subscription"
	"crypto-exchange-screener-bot/internal/core/domain/users"
	core_factory "crypto-exchange-screener-bot/internal/core/package"
	redis_service "crypto-exchange-screener-bot/internal/infrastructure/cache/redis"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	redis_storage_factory "crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage/factory"
	sr_storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage/sr_storage"
	events "crypto-exchange-screener-bot/internal/infrastructure/transport/event_bus"
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"sync"
	"time"

	sr_engine "crypto-exchange-screener-bot/internal/core/domain/analysis/sr_engine"
)

// CoreLayer слой ядра (бизнес-логика)
type CoreLayer struct {
	*BaseLayer
	config            *config.Config
	infraLayer        *InfrastructureLayer
	coreFactory       *core_factory.CoreServiceFactory
	initialized       bool
	bybitPriceFetcher *fetchers.BybitPriceFetcher
	fetcherFactory    *fetchers.MarketFetcherFactory
	candleSystem      *candle.CandleSystem
	analysisEngine    *engine.AnalysisEngine
	srZoneEngine      *sr_engine.Engine
	srZoneStorage     *sr_storage.SRZoneStorage
}

// NewCoreLayer создает слой ядра
func NewCoreLayer(cfg *config.Config, infraLayer *InfrastructureLayer) *CoreLayer {
	layer := &CoreLayer{
		BaseLayer:  NewBaseLayer("CoreLayer", []string{"InfrastructureLayer"}),
		config:     cfg,
		infraLayer: infraLayer,
	}
	return layer
}

// SetDependencies устанавливает зависимости
func (cl *CoreLayer) SetDependencies(deps map[string]Layer) error {
	// Получаем слой инфраструктуры из зависимостей
	infraLayer, exists := deps["InfrastructureLayer"]
	if !exists {
		return fmt.Errorf("зависимость InfrastructureLayer не найдена")
	}

	// Приводим к правильному типу
	infra, ok := infraLayer.(*InfrastructureLayer)
	if !ok {
		return fmt.Errorf("неверный тип InfrastructureLayer")
	}

	cl.infraLayer = infra
	return nil
}

// Initialize инициализирует слой ядра
func (cl *CoreLayer) Initialize() error {
	if cl.initialized {
		return fmt.Errorf("слой ядра уже инициализирован")
	}

	// Проверяем зависимости
	if cl.infraLayer == nil {
		return fmt.Errorf("InfrastructureLayer не установлен")
	}

	if !cl.infraLayer.IsInitialized() {
		return fmt.Errorf("InfrastructureLayer не инициализирован")
	}

	cl.updateState(StateInitializing)
	logger.Info("🧠 Инициализация слоя ядра...")

	// Получаем фабрику инфраструктуры
	infraFactory := cl.infraLayer.GetInfrastructureFactory()
	if infraFactory == nil {
		return fmt.Errorf("фабрика инфраструктуры не создана")
	}

	// Создаем конфигурацию для фабрики ядра со значениями по умолчанию
	coreConfig := &core_factory.Config{
		Environment: cl.config.Environment, // ⭐ Передаем окружение
		UserConfig: users.Config{
			UserDefaults: struct {
				MinGrowthThreshold float64
				MinFallThreshold   float64
				Language           string
				Timezone           string
			}{
				MinGrowthThreshold: 2.0,
				MinFallThreshold:   2.0,
				Language:           "ru",
				Timezone:           "Europe/Moscow",
			},
			DefaultMaxSignalsPerDay: 1500,
			SessionTTL:              24 * time.Hour,
			MaxSessionsPerUser:      5,
		},
		SubscriptionConfig: subscription.Config{
			DefaultPlan:     "free",
			TrialPeriodDays: 1,
			GracePeriodDays: 3,
			AutoRenew:       true,
			IsDev:           cl.config.IsDev(),
		},
		PaymentsConfig: payment.Config{
			TelegramBotToken:           cl.config.Telegram.BotToken,
			TelegramStarsProviderToken: "",
			TelegramBotUsername:        cl.config.Telegram.BotUsername,
		},
	}

	// Создаем фабрику ядра
	deps := core_factory.CoreServiceDependencies{
		InfrastructureFactory: infraFactory,
		Config:                coreConfig,
		UserNotifier:          nil,
		Analytics:             nil,
	}

	var err error
	cl.coreFactory, err = core_factory.NewCoreServiceFactory(deps)
	if err != nil {
		cl.setError(err)
		return fmt.Errorf("не удалось создать фабрику ядра: %w", err)
	}

	// Инициализируем фабрику ядра
	if err := cl.coreFactory.Initialize(); err != nil {
		cl.setError(err)
		return fmt.Errorf("не удалось инициализировать фабрику ядра: %w", err)
	}

	// СОЗДАЕМ UserService СРАЗУ (не лениво) для ServiceFactory
	userService, err := cl.coreFactory.CreateUserService()
	if err != nil {
		cl.setError(err)
		return fmt.Errorf("не удалось создать UserService: %w", err)
	}

	// Сохраняем UserService для быстрого доступа
	cl.registerComponent("UserService", userService)
	logger.Info("✅ UserService создан и зарегистрирован")

	// НОВОЕ: Создаем фабрику фетчеров
	cl.fetcherFactory = fetchers.NewMarketFetcherFactory(cl.config)
	logger.Info("🏭 Фабрика MarketFetcher создана")

	// НОВОЕ: Создаем свечную систему (пока без priceStorage - создадим в Start)
	logger.Info("🕯️ CoreLayer: подготовка свечной системы...")

	// Регистрируем остальные компоненты
	cl.registerCoreComponents()

	cl.initialized = true
	cl.updateState(StateInitialized)
	logger.Info("✅ Слой ядра инициализирован")
	return nil
}

// Start запускает слой ядра
func (cl *CoreLayer) Start() error {
	if !cl.initialized {
		return fmt.Errorf("слой ядра не инициализирован")
	}

	if cl.IsRunning() {
		return fmt.Errorf("слой ядра уже запущен")
	}

	cl.updateState(StateStarting)
	logger.Info("🚀 Запуск слоя ядра...")

	// НОВОЕ: Запускаем свечную систему если включен Telegram
	if cl.config.Telegram.Enabled && cl.infraLayer != nil {
		if err := cl.setupAndStartCandleSystem(); err != nil {
			logger.Warn("⚠️ Не удалось запустить свечную систему: %v", err)
		}
	}

	// НОВОЕ: Запускаем BybitPriceFetcher если включен Telegram
	if cl.config.Telegram.Enabled && cl.infraLayer != nil {
		cl.startBybitPriceFetcher()
	}

	// НОВОЕ: Запускаем SRZoneEngine если включен Telegram
	if cl.config.Telegram.Enabled && cl.infraLayer != nil {
		if err := cl.startSRZoneEngine(); err != nil {
			logger.Warn("⚠️ Не удалось запустить SRZoneEngine: %v (зоны S/R будут недоступны)", err)
		}
	}

	// НОВОЕ: Запускаем AnalysisEngine если CounterAnalyzer включен в конфигурации
	if cl.config.Telegram.Enabled && cl.infraLayer != nil {
		logger.Info("🔧 Проверка условий запуска AnalysisEngine:")
		logger.Info("   - TelegramEnabled: %v", cl.config.Telegram.Enabled)
		logger.Info("   - InfraLayer: %v", cl.infraLayer != nil)
		logger.Info("   - CounterAnalyzer.Enabled: %v", cl.config.AnalyzerConfigs.CounterAnalyzer.Enabled)

		if cl.config.AnalyzerConfigs.CounterAnalyzer.Enabled {
			if err := cl.startAnalysisEngine(); err != nil {
				logger.Warn("⚠️ Не удалось запустить AnalysisEngine: %v", err)
			}
		} else {
			logger.Info("ℹ️ CounterAnalyzer отключен в конфигурации, AnalysisEngine не запускается")
		}
	}

	// ⭐ ПРИНУДИТЕЛЬНО СОЗДАЕМ SUBSCRIPTIONSERVICE (ЧТОБЫ ЗАПУСТИТЬ ВАЛИДАТОР)
	logger.Info("🔧 Инициализация SubscriptionService для запуска валидатора...")
	if _, err := cl.GetSubscriptionService(); err != nil {
		logger.Warn("⚠️ Не удалось создать SubscriptionService: %v", err)
	} else {
		logger.Info("✅ SubscriptionService создан, валидатор запущен")
	}

	// Фабрика ядра не требует отдельного запуска,
	// так как сервисы создаются лениво

	cl.running = true
	cl.updateState(StateRunning)
	logger.Info("✅ Слой ядра запущен")
	return nil
}

// startAnalysisEngine запуск движка анализа
func (cl *CoreLayer) startAnalysisEngine() error {
	logger.Info("🔧 CoreLayer: запуск AnalysisEngine...")

	// 1. Получаем EventBus
	eventBusComp, exists := cl.infraLayer.GetComponent("EventBus")
	if !exists {
		return fmt.Errorf("EventBus не найден в инфраструктуре")
	}

	eventBusInterface, err := cl.getComponentValue(eventBusComp)
	if err != nil {
		return fmt.Errorf("не удалось получить EventBus: %w", err)
	}

	eventBus, ok := eventBusInterface.(*events.EventBus)
	if !ok {
		return fmt.Errorf("неверный тип EventBus")
	}

	// 2. Получаем StorageFactory для создания priceStorage
	storageFactoryComp, exists := cl.infraLayer.GetComponent("StorageFactory")
	if !exists {
		return fmt.Errorf("StorageFactory не найден")
	}

	storageInterface, err := cl.getComponentValue(storageFactoryComp)
	if err != nil {
		return fmt.Errorf("не удалось получить StorageFactory: %w", err)
	}

	storageFactory, ok := storageInterface.(*redis_storage_factory.StorageFactory)
	if !ok {
		return fmt.Errorf("неверный тип StorageFactory")
	}

	// 3. Создаем хранилище цен для AnalysisEngine
	priceStorage, err := storageFactory.CreateDefaultStorage()
	if err != nil {
		return fmt.Errorf("ошибка создания хранилища цен: %w", err)
	}

	// 4. Проверяем наличие BybitPriceFetcher
	var priceFetcher interface{}
	if cl.bybitPriceFetcher != nil {
		priceFetcher = cl.bybitPriceFetcher
		logger.Info("✅ Используем существующий BybitPriceFetcher")
	} else {
		logger.Warn("⚠️ BybitPriceFetcher не создан, создаем новый...")
		// Попробуем создать фетчер
		if err := cl.ensureBybitPriceFetcher(); err != nil {
			return fmt.Errorf("не удалось создать BybitPriceFetcher: %w", err)
		}
		priceFetcher = cl.bybitPriceFetcher
	}

	// 6. Создаем фабрику движка анализа
	engineFactory := engine.NewFactory(priceFetcher, cl.candleSystem)

	// Передаем SRZoneStorage если уже создано
	if cl.srZoneStorage != nil {
		engineFactory.SetSRZoneStorage(cl.srZoneStorage)
		logger.Info("✅ SRZoneStorage передан в AnalysisEngine Factory")
	}

	// 7. Создаем движок анализа через фабрику
	analysisEngine := engineFactory.NewAnalysisEngineFromConfig(
		priceStorage,
		eventBus,
		cl.config,
	)

	if analysisEngine == nil {
		return fmt.Errorf("не удалось создать AnalysisEngine")
	}

	// Сохраняем движок
	cl.analysisEngine = analysisEngine

	// 8. Регистрируем компонент
	cl.registerComponent("AnalysisEngine", cl.analysisEngine)
	logger.Info("✅ AnalysisEngine создан и зарегистрирован")

	// 9. Запускаем движок анализа
	if err := cl.analysisEngine.Start(); err != nil {
		return fmt.Errorf("ошибка запуска AnalysisEngine: %w", err)
	}

	logger.Info("🚀 AnalysisEngine запущен")
	return nil
}

// ensureBybitPriceFetcher создает BybitPriceFetcher если не создан
func (cl *CoreLayer) ensureBybitPriceFetcher() error {
	if cl.bybitPriceFetcher != nil {
		return nil
	}

	logger.Info("🔄 Создание BybitPriceFetcher для AnalysisEngine...")

	// Получаем EventBus
	eventBusComp, exists := cl.infraLayer.GetComponent("EventBus")
	if !exists {
		return fmt.Errorf("EventBus не найден")
	}

	eventBusInterface, err := cl.getComponentValue(eventBusComp)
	if err != nil {
		return fmt.Errorf("не удалось получить EventBus: %w", err)
	}

	eventBus, ok := eventBusInterface.(*events.EventBus)
	if !ok {
		return fmt.Errorf("неверный тип EventBus")
	}

	// Получаем StorageFactory
	storageFactoryComp, exists := cl.infraLayer.GetComponent("StorageFactory")
	if !exists {
		return fmt.Errorf("StorageFactory не найден")
	}

	storageInterface, err := cl.getComponentValue(storageFactoryComp)
	if err != nil {
		return fmt.Errorf("не удалось получить StorageFactory: %w", err)
	}

	storageFactory, ok := storageInterface.(*redis_storage_factory.StorageFactory)
	if !ok {
		return fmt.Errorf("неверный тип StorageFactory")
	}

	// Создаем хранилище цен
	priceStorage, err := storageFactory.CreateDefaultStorage()
	if err != nil {
		return fmt.Errorf("ошибка создания хранилища цен: %w", err)
	}

	// Создаем фетчер
	var fetcher *fetchers.BybitPriceFetcher
	if cl.candleSystem != nil {
		fetcher, err = cl.fetcherFactory.CreateBybitFetcherWithCandleSystem(
			priceStorage,
			eventBus,
			cl.candleSystem,
		)
	} else {
		fetcher, err = cl.fetcherFactory.CreateBybitFetcher(
			priceStorage,
			eventBus,
		)
	}

	if err != nil {
		return fmt.Errorf("ошибка создания BybitPriceFetcher: %w", err)
	}

	cl.bybitPriceFetcher = fetcher
	logger.Info("✅ BybitPriceFetcher создан для AnalysisEngine")
	return nil
}

// setupAndStartCandleSystem настройка и запуск свечной системы
func (cl *CoreLayer) setupAndStartCandleSystem() error {
	logger.Info("🕯️ CoreLayer: настройка свечной системы...")

	// Получаем StorageFactory
	storageFactoryComp, exists := cl.infraLayer.GetComponent("StorageFactory")
	if !exists {
		return fmt.Errorf("StorageFactory не найден")
	}

	storageInterface, err := cl.getComponentValue(storageFactoryComp)
	if err != nil {
		return fmt.Errorf("не удалось получить StorageFactory: %w", err)
	}

	storageFactory, ok := storageInterface.(*redis_storage_factory.StorageFactory)
	if !ok {
		return fmt.Errorf("неверный тип StorageFactory")
	}

	// Создаем хранилище цен для свечной системы
	priceStorage, err := storageFactory.CreateDefaultStorage()
	if err != nil {
		return fmt.Errorf("ошибка создания хранилища цен: %w", err)
	}

	// Создаем хранилище свечей Redis
	candleConfig := storage.CandleConfig{
		SupportedPeriods: []string{"5m", "15m", "30m", "1h", "4h", "1d"},
		MaxHistory:       1000,
		CleanupInterval:  5 * time.Minute,
		AutoBuild:        true,
	}

	candleStorage, err := storageFactory.CreateCandleStorage(candleConfig)
	if err != nil {
		return fmt.Errorf("ошибка создания хранилища свечей: %w", err)
	}

	// ⭐ ПОЛУЧАЕМ EventBus для свечной системы
	eventBusComp, exists := cl.infraLayer.GetComponent("EventBus")
	if !exists {
		return fmt.Errorf("EventBus не найден")
	}

	eventBusInterface, err := cl.getComponentValue(eventBusComp)
	if err != nil {
		return fmt.Errorf("не удалось получить EventBus: %w", err)
	}

	eventBus, ok := eventBusInterface.(*events.EventBus)
	if !ok {
		return fmt.Errorf("неверный тип EventBus")
	}

	// ⭐ ПОЛУЧАЕМ RedisService для создания CandleTracker
	redisServiceComp, exists := cl.infraLayer.GetComponent("RedisService")
	if !exists {
		logger.Warn("⚠️ RedisService не найден, CandleSystem будет создана без CandleTracker")
		// Создаем свечную систему С EventBus
		cl.candleSystem, err = candle.NewCandleSystemFactory().CreateSystem(priceStorage, candleStorage, eventBus)
	} else {
		redisServiceInterface, err := cl.getComponentValue(redisServiceComp)
		if err != nil {
			logger.Warn("⚠️ Не удалось получить RedisService: %v", err)
			// Создаем свечную систему С EventBus
			cl.candleSystem, err = candle.NewCandleSystemFactory().CreateSystem(priceStorage, candleStorage, eventBus)
		} else {
			redisService, ok := redisServiceInterface.(*redis_service.RedisService)
			if !ok {
				logger.Warn("⚠️ Неверный тип RedisService")
				// Создаем свечную систему С EventBus
				cl.candleSystem, err = candle.NewCandleSystemFactory().CreateSystem(priceStorage, candleStorage, eventBus)
			} else {
				// ⭐ СОЗДАЕМ СВЕЧНУЮ СИСТЕМУ С ТРЕКЕРОМ И EventBus
				cl.candleSystem, err = candle.NewCandleSystemFactory().CreateSystemWithRedis(
					priceStorage,
					candleStorage,
					redisService,
					eventBus,
				)
			}
		}
	}

	if err != nil {
		return fmt.Errorf("ошибка создания свечной системы: %w", err)
	}

	// Запускаем свечную систему
	if err := cl.candleSystem.Start(); err != nil {
		return fmt.Errorf("ошибка запуска свечной системы: %w", err)
	}

	// Регистрируем компонент
	cl.registerComponent("CandleSystem", cl.candleSystem)
	logger.Info("✅ Свечная система создана и запущена (с EventBus)")

	return nil
}

// startBybitPriceFetcher запуск BybitPriceFetcher
func (cl *CoreLayer) startBybitPriceFetcher() {
	logger.Info("🔄 CoreLayer: инициализация BybitPriceFetcher...")

	// Получаем EventBus из инфраструктуры
	eventBusComp, exists := cl.infraLayer.GetComponent("EventBus")
	if !exists {
		logger.Warn("⚠️ CoreLayer: EventBus не найден в инфраструктуре")
		return
	}

	// Получаем EventBus из LazyComponent
	eventBusInterface, err := cl.getComponentValue(eventBusComp)
	if err != nil {
		logger.Warn("⚠️ CoreLayer: не удалось получить EventBus: %v", err)
		return
	}

	if eventBusInterface == nil {
		logger.Warn("⚠️ CoreLayer: EventBus равен nil")
		return
	}

	eventBus, ok := eventBusInterface.(*events.EventBus)
	if !ok {
		logger.Warn("⚠️ CoreLayer: неверный тип EventBus")
		return
	}

	// Получаем StorageFactory
	storageFactoryComp, exists := cl.infraLayer.GetComponent("StorageFactory")
	if !exists {
		logger.Warn("⚠️ CoreLayer: StorageFactory не найден")
		return
	}

	// Получаем StorageFactory из LazyComponent
	storageInterface, err := cl.getComponentValue(storageFactoryComp)
	if err != nil {
		logger.Warn("⚠️ CoreLayer: не удалось получить StorageFactory: %v", err)
		return
	}

	storageFactory, ok := storageInterface.(*redis_storage_factory.StorageFactory)
	if !ok {
		logger.Warn("⚠️ CoreLayer: неверный тип StorageFactory")
		return
	}

	// Создаем хранилище цен
	priceStorage, err := storageFactory.CreateDefaultStorage()
	if err != nil {
		logger.Error("❌ CoreLayer: ошибка создания хранилища цен: %v", err)
		return
	}

	if priceStorage == nil {
		logger.Warn("⚠️ CoreLayer: хранилище цен равно nil")
		logger.Info("ℹ️  Пропускаем создание BybitPriceFetcher")
		return
	}

	// ⭐ Создаем фетчер БЕЗ CandleSystem (теперь взаимодействие через EventBus)
	fetcher, err := cl.fetcherFactory.CreateBybitFetcher(
		priceStorage,
		eventBus, // EventBus для публикации цен
	)

	if err != nil {
		logger.Error("❌ CoreLayer: ошибка создания BybitPriceFetcher: %v", err)
		return
	}

	// Сохраняем фетчер
	cl.bybitPriceFetcher = fetcher

	// Регистрируем компонент
	cl.registerComponent("BybitPriceFetcher", fetcher)
	logger.Info("✅ BybitPriceFetcher создан и зарегистрирован (взаимодействие через EventBus)")

	// Запускаем фетчер с интервалом из конфигурации
	interval := time.Duration(cl.config.UpdateInterval) * time.Second
	if interval == 0 {
		interval = 10 * time.Second
		logger.Info("ℹ️  Используется дефолтный интервал для BybitPriceFetcher: %v", interval)
	}

	if err := fetcher.Start(interval); err != nil {
		logger.Error("❌ CoreLayer: ошибка запуска BybitPriceFetcher: %v", err)
		cl.setError(err)
	} else {
		logger.Info("🚀 BybitPriceFetcher запущен с интервалом %v", interval)
	}
}

// startSRZoneEngine запускает движок зон S/R
func (cl *CoreLayer) startSRZoneEngine() error {
	logger.Info("📐 CoreLayer: запуск SRZoneEngine...")

	// 1. Получаем RedisService для SRZoneStorage
	redisServiceComp, exists := cl.infraLayer.GetComponent("RedisService")
	if !exists {
		return fmt.Errorf("RedisService не найден")
	}

	redisServiceInterface, err := cl.getComponentValue(redisServiceComp)
	if err != nil {
		return fmt.Errorf("не удалось получить RedisService: %w", err)
	}

	redisService, ok := redisServiceInterface.(*redis_service.RedisService)
	if !ok {
		return fmt.Errorf("неверный тип RedisService")
	}

	// 2. Создаем SRZoneStorage
	srStorage, err := sr_storage.NewSRZoneStorage(redisService)
	if err != nil {
		return fmt.Errorf("ошибка создания SRZoneStorage: %w", err)
	}
	cl.srZoneStorage = srStorage

	// 3. Получаем EventBus
	eventBusComp, exists := cl.infraLayer.GetComponent("EventBus")
	if !exists {
		return fmt.Errorf("EventBus не найден")
	}

	eventBusInterface, err := cl.getComponentValue(eventBusComp)
	if err != nil {
		return fmt.Errorf("не удалось получить EventBus: %w", err)
	}

	eventBus, ok := eventBusInterface.(*events.EventBus)
	if !ok {
		return fmt.Errorf("неверный тип EventBus")
	}

	// 4. Проверяем CandleSystem
	if cl.candleSystem == nil {
		return fmt.Errorf("CandleSystem не запущена")
	}

	// 5. Проверяем BybitPriceFetcher
	if cl.bybitPriceFetcher == nil {
		return fmt.Errorf("BybitPriceFetcher не создан")
	}

	// 6. Создаем SRZoneEngine
	cl.srZoneEngine = sr_engine.NewEngine(
		cl.candleSystem.Storage,
		srStorage,
		cl.bybitPriceFetcher, // OrderBookFetcher
		cl.bybitPriceFetcher, // Volume24hProvider
		eventBus,
	)

	// 7. Запускаем движок
	cl.srZoneEngine.Start()

	// 8. Прогреваем зоны при первом поступлении батча цен (one-shot горутина).
	// Это устраняет "холодный старт": без прогрева зоны появляются только
	// после первого EventCandleClosed (~60с для 1m), а сигналы идут сразу.
	go cl.warmupSRZonesOnFirstPriceEvent(eventBus)

	cl.registerComponent("SRZoneEngine", cl.srZoneEngine)
	logger.Info("✅ SRZoneEngine запущен и зарегистрирован")
	return nil
}

// warmupSRZonesOnFirstPriceEvent подписывается на EventPriceUpdated,
// берёт символы из первого батча и запускает Warmup, затем отписывается.
func (cl *CoreLayer) warmupSRZonesOnFirstPriceEvent(eventBus *events.EventBus) {
	if cl.srZoneEngine == nil {
		return
	}

	symbolsCh := make(chan []string, 1)
	var once sync.Once

	subscriber := events.NewBaseSubscriber(
		"sr_zone_warmup",
		[]types.EventType{types.EventPriceUpdated},
		func(event types.Event) error {
			once.Do(func() {
				if priceList, ok := event.Data.([]storage.PriceData); ok && len(priceList) > 0 {
					symbols := make([]string, 0, len(priceList))
					for _, p := range priceList {
						symbols = append(symbols, p.Symbol)
					}
					symbolsCh <- symbols
				}
			})
			return nil
		},
	)

	eventBus.Subscribe(types.EventPriceUpdated, subscriber)

	// Ждём первый батч (приходит через ~10с после старта)
	symbols := <-symbolsCh

	// Одноразовый подписчик — сразу отписываемся
	eventBus.Unsubscribe(types.EventPriceUpdated, subscriber)

	// Прогреваем все основные периоды параллельно
	periods := []string{"1m", "5m", "15m", "30m", "1h", "4h"}
	logger.Info("🔥 CoreLayer: прогрев S/R зон для %d символов × %d периодов", len(symbols), len(periods))
	cl.srZoneEngine.Warmup(symbols, periods)
}

// Stop останавливает слой ядра
func (cl *CoreLayer) Stop() error {
	if !cl.IsRunning() {
		return nil
	}

	cl.updateState(StateStopping)
	logger.Info("🛑 Остановка слоя ядра...")

	// Останавливаем SRZoneEngine если запущен
	if cl.srZoneEngine != nil {
		cl.srZoneEngine.Stop()
		logger.Info("📐 SRZoneEngine остановлен")
	}

	// Останавливаем AnalysisEngine если запущен
	if cl.analysisEngine != nil {
		// ✅ ИСПРАВЛЕНИЕ: Вызываем Stop() без проверки возвращаемого значения
		cl.analysisEngine.Stop() // Метод Stop() может не возвращать ошибку
		logger.Info("🛑 AnalysisEngine остановлен")
	}

	// Останавливаем свечную систему если запущена
	if cl.candleSystem != nil {
		if err := cl.candleSystem.Stop(); err != nil {
			logger.Warn("⚠️ Ошибка остановки свечной системы: %v", err)
		} else {
			logger.Info("🕯️ Свечная система остановлена")
		}
	}

	// Останавливаем BybitPriceFetcher если запущен
	if cl.bybitPriceFetcher != nil && cl.bybitPriceFetcher.IsRunning() {
		if err := cl.bybitPriceFetcher.Stop(); err != nil {
			logger.Warn("⚠️ Ошибка остановки BybitPriceFetcher: %v", err)
		} else {
			logger.Info("🛑 BybitPriceFetcher остановлен")
		}
	}

	cl.running = false
	cl.updateState(StateStopped)
	logger.Info("✅ Слой ядра остановлен")
	return nil
}

// Reset сбрасывает слой ядра
func (cl *CoreLayer) Reset() error {
	logger.Info("🔄 Сброс слоя ядра...")

	// Останавливаем если запущен
	if cl.IsRunning() {
		cl.Stop()
	}

	// Сбрасываем фабрику
	if cl.coreFactory != nil {
		cl.coreFactory.Reset()
	}

	// Сбрасываем SRZoneEngine
	if cl.srZoneEngine != nil {
		cl.srZoneEngine = nil
	}
	if cl.srZoneStorage != nil {
		cl.srZoneStorage = nil
	}

	// Сбрасываем AnalysisEngine
	if cl.analysisEngine != nil {
		cl.analysisEngine = nil
	}

	// Сбрасываем свечную систему
	if cl.candleSystem != nil {
		cl.candleSystem = nil
	}

	// Сбрасываем фетчер
	if cl.bybitPriceFetcher != nil {
		cl.bybitPriceFetcher = nil
	}
	if cl.fetcherFactory != nil {
		cl.fetcherFactory = nil
	}

	// Сбрасываем базовый слой
	cl.BaseLayer.Reset()
	cl.coreFactory = nil
	cl.initialized = false
	logger.Info("✅ Слой ядра сброшен")
	return nil
}

// IsInitialized проверяет инициализацию
func (cl *CoreLayer) IsInitialized() bool {
	return cl.initialized
}

// GetCoreFactory возвращает фабрику ядра
func (cl *CoreLayer) GetCoreFactory() *core_factory.CoreServiceFactory {
	cl.mu.RLock()
	defer cl.mu.RUnlock()
	return cl.coreFactory
}

// getComponentValue получает значение компонента из LazyComponent
func (cl *CoreLayer) getComponentValue(component interface{}) (interface{}, error) {
	if lc, ok := component.(*LazyComponent); ok {
		return lc.Get()
	}
	return nil, fmt.Errorf("компонент не является LazyComponent")
}

// GetBybitPriceFetcher возвращает BybitPriceFetcher
func (cl *CoreLayer) GetBybitPriceFetcher() *fetchers.BybitPriceFetcher {
	cl.mu.RLock()
	defer cl.mu.RUnlock()
	return cl.bybitPriceFetcher
}

// GetCandleSystem возвращает свечную систему
func (cl *CoreLayer) GetCandleSystem() *candle.CandleSystem {
	cl.mu.RLock()
	defer cl.mu.RUnlock()
	return cl.candleSystem
}

// GetAnalysisEngine возвращает AnalysisEngine
func (cl *CoreLayer) GetAnalysisEngine() *engine.AnalysisEngine {
	cl.mu.RLock()
	defer cl.mu.RUnlock()
	return cl.analysisEngine
}

// registerCoreComponents регистрирует компоненты ядра
func (cl *CoreLayer) registerCoreComponents() {
	if cl.coreFactory == nil {
		return
	}

	// Регистрируем компоненты ядра
	components := map[string]string{
		"UserService":         "сервис пользователей",
		"SubscriptionService": "сервис подписок",
		"AnalysisEngine":      "движок анализа сигналов",
	}

	for name, description := range components {
		// Пропускаем AnalysisEngine - он создается в Start()
		if name == "AnalysisEngine" {
			continue
		}

		cl.registerComponent(name, &LazyComponent{
			name:        name,
			description: description,
			getter:      cl.getCoreComponent(name),
		})
		logger.Debug("🧩 Зарегистрирован компонент ядра: %s (%s)", name, description)
	}
}

// getCoreComponent возвращает геттер для компонента ядра
func (cl *CoreLayer) getCoreComponent(name string) func() (interface{}, error) {
	return func() (interface{}, error) {
		if cl.coreFactory == nil {
			return nil, fmt.Errorf("фабрика ядра не создана")
		}

		switch name {
		case "UserService":
			// Если UserService уже создан, возвращаем его
			if userService, exists := cl.GetComponent("UserService"); exists {
				return userService, nil
			}
			// Иначе создаем через фабрику
			return cl.coreFactory.CreateUserService()
		case "SubscriptionService":
			return cl.coreFactory.CreateSubscriptionService()
		case "AnalysisEngine":
			return cl.analysisEngine, nil
		default:
			return nil, fmt.Errorf("неизвестный компонент ядра: %s", name)
		}
	}
}

// GetUserService возвращает UserService (ленивое создание)
func (cl *CoreLayer) GetUserService() (interface{}, error) {
	comp, exists := cl.GetComponent("UserService")
	if !exists {
		return nil, fmt.Errorf("UserService не зарегистрирован")
	}

	lc, ok := comp.(*LazyComponent)
	if !ok {
		return nil, fmt.Errorf("неверный тип компонента UserService")
	}

	return lc.Get()
}

// GetSubscriptionService возвращает SubscriptionService (ленивое создание)
func (cl *CoreLayer) GetSubscriptionService() (interface{}, error) {
	comp, exists := cl.GetComponent("SubscriptionService")
	if !exists {
		return nil, fmt.Errorf("SubscriptionService не зарегистрирован")
	}

	lc, ok := comp.(*LazyComponent)
	if !ok {
		return nil, fmt.Errorf("неверный тип компонента SubscriptionService")
	}

	return lc.Get()
}
