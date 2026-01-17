// application/services/orchestrator/initialization.go
package orchestrator

import (
	"crypto-exchange-screener-bot/internal/core/domain/candle"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"time"

	fetcher "crypto-exchange-screener-bot/internal/adapters/market"
	notifier "crypto-exchange-screener-bot/internal/adapters/notification"
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

// initTelegramAndNotifications инициализирует Telegram и уведомления (ТОЛЬКО СОЗДАНИЕ)
func (dm *DataManager) initTelegramAndNotifications(testMode bool) error {
	// 4.1 Telegram Delivery Package - БУДЕТ СОЗДАН ПОЗЖЕ в tryCreateTelegramDeliveryPackage()
	if dm.config.TelegramEnabled {
		logger.Info("📦 TelegramDeliveryPackage будет создан позже (после UserService и SubscriptionService)...")
	} else {
		logger.Info("🤖 Telegram отключен в конфигурации")
	}

	// 4.2 Составной сервис уведомлений - ТОЛЬКО СОЗДАНИЕ
	logger.Info("📱 Создание CompositeNotificationService...")
	notifierFactory := notifier.NewNotifierFactory(dm.eventBus)
	dm.notification = notifierFactory.CreateCompositeNotifier(dm.config)
	if dm.notification == nil {
		return fmt.Errorf("не удалось создать CompositeNotificationService")
	}
	logger.Info("✅ CompositeNotificationService создан")

	return nil
}
