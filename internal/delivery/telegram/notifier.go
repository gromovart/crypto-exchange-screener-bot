// internal/delivery/telegram/notifier.go
package telegram

import (
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"crypto-exchange-screener-bot/internal/types"
	"log"
	"sync"
	"time"
)

// Notifier - нотификатор
type Notifier struct {
	config           *config.Config
	messageSender    *MessageSender
	messageFormatter *MarketMessageFormatter
	rateLimiter      *RateLimiter
	lastSendTime     time.Time
	minInterval      time.Duration
	enabled          bool
	mu               sync.RWMutex
}

// NewNotifier создает новый нотификатор
func NewNotifier(cfg *config.Config) *Notifier {
	// Используем биржу из конфигурации
	exchange := cfg.Exchange
	if exchange == "" {
		exchange = "bybit" // значение по умолчанию
	}

	return &Notifier{
		config:           cfg,
		messageFormatter: NewMarketMessageFormatter(exchange),
		rateLimiter:      NewRateLimiter(2 * time.Second),
		minInterval:      2 * time.Second,
		enabled:          cfg.TelegramEnabled,
	}
}

// SetMessageSender устанавливает отправителя сообщений
func (n *Notifier) SetMessageSender(sender *MessageSender) {
	n.messageSender = sender
}

// SendNotification отправляет уведомление
func (n *Notifier) SendNotification(signal types.GrowthSignal, menuEnabled bool) error {
	// 🔴 ОТКЛЮЧАЕМ - торговые сигналы отправляет только CounterAnalyzer через CounterNotifier

	if !n.IsEnabled() {
		return nil
	}

	log.Printf("⚠️ Notifier: Торговые сигналы ОТКЛЮЧЕНЫ. Используйте CounterAnalyzer для %s %.2f%% (%s)",
		signal.Symbol, signal.GrowthPercent+signal.FallPercent, signal.Direction)

	// Возвращаем успех, но не отправляем сообщение
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
