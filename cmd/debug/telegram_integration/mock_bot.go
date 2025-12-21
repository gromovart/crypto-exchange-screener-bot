// cmd/debug/telegram_integration/mock_bot.go
package main

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// MockTelegramBot - мок Telegram бота для тестирования
type MockTelegramBot struct {
	mu                sync.RWMutex
	enabled           bool
	sentMessages      []string
	sentNotifications []MockCounterNotification // Используем локальный тип
	rateLimiter       *MockRateLimiter
	callbacks         map[string]func() string
	config            *MockConfig
}

// MockCounterNotification - уведомление счетчика (локальный тип)
type MockCounterNotification struct {
	Symbol          string
	SignalType      string
	CurrentCount    int
	Period          string
	PeriodStartTime time.Time
	Timestamp       time.Time
	MaxSignals      int
	Percentage      float64
}

// MockRateLimiter - мок ограничителя частоты
type MockRateLimiter struct {
	mu       sync.Mutex
	lastSent map[string]time.Time
	minDelay time.Duration
}

// MockConfig - мок конфигурации
type MockConfig struct {
	TelegramEnabled            bool
	TelegramNotifyGrowth       bool
	TelegramNotifyFall         bool
	MessageFormat              string
	CounterChartProvider       string
	CounterNotificationEnabled bool
}

// NewMockTelegramBot создает новый мок бота
func NewMockTelegramBot() *MockTelegramBot {
	return &MockTelegramBot{
		enabled:           true,
		sentMessages:      []string{},
		sentNotifications: []MockCounterNotification{},
		rateLimiter:       NewMockRateLimiter(2 * time.Second),
		callbacks:         make(map[string]func() string),
		config: &MockConfig{
			TelegramEnabled:            true,
			TelegramNotifyGrowth:       true,
			TelegramNotifyFall:         true,
			MessageFormat:              "compact",
			CounterChartProvider:       "coinglass",
			CounterNotificationEnabled: true,
		},
	}
}

// NewMockRateLimiter создает мок ограничителя
func NewMockRateLimiter(minDelay time.Duration) *MockRateLimiter {
	return &MockRateLimiter{
		lastSent: make(map[string]time.Time),
		minDelay: minDelay,
	}
}

// SetEnabled включает/выключает бота
func (m *MockTelegramBot) SetEnabled(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = enabled
}

// SendCounterNotification отправляет уведомление счетчика
func (m *MockTelegramBot) SendCounterNotification(notification MockCounterNotification) error {
	if !m.enabled || !m.config.CounterNotificationEnabled {
		return nil
	}

	// Проверка лимита частоты
	key := fmt.Sprintf("counter_%s_%s", notification.SignalType, notification.Symbol)
	if !m.rateLimiter.CanSend(key) {
		log.Printf("⚠️ Пропуск уведомления для %s (лимит частоты)", notification.Symbol)
		return nil
	}

	// Форматируем сообщение
	message := m.FormatCounterMessage(notification)

	// Сохраняем уведомление
	m.mu.Lock()
	m.sentNotifications = append(m.sentNotifications, notification)
	m.sentMessages = append(m.sentMessages, message)
	m.mu.Unlock()

	log.Printf("📨 Mock Telegram: Отправлено уведомление для %s", notification.Symbol)
	log.Printf("   • Счетчик: %d/%d (%.0f%%)",
		notification.CurrentCount, notification.MaxSignals, notification.Percentage)

	return nil
}

// FormatCounterMessage форматирует сообщение счетчика
func (m *MockTelegramBot) FormatCounterMessage(notification MockCounterNotification) string {
	icon := "🟢"
	directionStr := "РОСТ"
	if notification.SignalType == "fall" {
		icon = "🔴"
		directionStr = "ПАДЕНИЕ"
	}

	timeStr := notification.Timestamp.Format("2006/01/02 15:04:05")

	return fmt.Sprintf(
		"📊 *Счетчик сигналов*\n"+
			"⚫ Символ: %s\n"+
			"🕐 Время: %s\n"+
			"⏱️  Период: %s\n"+
			"%s Направление: %s\n"+
			"📈 Счетчик: %d/%d (%.0f%%)\n"+
			"📊 Базовый период: %d мин",
		notification.Symbol,
		timeStr,
		notification.Period,
		icon, directionStr,
		notification.CurrentCount, notification.MaxSignals, notification.Percentage,
		1, // базовый период
	)
}

// CanSend проверяет лимит частоты
func (rl *MockRateLimiter) CanSend(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	if last, exists := rl.lastSent[key]; exists {
		if now.Sub(last) < rl.minDelay {
			return false
		}
	}
	rl.lastSent[key] = now
	return true
}

// GetSentMessages возвращает отправленные сообщения
func (m *MockTelegramBot) GetSentMessages() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]string{}, m.sentMessages...)
}

// GetSentNotifications возвращает отправленные уведомления
func (m *MockTelegramBot) GetSentNotifications() []MockCounterNotification {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]MockCounterNotification{}, m.sentNotifications...)
}

// ClearMessages очищает историю сообщений
func (m *MockTelegramBot) ClearMessages() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentMessages = []string{}
	m.sentNotifications = []MockCounterNotification{}
}

// RegisterCallback регистрирует callback обработчик
func (m *MockTelegramBot) RegisterCallback(data string, handler func() string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callbacks[data] = handler
}

// HandleCallback обрабатывает callback
func (m *MockTelegramBot) HandleCallback(callbackData string) string {
	m.mu.RLock()
	handler, exists := m.callbacks[callbackData]
	m.mu.RUnlock()

	if exists && handler != nil {
		return handler()
	}

	// Обработка стандартных callback
	switch callbackData {
	case "counter_notify_on":
		m.config.CounterNotificationEnabled = true
		return "✅ Уведомления счетчика включены"
	case "counter_notify_off":
		m.config.CounterNotificationEnabled = false
		return "❌ Уведомления счетчика выключены"
	case "counter_settings":
		return m.ShowCounterSettings()
	default:
		return fmt.Sprintf("❓ Неизвестный callback: %s", callbackData)
	}
}

// ShowCounterSettings показывает настройки счетчика
func (m *MockTelegramBot) ShowCounterSettings() string {
	return `⚙️ *Настройки счетчика сигналов*

Выберите период анализа:
[5 минут] [15 минут] [30 минут]
[1 час] [4 часа] [1 день]

Настройки:
• Уведомления: ✅ включены
• Чарт: coinglass
• Отслеживать рост: ✅
• Отслеживать падение: ✅`
}

// CreateTestKeyboard создает тестовую клавиатуру
func (m *MockTelegramBot) CreateTestKeyboard() string {
	return `Клавиатура:
[📊 График] [💱 Торговать]
[🔔 Уведомлять] [🔕 Игнорировать]
[⚙️ Настройки счетчика]`
}
