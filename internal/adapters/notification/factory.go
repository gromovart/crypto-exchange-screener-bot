// internal/adapters/notification/factory.go
package notification

import (
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	events "crypto-exchange-screener-bot/internal/types"
	"log"
	"time"
)

// NotifierFactory фабрика для создания нотификаторов
type NotifierFactory struct {
	eventBus events.EventBus // Добавить EventBus
}

// NewNotifierFactory создает новую фабрику нотификаторов
func NewNotifierFactory(eventBus events.EventBus) *NotifierFactory {
	return &NotifierFactory{
		eventBus: eventBus,
	}
}

// CreateNotifier создает нотификатор на основе конфигурации
func (nf *NotifierFactory) CreateNotifier(cfg *config.Config) Notifier {
	if cfg == nil {
		return nil
	}

	// Если включен Telegram, создаем TelegramNotifier
	if cfg.Telegram.Enabled {
		notifier := NewTelegramNotifier(cfg, nf.eventBus)
		if notifier != nil {
			log.Println("✅ Создан TelegramNotifier с EventBus")
			return notifier
		}
	}

	// Fallback на консольный нотификатор
	log.Println("⚠️ Telegram не настроен, использую консольный нотификатор")
	return NewConsoleNotifier(cfg.Display.Mode == "compact")
}

// CreateCompositeNotifier создает композитный нотификатор с подходящими типами
func (nf *NotifierFactory) CreateCompositeNotifier(cfg *config.Config) *CompositeNotificationService {
	service := NewCompositeNotificationService()

	// Всегда добавляем консольный нотификатор
	consoleNotifier := NewConsoleNotifier(cfg.Display.Mode == "compact")
	service.AddNotifier(consoleNotifier)

	// Добавляем Telegram нотификатор если включен
	if cfg.Telegram.Enabled {
		telegramNotifier := NewTelegramNotifier(cfg, nf.eventBus)
		if telegramNotifier != nil {
			service.AddNotifier(telegramNotifier)

			// Отправляем системное сообщение о запуске
			go func() {
				time.Sleep(2 * time.Second)
				if err := telegramNotifier.SendStartupMessage("Crypto Exchange Screener Bot", "1.0.0"); err != nil {
					log.Printf("⚠️ Не удалось отправить startup сообщение: %v", err)
				}
			}()
		}
	}

	return service
}
