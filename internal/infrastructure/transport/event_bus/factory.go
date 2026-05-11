// internal/infrastructure/transport/event_bus/factory.go
package events

import (
	notifier "crypto-exchange-screener-bot/internal/adapters/notification"
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	telegrambot "crypto-exchange-screener-bot/internal/delivery/telegram/app/bot"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
	"time"

	// ДОБАВЛЕНО: импорт контроллера счетчика
	countercontroller "crypto-exchange-screener-bot/internal/delivery/telegram/controllers/counter"
	counterservice "crypto-exchange-screener-bot/internal/delivery/telegram/services/counter"
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

	return bus
}

// RegisterDefaultSubscribers регистрирует стандартных подписчиков
// ДОБАВЛЕН параметр counterService
func (f *Factory) RegisterDefaultSubscribers(
	bus *EventBus,
	cfg *config.Config,
	telegramBot *telegrambot.TelegramBot,
	notificationService *notifier.CompositeNotificationService,
	counterService counterservice.Service, // НОВЫЙ параметр
) {
	// Консольный логгер (всегда включен)
	consoleLogger := f.createConsoleLoggerSubscriber()
	bus.Subscribe(types.EventPriceUpdated, consoleLogger)
	bus.Subscribe(types.EventSignalDetected, consoleLogger)
	bus.Subscribe(types.EventError, consoleLogger)

	// Регистрация контроллера счетчика если есть сервис
	if counterService != nil {
		logger.Info("🔢 Регистрация CounterController подписчика...")

		// Создаем контроллер счетчика
		counterController := countercontroller.NewController(counterService)

		// Создаем подписчика-обертку для контроллера
		counterSubscriber := NewCounterControllerWrapper(counterController)

		// Регистрируем в EventBus
		bus.Subscribe(types.EventCounterSignalDetected, counterSubscriber)
		logger.Info("✅ CounterController подписчик зарегистрирован")
	} else {
		logger.Warn("⚠️ CounterService не предоставлен, CounterController не будет зарегистрирован")
	}

}

// createConsoleLoggerSubscriber создает подписчика для консольного логирования
func (f *Factory) createConsoleLoggerSubscriber() *BaseSubscriber {
	return NewBaseSubscriber(
		"console_logger",
		[]types.EventType{types.EventPriceUpdated, types.EventSignalDetected, types.EventError},
		func(event types.Event) error {
			// Реализация консольного логирования
			logger.Debug("[Console Logger] Event: %v, Type: %v", event.Type, event.Timestamp)
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
