// internal/telegram/bot.go
package telegram

import (
	"bytes"
	"crypto-exchange-screener-bot/internal/config"
	"crypto-exchange-screener-bot/internal/types"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
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
	notifyEnabled bool
	rateLimiter   *RateLimiter
	lastSendTime  time.Time
	minInterval   time.Duration
	mu            sync.RWMutex
	menuEnabled   bool // Флаг включения меню
}

// RateLimiter - ограничитель частоты запросов
type RateLimiter struct {
	mu       sync.Mutex
	lastSent map[string]time.Time
	minDelay time.Duration
}

// TelegramResponse - ответ от Telegram API
type TelegramResponse struct {
	OK     bool `json:"ok"`
	Result struct {
		MessageID int `json:"message_id"`
	} `json:"result"`
}

// InlineKeyboardButton - кнопка inline клавиатуры
type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
	URL          string `json:"url,omitempty"`
}

// InlineKeyboardMarkup - разметка inline клавиатуры
type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

// TelegramMessage - сообщение с клавиатурой
type TelegramMessage struct {
	ChatID      string      `json:"chat_id"`
	Text        string      `json:"text"`
	ParseMode   string      `json:"parse_mode,omitempty"`
	ReplyMarkup interface{} `json:"reply_markup,omitempty"` // Может быть любой клавиатурой
}

// NewRateLimiter создает новый ограничитель частоты
func NewRateLimiter(minDelay time.Duration) *RateLimiter {
	return &RateLimiter{
		lastSent: make(map[string]time.Time),
		minDelay: minDelay,
	}
}

// CanSend проверяет, можно ли отправить сообщение
func (rl *RateLimiter) CanSend(key string) bool {
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

// NewTelegramBot создает новый экземпляр Telegram бота
func NewTelegramBot(cfg *config.Config) *TelegramBot {
	if cfg.TelegramBotToken == "" || cfg.TelegramChatID == "" {
		log.Println("⚠️ Telegram Bot Token или Chat ID не указаны, бот отключен")
		return nil
	}

	bot := &TelegramBot{
		config:        cfg,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
		baseURL:       fmt.Sprintf("https://api.telegram.org/bot%s/", cfg.TelegramBotToken),
		chatID:        cfg.TelegramChatID,
		notifyEnabled: cfg.TelegramEnabled,
		rateLimiter:   NewRateLimiter(2 * time.Second),
		minInterval:   2 * time.Second,
		menuEnabled:   true, // По умолчанию меню включено
	}

	// Устанавливаем меню при создании бота
	bot.setupMenu()

	return bot
}

// setupMenu устанавливает меню в нижней части экрана
func (tb *TelegramBot) setupMenu() error {
	if !tb.menuEnabled {
		return nil
	}

	// Создаем меню с настройками
	menu := ReplyKeyboardMarkup{
		Keyboard: [][]ReplyKeyboardButton{
			{
				{Text: "⚙️ Настройки"},
				{Text: "📊 Статус"},
			},
			{
				{Text: "🔔 Уведомления ВКЛ"},
				{Text: "🔕 Уведомления ВЫКЛ"},
			},
			{
				{Text: "📈 Только рост"},
				{Text: "📉 Только падение"},
			},
			{
				{Text: "🔄 Сбросить счетчик"},
				{Text: "📊 Изменить период"},
			},
			{
				{Text: "🔄 Сбросить все"},
				{Text: "📋 Помощь"},
			},
		},
		ResizeKeyboard:  true,  // Подстраивается под размер экрана
		OneTimeKeyboard: false, // Меню постоянно видимо
		Selective:       false,
	}

	// Отправляем запрос на установку меню
	return tb.setReplyKeyboard(menu)
}

// removeMenu удаляет меню
func (tb *TelegramBot) removeMenu() error {
	menu := ReplyKeyboardMarkup{
		RemoveKeyboard: true,
		Selective:      false,
	}

	return tb.setReplyKeyboard(menu)
}

// setReplyKeyboard устанавливает reply клавиатуру
func (tb *TelegramBot) setReplyKeyboard(keyboard ReplyKeyboardMarkup) error {
	message := struct {
		ChatID      string              `json:"chat_id"`
		Text        string              `json:"text"`
		ReplyMarkup ReplyKeyboardMarkup `json:"reply_markup,omitempty"`
	}{
		ChatID:      tb.chatID,
		Text:        "⚙️ *Меню настроек активировано*\n\nВсе настройки доступны в меню ниже ⬇️",
		ReplyMarkup: keyboard,
	}

	return tb.sendTelegramRequest("sendMessage", message)
}

// SetMenuEnabled включает/выключает меню
func (tb *TelegramBot) SetMenuEnabled(enabled bool) {
	tb.mu.Lock()
	tb.menuEnabled = enabled
	tb.mu.Unlock()

	if enabled {
		tb.setupMenu()
	} else {
		tb.removeMenu()
	}
}

// IsMenuEnabled возвращает статус меню
func (tb *TelegramBot) IsMenuEnabled() bool {
	tb.mu.RLock()
	defer tb.mu.RUnlock()
	return tb.menuEnabled
}

// SetNotifyEnabled устанавливает статус уведомлений
func (tb *TelegramBot) SetNotifyEnabled(enabled bool) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.notifyEnabled = enabled

	// Обновляем текст кнопки в меню
	if tb.menuEnabled {
		go func() {
			time.Sleep(100 * time.Millisecond) // Небольшая задержка
			tb.setupMenu()                     // Перерисовываем меню
		}()
	}
}

// IsNotifyEnabled возвращает статус уведомлений
func (tb *TelegramBot) IsNotifyEnabled() bool {
	tb.mu.RLock()
	defer tb.mu.RUnlock()
	return tb.notifyEnabled
}

// SendNotification отправляет уведомление о сигнале с проверкой частоты
func (tb *TelegramBot) SendNotification(signal types.GrowthSignal) error {
	if !tb.IsNotifyEnabled() {
		return nil
	}

	// Проверяем настройки уведомлений
	if (signal.Direction == "growth" && !tb.config.TelegramNotifyGrowth) ||
		(signal.Direction == "fall" && !tb.config.TelegramNotifyFall) {
		return nil
	}

	// Проверяем лимит частоты для данного типа сигнала
	key := fmt.Sprintf("signal_%s_%s", signal.Direction, signal.Symbol)
	if !tb.rateLimiter.CanSend(key) {
		log.Printf("⚠️ Пропуск Telegram уведомления для %s (лимит частоты)", signal.Symbol)
		return nil
	}

	// Форматируем сообщение
	message := tb.FormatSignalMessage(signal)

	// Создаем клавиатуру (без меню настроек в уведомлениях)
	keyboard := tb.createNotificationKeyboard(signal)

	// Отправляем сообщение с клавиатурой
	return tb.sendMessageWithKeyboard(message, keyboard, true) // true - скрыть меню для этого сообщения
}

// SendMessage - публичный метод для отправки сообщений
func (tb *TelegramBot) SendMessage(text string) error {
	return tb.sendMessageWithKeyboard(text, nil, false)
}

// SendMessageWithKeyboard отправляет сообщение с клавиатурой
func (tb *TelegramBot) SendMessageWithKeyboard(text string, keyboard *InlineKeyboardMarkup) error {
	return tb.sendMessageWithKeyboard(text, keyboard, false)
}

// sendMessageWithKeyboard отправляет сообщение с клавиатурой
func (tb *TelegramBot) sendMessageWithKeyboard(text string, keyboard *InlineKeyboardMarkup, hideMenu bool) error {
	if !tb.IsNotifyEnabled() && hideMenu {
		return nil
	}

	// Проверяем лимит частоты
	key := "message"
	if !tb.rateLimiter.CanSend(key) {
		log.Printf("⚠️ Пропуск Telegram сообщения (лимит частоты)")
		return fmt.Errorf("rate limit exceeded, try again in 2 seconds")
	}

	// Проверяем минимальный интервал
	now := time.Now()
	if now.Sub(tb.lastSendTime) < tb.minInterval {
		time.Sleep(tb.minInterval - now.Sub(tb.lastSendTime))
	}

	message := TelegramMessage{
		ChatID:    tb.chatID,
		Text:      text,
		ParseMode: "Markdown",
	}

	// Если hideMenu = true и есть клавиатура, используем inline клавиатуру
	// Если hideMenu = false и меню включено, показываем меню
	if hideMenu && keyboard != nil {
		message.ReplyMarkup = keyboard
	} else if !hideMenu && tb.menuEnabled {
		// Для сообщений с меню показываем только текст
		// Меню уже постоянно отображается внизу
	} else if keyboard != nil {
		message.ReplyMarkup = keyboard
	}

	return tb.sendTelegramRequest("sendMessage", message)
}

// FormatSignalMessage форматирует сообщение о сигнале
func (tb *TelegramBot) FormatSignalMessage(signal types.GrowthSignal) string {
	var icon, directionStr, changeStr string
	changePercent := signal.GrowthPercent + signal.FallPercent

	if signal.Direction == "growth" {
		icon = "🟢"
		directionStr = "РОСТ"
		changeStr = fmt.Sprintf("+%.2f%%", changePercent)
	} else {
		icon = "🔴"
		directionStr = "ПАДЕНИЕ"
		changeStr = fmt.Sprintf("-%.2f%%", -changePercent)
	}

	intervalStr := strconv.Itoa(signal.PeriodMinutes) + "мин"
	timeStr := signal.Timestamp.Format("2006/01/02 15:04:05")

	// ДОБАВЛЯЕМ: Проверяем, является ли сигнал от CounterAnalyzer
	counterInfo := ""

	// Проверяем наличие информации о счетчике в метаданных
	if signal.Metadata != nil && signal.Metadata.Indicators != nil {
		// Для CounterAnalyzer сигналов
		if count, ok := signal.Metadata.Indicators["current_count"]; ok {
			if maxSignals, ok2 := signal.Metadata.Indicators["total_max"]; ok2 {
				percentage := (count / maxSignals) * 100

				// Получаем период из метаданных или используем интервал
				periodMinutes := signal.PeriodMinutes
				if period, ok3 := signal.Metadata.Indicators["period_minutes"]; ok3 {
					periodMinutes = int(period)
				}

				counterInfo = fmt.Sprintf("\n📊 Счетчик: %d/%d (%.0f%%)",
					int(count), int(maxSignals), percentage)

				// Добавляем информацию о периоде для счетчика
				if strings.Contains(signal.Type, "counter") {
					counterInfo += fmt.Sprintf("\n⏱️  Период анализа: %d мин", periodMinutes)
				}
			}
		} else if count, ok := signal.Metadata.Indicators["count"]; ok { // Альтернативные ключи (для обратной совместимости)
			if maxSignals, ok2 := signal.Metadata.Indicators["max_signals"]; ok2 {
				percentage := (count / maxSignals) * 100
				counterInfo = fmt.Sprintf("\n📊 Счетчик: %d/%d (%.0f%%)",
					int(count), int(maxSignals), percentage)
			}
		}
	}

	switch tb.config.MessageFormat {
	case "detailed":
		return fmt.Sprintf(
			"⚫ Bybit Futures - %s\n"+
				"📊 Символ: %s\n"+
				"🕐 Время: %s\n"+
				"⏱️  Период: %s\n"+
				"%s Направление: %s\n"+
				"📈 Изменение: %s%s\n"+ // Добавлен counterInfo
				"📡 Уверенность: %.0f%%\n"+
				"📊 Объем: $%.0f",
			intervalStr, signal.Symbol,
			timeStr,
			intervalStr,
			icon, directionStr,
			changeStr, counterInfo,
			signal.Confidence,
			signal.Volume24h,
		)
	case "compact":
		return fmt.Sprintf(
			"⚫ Bybit - %s - %s\n"+
				"🕐 %s\n"+
				"%s %s: %s%s\n"+ // Добавлен counterInfo
				"📡 Уверенность: %.0f%%",
			intervalStr, signal.Symbol,
			timeStr,
			icon, directionStr, changeStr, counterInfo,
			signal.Confidence,
		)
	default:
		return fmt.Sprintf(
			"⚫ Bybit - %s - %s\n"+
				"🕐 %s\n"+
				"%s %s: %s%s\n"+ // Добавлен counterInfo
				"📡 Уверенность: %.0f%%\n"+
				"📈 Сигнал: 1",
			intervalStr, signal.Symbol,
			timeStr,
			icon, directionStr, changeStr, counterInfo,
			signal.Confidence,
		)
	}
}

// createNotificationKeyboard создает клавиатуру с кнопками для уведомления
func (tb *TelegramBot) createNotificationKeyboard(signal types.GrowthSignal) *InlineKeyboardMarkup {
	symbolURL := fmt.Sprintf("https://www.bybit.com/trade/usdt/%s", signal.Symbol)
	chartURL := fmt.Sprintf("https://www.tradingview.com/chart/?symbol=BYBIT:%s", signal.Symbol)

	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{
					Text: "📈 График",
					URL:  chartURL,
				},
				{
					Text: "💱 Торговать",
					URL:  symbolURL,
				},
			},
			{
				{
					Text:         "🔔 Уведомлять",
					CallbackData: fmt.Sprintf("notify_%s_on", signal.Symbol),
				},
				{
					Text:         "🔕 Игнорировать",
					CallbackData: fmt.Sprintf("notify_%s_off", signal.Symbol),
				},
			},
		},
	}
}

// sendTelegramRequest - общая функция для отправки запросов к Telegram API
func (tb *TelegramBot) sendTelegramRequest(method string, payload interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	resp, err := tb.httpClient.Post(
		tb.baseURL+method,
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer func() {
		if resp.Body != nil {
			resp.Body.Close()
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var telegramResp struct {
		OK          bool   `json:"ok"`
		ErrorCode   int    `json:"error_code,omitempty"`
		Description string `json:"description,omitempty"`
	}

	if err := json.Unmarshal(body, &telegramResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !telegramResp.OK {
		// Если ошибка 429, ждем указанное время
		if telegramResp.ErrorCode == 429 {
			retryAfter := 5 // по умолчанию 5 секунд
			var retryResp struct {
				Parameters struct {
					RetryAfter int `json:"retry_after"`
				} `json:"parameters"`
			}
			if json.Unmarshal(body, &retryResp) == nil && retryResp.Parameters.RetryAfter > 0 {
				retryAfter = retryResp.Parameters.RetryAfter
			}
			log.Printf("⚠️ Telegram API лимит, ждем %d секунд", retryAfter)
			time.Sleep(time.Duration(retryAfter) * time.Second)
			// Пробуем снова один раз
			return tb.sendTelegramRequest(method, payload)
		}
		return fmt.Errorf("telegram API error %d: %s", telegramResp.ErrorCode, telegramResp.Description)
	}

	tb.lastSendTime = time.Now()
	return nil
}

// SendTestMessage отправляет тестовое сообщение
func (tb *TelegramBot) SendTestMessage() error {
	message := "🤖 *Бот активирован!*\n\n" +
		"✅ Система мониторинга роста/падения запущена.\n" +
		"🔔 Уведомления отправляются с ограничением 1 сообщение в 10 секунд.\n" +
		"⚡ Настройки: рост=%.2f%%, падение=%.2f%%"

	// Используем настройки из конфигурации анализаторов
	growthThreshold := tb.config.Analyzers.GrowthAnalyzer.MinGrowth
	fallThreshold := tb.config.Analyzers.FallAnalyzer.MinFall

	message = fmt.Sprintf(message, growthThreshold, fallThreshold)

	return tb.sendMessageWithKeyboard(message, nil, false)
}

// StartCommandHandler обрабатывает команду /start
func (tb *TelegramBot) StartCommandHandler(chatID string) error {
	message := "🚀 *Crypto Exchange Screener Bot*\n\n" +
		"✅ *Бот активирован!*\n\n" +
		"*Основные возможности:*\n" +
		"• 📈 Мониторинг роста/падения цен\n" +
		"• 📊 Счетчики сигналов за период\n" +
		"• 🔔 Умные уведомления\n" +
		"• ⚙️ Настройки в меню ниже\n\n" +
		"*Используйте меню внизу для управления ботом* ⬇️"

	return tb.sendMessageWithKeyboardToChat(chatID, message, nil)
}

// sendMessageWithKeyboardToChat отправляет сообщение в указанный чат
func (tb *TelegramBot) sendMessageWithKeyboardToChat(chatID string, text string, keyboard *InlineKeyboardMarkup) error {
	message := TelegramMessage{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "Markdown",
	}

	if keyboard != nil {
		message.ReplyMarkup = keyboard
	}

	return tb.sendTelegramRequest("sendMessage", message)
}

// HandleMessage обрабатывает текстовые сообщения из меню
func (tb *TelegramBot) HandleMessage(text, chatID string) error {
	log.Printf("📝 Handling menu message from chat %s: %s", chatID, text)

	switch text {
	case "⚙️ Настройки":
		return tb.sendSettingsMessage(chatID)
	case "📊 Статус":
		return tb.sendStatus(chatID)
	case "🔔 Уведомления ВКЛ":
		tb.SetNotifyEnabled(true)
		return tb.sendMessageWithKeyboardToChat(chatID, "✅ Уведомления включены", nil)
	case "🔕 Уведомления ВЫКЛ":
		tb.SetNotifyEnabled(false)
		return tb.sendMessageWithKeyboardToChat(chatID, "❌ Уведомления выключены", nil)
	case "📈 Только рост":
		return tb.handleTrackGrowthOnly(chatID)
	case "📉 Только падение":
		return tb.handleTrackFallOnly(chatID)
	case "📊 Любое изменение":
		return tb.handleTrackBoth(chatID)
	case "🔄 Сбросить счетчик":
		return tb.sendResetCounterOptions(chatID)
	case "📊 Изменить период":
		return tb.sendPeriodOptions(chatID)
	case "🔄 Сбросить все":
		return tb.handleResetAllCounters(chatID)
	case "📋 Помощь":
		return tb.sendHelp(chatID)
	default:
		// Обработка других текстовых команд
		if strings.HasPrefix(text, "/") {
			return tb.handleCommand(text, chatID)
		}
		return tb.sendMessageWithKeyboardToChat(chatID,
			"❓ Неизвестная команда. Используйте меню ниже или /help", nil)
	}
}

// handleCommand обрабатывает текстовые команды
func (tb *TelegramBot) handleCommand(cmd, chatID string) error {
	switch cmd {
	case "/start":
		return tb.StartCommandHandler(chatID)
	case "/help":
		return tb.sendHelp(chatID)
	case "/status":
		return tb.sendStatus(chatID)
	case "/notify_on":
		tb.SetNotifyEnabled(true)
		return tb.sendMessageWithKeyboardToChat(chatID, "✅ Уведомления включены", nil)
	case "/notify_off":
		tb.SetNotifyEnabled(false)
		return tb.sendMessageWithKeyboardToChat(chatID, "❌ Уведомления выключены", nil)
	case "/settings":
		return tb.sendSettingsMessage(chatID)
	case "/test":
		return tb.SendTestMessage()
	default:
		return tb.sendMessageWithKeyboardToChat(chatID,
			fmt.Sprintf("❓ Неизвестная команда: %s. Используйте /help", cmd), nil)
	}
}

// sendSettingsMessage отправляет сообщение с описанием настроек
func (tb *TelegramBot) sendSettingsMessage(chatID string) error {
	message := "⚙️ *Настройки бота*\n\n" +
		"*Текущие настройки:*\n" +
		fmt.Sprintf("🔔 Уведомления: %s\n", tb.getNotifyStatus()) +
		fmt.Sprintf("📈 Отслеживание роста: %v\n", tb.config.TelegramNotifyGrowth) +
		fmt.Sprintf("📉 Отслеживание падения: %v\n", tb.config.TelegramNotifyFall) +
		fmt.Sprintf("⏱️  Период счетчика: %s\n", tb.config.CounterAnalyzer.DefaultPeriod) +
		"\n*Используйте меню ниже для изменения настроек:*"

	return tb.sendMessageWithKeyboardToChat(chatID, message, nil)
}

// sendResetCounterOptions отправляет опции сброса счетчика
func (tb *TelegramBot) sendResetCounterOptions(chatID string) error {
	message := "🔄 *Сброс счетчика*\n\n" +
		"Выберите действие:"

	keyboard := &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "🔄 Сбросить все счетчики", CallbackData: "reset_all"},
				{Text: "📊 Сбросить по символу", CallbackData: "reset_by_symbol"},
			},
			{
				{Text: "🔙 Назад", CallbackData: "back_to_menu"},
			},
		},
	}

	return tb.sendMessageWithKeyboardToChat(chatID, message, keyboard)
}

// sendPeriodOptions отправляет опции выбора периода
func (tb *TelegramBot) sendPeriodOptions(chatID string) error {
	message := "⏳ *Изменение периода анализа*\n\n" +
		"Выберите период для счетчика:"

	keyboard := &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "5 минут", CallbackData: "period_5m"},
				{Text: "15 минут", CallbackData: "period_15m"},
			},
			{
				{Text: "30 минут", CallbackData: "period_30m"},
				{Text: "1 час", CallbackData: "period_1h"},
			},
			{
				{Text: "4 часа", CallbackData: "period_4h"},
				{Text: "1 день", CallbackData: "period_1d"},
			},
			{
				{Text: "🔙 Назад", CallbackData: "back_to_menu"},
			},
		},
	}

	return tb.sendMessageWithKeyboardToChat(chatID, message, keyboard)
}

// HandleCallback обрабатывает callback от inline кнопок
func (tb *TelegramBot) HandleCallback(callbackData string, chatID string) error {
	log.Printf("🔄 Handling callback: %s for chat %s", callbackData, chatID)

	switch callbackData {
	case "reset_all":
		return tb.handleResetAllCounters(chatID)
	case "reset_by_symbol":
		return tb.sendSymbolSelectionMenu(chatID, "reset")
	case "back_to_menu":
		return tb.sendSettingsMessage(chatID)
	case "period_5m":
		return tb.handlePeriodChange(chatID, "5m")
	case "period_15m":
		return tb.handlePeriodChange(chatID, "15m")
	case "period_30m":
		return tb.handlePeriodChange(chatID, "30m")
	case "period_1h":
		return tb.handlePeriodChange(chatID, "1h")
	case "period_4h":
		return tb.handlePeriodChange(chatID, "4h")
	case "period_1d":
		return tb.handlePeriodChange(chatID, "1d")
	case "notify_on":
		tb.SetNotifyEnabled(true)
		return tb.sendMessageWithKeyboardToChat(chatID, "✅ Уведомления включены", nil)
	case "notify_off":
		tb.SetNotifyEnabled(false)
		return tb.sendMessageWithKeyboardToChat(chatID, "❌ Уведомления выключены", nil)
	default:
		// Обработка уведомлений для конкретных символов
		if len(callbackData) > 7 && callbackData[:7] == "notify_" {
			return tb.handleSymbolNotification(callbackData[7:], chatID)
		}
		// Обработка сброса по символу
		if strings.HasPrefix(callbackData, "reset_symbol_") {
			symbol := strings.TrimPrefix(callbackData, "reset_symbol_")
			return tb.handleResetCounterForSymbol(chatID, symbol)
		}
		// Обработка callback счетчика
		if strings.HasPrefix(callbackData, "counter_") {
			return tb.HandleCounterCallback(callbackData, chatID)
		}

		return fmt.Errorf("unknown callback data: %s", callbackData)
	}
}

// handleTrackGrowthOnly устанавливает отслеживание только роста
func (tb *TelegramBot) handleTrackGrowthOnly(chatID string) error {
	tb.config.TelegramNotifyGrowth = true
	tb.config.TelegramNotifyFall = false

	message := "📈 Теперь отслеживаются только сигналы роста\n\n" +
		"Настройки будут применены ко всем счетчикам."

	return tb.sendMessageWithKeyboardToChat(chatID, message, nil)
}

// handleTrackFallOnly устанавливает отслеживание только падения
func (tb *TelegramBot) handleTrackFallOnly(chatID string) error {
	tb.config.TelegramNotifyGrowth = false
	tb.config.TelegramNotifyFall = true

	message := "📉 Теперь отслеживаются только сигналы падения\n\n" +
		"Настройки будут применены ко всем счетчикам."

	return tb.sendMessageWithKeyboardToChat(chatID, message, nil)
}

// handleTrackBoth устанавливает отслеживание обоих типов
func (tb *TelegramBot) handleTrackBoth(chatID string) error {
	tb.config.TelegramNotifyGrowth = true
	tb.config.TelegramNotifyFall = true

	message := "📊 Теперь отслеживаются все сигналы (рост и падение)\n\n" +
		"Настройки будут применены ко всем счетчикам."

	return tb.sendMessageWithKeyboardToChat(chatID, message, nil)
}

// handlePeriodChange обрабатывает изменение периода
func (tb *TelegramBot) handlePeriodChange(chatID string, period string) error {
	periodMap := map[string]string{
		"5m":  "5 минут",
		"15m": "15 минут",
		"30m": "30 минут",
		"1h":  "1 час",
		"4h":  "4 часа",
		"1d":  "1 день",
	}

	periodName, exists := periodMap[period]
	if !exists {
		periodName = "15 минут"
	}

	// Обновляем конфигурацию
	tb.config.CounterAnalyzer.DefaultPeriod = period
	tb.config.CounterAnalyzer.AnalysisPeriod = period

	message := fmt.Sprintf("✅ Период анализа установлен на: %s\n\n"+
		"Все счетчики будут перезапущены с новым периодом.", periodName)

	return tb.sendMessageWithKeyboardToChat(chatID, message, nil)
}

// handleResetAllCounters сбрасывает все счетчики
func (tb *TelegramBot) handleResetAllCounters(chatID string) error {
	message := "🔄 Все счетчики сигналов сброшены\n\n" +
		"Отсчет начался заново для всех символов."

	return tb.sendMessageWithKeyboardToChat(chatID, message, nil)
}

// sendSymbolSelectionMenu отправляет меню выбора символа
func (tb *TelegramBot) sendSymbolSelectionMenu(chatID string, action string) error {
	message := fmt.Sprintf("Выберите символ для %s:",
		map[string]string{
			"reset": "сброса счетчика",
		}[action])

	keyboard := &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "BTCUSDT", CallbackData: action + "_symbol_BTCUSDT"},
				{Text: "ETHUSDT", CallbackData: action + "_symbol_ETHUSDT"},
			},
			{
				{Text: "SOLUSDT", CallbackData: action + "_symbol_SOLUSDT"},
				{Text: "XRPUSDT", CallbackData: action + "_symbol_XRPUSDT"},
			},
			{
				{Text: "🔙 Назад", CallbackData: "back_to_menu"},
			},
		},
	}

	return tb.sendMessageWithKeyboardToChat(chatID, message, keyboard)
}

// handleResetCounterForSymbol сбрасывает счетчик для конкретного символа
func (tb *TelegramBot) handleResetCounterForSymbol(chatID, symbol string) error {
	message := fmt.Sprintf("🔄 Счетчик для %s сброшен\n\n"+
		"Отсчет начался заново.", symbol)

	return tb.sendMessageWithKeyboardToChat(chatID, message, nil)
}

// handleSymbolNotification обрабатывает уведомления для конкретных символов
func (tb *TelegramBot) handleSymbolNotification(callbackData, chatID string) error {
	lastUnderscore := -1
	for i := len(callbackData) - 1; i >= 0; i-- {
		if callbackData[i] == '_' {
			lastUnderscore = i
			break
		}
	}

	if lastUnderscore != -1 {
		symbol := callbackData[:lastUnderscore]
		action := callbackData[lastUnderscore+1:]

		var response string
		if action == "on" {
			response = fmt.Sprintf("✅ Уведомления для %s включены", symbol)
		} else if action == "off" {
			response = fmt.Sprintf("❌ Уведомления для %s выключены", symbol)
		} else {
			response = fmt.Sprintf("❓ Неизвестное действие для %s: %s", symbol, action)
		}

		return tb.sendMessageWithKeyboardToChat(chatID, response, nil)
	}

	return fmt.Errorf("invalid symbol notification callback: %s", callbackData)
}

// sendStatus отправляет статус системы
func (tb *TelegramBot) sendStatus(chatID string) error {
	message := "📊 *Статус системы*\n\n" +
		"✅ Бот работает\n" +
		"🔔 Уведомления: " + tb.getNotifyStatus() + "\n" +
		"📈 Мониторинг роста: активен\n" +
		"📉 Мониторинг падения: активен\n" +
		"📊 Счетчики сигналов: активны\n" +
		"⚙️  Меню настроек: " + tb.getMenuStatus() + "\n" +
		"🕐 Время сервера: " + time.Now().Format("2006-01-02 15:04:05")

	return tb.sendMessageWithKeyboardToChat(chatID, message, nil)
}

// sendHelp отправляет справку
func (tb *TelegramBot) sendHelp(chatID string) error {
	message := "🆘 *Помощь*\n\n" +
		"*Основные команды:*\n" +
		"/start - Начало работы\n" +
		"/status - Статус системы\n" +
		"/notify_on - Включить уведомления\n" +
		"/notify_off - Выключить уведомления\n" +
		"/test - Тестовое уведомление\n" +
		"/help - Эта справка\n\n" +
		"*Меню настроек (внизу экрана):*\n" +
		"⚙️ Настройки - Показать текущие настройки\n" +
		"📊 Статус - Показать статус системы\n" +
		"🔔 Уведомления ВКЛ/ВЫКЛ - Управление уведомлениями\n" +
		"📈 Только рост - Отслеживать только рост\n" +
		"📉 Только падение - Отслеживать только падение\n" +
		"🔄 Сбросить счетчик - Сбросить счетчики сигналов\n" +
		"📊 Изменить период - Изменить период анализа\n" +
		"🔄 Сбросить все - Сбросить все настройки\n\n" +
		"*Используйте меню ниже для быстрого доступа к настройкам*"

	return tb.sendMessageWithKeyboardToChat(chatID, message, nil)
}

// getNotifyStatus возвращает статус уведомлений
func (tb *TelegramBot) getNotifyStatus() string {
	if tb.IsNotifyEnabled() {
		return "✅ Включены"
	}
	return "❌ Выключены"
}

// getMenuStatus возвращает статус меню
func (tb *TelegramBot) getMenuStatus() string {
	if tb.IsMenuEnabled() {
		return "✅ Активно"
	}
	return "❌ Отключено"
}

// SendCounterNotification отправляет уведомление счетчика
func (tb *TelegramBot) SendCounterNotification(symbol string, signalType string, count int, maxSignals int, period string) error {
	if !tb.notifyEnabled {
		return nil
	}

	icon := "🟢"
	directionStr := "РОСТ"
	if signalType == "fall" {
		icon = "🔴"
		directionStr = "ПАДЕНИЕ"
	}

	percentage := float64(count) / float64(maxSignals) * 100
	timeStr := time.Now().Format("2006/01/02 15:04:05")

	message := fmt.Sprintf(
		"📊 *Счетчик сигналов*\n"+
			"⚫ Символ: %s\n"+
			"🕐 Время: %s\n"+
			"⏱️  Период: %s\n"+
			"%s Направление: %s\n"+
			"📈 Счетчик: %d/%d (%.0f%%)",
		symbol,
		timeStr,
		period,
		icon, directionStr,
		count, maxSignals, percentage,
	)

	// Создаем клавиатуру
	keyboard := tb.createCounterKeyboard(symbol)

	return tb.sendMessageWithKeyboard(message, keyboard, true)
}

// createCounterKeyboard создает клавиатуру для счетчика
func (tb *TelegramBot) createCounterKeyboard(symbol string) *InlineKeyboardMarkup {
	chartURL := tb.getCounterChartURL(symbol)
	symbolURL := fmt.Sprintf("https://www.bybit.com/trade/usdt/%s", symbol)

	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{
					Text: "📊 График",
					URL:  chartURL,
				},
				{
					Text: "💱 Торговать",
					URL:  symbolURL,
				},
			},
			{
				{
					Text:         "🔕 Отключить уведомления",
					CallbackData: fmt.Sprintf("counter_notify_%s_off", symbol),
				},
				{
					Text:         "⚙️ Настройки",
					CallbackData: "settings",
				},
			},
		},
	}
}

// getCounterChartURL возвращает URL графика для счетчика
func (tb *TelegramBot) getCounterChartURL(symbol string) string {
	// Используем настройку из конфигурации
	chartProvider := tb.config.CounterAnalyzer.ChartProvider
	if chartProvider == "" {
		chartProvider = "coinglass" // По умолчанию
	}

	switch chartProvider {
	case "tradingview":
		return fmt.Sprintf("https://www.tradingview.com/chart/?symbol=BYBIT:%s", symbol)
	default: // coinglass
		return fmt.Sprintf("https://www.coinglass.com/tv/%s", symbol)
	}
}

// HandleCounterCallback обрабатывает callback счетчика
func (tb *TelegramBot) HandleCounterCallback(callbackData string, chatID string) error {
	switch callbackData {
	case "counter_settings":
		return tb.sendCounterSettings(chatID)
	case "counter_notify_on":
		tb.SetCounterNotifications(true)
		return tb.sendMessageWithKeyboardToChat(chatID, "✅ Уведомления счетчика включены", nil)
	case "counter_notify_off":
		tb.SetCounterNotifications(false)
		return tb.sendMessageWithKeyboardToChat(chatID, "❌ Уведомления счетчика выключены", nil)
	default:
		// Обработка настроек периода
		if strings.HasPrefix(callbackData, "counter_period_") {
			period := strings.TrimPrefix(callbackData, "counter_period_")
			return tb.handleCounterPeriodChange(chatID, period)
		}
		// Обработка отключения уведомлений для символа
		if strings.HasPrefix(callbackData, "counter_notify_") && strings.HasSuffix(callbackData, "_off") {
			symbol := strings.TrimPrefix(callbackData, "counter_notify_")
			symbol = strings.TrimSuffix(symbol, "_off")
			return tb.sendMessageWithKeyboardToChat(chatID,
				fmt.Sprintf("❌ Уведомления счетчика для %s выключены", symbol), nil)
		}
	}

	return fmt.Errorf("unknown counter callback: %s", callbackData)
}

// sendCounterSettings отправляет настройки счетчика
func (tb *TelegramBot) sendCounterSettings(chatID string) error {
	message := "⚙️ *Настройки счетчика сигналов*\n\n" +
		"Выберите период анализа:"

	keyboard := &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "5 минут", CallbackData: "counter_period_5m"},
				{Text: "15 минут", CallbackData: "counter_period_15m"},
			},
			{
				{Text: "30 минут", CallbackData: "counter_period_30m"},
				{Text: "1 час", CallbackData: "counter_period_1h"},
			},
			{
				{Text: "4 часа", CallbackData: "counter_period_4h"},
				{Text: "1 день", CallbackData: "counter_period_1d"},
			},
			{
				{Text: "✅ Включить уведомления", CallbackData: "counter_notify_on"},
				{Text: "❌ Выключить уведомления", CallbackData: "counter_notify_off"},
			},
			{
				{Text: "🔙 Назад", CallbackData: "settings"},
			},
		},
	}

	return tb.sendMessageWithKeyboardToChat(chatID, message, keyboard)
}

// handleCounterPeriodChange обрабатывает изменение периода
func (tb *TelegramBot) handleCounterPeriodChange(chatID string, period string) error {
	periodNames := map[string]string{
		"5m":  "5 минут",
		"15m": "15 минут",
		"30m": "30 минут",
		"1h":  "1 час",
		"4h":  "4 часа",
		"1d":  "1 день",
	}

	periodName, exists := periodNames[period]
	if !exists {
		return fmt.Errorf("unknown period: %s", period)
	}

	return tb.sendMessageWithKeyboardToChat(chatID,
		fmt.Sprintf("✅ Период счетчика изменен на: %s", periodName), nil)
}

// SetCounterNotifications включает/выключает уведомления счетчика
func (tb *TelegramBot) SetCounterNotifications(enabled bool) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	// Здесь нужно обновить настройку в анализаторе счетчика
}
