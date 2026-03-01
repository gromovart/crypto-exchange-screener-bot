// internal/delivery/max/package.go
package max

import (
	"fmt"
	"sync"

	events "crypto-exchange-screener-bot/internal/infrastructure/transport/event_bus"
	"crypto-exchange-screener-bot/pkg/logger"
)

// Package упаковывает всё необходимое для доставки сигналов через MAX
type Package struct {
	mu          sync.RWMutex
	client      *Client
	controller  *Controller
	chatID      int64
	eventBus    *events.EventBus
	initialized bool
	running     bool
}

// NewPackage создаёт новый пакет доставки MAX
func NewPackage(token string, chatID int64) *Package {
	return &Package{
		client: NewClient(token),
		chatID: chatID,
	}
}

// GetClient возвращает HTTP-клиент MAX (для использования в боте)
func (p *Package) GetClient() *Client {
	return p.client
}

// Initialize подписывает контроллер на EventBus
func (p *Package) Initialize(eventBus *events.EventBus) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.initialized {
		return fmt.Errorf("MAX package уже инициализирован")
	}

	if eventBus == nil {
		return fmt.Errorf("eventBus не может быть nil")
	}

	p.eventBus = eventBus
	p.controller = NewController(p.client, p.chatID)

	// Подписываем контроллер на все нужные события
	for _, eventType := range p.controller.GetSubscribedEvents() {
		eventBus.Subscribe(eventType, p.controller)
		logger.Debug("📬 MAX: подписка на событие %s", eventType)
	}

	p.initialized = true
	logger.Info("✅ MAX Package инициализирован (chatID: %d)", p.chatID)
	return nil
}

// Start запускает пакет
func (p *Package) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.initialized {
		return fmt.Errorf("MAX package не инициализирован")
	}
	if p.running {
		return nil
	}
	p.running = true
	logger.Info("🚀 MAX Package запущен")
	return nil
}

// Stop останавливает пакет и отписывает контроллер
func (p *Package) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return
	}

	if p.eventBus != nil && p.controller != nil {
		for _, eventType := range p.controller.GetSubscribedEvents() {
			p.eventBus.Unsubscribe(eventType, p.controller)
		}
	}

	p.running = false
	logger.Info("🛑 MAX Package остановлен")
}
