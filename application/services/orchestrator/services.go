// application/services/orchestrator/services.go
package orchestrator

import (
	"crypto-exchange-screener-bot/internal/adapters"
	"crypto-exchange-screener-bot/internal/adapters/notification"
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/core/domain/signals/engine"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"time"

	"crypto-exchange-screener-bot/application/pipeline"
	events "crypto-exchange-screener-bot/internal/infrastructure/transport/event_bus"
	"crypto-exchange-screener-bot/internal/types"
)

// initAnalysisAndProcessing инициализирует анализ и обработку
func (dm *DataManager) initAnalysisAndProcessing() error {
	// 5.1 Движок анализа
	logger.Info("🔧 Создание AnalysisEngine...")
	analysisFactory := engine.NewFactory(dm.priceFetcher, dm.candleSystem)

	var telegramNotifier *notification.TelegramNotifier
	if dm.notification != nil {
		for _, notifier := range dm.notification.GetNotifiers() {
			if tn, ok := notifier.(*notification.TelegramNotifier); ok {
				telegramNotifier = tn
				break
			}
		}
	}

	dm.analysisEngine = analysisFactory.NewAnalysisEngineFromConfig(
		dm.storage,
		dm.eventBus,
		dm.config,
		telegramNotifier,
	)
	logger.Info("✅ AnalysisEngine создан")

	// 5.2 Пайплайн сигналов
	logger.Info("🔄 Создание SignalPipeline...")
	dm.signalPipeline = pipeline.NewSignalPipeline(dm.eventBus)
	logger.Info("✅ SignalPipeline создан")

	return nil
}

// initRegistrationAndSetup инициализирует регистрацию и настройку
func (dm *DataManager) initRegistrationAndSetup() error {
	// 6.1 Регистрация подписчиков
	logger.Info("📋 Регистрация подписчиков EventBus...")
	dm.registerBasicSubscribers()

	// 6.2 Реестр сервисов
	logger.Info("📝 Создание реестра сервисов...")
	dm.registry = NewServiceRegistry()

	// 6.3 Менеджер жизненного цикла
	logger.Info("⚙️ Создание менеджера жизненного цикла...")
	coordinatorConfig := CoordinatorConfig{
		EnableEventLogging:  true,
		EventBufferSize:     1000,
		HealthCheckInterval: 30 * time.Second,
		RestartOnFailure:    true,
		MaxRestartAttempts:  3,
		RestartDelay:        5 * time.Second,
		EnableMetrics:       true,
		MetricsPort:         "9090",
	}
	dm.lifecycle = NewLifecycleManager(dm.registry, dm.eventBus, coordinatorConfig)
	logger.Info("✅ Менеджер жизненного цикла создан")

	// 6.4 Настройка пайплайна
	logger.Info("🔗 Настройка пайплайна...")
	dm.setupPipeline()

	// 6.5 Регистрация сервисов
	logger.Info("🏷️ Регистрация сервисов...")
	if err := dm.registerServices(); err != nil {
		return err
	}
	logger.Info("✅ Сервисы зарегистрированы")

	return nil
}

// registerBasicSubscribers регистрирует только базовых подписчиков
func (dm *DataManager) registerBasicSubscribers() {
	logger.Info("📋 Начало регистрации подписчиков...")

	// Консольный логгер для ошибок и сигналов
	consoleSubscriber := events.NewConsoleLoggerSubscriber()
	dm.eventBus.Subscribe(types.EventSignalDetected, consoleSubscriber)
	dm.eventBus.Subscribe(types.EventPriceUpdated, consoleSubscriber)
	dm.eventBus.Subscribe(types.EventError, consoleSubscriber)
	logger.Info("✅ Консольный логгер подписан")

	// TelegramDeliveryPackage автоматически подписывает контроллеры в методе Initialize()
	// Нет необходимости вручную подписывать здесь
	logger.Info("ℹ️ TelegramDeliveryPackage автоматически подписывает контроллеры при инициализации")

	logger.Info("🎯 Регистрация подписчиков завершена")
}

// subscribeNotificationService подписывает notification service на события сигналов
func (dm *DataManager) subscribeNotificationService() {
	if dm.notification == nil {
		return
	}

	notificationSubscriber := events.NewBaseSubscriber(
		"notification_service",
		[]types.EventType{types.EventSignalDetected},
		func(event types.Event) error {
			if dm.notification != nil && dm.notification.IsEnabled() {
				if signal, ok := event.Data.(analysis.Signal); ok {
					trendSignal := adapters.AnalysisSignalToTrendSignal(signal)
					return dm.notification.Send(trendSignal)
				}
			}
			return nil
		},
	)

	dm.eventBus.Subscribe(types.EventSignalDetected, notificationSubscriber)
	logger.Info("✅ Сервис уведомлений подписан на события сигналов")
}

// setupPipeline настраивает этапы обработки сигналов
func (dm *DataManager) setupPipeline() {
	dm.signalPipeline.AddStage(&pipeline.ValidationStage{})
	dm.signalPipeline.AddStage(&pipeline.EnrichmentStage{})
	logger.Info("✅ Этапы пайплайна настроены")
}

// registerServices регистрирует сервисы в реестре
func (dm *DataManager) registerServices() error {
	// Создаем обертки для сервисов (используем специализированные обертки где нужно)
	services := map[string]Service{
		// Инфраструктурные сервисы (требуют Start/Stop)
		"DatabaseService": &DatabaseServiceWrapper{DatabaseService: dm.databaseService},
		"RedisService":    &RedisServiceWrapper{RedisService: dm.redisService},
		"EventBus":        &EventBusWrapper{EventBus: dm.eventBus},

		// Хранение и данные (используем специализированные обертки)
		"PriceStorage": &PriceStorageWrapper{PriceStorage: dm.storage},
		"PriceFetcher": &PriceFetcherWrapper{PriceFetcher: dm.priceFetcher},
		"CandleSystem": &CandleSystemWrapper{CandleSystem: dm.candleSystem},

		// Анализ и обработка
		"AnalysisEngine": &AnalysisEngineWrapper{AnalysisEngine: dm.analysisEngine},
		"SignalPipeline": &SignalPipelineWrapper{SignalPipeline: dm.signalPipeline},

		// Уведомления
		"NotificationService": &NotificationServiceWrapper{CompositeNotificationService: dm.notification},

		// Telegram сервисы
		"TelegramBot": &TelegramBotWrapper{TelegramBot: dm.telegramBot},

		// Бизнес-сервисы (не требуют Start/Stop)
		"UserService":         NewUniversalServiceWrapper("UserService", dm.userService, false, false),
		"SubscriptionService": NewUniversalServiceWrapper("SubscriptionService", dm.subscriptionService, false, false),
	}

	// TelegramDeliveryPackage добавляем только если создан
	if dm.telegramDeliveryPackage != nil {
		services["TelegramDeliveryPackage"] = NewUniversalServiceWrapper("TelegramDeliveryPackage", dm.telegramDeliveryPackage, true, true)
	}

	// WebhookServer добавляем только если он создан
	if dm.webhookServer != nil {
		services["WebhookServer"] = NewUniversalServiceWrapper("WebhookServer", dm.webhookServer, true, true)
	}

	for name, service := range services {
		// Проверяем что сервис не nil
		if service == nil {
			logger.Warn("⚠️ Сервис %s равен nil, пропускаем регистрацию", name)
			continue
		}

		// Для UniversalServiceWrapper проверяем что обернутый сервис не nil
		if wrapper, ok := service.(*UniversalServiceWrapper); ok {
			if wrapper.service == nil {
				logger.Warn("⚠️ Базовый сервис %s не создан, пропускаем регистрацию", name)
				continue
			}
		}

		// Для специализированных оберток проверяем внутренний сервис
		switch s := service.(type) {
		case *PriceFetcherWrapper:
			if s.PriceFetcher == nil {
				logger.Warn("⚠️ PriceFetcher не создан, пропускаем регистрацию")
				continue
			}
		case *DatabaseServiceWrapper:
			if s.DatabaseService == nil {
				logger.Warn("⚠️ DatabaseService не создан, пропускаем регистрацию")
				continue
			}
		case *RedisServiceWrapper:
			if s.RedisService == nil {
				logger.Warn("⚠️ RedisService не создан, пропускаем регистрацию")
				continue
			}
		case *EventBusWrapper:
			if s.EventBus == nil {
				logger.Warn("⚠️ EventBus не создан, пропускаем регистрацию")
				continue
			}
		case *CandleSystemWrapper:
			if s.CandleSystem == nil {
				logger.Warn("⚠️ CandleSystem не создан, пропускаем регистрацию")
				continue
			}
		case *AnalysisEngineWrapper:
			if s.AnalysisEngine == nil {
				logger.Warn("⚠️ AnalysisEngine не создан, пропускаем регистрацию")
				continue
			}
		case *SignalPipelineWrapper:
			if s.SignalPipeline == nil {
				logger.Warn("⚠️ SignalPipeline не создан, пропускаем регистрацию")
				continue
			}
		case *NotificationServiceWrapper:
			if s.CompositeNotificationService == nil {
				logger.Warn("⚠️ NotificationService не создан, пропускаем регистрацию")
				continue
			}
		case *TelegramBotWrapper:
			if s.TelegramBot == nil {
				logger.Warn("⚠️ TelegramBot не создан, пропускаем регистрацию")
				continue
			}
		case *PriceStorageWrapper:
			if s.PriceStorage == nil {
				logger.Warn("⚠️ PriceStorage не создан, пропускаем регистрацию")
				continue
			}
		}

		if err := dm.registry.Register(name, service); err != nil {
			return fmt.Errorf("не удалось зарегистрировать сервис %s: %w", name, err)
		}
		logger.Info("✅ Зарегистрирован сервис: %s", name)
	}

	return nil
}

// setupDependencies настраивает зависимости между сервисами
func (dm *DataManager) setupDependencies() {
	// Анализ зависит от хранилища и EventBus
	dm.lifecycle.AddDependency("AnalysisEngine", "PriceStorage")
	dm.lifecycle.AddDependency("AnalysisEngine", "EventBus")

	// Пайплайн зависит от EventBus
	dm.lifecycle.AddDependency("SignalPipeline", "EventBus")

	// NotificationService зависит от EventBus
	dm.lifecycle.AddDependency("NotificationService", "EventBus")

	// TelegramBot зависит от EventBus
	if dm.telegramBot != nil {
		dm.lifecycle.AddDependency("TelegramBot", "EventBus")
	}

	// WebhookServer зависит от TelegramBot
	if dm.webhookServer != nil {
		dm.lifecycle.AddDependency("WebhookServer", "TelegramBot")
	}

	// TelegramDeliveryPackage зависит от EventBus
	if dm.telegramDeliveryPackage != nil {
		dm.lifecycle.AddDependency("TelegramDeliveryPackage", "EventBus")
	}
}
