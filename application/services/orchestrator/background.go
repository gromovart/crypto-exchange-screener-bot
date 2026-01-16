// application/services/orchestrator/background.go
package orchestrator

import (
	subscriptiontypes "crypto-exchange-screener-bot/internal/core/domain/subscription"
	"crypto-exchange-screener-bot/internal/core/domain/users"
	telegramintegrations "crypto-exchange-screener-bot/internal/delivery/telegram/integrations"
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
	"runtime"
	"time"
)

// startBackgroundTasks запускает фоновые задачи
func (dm *DataManager) startBackgroundTasks() {
	// Обновление статистики системы
	dm.wg.Add(1)
	go func() {
		defer dm.wg.Done()
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				dm.updateSystemStats()
			case <-dm.stopChan:
				return
			}
		}
	}()

	// Очистка старых данных
	dm.wg.Add(1)
	go func() {
		defer dm.wg.Done()
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if _, err := dm.storage.CleanOldData(24 * time.Hour); err != nil {
					logger.Info("⚠️ Не удалось очистить старые данные: %v", err)
				}
			case <-dm.stopChan:
				return
			}
		}
	}()

	// Мониторинг здоровья
	dm.wg.Add(1)
	go func() {
		defer dm.wg.Done()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				dm.checkHealth()
			case <-dm.stopChan:
				return
			}
		}
	}()

	// Проверка и создание UserService/SubscriptionService когда зависимости готовы
	dm.wg.Add(1)
	go func() {
		defer dm.wg.Done()
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		userServiceCreated := false
		subscriptionServiceCreated := false

		for {
			select {
			case <-ticker.C:
				// Создаем UserService если еще не создан
				if !userServiceCreated {
					created := dm.tryCreateUserService()
					if created {
						userServiceCreated = true
						logger.Info("✅ UserService успешно создан")
					}
				}

				// Создаем SubscriptionService если еще не создан
				if !subscriptionServiceCreated {
					created := dm.tryCreateSubscriptionService()
					if created {
						subscriptionServiceCreated = true
						logger.Info("✅ SubscriptionService успешно создан")
					}
				}

			case <-dm.stopChan:
				return
			}
		}
	}()

	// Проверка и создание TelegramPackageService когда зависимости готовы
	dm.wg.Add(1)
	go func() {
		defer dm.wg.Done()
		ticker := time.NewTicker(2 * time.Second) // Проверяем каждые 2 секунды
		defer ticker.Stop()

		telegramServiceCreated := false

		for {
			select {
			case <-ticker.C:
				if !telegramServiceCreated && dm.config.TelegramEnabled {
					created := dm.tryCreateTelegramPackageService()
					if created {
						telegramServiceCreated = true
						logger.Info("✅ TelegramPackageService успешно создан и запущен")
					}
				}
			case <-dm.stopChan:
				return
			}
		}
	}()
}

// tryCreateUserService пытается создать UserService
func (dm *DataManager) tryCreateUserService() bool {
	if dm.userService != nil {
		return true
	}

	if dm.databaseService == nil || dm.redisService == nil {
		logger.Debug("⏳ Ожидание DatabaseService и RedisService для UserService...")
		return false
	}

	db := dm.databaseService.GetDB()
	redisCache := dm.redisService.GetCache()

	if db == nil || redisCache == nil {
		logger.Debug("⏳ Ожидание подключения к БД/Redis для UserService...")
		return false
	}

	logger.Info("👤 Создание UserService (зависимости доступны)...")

	userConfig := users.Config{
		DefaultMinGrowthThreshold: 2.0,
		DefaultMaxSignalsPerDay:   50,
		SessionTTL:                24 * time.Hour,
		MaxSessionsPerUser:        5,
	}

	var err error
	dm.userService, err = users.NewService(db, redisCache, nil, userConfig)
	if err != nil {
		logger.Warn("Не удалось создать UserService: %v", err)
		return false
	}

	logger.Info("✅ UserService создан")
	return true
}

// tryCreateSubscriptionService пытается создать SubscriptionService
func (dm *DataManager) tryCreateSubscriptionService() bool {
	if dm.subscriptionService != nil {
		return true
	}

	if dm.databaseService == nil || dm.redisService == nil {
		logger.Debug("⏳ Ожидание DatabaseService и RedisService для SubscriptionService...")
		return false
	}

	db := dm.databaseService.GetDB()
	redisCache := dm.redisService.GetCache()

	if db == nil || redisCache == nil {
		logger.Debug("⏳ Ожидание подключения к БД/Redis для SubscriptionService...")
		return false
	}

	logger.Info("💎 Создание SubscriptionService (зависимости доступны)...")

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
		logger.Warn("Не удалось создать SubscriptionService: %v", err)
		return false
	}

	dm.subscriptionService = subService
	logger.Info("✅ SubscriptionService создан")
	return true
}

// tryCreateTelegramPackageService пытается создать TelegramPackageService
func (dm *DataManager) tryCreateTelegramPackageService() bool {
	// Проверяем, не создан ли уже сервис
	if dm.telegramPackageService != nil {
		return true
	}

	// Проверяем все зависимости
	if dm.userService == nil {
		logger.Debug("⏳ Ожидание UserService для TelegramPackageService...")
		return false
	}

	if dm.subscriptionService == nil {
		logger.Debug("⏳ Ожидание SubscriptionService для TelegramPackageService...")
		return false
	}

	if dm.eventBus == nil {
		logger.Debug("⏳ Ожидание EventBus для TelegramPackageService...")
		return false
	}

	logger.Info("📦 Создание TelegramPackageService (все зависимости доступны)...")

	telegramService, err := telegramintegrations.NewTelegramPackageServiceWithDefaults(
		dm.config,
		dm.userService,
		dm.subscriptionService,
		dm.eventBus,
	)

	if err != nil {
		logger.Warn("Не удалось создать TelegramPackageService: %v", err)
		return false
	}

	dm.telegramPackageService = telegramService
	logger.Info("✅ TelegramPackageService создан как единственная точка взаимодействия с Telegram")

	// Запускаем сервис
	if err := dm.telegramPackageService.Start(); err != nil {
		logger.Warn("Не удалось запустить TelegramPackageService: %v", err)
		return false
	}

	// Регистрируем в реестре сервисов
	if dm.registry != nil {
		dm.registry.Register("TelegramPackageService",
			NewUniversalServiceWrapper("TelegramPackageService", dm.telegramPackageService, true, true))
		logger.Info("✅ TelegramPackageService зарегистрирован в реестре")
	}

	// Обновляем зависимости в lifecycle
	if dm.lifecycle != nil {
		dm.lifecycle.AddDependency("TelegramPackageService", "EventBus")
		logger.Info("✅ Зависимости TelegramPackageService обновлены")
	}

	return true
}

// updateSystemStats обновляет статистику системы
func (dm *DataManager) updateSystemStats() {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	servicesInfo := dm.registry.GetAllInfo()
	storageStats := dm.storage.GetStats()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	var analysisStats interface{}
	if dm.analysisEngine != nil {
		analysisStats = dm.analysisEngine.GetStats()
	}

	var eventBusStats interface{}
	if dm.eventBus != nil {
		eventBusStats = dm.eventBus.GetMetrics()
	}

	dm.systemStats = SystemStats{
		Services:      servicesInfo,
		StorageStats:  storageStats,
		AnalysisStats: analysisStats,
		EventBusStats: eventBusStats,
		Uptime:        time.Since(dm.startTime),
		TotalRequests: 0,
		MemoryUsageMB: float64(m.Alloc) / 1024 / 1024,
		CPUUsage:      0,
		ActiveSymbols: storageStats.TotalSymbols,
		LastUpdated:   time.Now(),
	}
}

// checkHealth проверяет здоровье системы
func (dm *DataManager) checkHealth() {
	health := dm.GetHealthStatus()
	if health.Status != "healthy" {
		dm.eventBus.Publish(types.Event{
			Type:   types.EventError,
			Source: "DataManager",
			Data: map[string]interface{}{
				"status":  health.Status,
				"message": "Проверка здоровья системы не пройдена",
			},
		})
		logger.Info("⚠️ Проверка здоровья системы не пройдена: %s", health.Status)
	}
}
	