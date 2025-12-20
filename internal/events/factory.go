// internal/events/factory.go (исправленная версия)
package events

import (
	"crypto-exchange-screener-bot/internal/analysis"
	"crypto-exchange-screener-bot/internal/config"
	"crypto-exchange-screener-bot/internal/notifier"
	"crypto-exchange-screener-bot/internal/telegram"
	"crypto-exchange-screener-bot/internal/types"
	"fmt"
	"time"
)

// Factory - фабрика для создания EventBus
type Factory struct{}

// NewEventBusFromConfig создает EventBus из конфигурации
func (f *Factory) NewEventBusFromConfig(cfg *config.Config) *EventBus {
	// Настраиваем конфигурацию EventBus на основе конфигурации приложения
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
func (f *Factory) RegisterDefaultSubscribers(bus *EventBus, cfg *config.Config) {
	// Консольный логгер (всегда включен)
	consoleLogger := f.createConsoleLoggerSubscriber()
	bus.Subscribe(EventPriceUpdated, consoleLogger)
	bus.Subscribe(EventSignalDetected, consoleLogger)
	bus.Subscribe(EventError, consoleLogger)

	// Telegram нотификатор если включен
	if cfg.TelegramEnabled && cfg.TelegramBotToken != "" {
		// Создаем телеграм бота
		telegramBot := telegram.NewTelegramBot(cfg)
		if telegramBot != nil {
			// Создаем нотификатор
			telegramNotifier := notifier.NewTelegramNotifier(cfg)
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
			}
		}
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

// createFileLoggerSubscriber создает подписчика для логирования в файл
func (f *Factory) createFileLoggerSubscriber(logFile string) *BaseSubscriber {
	return NewBaseSubscriber(
		"file_logger",
		[]EventType{EventPriceUpdated, EventSignalDetected, EventError},
		func(event Event) error {
			// Реализация логирования в файл
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
