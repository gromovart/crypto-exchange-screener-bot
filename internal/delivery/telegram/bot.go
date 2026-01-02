// internal/delivery/telegram/bot.go
package telegram

import (
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"crypto-exchange-screener-bot/internal/types"
	"fmt"
	"log"
	"net/http"
	"strings"
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
	testMode      bool
	testModeMu    sync.RWMutex
	buttonBuilder *ButtonURLBuilder
	menuUtils     *MenuUtils
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
	menuUtils := NewMenuUtils(cfg.Exchange)
	notifier := NewNotifier(&chatConfig)
	notifier.SetMessageSender(messageSender)

	// Используем menuUtils для создания менеджера меню
	menuManager := NewMenuManagerWithUtils(&chatConfig, messageSender, menuUtils)

	// Создаем buttonBuilder для кнопок
	buttonBuilder := NewButtonURLBuilder(cfg.Exchange)

	bot := &TelegramBot{
		config:        &chatConfig,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
		baseURL:       fmt.Sprintf("https://api.telegram.org/bot%s/", cfg.TelegramBotToken),
		chatID:        chatID,
		notifier:      notifier,
		menuManager:   menuManager,
		messageSender: messageSender,
		startupTime:   time.Now(),
		welcomeSent:   true, // НЕ отправляем приветствие для мониторинг-бота!
		testMode:      cfg.MonitoringTestMode || false,
		buttonBuilder: buttonBuilder,
		menuUtils:     menuUtils,
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

	message := fmt.Sprintf(
		"🤖 *Бот активирован!*\n\n"+
			"✅ Система мониторинга роста/падения запущена.\n"+
			"🔔 Уведомления отправляются с ограничением 1 сообщение в 10 секунд.\n"+
			"⚡ Настройки: рост=%.2f%%, падение=%.2f%%\n\n"+
			"Используйте меню ниже для управления ботом ⬇️",
		tb.config.AnalyzerConfigs.GrowthAnalyzer.MinGrowth,
		tb.config.AnalyzerConfigs.FallAnalyzer.MinFall,
	)

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

	// Используем notifier, если он есть
	if tb.notifier != nil {
		return tb.notifier.SendNotification(signal, tb.menuManager.IsEnabled())
	}

	// Если notifier не работает, отправляем напрямую
	message := tb.formatSignalMessage(signal)

	var keyboard *InlineKeyboardMarkup
	if tb.buttonBuilder != nil && signal.Symbol != "" {
		periodMinutes := signal.PeriodMinutes
		if periodMinutes == 0 {
			periodMinutes = tb.getDefaultPeriod()
		}

		// Определяем, нужна ли расширенная клавиатура
		changePercent := tb.getSignalChangePercent(signal)
		volume := signal.Volume24h

		if changePercent >= 5.0 || volume >= 1000000 {
			keyboard = tb.buttonBuilder.EnhancedNotificationKeyboard(signal.Symbol, periodMinutes)
		} else {
			keyboard = tb.buttonBuilder.StandardNotificationKeyboard(signal.Symbol, periodMinutes)
		}
	}

	return tb.messageSender.SendTextMessage(message, keyboard, true)
}

// createSimpleKeyboard создает простую клавиатуру (fallback)
func (tb *TelegramBot) createSimpleKeyboard(symbol string) *InlineKeyboardMarkup {
	// Создаем кнопки вручную, если buttonBuilder недоступен
	chartButton := InlineKeyboardButton{
		Text: ButtonTexts.Chart,
		URL: fmt.Sprintf("https://www.tradingview.com/chart/?symbol=%s:%s",
			strings.ToUpper(tb.config.Exchange), symbol),
	}

	tradeButton := InlineKeyboardButton{
		Text: ButtonTexts.Trade,
		URL: fmt.Sprintf("%s/trade/usdt/%s?interval=15",
			tb.getExchangeBaseURL(), symbol),
	}

	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{chartButton, tradeButton},
		},
	}
}

// getExchangeBaseURL возвращает базовый URL биржи
func (tb *TelegramBot) getExchangeBaseURL() string {
	switch strings.ToLower(tb.config.Exchange) {
	case "binance":
		return "https://www.binance.com"
	case "kucoin":
		return "https://www.kucoin.com"
	case "okx":
		return "https://www.okx.com"
	default:
		return "https://www.bybit.com"
	}
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

	// Используем CreateTestKeyboard() - статический метод
	keyboard := CreateTestKeyboard()

	return tb.messageSender.SendTextMessage(message, keyboard, false)
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
	if tb.menuManager != nil {
		return tb.menuManager.StartCommandHandler(chatID)
	}
	return nil
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

	// Форматируем сообщение
	message := tb.formatCounterMessage(symbol, signalType, count, maxSignals, period)

	// Создаем клавиатуру с использованием ButtonURLBuilder
	var keyboard *InlineKeyboardMarkup
	if tb.buttonBuilder != nil {
		periodMinutes := tb.parsePeriodToMinutes(period)
		keyboard = tb.buttonBuilder.CounterNotificationKeyboard(symbol, periodMinutes)
	} else {
		// Fallback на простую клавиатуру
		keyboard = tb.createSimpleKeyboard(symbol)
	}

	return tb.messageSender.SendTextMessage(message, keyboard, true)
}

// formatSignalMessage форматирует сообщение о сигнале
func (tb *TelegramBot) formatSignalMessage(signal types.GrowthSignal) string {
	// Определяем иконку и направление
	icon, directionStr, changePercent := tb.getSignalInfo(signal)

	// Форматируем сообщение
	return fmt.Sprintf(
		"%s *%s %s на %.2f%%*\n\n"+
			"💰 Цена: $%.2f → $%.2f\n"+
			"📊 Точок данных: %d\n"+
			"📈 Уверенность: %.1f%%\n"+
			"🕐 Время: %s\n\n"+
			"Используйте кнопки ниже для торговли ⬇️",
		icon,
		directionStr,
		signal.Symbol,
		changePercent,
		signal.StartPrice,
		signal.EndPrice,
		signal.DataPoints,
		signal.Confidence,
		signal.Timestamp.Format("15:04:05"),
	)
}

// getSignalInfo возвращает информацию о сигнале
func (tb *TelegramBot) getSignalInfo(signal types.GrowthSignal) (icon, direction string, changePercent float64) {
	// Определяем направление и процент изменения
	if signal.Direction == "growth" {
		icon = "🚀"
		direction = "РОСТ"
		changePercent = signal.GrowthPercent
	} else {
		icon = "📉"
		direction = "ПАДЕНИЕ"
		changePercent = signal.FallPercent
	}
	return
}

// formatCounterMessage форматирует сообщение счетчика
func (tb *TelegramBot) formatCounterMessage(symbol string, signalType string, count int, maxSignals int, period string) string {
	icon := "🟢"
	directionStr := "РОСТ"
	if signalType == "fall" {
		icon = "🔴"
		directionStr = "ПАДЕНИЕ"
	}

	percentage := float64(count) / float64(maxSignals) * 100
	timeStr := time.Now().Format("15:04:05")

	return fmt.Sprintf(
		"📊 *Счетчик сигналов*\n"+
			"%s %s %s\n"+
			"📈 Текущее: %d/%d (%.0f%%)\n"+
			"⏱️ Период: %s\n"+
			"🕐 %s",
		icon,
		directionStr,
		symbol,
		count,
		maxSignals,
		percentage,
		period,
		timeStr,
	)
}

// parsePeriodToMinutes преобразует строку периода в минуты
func (tb *TelegramBot) parsePeriodToMinutes(period string) int {
	switch strings.ToLower(period) {
	case "5m", "5 минут":
		return 5
	case "15m", "15 минут":
		return 15
	case "30m", "30 минут":
		return 30
	case "1h", "1 час":
		return 60
	case "4h", "4 часа":
		return 240
	case "1d", "1 день":
		return 1440
	default:
		return tb.getDefaultPeriod()
	}
}

// getDefaultPeriod возвращает период по умолчанию
func (tb *TelegramBot) getDefaultPeriod() int {
	return 15 // по умолчанию 15 минут
}

// getSignalChangePercent получает процент изменения из сигнала
func (tb *TelegramBot) getSignalChangePercent(signal types.GrowthSignal) float64 {
	if signal.Direction == "growth" {
		return signal.GrowthPercent
	}
	return signal.FallPercent
}

// GetSettingsKeyboard возвращает клавиатуру настроек
func (tb *TelegramBot) GetSettingsKeyboard() *InlineKeyboardMarkup {
	// Используем buttonBuilder если есть, чтобы получить клавиатуру с актуальными статусами
	if tb.buttonBuilder != nil {
		return tb.buttonBuilder.UpdateSettingsKeyboard(tb)
	}

	// Fallback на статическую клавиатуру без статусов
	return CreateSettingsKeyboard()
}

// GetButtonBuilder возвращает строитель кнопок (для тестирования)
func (tb *TelegramBot) GetButtonBuilder() *ButtonURLBuilder {
	return tb.buttonBuilder
}

// GetMenuUtils возвращает утилиты меню (для тестирования)
func (tb *TelegramBot) GetMenuUtils() *MenuUtils {
	return tb.menuUtils
}

// GetStats возвращает статистику бота
func (tb *TelegramBot) GetStats() string {
	tb.mu.RLock()
	defer tb.mu.RUnlock()

	uptime := time.Since(tb.startupTime).Round(time.Second)

	return fmt.Sprintf(
		"%s *Статистика бота*\n\n"+
			"⏱️ Аптайм: %s\n"+
			"📊 Уведомления: %v\n"+
			"🔄 Меню: %v\n"+
			"🧪 Тестовый режим: %v\n"+
			"🏦 Биржа: %s",
		ButtonTexts.Status,
		uptime,
		tb.config.TelegramEnabled,
		tb.menuManager != nil && tb.menuManager.IsEnabled(),
		tb.testMode,
		tb.config.Exchange,
	)
}

// GetNotificationStatus возвращает статус уведомлений в текстовом формате
func (tb *TelegramBot) GetNotificationStatus() string {
	if tb.IsNotifyEnabled() {
		return "✅ Включены"
	}
	return "❌ Выключены"
}

// GetTestModeStatus возвращает статус тестового режима в текстовом формате
func (tb *TelegramBot) GetTestModeStatus() string {
	if tb.IsTestMode() {
		return "✅ Включен"
	}
	return "❌ Выключен"
}

// =============================================
// Статические методы для клавиатур (доступны без экземпляра)
// =============================================

// CreateWelcomeKeyboard создает приветственную клавиатуру
func CreateWelcomeKeyboard() *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: ButtonTexts.Status, CallbackData: CallbackStats},
				{Text: ButtonTexts.Settings, CallbackData: CallbackSettings},
			},
			{
				{Text: ButtonTexts.Help, CallbackData: "help"},
				{Text: ButtonTexts.Chart, CallbackData: "chart"},
			},
		},
	}
}

// CreateSettingsKeyboard создает клавиатуру настроек (статическая версия)
func CreateSettingsKeyboard() *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "🔔 Включить уведомления", CallbackData: CallbackSettingsNotifyToggle},
				{Text: "⚙️ Изменить пороги", CallbackData: "change_thresholds"},
			},
			{
				{Text: "📊 Изменить период", CallbackData: CallbackSettingsChangePeriod},
				{Text: "🧪 Тестовый режим", CallbackData: "toggle_test_mode"},
			},
			{
				{Text: ButtonTexts.Back, CallbackData: CallbackSettingsBack},
			},
		},
	}
}

// CreateTestKeyboard создает тестовую клавиатуру
func CreateTestKeyboard() *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "✅ Тест", CallbackData: "test_ok"},
				{Text: "❌ Отмена", CallbackData: "test_cancel"},
			},
			{
				{Text: ButtonTexts.Status, CallbackData: CallbackStats},
				{Text: ButtonTexts.Settings, CallbackData: CallbackSettings},
			},
		},
	}
}
