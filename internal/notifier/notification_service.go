package notifier

import (
	"crypto-exchange-screener-bot/internal/types"
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
