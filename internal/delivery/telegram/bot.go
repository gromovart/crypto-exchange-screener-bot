// internal/telegram/bot.go
package telegram

import (
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"crypto-exchange-screener-bot/internal/types"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// TelegramBot - бот для отправки уведомлений в Telegram
type TelegramBot struct {
	config        *config.Config
	httpClient    *http.Client
	baseURL       string
	chatID        string
	notifier      *Notifier
	menuManager   *MenuManager
	messageSender *MessageSender
	mu            sync.RWMutex
	startupTime   time.Time
	welcomeSent   bool

	// ДОБАВЛЕНО: флаг для тестового режима
	testMode   bool
	testModeMu sync.RWMutex
}

// NewTelegramBot создает новый экземпляр Telegram бота
func NewTelegramBot(cfg *config.Config) *TelegramBot {
	return GetOrCreateBot(cfg)
}

// NewTelegramBotWithChatID создает бота для конкретного чата (для мониторинга)
func NewTelegramBotWithChatID(cfg *config.Config, chatID string) *TelegramBot {
	if cfg == nil || cfg.TelegramBotToken == "" || chatID == "" {
		log.Println("⚠️ Telegram Bot Token или Chat ID не указаны")
		return nil
	}

	// Создаем копию конфигурации с новым chat_id
	chatConfig := *cfg
	chatConfig.TelegramChatID = chatID

	// Создаем новый бот для мониторинга (не Singleton!)
	messageSender := NewMessageSender(&chatConfig)
	notifier := NewNotifier(&chatConfig)
	notifier.SetMessageSender(messageSender)

	bot := &TelegramBot{
		config:        &chatConfig,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
		baseURL:       fmt.Sprintf("https://api.telegram.org/bot%s/", cfg.TelegramBotToken),
		chatID:        chatID,
		notifier:      notifier,
		menuManager:   NewMenuManager(&chatConfig, messageSender),
		messageSender: messageSender,
		startupTime:   time.Now(),
		welcomeSent:   true, // НЕ отправляем приветствие для мониторинг-бота!
		testMode:      cfg.MonitoringTestMode || false,
	}

	log.Printf("🤖 Создан Telegram бот для мониторинга (chat_id: %s)", chatID)
	return bot
}

// SetTestMode включает/выключает тестовый режим
func (tb *TelegramBot) SetTestMode(enabled bool) {
	tb.testModeMu.Lock()
	tb.testMode = enabled
	tb.testModeMu.Unlock()

	if enabled {
		log.Println("📱 Telegram бот переключен в тестовый режим")
	}
}

// IsTestMode возвращает статус тестового режима
func (tb *TelegramBot) IsTestMode() bool {
	tb.testModeMu.RLock()
	defer tb.testModeMu.RUnlock()
	return tb.testMode
}

// SendWelcomeMessage отправляет приветственное сообщение один раз
func (tb *TelegramBot) SendWelcomeMessage() error {
	// Проверяем, что это основной Singleton бот
	if tb != GetBot() {
		log.Println("📱 Это не основной бот - пропуск приветственного сообщения")
		return nil
	}

	// ПРОВЕРЯЕМ ТЕСТОВЫЙ РЕЖИМ
	if tb.IsTestMode() {
		log.Println("📱 Тестовый режим - пропуск приветственного сообщения")
		return nil
	}

	tb.mu.Lock()
	defer tb.mu.Unlock()

	if tb.welcomeSent {
		log.Println("📱 Приветственное сообщение уже отправлено")
		return nil
	}

	message := "🤖 *Бот активирован!*\n\n" +
		"✅ Система мониторинга роста/падения запущена.\n" +
		"🔔 Уведомления отправляются с ограничением 1 сообщение в 10 секунд.\n" +
		"⚡ Настройки: рост=%.2f%%, падение=%.2f%%\n\n" +
		"Используйте меню ниже для управления ботом ⬇️"

	// Используем настройки из конфигурации анализаторов
	growthThreshold := tb.config.Analyzers.GrowthAnalyzer.MinGrowth
	fallThreshold := tb.config.Analyzers.FallAnalyzer.MinFall

	message = fmt.Sprintf(message, growthThreshold, fallThreshold)

	err := tb.messageSender.SendTextMessage(message, nil, false)
	if err == nil {
		tb.welcomeSent = true
		log.Println("✅ Приветственное сообщение отправлено (Singleton)")
	} else {
		log.Printf("❌ Ошибка отправки приветственного сообщения: %v", err)
	}

	return err
}

// IsNotifyEnabled возвращает статус уведомлений
func (tb *TelegramBot) IsNotifyEnabled() bool {
	tb.mu.RLock()
	defer tb.mu.RUnlock()
	return tb.config.TelegramEnabled
}

// SendNotification отправляет уведомление о сигнале
func (tb *TelegramBot) SendNotification(signal types.GrowthSignal) error {
	if !tb.IsNotifyEnabled() {
		return nil
	}

	return tb.notifier.SendNotification(signal, tb.menuManager.IsEnabled())
}

// SendMessage отправляет текстовое сообщение
func (tb *TelegramBot) SendMessage(text string) error {
	return tb.messageSender.SendTextMessage(text, nil, false)
}

// SendMessageWithKeyboard отправляет сообщение с клавиатурой
func (tb *TelegramBot) SendMessageWithKeyboard(text string, keyboard *InlineKeyboardMarkup) error {
	return tb.messageSender.SendTextMessage(text, keyboard, false)
}

// SendTestMessage отправляет тестовое сообщение (ТОЛЬКО В ТЕСТОВОМ РЕЖИМЕ)
func (tb *TelegramBot) SendTestMessage() error {
	// Если не в тестовом режиме, используем обычное приветствие
	if !tb.IsTestMode() {
		return tb.SendWelcomeMessage()
	}

	// В тестовом режиме - простое сообщение
	message := "🧪 *Тестовое сообщение от бота*\n\n" +
		"Проверка работоспособности системы..."

	return tb.messageSender.SendTextMessage(message, nil, false)
}

// HandleMessage обрабатывает текстовые сообщения из меню
func (tb *TelegramBot) HandleMessage(text, chatID string) error {
	if tb.menuManager == nil {
		return fmt.Errorf("menu manager not initialized")
	}
	return tb.menuManager.HandleMessage(text, chatID)
}

// HandleCallback обрабатывает callback от inline кнопок
func (tb *TelegramBot) HandleCallback(callbackData string, chatID string) error {
	if tb.menuManager == nil {
		return fmt.Errorf("menu manager not initialized")
	}
	return tb.menuManager.HandleCallback(callbackData, chatID)
}

// StartCommandHandler обрабатывает команду /start
func (tb *TelegramBot) StartCommandHandler(chatID string) error {
	if tb.menuManager == nil {
		return fmt.Errorf("menu manager not initialized")
	}
	return tb.menuManager.StartCommandHandler(chatID)
}

// SetMenuEnabled включает/выключает меню
func (tb *TelegramBot) SetMenuEnabled(enabled bool) {
	if tb.menuManager != nil {
		tb.menuManager.SetEnabled(enabled)
	}
}

// IsMenuEnabled возвращает статус меню
func (tb *TelegramBot) IsMenuEnabled() bool {
	if tb.menuManager != nil {
		return tb.menuManager.IsEnabled()
	}
	return false
}

// SendCounterNotification отправляет уведомление счетчика
func (tb *TelegramBot) SendCounterNotification(symbol string, signalType string, count int, maxSignals int, period string) error {
	if !tb.IsNotifyEnabled() {
		return nil
	}

	icon := "🟢"
	directionStr := "РОСТ"
	if signalType == "fall" {
		icon = "🔴"
		directionStr = "ПАДЕНИЕ"
	}

	percentage := float64(count) / float64(maxSignals) * 100
	timeStr := time.Now().Format("15:04:05")

	message := fmt.Sprintf(
		"📊 *Счетчик сигналов*\n"+
			"%s %s\n"+
			"Символ: %s\n"+
			"Текущее: %d/%d (%.0f%%)\n"+
			"Период: %s\n"+
			"🕐 %s",
		icon, directionStr,
		symbol,
		count, maxSignals, percentage,
		period,
		timeStr,
	)

	// Простая клавиатура для счетчика
	keyboard := &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "📊 График", URL: fmt.Sprintf("https://www.tradingview.com/chart/?symbol=BYBIT:%s", symbol)},
				{Text: "💱 Торговать", URL: fmt.Sprintf("https://www.bybit.com/trade/usdt/%s", symbol)},
			},
		},
	}

	return tb.messageSender.SendTextMessage(message, keyboard, true)
}
