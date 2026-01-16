// application/services/orchestrator/initialization.go
package orchestrator

import (
	"crypto-exchange-screener-bot/internal/core/domain/candle"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"time"

	fetcher "crypto-exchange-screener-bot/internal/adapters/market"
	notifier "crypto-exchange-screener-bot/internal/adapters/notification"
	subscriptiontypes "crypto-exchange-screener-bot/internal/core/domain/subscription"
	"crypto-exchange-screener-bot/internal/core/domain/users"
	telegrambot "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot"
	telegramintegrations "crypto-exchange-screener-bot/internal/delivery/telegram/integrations"
	"crypto-exchange-screener-bot/internal/infrastructure/api/exchanges/bybit"
	redis "crypto-exchange-screener-bot/internal/infrastructure/cache/redis"
	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/in_memory_storage"
	database "crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/database"
	events "crypto-exchange-screener-bot/internal/infrastructure/transport/event_bus"
)

// initInfrastructure инициализирует инфраструктурные компоненты (ТОЛЬКО СОЗДАНИЕ)
func (dm *DataManager) initInfrastructure(testMode bool) error {
	// 1.1 База данных - ТОЛЬКО СОЗДАНИЕ
	logger.Info("🗄️ Создание сервиса базы данных...")
	dm.databaseService = database.NewDatabaseService(dm.config)
	logger.Info("✅ Сервис базы данных создан (не запущен)")

	// 1.2 Redis - ТОЛЬКО СОЗДАНИЕ
	logger.Info("🔴 Создание Redis сервиса...")
	dm.redisService = redis.NewRedisService(dm.config)
	logger.Info("✅ Redis сервис создан (не запущен)")

	// 1.3 EventBus - ТОЛЬКО СОЗДАНИЕ
	logger.Info("🚌 Создание EventBus...")
	eventBusConfig := events.EventBusConfig{
		BufferSize:    dm.config.EventBus.BufferSize,
		WorkerCount:   dm.config.EventBus.WorkerCount,
		EnableMetrics: dm.config.EventBus.EnableMetrics,
		EnableLogging: dm.config.EventBus.EnableLogging,
	}
	dm.eventBus = events.NewEventBus(eventBusConfig)
	logger.Info("✅ EventBus создан (не запущен)")

	return nil
}

// initStorageAndFetchers инициализирует хранение и получение данных (ТОЛЬКО СОЗДАНИЕ)
func (dm *DataManager) initStorageAndFetchers() (*candle.CandleSystem, error) {
	// 2.1 Хранилище цен - ТОЛЬКО СОЗДАНИЕ
	logger.Info("💾 Создание хранилища цен...")
	storageConfig := &storage.StorageConfig{
		MaxHistoryPerSymbol: 10000,
		MaxSymbols:          1000,
		CleanupInterval:     5 * time.Minute,
		RetentionPeriod:     24 * time.Hour,
	}
	dm.storage = storage.NewInMemoryPriceStorage(storageConfig)
	logger.Info("✅ Хранилище цен создано")

	// 2.2 Создаем свечную систему - ТОЛЬКО СОЗДАНИЕ
	logger.Info("🕯️ Создание системы свечей...")
	candleSystem, err := candle.CreateSimpleSystem(dm.storage)
	if err != nil {
		logger.Warn("Не удалось создать систему свечей: %v", err)
		logger.Warn("Приложение будет работать без системы свечей")
	} else {
		logger.Info("✅ Система свечей создана")
	}

	// 2.3 API клиент - ТОЛЬКО СОЗДАНИЕ
	logger.Info("🌐 Создание API клиента...")
	apiClient := bybit.NewBybitClient(dm.config)

	// 2.4 Получение цен - ТОЛЬКО СОЗДАНИЕ
	logger.Info("📡 Создание PriceFetcher...")
	dm.priceFetcher = fetcher.NewPriceFetcher(apiClient, dm.storage, dm.eventBus, candleSystem)
	logger.Info("✅ PriceFetcher создан")

	return candleSystem, nil
}

// initUsersAndAuth инициализирует пользователей и авторизацию (ТОЛЬКО СОЗДАНИЕ - ОТЛОЖЕННО)
func (dm *DataManager) initUsersAndAuth() error {
	// 3.1 Сервис пользователей - ТОЛЬКО СОЗДАНИЕ (отложенное)
	logger.Info("👤 Инициализация UserService будет выполнена после запуска БД/Redis...")
	// UserService будет создан позже в initPostStartServices()

	// 3.2 Сервис подписок - ТОЛЬКО СОЗДАНИЕ (отложенное)
	logger.Info("💎 Инициализация SubscriptionService будет выполнена после запуска БД/Redis...")
	// SubscriptionService будет создан позже в initPostStartServices()

	return nil
}

// initPostStartServices инициализирует сервисы, зависящие от запущенных БД/Redis
func (dm *DataManager) initPostStartServices() error {
	logger.Info("🔄 Инициализация сервисов, зависящих от БД/Redis...")

	// 1. Создаем UserService если еще не создан и БД/Redis доступны
	if dm.userService == nil && dm.databaseService != nil && dm.redisService != nil {
		logger.Info("👤 Создание UserService...")
		db := dm.databaseService.GetDB()
		redisCache := dm.redisService.GetCache()

		if db != nil && redisCache != nil {
			userConfig := users.Config{
				DefaultMinGrowthThreshold: 2.0,
				DefaultMaxSignalsPerDay:   50,
				SessionTTL:                24 * time.Hour,
				MaxSessionsPerUser:        5,
			}

			var err error
			dm.userService, err = users.NewService(db, redisCache, nil, userConfig)
			if err != nil {
				logger.Warn("Не удалось создать сервис пользователей: %v", err)
			} else {
				logger.Info("✅ Сервис пользователей создан")
			}
		} else {
			logger.Warn("Подключение к базе данных или Redis недоступно для UserService")
		}
	}

	// 2. Создаем SubscriptionService если еще не создан и БД/Redis доступны
	if dm.subscriptionService == nil && dm.databaseService != nil && dm.redisService != nil {
		logger.Info("💎 Создание SubscriptionService...")
		db := dm.databaseService.GetDB()
		if db != nil && dm.redisService != nil {
			redisCache := dm.redisService.GetCache()
			if redisCache != nil {
				subscriptionConfig := subscriptiontypes.Config{
					StripeSecretKey:  "",
					StripeWebhookKey: "",
					DefaultPlan:      "free",
					TrialPeriodDays:  7,
					GracePeriodDays:  3,
					AutoRenew:        true,
				}

				subService, err := subscriptiontypes.NewService(
					db,
					redisCache,
					nil,
					nil,
					subscriptionConfig,
				)

				if err != nil {
					logger.Warn("Не удалось создать сервис подписок: %v", err)
				} else {
					dm.subscriptionService = subService
					logger.Info("✅ Сервис подписок создан")
				}
			} else {
				logger.Warn("Redis кэш недоступен для сервиса подписок")
			}
		} else {
			logger.Warn("Подключение к базе данных или Redis недоступно для сервиса подписок")
		}
	}

	// 3. Создаем TelegramPackageService если условия выполнены
	if dm.telegramPackageService == nil && dm.config.TelegramEnabled && dm.userService != nil && dm.subscriptionService != nil && dm.eventBus != nil {
		logger.Info("📦 Создание TelegramPackageService...")
		telegramService, err := telegramintegrations.NewTelegramPackageServiceWithDefaults(
			dm.config,
			dm.userService,
			dm.subscriptionService,
			dm.eventBus,
		)

		if err != nil {
			logger.Warn("Не удалось создать Telegram package service: %v", err)
		} else {
			dm.telegramPackageService = telegramService
			logger.Info("✅ Telegram package service создан")
		}
	}

	// 4. Создаем TelegramBot если еще не создан и UserService доступен
	if dm.telegramBot == nil && dm.config.TelegramEnabled && dm.config.TelegramBotToken != "" && dm.userService != nil {
		logger.Info("🤖 Создание Telegram бота с UserService...")
		dm.telegramBot = telegrambot.GetOrCreateBotWithDeps(dm.config, &telegrambot.Dependencies{
			UserService: dm.userService,
		})
		logger.Info("✅ Telegram бот создан с аутентификацией (Singleton)")
	} else if dm.telegramBot == nil && dm.config.TelegramEnabled && dm.config.TelegramBotToken != "" {
		logger.Info("🤖 Создание Telegram бота без UserService...")
		dm.telegramBot = telegrambot.GetOrCreateBot(dm.config)
		logger.Info("✅ Telegram бот создан без аутентификации (Singleton)")
	}

	return nil
}

// initTelegramAndNotifications инициализирует Telegram и уведомления (ТОЛЬКО СОЗДАНИЕ)
func (dm *DataManager) initTelegramAndNotifications(testMode bool) error {
	// 4.1 Telegram бот - БУДЕТ СОЗДАН ПОЗЖЕ в initPostStartServices()
	if dm.config.TelegramEnabled && dm.config.TelegramBotToken != "" {
		logger.Info("🤖 Telegram бот будет создан после инициализации UserService...")
	} else {
		logger.Info("🤖 Telegram бот отключен в конфигурации")
	}

	// 4.2 Telegram Package Service - будет создан в initPostStartServices()
	if dm.config.TelegramEnabled {
		logger.Info("📦 TelegramPackageService будет создан при наличии всех зависимостей...")
	}

	// 4.3 Составной сервис уведомлений - ТОЛЬКО СОЗДАНИЕ
	logger.Info("📱 Создание CompositeNotificationService...")
	notifierFactory := notifier.NewNotifierFactory(dm.eventBus)
	dm.notification = notifierFactory.CreateCompositeNotifier(dm.config)
	if dm.notification == nil {
		return fmt.Errorf("не удалось создать CompositeNotificationService")
	}
	logger.Info("✅ CompositeNotificationService создан")

	return nil
}
