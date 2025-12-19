// internal/events/factory.go
package events

import (
	"crypto-exchange-screener-bot/internal/analysis"
	"crypto-exchange-screener-bot/internal/config"
	"crypto-exchange-screener-bot/internal/notifier"
	"crypto-exchange-screener-bot/internal/telegram"
	"crypto-exchange-screener-bot/internal/types"
	"time"
)

// Factory - фабрика для создания EventBus
type Factory struct{}

// NewEventBusFromConfig создает EventBus из конфигурации
func (f *Factory) NewEventBusFromConfig(cfg *config.Config) *EventBus {
	// Настраиваем конфигурацию EventBus на основе конфигурации приложения
	eventBusConfig := EventBusConfig{
		BufferSize:      cfg.MaxConcurrentRequests * 10,
		WorkerCount:     cfg.MaxConcurrentRequests,
		MaxRetries:      3,
		RetryDelay:      100 * time.Millisecond,
		EnableMetrics:   true,
		EnableLogging:   cfg.LogLevel == "debug",
		DeadLetterQueue: true,
	}

	bus := NewEventBus(eventBusConfig)

	// Добавляем middleware в зависимости от конфигурации
	if cfg.LogLevel == "debug" {
		bus.AddMiddleware(&LoggingMiddleware{})
	}

	bus.AddMiddleware(&ValidationMiddleware{})
	bus.AddMiddleware(&MetricsMiddleware{metrics: bus.metrics})

	// Добавляем rate limiting если нужно
	if cfg.RateLimitDelay > 0 {
		rateLimits := map[EventType]time.Duration{
			EventPriceUpdated:   cfg.RateLimitDelay,
			EventSignalDetected: 2 * time.Second,
		}
		bus.AddMiddleware(NewRateLimitingMiddleware(rateLimits))
	}

	return bus
}

// RegisterDefaultSubscribers регистрирует стандартных подписчиков
func (f *Factory) RegisterDefaultSubscribers(bus *EventBus, cfg *config.Config) {
	// Консольный логгер (всегда включен)
	consoleLogger := NewConsoleLoggerSubscriber()
	bus.Subscribe(EventPriceUpdated, consoleLogger)
	bus.Subscribe(EventSignalDetected, consoleLogger)
	bus.Subscribe(EventError, consoleLogger)

	// Логгер в файл если включено
	if cfg.LogFile != "" {
		fileLogger := NewFileLoggerSubscriber(cfg.LogFile)
		bus.Subscribe(EventPriceUpdated, fileLogger)
		bus.Subscribe(EventSignalDetected, fileLogger)
		bus.Subscribe(EventError, fileLogger)
	}

	// Telegram нотификатор если включен
	if cfg.TelegramEnabled && cfg.TelegramAPIKey != "" {
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

// NewFileLoggerSubscriber создает подписчика для логирования в файл
func NewFileLoggerSubscriber(logFile string) *BaseSubscriber {
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
