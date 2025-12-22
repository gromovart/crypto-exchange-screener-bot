// internal/events/factory.go
package events

import (
	"crypto_exchange_screener_bot/internal/config"
	"crypto_exchange_screener_bot/internal/notifier"
	"crypto_exchange_screener_bot/internal/telegram"
	"crypto_exchange_screener_bot/internal/types/analysis"
	"crypto_exchange_screener_bot/internal/types/common"
	"crypto_exchange_screener_bot/internal/types/events"
	"fmt"
	"log"
)

// Factory - фабрика для создания EventBus
type Factory struct{}

// RegisterDefaultSubscribers регистрирует стандартных подписчиков
func (f *Factory) RegisterDefaultSubscribers(
	bus *EventBus,
	cfg *config.Config,
	telegramBot *telegram.TelegramBot,
	notificationService *notifier.CompositeNotificationService,
) {
	// Консольный логгер (всегда включен)
	consoleLogger := f.createConsoleLoggerSubscriber()
	bus.Subscribe(events.EventPriceUpdated, consoleLogger)
	bus.Subscribe(events.EventSignalDetected, consoleLogger)
	bus.Subscribe(events.EventError, consoleLogger)

	// Telegram нотификатор если включен
	if cfg.TelegramEnabled && telegramBot != nil {
		log.Println("📱 Регистрация EnhancedTelegramNotifier...")

		// Ищем существующий EnhancedTelegramNotifier
		var enhancedNotifier *notifier.EnhancedTelegramNotifier
		if notificationService != nil {
			for _, n := range notificationService.GetNotifiers() {
				if enh, ok := n.(*notifier.EnhancedTelegramNotifier); ok {
					enhancedNotifier = enh
					break
				}
			}
		}

		// Если не нашли, создаем новый
		if enhancedNotifier == nil {
			enhancedNotifier = notifier.NewEnhancedTelegramNotifier(cfg)
			if enhancedNotifier != nil && notificationService != nil {
				notificationService.AddNotifier(enhancedNotifier)
				log.Println("✅ EnhancedTelegramNotifier создан и добавлен")
			}
		}

		if enhancedNotifier != nil {
			// Подписчик для Telegram
			telegramSubscriber := events.NewBaseSubscriber(
				"enhanced_telegram_notifier",
				[]events.EventType{events.EventSignalDetected},
				func(event events.Event) error {
					// Ищем сигнал в Payload
					if signal, ok := event.Payload.(analysis.TrendSignal); ok {
						return enhancedNotifier.Send(signal)
					}
					// Или конвертируем Signal в TrendSignal
					if analysisSignal, ok := event.Payload.(analysis.Signal); ok {
						trendSignal := convertAnalysisSignalToTrendSignal(analysisSignal)
						return enhancedNotifier.Send(trendSignal)
					}

					// Логируем, если не нашли сигнал
					log.Printf("⚠️ Не удалось извлечь сигнал из Payload события %s", event.Type)
					return nil
				},
			)
			bus.Subscribe(events.EventSignalDetected, telegramSubscriber)
			log.Println("✅ EnhancedTelegramNotifier подписчик зарегистрирован")
		}
	} else if cfg.TelegramEnabled && telegramBot == nil {
		log.Println("⚠️ Telegram включен в конфигурации, но бот не передан")
	}
}

// createConsoleLoggerSubscriber создает подписчика для консольного логирования
func (f *Factory) createConsoleLoggerSubscriber() events.Subscriber {
	return events.NewBaseSubscriber(
		"console_logger",
		[]events.EventType{events.EventPriceUpdated, events.EventSignalDetected, events.EventError},
		func(event events.Event) error {
			// Логируем в зависимости от типа события
			switch event.Type {
			case events.EventPriceUpdated:
				if priceData, ok := event.Payload.(common.PriceData); ok {
					fmt.Printf("💰 [%s] Цена обновлена: %s = $%.2f (объем: $%.0f)\n",
						event.Timestamp.Format("15:04:05"),
						priceData.Symbol,
						priceData.Price,
						priceData.Volume24h)
				} else {
					fmt.Printf("💰 [%s] Цена обновлена (неизвестный формат)\n",
						event.Timestamp.Format("15:04:05"))
				}

			case events.EventSignalDetected:
				if signal, ok := event.Payload.(analysis.TrendSignal); ok {
					emoji := "📈"
					if signal.Direction == "fall" {
						emoji = "📉"
					}
					fmt.Printf("%s [%s] Сигнал: %s %s %.2f%% (уверенность: %.0f%%)\n",
						emoji,
						event.Timestamp.Format("15:04:05"),
						signal.Symbol,
						signal.Direction,
						signal.ChangePercent,
						signal.Confidence)
				} else if signal, ok := event.Payload.(analysis.Signal); ok {
					emoji := "📈"
					if string(signal.Direction) == "down" || string(signal.Direction) == "bearish" {
						emoji = "📉"
					}
					fmt.Printf("%s [%s] Сигнал: %s %s %.2f%%\n",
						emoji,
						event.Timestamp.Format("15:04:05"),
						signal.Symbol,
						signal.Direction,
						signal.ChangePercent)
				}

			case events.EventError:
				if err, ok := event.Payload.(error); ok {
					fmt.Printf("❌ [%s] Ошибка: %v\n",
						event.Timestamp.Format("15:04:05"), err)
				} else if errorData, ok := event.Payload.(events.ErrorEvent); ok {
					fmt.Printf("❌ [%s] Ошибка: %v (компонент: %s)\n",
						event.Timestamp.Format("15:04:05"),
						errorData.Error,
						errorData.Component)
				}
			}
			return nil
		},
	)
}

// convertAnalysisSignalToTrendSignal конвертирует analysis.Signal в types.TrendSignal
func convertAnalysisSignalToTrendSignal(signal analysis.Signal) analysis.TrendSignal {
	direction := "neutral"
	if signal.Direction == "up" || signal.Type == "growth" {
		direction = "growth"
	} else if signal.Direction == "down" || signal.Type == "fall" {
		direction = "fall"
	}

	return analysis.TrendSignal{
		Symbol:        signal.Symbol,
		Direction:     direction,
		ChangePercent: signal.ChangePercent,
		PeriodMinutes: signal.Period,
		Timestamp:     signal.Timestamp,
		Confidence:    signal.Confidence,
		DataPoints:    signal.DataPoints,
	}
}
