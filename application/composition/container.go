// application/composition/container.go
package composition

import (
	"crypto-exchange-screener-bot/internal/infrastructure/config"
)

// Container - DI контейнер
type Container struct {
	Config *config.Config

	// // Инфраструктура
	// BybitClient    *bybit.Client
	// TelegramClient *messaging.TelegramClient

	// // Адаптеры
	// MarketFetcher  ports.MarketFetcher
	// SignalDetector ports.SignalDetector
	// Notifier       ports.Notifier
	// PriceStorage   ports.PriceStorage

	// // Сервисы приложения
	// DataManager *services.DataManager
}

// // NewContainer создает и настраивает контейнер
// func NewContainer(cfg *config.Config) (*Container, error) {
// 	c := &Container{
// 		Config: cfg,
// 	}

// 	// 1. Создаем инфраструктурные компоненты
// 	c.BybitClient = bybit.NewClient(cfg.BybitAPIKey, cfg.BybitAPISecret)
// 	c.TelegramClient = messaging.NewTelegramClient(cfg.TelegramBotToken)

// 	// 2. Создаем адаптеры (реализации портов)
// 	c.PriceStorage = storage.NewMemoryStorage()
// 	c.MarketFetcher = market.NewBybitFetcher(c.BybitClient, c.PriceStorage)

// 	// 3. Создаем доменные сервисы
// 	detectorService := signals.NewDetectorService(c.PriceStorage)
// 	c.SignalDetector = detectorService

// 	// 4. Создаем нотификаторы
// 	compositeNotifier := notification.NewCompositeNotifier()
// 	compositeNotifier.AddNotifier(notification.NewConsoleNotifier())
// 	compositeNotifier.AddNotifier(messaging.NewTelegramNotifier(c.TelegramClient))
// 	c.Notifier = compositeNotifier

// 	// 5. Создаем сервисы приложения
// 	c.DataManager = services.NewDataManager(
// 		c.MarketFetcher,
// 		c.SignalDetector,
// 		c.Notifier,
// 		c.PriceStorage,
// 		nil, // userRepository если нужен
// 	)

// 	return c, nil
// }
