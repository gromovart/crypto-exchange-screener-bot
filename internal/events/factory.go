// internal/events/factory.go
package events

import (
	"crypto-exchange-screener-bot/internal/analysis"
	"crypto-exchange-screener-bot/internal/config"
	"crypto-exchange-screener-bot/internal/notifier"
	"crypto-exchange-screener-bot/internal/telegram"
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
	telegramBot *telegram.TelegramBot, // ПЕРЕДАЕМ БОТА ЧЕРЕЗ DI
	notificationService *notifier.CompositeNotificationService, // И notification service тоже
) {
	// Консольный логгер (всегда включен)
	consoleLogger := f.createConsoleLoggerSubscriber()
	bus.Subscribe(EventPriceUpdated, consoleLogger)
	bus.Subscribe(EventSignalDetected, consoleLogger)
	bus.Subscribe(EventError, consoleLogger)

	// Telegram нотификатор если включен И бот передан
	if cfg.TelegramEnabled && telegramBot != nil {
		log.Println("📱 Регистрация Telegram подписчика с переданным ботом")

		// Используем уже созданный TelegramNotifier или создаем новый с переданным ботом
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

		// Если не нашли существующий, создаем новый
		if telegramNotifier == nil {
			telegramNotifier = notifier.NewTelegramNotifier(cfg, telegramBot)
		}

		if telegramNotifier != nil {
			// Обертка в BaseSubscriber
			telegramSubscriber := NewBaseSubscriber(
				"telegram_notifier",
				[]EventType{EventSignalDetected},
				func(event Event) error {
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
			bus.Subscribe(EventSignalDetected, telegramSubscriber)
			log.Println("✅ Telegram подписчик успешно зарегистрирован")
		}
	} else if cfg.TelegramEnabled && telegramBot == nil {
		log.Println("⚠️ Telegram включен в конфигурации, но бот не передан в RegisterDefaultSubscribers")
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
