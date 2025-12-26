package events

import (
	"crypto-exchange-screener-bot/internal/analysis"
	"crypto-exchange-screener-bot/internal/delivery/telegram"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"crypto-exchange-screener-bot/internal/notifier"
	"crypto-exchange-screener-bot/internal/types"
	"fmt"
	"log"
	"time"
)

// Factory - фабрика для создания EventBus
type Factory struct{}

// NewEventBusFromConfig создает EventBus из конфигурации
func (f *Factory) NewEventBusFromConfig(cfg *config.Config) *EventBus {
	// Настраиваем конфигурацию EventBus
	eventBusConfig := EventBusConfig{
		BufferSize:      cfg.EventBus.BufferSize,
		WorkerCount:     cfg.EventBus.WorkerCount,
		MaxRetries:      3,
		RetryDelay:      100 * time.Millisecond,
		EnableMetrics:   cfg.EventBus.EnableMetrics,
		EnableLogging:   cfg.EventBus.EnableLogging,
		DeadLetterQueue: true,
	}

	bus := NewEventBus(eventBusConfig)

	// Добавляем middleware в зависимости от конфигурации
	if cfg.LogLevel == "debug" {
		bus.AddMiddleware(&LoggingMiddleware{})
	}

	bus.AddMiddleware(&ValidationMiddleware{})
	bus.AddMiddleware(&MetricsMiddleware{metrics: bus.metrics})

	return bus
}

// RegisterDefaultSubscribers регистрирует стандартных подписчиков
func (f *Factory) RegisterDefaultSubscribers(
	bus *EventBus,
	cfg *config.Config,
	telegramBot *telegram.TelegramBot,
	notificationService *notifier.CompositeNotificationService,
) {
	// Консольный логгер (всегда включен)
	consoleLogger := f.createConsoleLoggerSubscriber()
	bus.Subscribe(EventPriceUpdated, consoleLogger)
	bus.Subscribe(EventSignalDetected, consoleLogger)
	bus.Subscribe(EventError, consoleLogger)

	// Telegram нотификатор если включен
	if cfg.TelegramEnabled && telegramBot != nil {
		log.Println("📱 Регистрация EnhancedTelegramNotifier...")
		telegramBot := telegram.GetBot()

		if telegramBot == nil {
			log.Println("⚠️ Telegram бот не инициализирован")
			return
		}
		// Ищем существующий EnhancedTelegramNotifier в CompositeNotificationService
		var enhancedNotifier *notifier.EnhancedTelegramNotifier

		if notificationService != nil {
			// Пробуем получить существующий EnhancedTelegramNotifier
			for _, n := range notificationService.GetNotifiers() {
				if enh, ok := n.(*notifier.EnhancedTelegramNotifier); ok {
					enhancedNotifier = enh
					break
				}
			}
		}

		// Если не нашли существующий, создаем новый
		if enhancedNotifier == nil {
			enhancedNotifier = notifier.NewEnhancedTelegramNotifier(cfg)
			if enhancedNotifier != nil && notificationService != nil {
				notificationService.AddNotifier(enhancedNotifier)
				log.Println("✅ EnhancedTelegramNotifier создан и добавлен")
			}
		}

		if enhancedNotifier != nil {
			// Обертка в BaseSubscriber
			telegramSubscriber := NewBaseSubscriber(
				"enhanced_telegram_notifier",
				[]EventType{EventSignalDetected},
				func(event Event) error {
					// Получаем сигнал из события
					if signal, ok := event.Data.(types.TrendSignal); ok {
						return enhancedNotifier.Send(signal)
					}
					// Если это другой тип сигнала (например, analysis.Signal), конвертируем
					if analysisSignal, ok := event.Data.(analysis.Signal); ok {
						// Конвертируем analysis.Signal в types.TrendSignal
						trendSignal := convertAnalysisSignalToTrendSignal(analysisSignal)
						return enhancedNotifier.Send(trendSignal)
					}
					return nil
				},
			)
			bus.Subscribe(EventSignalDetected, telegramSubscriber)
			log.Println("✅ EnhancedTelegramNotifier подписчик зарегистрирован")
		} else {
			log.Println("⚠️ Не удалось создать EnhancedTelegramNotifier")
		}
	} else if cfg.TelegramEnabled && telegramBot == nil {
		log.Println("⚠️ Telegram включен в конфигурации, но бот не передан")
	}
}

// createConsoleLoggerSubscriber создает подписчика для консольного логирования
func (f *Factory) createConsoleLoggerSubscriber() *BaseSubscriber {
	return NewBaseSubscriber(
		"console_logger",
		[]EventType{EventPriceUpdated, EventSignalDetected, EventError},
		func(event Event) error {
			// Реализация консольного логирования
			fmt.Printf("[Console Logger] Event: %v, Type: %v\n", event.Type, event.Timestamp)
			return nil
		},
	)
}

// convertAnalysisSignalToTrendSignal конвертирует analysis.Signal в types.TrendSignal
func convertAnalysisSignalToTrendSignal(signal analysis.Signal) types.TrendSignal {
	direction := "neutral"
	if signal.Direction == "up" || signal.Type == "growth" {
		direction = "growth"
	} else if signal.Direction == "down" || signal.Type == "fall" {
		direction = "fall"
	}

	return types.TrendSignal{
		Symbol:        signal.Symbol,
		Direction:     direction,
		ChangePercent: signal.ChangePercent,
		PeriodMinutes: signal.Period,
		Timestamp:     signal.Timestamp,
		Confidence:    signal.Confidence,
		DataPoints:    signal.DataPoints,
	}
}
