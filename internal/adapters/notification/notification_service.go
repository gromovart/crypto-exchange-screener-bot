// internal/adapters/notification/notification_service.go
package notification

import (
	"crypto-exchange-screener-bot/internal/types"
	"fmt"
	"log"
	"sync"
	"time"
)

// CompositeNotificationService композитный сервис уведомлений
type CompositeNotificationService struct {
	notifiers []Notifier
	enabled   bool
	mu        sync.RWMutex
	stats     map[string]interface{}
}

// Notifier интерфейс отдельного нотификатора
type Notifier interface {
	Send(signal types.TrendSignal) error
	Name() string
	IsEnabled() bool
	SetEnabled(bool)
	GetStats() map[string]interface{}
}

// NewCompositeNotificationService создает композитный сервис
func NewCompositeNotificationService() *CompositeNotificationService {
	return &CompositeNotificationService{
		notifiers: make([]Notifier, 0),
		enabled:   true,
		stats: map[string]interface{}{
			"total_sent":     0,
			"successful":     0,
			"failed":         0,
			"last_sent_time": time.Time{},
		},
	}
}

// GetNotifiers возвращает все зарегистрированные нотификаторы
func (c *CompositeNotificationService) GetNotifiers() []Notifier {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Возвращаем копию списка
	notifiers := make([]Notifier, len(c.notifiers))
	copy(notifiers, c.notifiers)
	return notifiers
}

// GetNotifierByName возвращает нотификатор по имени
func (c *CompositeNotificationService) GetNotifierByName(name string) Notifier {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, notifier := range c.notifiers {
		if notifier.Name() == name {
			return notifier
		}
	}
	return nil
}

// Name возвращает имя сервиса
func (c *CompositeNotificationService) Name() string {
	return "composite_notification_service"
}

// Send отправляет сигнал через все нотификаторы
func (c *CompositeNotificationService) Send(signal types.TrendSignal) error {
	if !c.enabled {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	var lastError error
	sentCount := 0

	for _, notifier := range c.notifiers {
		if notifier.IsEnabled() {
			if err := notifier.Send(signal); err != nil {
				log.Printf("❌ Ошибка отправки через %s: %v", notifier.Name(), err)
				lastError = err
			} else {
				sentCount++
			}
		}
	}

	// Обновляем статистику
	c.stats["total_sent"] = c.stats["total_sent"].(int) + 1
	if sentCount == len(c.notifiers) {
		c.stats["successful"] = c.stats["successful"].(int) + 1
	} else {
		c.stats["failed"] = c.stats["failed"].(int) + 1
	}
	c.stats["last_sent_time"] = time.Now()

	if sentCount == 0 {
		return lastError
	}

	return nil
}

// SendBatch отправляет пакет сигналов
func (c *CompositeNotificationService) SendBatch(signals []types.TrendSignal) error {
	if !c.enabled || len(signals) == 0 {
		return nil
	}

	for _, signal := range signals {
		if err := c.Send(signal); err != nil {
			return err
		}
	}

	return nil
}

// SetEnabled включает/выключает сервис
func (c *CompositeNotificationService) SetEnabled(enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.enabled = enabled
}

// IsEnabled возвращает статус
func (c *CompositeNotificationService) IsEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.enabled
}

// GetStats возвращает статистику
func (c *CompositeNotificationService) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Объединяем статистику всех нотификаторов
	result := make(map[string]interface{})
	for k, v := range c.stats {
		result[k] = v
	}

	notifierStats := make(map[string]interface{})
	for _, notifier := range c.notifiers {
		notifierStats[notifier.Name()] = notifier.GetStats()
	}
	result["notifiers"] = notifierStats

	return result
}

// AddNotifier добавляет нотификатор
func (c *CompositeNotificationService) AddNotifier(notifier Notifier) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.notifiers = append(c.notifiers, notifier)
}

// RemoveNotifier удаляет нотификатор
func (c *CompositeNotificationService) RemoveNotifier(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, notifier := range c.notifiers {
		if notifier.Name() == name {
			c.notifiers = append(c.notifiers[:i], c.notifiers[i+1:]...)
			break
		}
	}
}

// Start запускает сервис уведомлений
func (c *CompositeNotificationService) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.enabled = true
	c.stats["started_at"] = time.Now()
	c.stats["enabled"] = true

	return nil
}

// Stop останавливает сервис уведомлений
func (c *CompositeNotificationService) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.enabled = false
	c.stats["stopped_at"] = time.Now()
	c.stats["enabled"] = false

	return nil
}

// State возвращает состояние сервиса
func (c *CompositeNotificationService) State() string {
	if c.enabled {
		return "running"
	}
	return "stopped"
}

// IsRunning возвращает true если сервис запущен
func (c *CompositeNotificationService) IsRunning() bool {
	return c.enabled
}

// HealthCheck проверяет здоровье сервиса
func (c *CompositeNotificationService) HealthCheck() bool {
	// Базовые проверки
	if !c.enabled {
		return false
	}

	// Проверяем наличие нотификаторов
	if len(c.notifiers) == 0 {
		return false
	}

	// Проверяем что хотя бы один нотификатор работает
	for _, notifier := range c.notifiers {
		// Предполагаем, что нотификатор имеет метод IsEnabled или аналогичный
		if enabled, ok := notifier.(interface{ IsEnabled() bool }); ok {
			if enabled.IsEnabled() {
				return true
			}
		}
	}

	return false
}

// GetStatus возвращает подробный статус
func (c *CompositeNotificationService) GetStatus() map[string]interface{} {
	stats := c.GetStats()

	status := map[string]interface{}{
		"name":        c.Name(),
		"running":     c.enabled,
		"state":       c.State(),
		"healthy":     c.HealthCheck(),
		"total_stats": stats,
	}

	// Информация о нотификаторах
	notifierInfo := make(map[string]interface{})
	for _, notifier := range c.notifiers {
		notifierInfo[notifier.Name()] = map[string]interface{}{
			"type": fmt.Sprintf("%T", notifier),
		}
	}
	status["notifiers"] = notifierInfo
	status["notifier_count"] = len(c.notifiers)

	return status
}
