// internal/telegram/notifier.go
package telegram

import (
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"crypto-exchange-screener-bot/internal/types"
	"fmt"
	"log"
	"sync"
	"time"
)

// Notifier - нотификатор
type Notifier struct {
	config        *config.Config
	messageSender *MessageSender
	menuUtils     *MenuUtils
	rateLimiter   *RateLimiter
	lastSendTime  time.Time
	minInterval   time.Duration
	enabled       bool
	mu            sync.RWMutex
}

// NewNotifier создает новый нотификатор
func NewNotifier(cfg *config.Config) *Notifier {
	return &Notifier{
		config:      cfg,
		menuUtils:   NewMenuUtils(),
		rateLimiter: NewRateLimiter(2 * time.Second),
		minInterval: 2 * time.Second,
		enabled:     cfg.TelegramEnabled,
	}
}

// SetMessageSender устанавливает отправителя сообщений
func (n *Notifier) SetMessageSender(sender *MessageSender) {
	n.messageSender = sender
}

// SetMenuUtils устанавливает утилиты меню
func (n *Notifier) SetMenuUtils(utils *MenuUtils) {
	n.menuUtils = utils
}

// SendNotification отправляет уведомление
func (n *Notifier) SendNotification(signal types.GrowthSignal, menuEnabled bool) error {
	if !n.IsEnabled() {
		return nil
	}

	// Проверяем настройки уведомлений
	if (signal.Direction == "growth" && !n.config.TelegramNotifyGrowth) ||
		(signal.Direction == "fall" && !n.config.TelegramNotifyFall) {
		return nil
	}

	// Проверяем лимит частоты
	key := fmt.Sprintf("signal_%s_%s", signal.Direction, signal.Symbol)
	if !n.rateLimiter.CanSend(key) {
		log.Printf("⚠️ Пропуск Telegram уведомления для %s (лимит частоты)", signal.Symbol)
		return nil
	}

	// Форматируем сообщение (компактный формат)
	message := n.menuUtils.FormatSignalMessage(signal, "compact")

	// Создаем компактную клавиатуру
	keyboard := n.menuUtils.FormatNotificationKeyboard(signal)

	// Отправляем сообщение через MessageSender
	if n.messageSender != nil {
		return n.messageSender.SendTextMessage(message, keyboard, !menuEnabled)
	}

	log.Printf("📨 Отправка уведомления: %s", message)
	return nil
}

// SetEnabled включает/выключает уведомления
func (n *Notifier) SetEnabled(enabled bool) {
	n.mu.Lock()
	n.enabled = enabled
	n.mu.Unlock()
}

// IsEnabled возвращает статус уведомлений
func (n *Notifier) IsEnabled() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.enabled
}
