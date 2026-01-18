// internal/adapters/notification/factory.go
package notification

import (
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	events "crypto-exchange-screener-bot/internal/types"
	"log"
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

	return service
}
