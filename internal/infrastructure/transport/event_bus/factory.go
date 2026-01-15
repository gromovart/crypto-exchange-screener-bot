// internal/infrastructure/transport/event_bus/factory.go
package events

import (
	notifier "crypto-exchange-screener-bot/internal/adapters/notification"
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	telegrambot "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot" // ИЗМЕНЕНО
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
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
	telegramBot *telegrambot.TelegramBot, // ИЗМЕНЕНО тип
	notificationService *notifier.CompositeNotificationService,
) {
	// Консольный логгер (всегда включен)
	consoleLogger := f.createConsoleLoggerSubscriber()
	bus.Subscribe(types.EventPriceUpdated, consoleLogger)
	bus.Subscribe(types.EventSignalDetected, consoleLogger)
	bus.Subscribe(types.EventError, consoleLogger)

	// Telegram нотификатор если включен
	if cfg.TelegramEnabled && telegramBot != nil {
		log.Println("📱 Регистрация TelegramNotifier подписчика...")

		// Ищем существующий TelegramNotifier в CompositeNotificationService
		var telegramNotifier *notifier.TelegramNotifier

		if notificationService != nil {
			// Пробуем получить существующий TelegramNotifier
			for _, n := range notificationService.GetNotifiers() {
				if tn, ok := n.(*notifier.TelegramNotifier); ok {
					telegramNotifier = tn
					break
				}
			}
		}

		// Если не нашли существующий, создаем новый с EventBus
		if telegramNotifier == nil {
			telegramNotifier = notifier.NewTelegramNotifier(cfg, bus) // ПЕРЕДАЕМ bus
			if telegramNotifier != nil && notificationService != nil {
				notificationService.AddNotifier(telegramNotifier)
				log.Println("✅ TelegramNotifier создан и добавлен")
			}
		}

		if telegramNotifier != nil {
			// Обертка в BaseSubscriber
			telegramSubscriber := NewBaseSubscriber(
				"telegram_notifier",
				[]types.EventType{types.EventSignalDetected},
				func(event types.Event) error {
					// Получаем сигнал из события
					if signal, ok := event.Data.(types.TrendSignal); ok {
						return telegramNotifier.Send(signal)
					}
					// Если это другой тип сигнала (например, analysis.Signal), конвертируем
					if analysisSignal, ok := event.Data.(analysis.Signal); ok {
						// Конвертируем analysis.Signal в types.TrendSignal
						trendSignal := convertAnalysisSignalToTrendSignal(analysisSignal)
						return telegramNotifier.Send(trendSignal)
					}
					return nil
				},
			)
			bus.Subscribe(types.EventSignalDetected, telegramSubscriber)
			log.Println("✅ TelegramNotifier подписчик зарегистрирован")
		} else {
			log.Println("⚠️ Не удалось создать TelegramNotifier")
		}
	} else if cfg.TelegramEnabled && telegramBot == nil {
		log.Println("⚠️ Telegram включен в конфигурации, но бот не передан")
	}
}

// createConsoleLoggerSubscriber создает подписчика для консольного логирования
func (f *Factory) createConsoleLoggerSubscriber() *BaseSubscriber {
	return NewBaseSubscriber(
		"console_logger",
		[]types.EventType{types.EventPriceUpdated, types.EventSignalDetected, types.EventError},
		func(event types.Event) error {
			// Реализация консольного логирования
			logger.Debug("[Console Logger] Event: %v, Type: %v\n", event.Type, event.Timestamp)
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
