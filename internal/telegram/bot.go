// internal/telegram/bot.go (исправленная версия)
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
	"sync"
	"time"
)

// TelegramBot - бот для отправки уведомлений в Telegram
type TelegramBot struct {
	config        *config.Config
	httpClient    *http.Client
	baseURL       string
	chatID        int64
	notifyEnabled bool
	rateLimiter   *RateLimiter
	lastSendTime  time.Time
	minInterval   time.Duration
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
	ChatID      int64                 `json:"chat_id"`
	Text        string                `json:"text"`
	ParseMode   string                `json:"parse_mode,omitempty"`
	ReplyMarkup *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
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

// CanSend проверяет, можно ли отправить сообщение
func NewTelegramBot(cfg *config.Config) *TelegramBot {
	if cfg.TelegramAPIKey == "" || cfg.TelegramChatID == 0 {
		log.Println("⚠️ Telegram API ключ или Chat ID не указаны, бот отключен")
		return nil
	}

	return &TelegramBot{
		config:        cfg,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
		baseURL:       fmt.Sprintf("https://api.telegram.org/bot%s/", cfg.TelegramAPIKey),
		chatID:        cfg.TelegramChatID,
		notifyEnabled: cfg.TelegramEnabled,
		rateLimiter:   NewRateLimiter(10 * time.Second), // УВЕЛИЧЕН ДО 10 СЕКУНД
		minInterval:   10 * time.Second,                 // УВЕЛИЧЕН ДО 10 СЕКУНД
	}
}

// SendNotification отправляет уведомление о сигнале с проверкой частоты
func (tb *TelegramBot) SendNotification(signal types.GrowthSignal) error {
	if !tb.notifyEnabled {
		return nil
	}

	// Проверяем настройки уведомлений
	if (signal.Direction == "growth" && !tb.config.TelegramNotifyOn.Growth) ||
		(signal.Direction == "fall" && !tb.config.TelegramNotifyOn.Fall) {
		return nil
	}

	// Проверяем лимит частоты для данного типа сигнала
	key := fmt.Sprintf("signal_%s", signal.Direction)
	if !tb.rateLimiter.CanSend(key) {
		log.Printf("⚠️ Пропуск Telegram уведомления для %s (лимит частоты)", signal.Symbol)
		return nil
	}

	// Форматируем сообщение в новом формате
	message := tb.FormatSignalMessage(signal)

	// Создаем клавиатуру
	keyboard := tb.createNotificationKeyboard(signal)

	// Отправляем сообщение с клавиатурой
	return tb.sendMessageWithKeyboard(message, keyboard)
}

// SendMessage - публичный метод для отправки сообщений
func (tb *TelegramBot) SendMessage(text string) error {
	return tb.sendMessageWithKeyboard(text, nil)
}

// SendMessageWithKeyboard отправляет сообщение с клавиатурой
func (tb *TelegramBot) SendMessageWithKeyboard(text string, keyboard *InlineKeyboardMarkup) error {
	return tb.sendMessageWithKeyboard(text, keyboard)
}

// sendMessage отправляет простое сообщение без клавиатуры
func (tb *TelegramBot) sendMessage(text string) error {
	if !tb.notifyEnabled {
		return nil
	}

	// Проверяем лимит частоты
	key := "message"
	if !tb.rateLimiter.CanSend(key) {
		return fmt.Errorf("rate limit exceeded, try again in 2 seconds")
	}

	// Проверяем минимальный интервал
	now := time.Now()
	if now.Sub(tb.lastSendTime) < tb.minInterval {
		time.Sleep(tb.minInterval - now.Sub(tb.lastSendTime))
	}

	message := struct {
		ChatID    int64  `json:"chat_id"`
		Text      string `json:"text"`
		ParseMode string `json:"parse_mode,omitempty"`
	}{
		ChatID:    tb.chatID,
		Text:      text,
		ParseMode: "Markdown",
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	resp, err := tb.httpClient.Post(
		tb.baseURL+"sendMessage",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

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
			return tb.sendMessage(text)
		}
		return fmt.Errorf("telegram API error %d: %s", telegramResp.ErrorCode, telegramResp.Description)
	}

	tb.lastSendTime = time.Now()
	return nil
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

	// НОВЫЙ ФОРМАТ СООБЩЕНИЯ
	return fmt.Sprintf(
		"⚫ Bybit - %s - %s\n"+
			"🕐 %s\n"+
			"%s %s: %s\n"+
			"📡 Уверенность: %.0f%%\n"+
			"📈 Сигнал: 1",
		intervalStr, signal.Symbol,
		timeStr,
		icon, directionStr, changeStr,
		signal.Confidence,
	)
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

// sendMessageWithKeyboard отправляет сообщение с клавиатурой
func (tb *TelegramBot) sendMessageWithKeyboard(text string, keyboard *InlineKeyboardMarkup) error {
	if !tb.notifyEnabled {
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

	if keyboard != nil {
		message.ReplyMarkup = keyboard
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	resp, err := tb.httpClient.Post(
		tb.baseURL+"sendMessage",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

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
			return tb.sendMessageWithKeyboard(text, keyboard)
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
		"🔔 Уведомления отправляются с ограничением 1 сообщение в 2 секунды.\n" +
		"⚡ Настройки: рост=%.2f%%, падение=%.2f%%"

	// Используем настройки из конфигурации анализаторов
	growthThreshold := tb.config.Analyzers.GrowthAnalyzer.MinGrowth
	fallThreshold := tb.config.Analyzers.FallAnalyzer.MinFall

	message = fmt.Sprintf(message, growthThreshold, fallThreshold)

	keyboard := &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "📊 Статус", CallbackData: "status"},
				{Text: "⚙️ Настройки", CallbackData: "settings"},
			},
		},
	}

	return tb.sendMessageWithKeyboard(message, keyboard)
}

// StartCommandHandler обрабатывает команду /start
func (tb *TelegramBot) StartCommandHandler(chatID int64) error {
	message := "🚀 *Crypto Exchange Screener Bot*\n\n" +
		"*Доступные команды:*\n" +
		"/start - Начало работы\n" +
		"/status - Статус системы\n" +
		"/notify_on - Включить уведомления\n" +
		"/notify_off - Выключить уведомления\n" +
		"/test - Тестовое уведомление\n\n" +
		"⚡ Бот мониторит рост/падение фьючерсов на Bybit"

	keyboard := &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "✅ Уведомлять", CallbackData: "notify_on"},
				{Text: "❌ Не уведомлять", CallbackData: "notify_off"},
			},
			{
				{Text: "📊 Статистика", CallbackData: "stats"},
				{Text: "⚙️ Настройки", CallbackData: "config"},
			},
		},
	}

	return tb.sendMessageWithKeyboardToChat(chatID, message, keyboard)
}

// sendMessageWithKeyboardToChat отправляет сообщение в указанный чат
func (tb *TelegramBot) sendMessageWithKeyboardToChat(chatID int64, text string, keyboard *InlineKeyboardMarkup) error {
	message := TelegramMessage{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "Markdown",
	}

	if keyboard != nil {
		message.ReplyMarkup = keyboard
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	resp, err := tb.httpClient.Post(
		tb.baseURL+"sendMessage",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

// HandleCallback обрабатывает callback от кнопок
func (tb *TelegramBot) HandleCallback(callbackData string, chatID int64) error {
	switch callbackData {
	case "notify_on":
		tb.notifyEnabled = true
		return tb.sendMessageWithKeyboardToChat(chatID, "✅ Уведомления включены", nil)
	case "notify_off":
		tb.notifyEnabled = false
		return tb.sendMessageWithKeyboardToChat(chatID, "❌ Уведомления выключены", nil)
	case "status":
		return tb.sendStatus(chatID)
	case "stats":
		return tb.sendStats(chatID)
	case "config":
		return tb.sendConfig(chatID)
	default:
		// Обработка notify_SYMBOL_on/off
		if len(callbackData) > 7 && callbackData[:7] == "notify_" {
			symbol := callbackData[7 : len(callbackData)-3]
			action := callbackData[len(callbackData)-2:]

			var response string
			if action == "on" {
				response = fmt.Sprintf("✅ Уведомления для %s включены", symbol)
			} else {
				response = fmt.Sprintf("❌ Уведомления для %s выключены", symbol)
			}

			return tb.sendMessageWithKeyboardToChat(chatID, response, nil)
		}
	}

	return nil
}

// sendStatus отправляет статус системы
func (tb *TelegramBot) sendStatus(chatID int64) error {
	message := "📊 *Статус системы*\n\n" +
		"✅ Бот работает\n" +
		"🔔 Уведомления: " + tb.getNotifyStatus() + "\n" +
		"📈 Мониторинг роста: активен\n" +
		"🕐 Время сервера: " + time.Now().Format("2006-01-02 15:04:05")

	return tb.sendMessageWithKeyboardToChat(chatID, message, nil)
}

// sendStats отправляет статистику
func (tb *TelegramBot) sendStats(chatID int64) error {
	// Здесь можно добавить реальную статистику
	message := "📈 *Статистика*\n\n" +
		"Сигналов сегодня: 0\n" +
		"Рост: 0\n" +
		"Падение: 0\n" +
		"Топ сигнал: Нет данных"

	return tb.sendMessageWithKeyboardToChat(chatID, message, nil)
}

// sendConfig отправляет текущую конфигурацию
func (tb *TelegramBot) sendConfig(chatID int64) error {
	message := fmt.Sprintf(
		"⚙️ *Конфигурация*\n\n"+
			"Бот: %s\n"+
			"Уведомления: %s\n"+
			"Рост: %v\n"+
			"Падение: %v\n"+
			"Формат: %s",
		tb.config.FuturesCategory,
		tb.getNotifyStatus(),
		tb.config.TelegramNotifyOn.Growth,
		tb.config.TelegramNotifyOn.Fall,
		tb.config.MessageFormat,
	)

	return tb.sendMessageWithKeyboardToChat(chatID, message, nil)
}

// getNotifyStatus возвращает статус уведомлений
func (tb *TelegramBot) getNotifyStatus() string {
	if tb.notifyEnabled {
		return "✅ Включены"
	}
	return "❌ Выключены"
}
