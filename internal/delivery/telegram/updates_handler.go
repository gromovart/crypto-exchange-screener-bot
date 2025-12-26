package telegram

import (
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// UpdatesHandler - обработчик обновлений (поддерживает и webhook, и polling)
type UpdatesHandler struct {
	config        *config.Config
	bot           *TelegramBot
	pollingActive bool
	lastUpdateID  int64
	httpClient    *http.Client
}

// NewUpdatesHandler создает новый обработчик обновлений
func NewUpdatesHandler(cfg *config.Config, bot *TelegramBot) *UpdatesHandler {
	return &UpdatesHandler{
		config:       cfg,
		bot:          bot,
		lastUpdateID: 0,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Start запускает обработчик обновлений
func (uh *UpdatesHandler) Start() error {
	if uh.config.HTTPEnabled && uh.config.HTTPPort > 0 {
		// Запуск в режиме webhook
		return uh.startWebhook()
	} else {
		// Запуск в режиме polling
		return uh.startPolling()
	}
}

// startWebhook запускает webhook сервер
func (uh *UpdatesHandler) startWebhook() error {
	log.Println("🌐 Запуск в режиме Webhook...")

	// Создаем webhook сервер
	webhookServer := NewWebhookServer(uh.config, uh.bot)

	// Настраиваем webhook в Telegram
	if err := uh.setWebhook(); err != nil {
		return fmt.Errorf("failed to set webhook: %w", err)
	}

	// Запускаем сервер
	return webhookServer.Start()
}

// startPolling запускает polling (опрос) обновлений
func (uh *UpdatesHandler) startPolling() error {
	log.Println("🔄 Запуск в режиме Polling (локальная разработка)...")

	// Удаляем webhook если был установлен
	if err := uh.deleteWebhook(); err != nil {
		log.Printf("⚠️ Не удалось удалить webhook: %v", err)
	}

	// Запускаем polling
	uh.pollingActive = true
	go uh.pollUpdates()

	return nil
}

// Stop останавливает обработчик
func (uh *UpdatesHandler) Stop() error {
	uh.pollingActive = false
	return nil
}

// pollUpdates опрашивает Telegram API на наличие обновлений
func (uh *UpdatesHandler) pollUpdates() {
	log.Println("🔄 Начало polling обновлений...")

	// Интервал опроса
	pollInterval := 1 * time.Second

	for uh.pollingActive {
		updates, err := uh.getUpdates()
		if err != nil {
			log.Printf("❌ Ошибка получения обновлений: %v", err)
			time.Sleep(pollInterval)
			continue
		}

		// Обрабатываем полученные обновления
		for _, update := range updates {
			uh.processUpdate(update)
			uh.lastUpdateID = update.UpdateID + 1
		}

		// Ждем перед следующим опросом
		time.Sleep(pollInterval)
	}

	log.Println("🛑 Polling остановлен")
}

// getUpdates получает обновления от Telegram API
func (uh *UpdatesHandler) getUpdates() ([]TelegramUpdate, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates", uh.config.TelegramBotToken)

	// Параметры запроса
	params := map[string]interface{}{
		"offset":  uh.lastUpdateID,
		"timeout": 30,
		"limit":   100,
	}

	// Отправляем запрос
	resp, err := uh.httpClient.Post(url, "application/json", toJSONReader(params))
	if err != nil {
		return nil, fmt.Errorf("failed to get updates: %w", err)
	}
	defer resp.Body.Close()

	// Читаем ответ
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Парсим ответ
	var response struct {
		OK     bool             `json:"ok"`
		Result []TelegramUpdate `json:"result"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !response.OK {
		return nil, fmt.Errorf("telegram API error: %s", string(body))
	}

	return response.Result, nil
}

// processUpdate обрабатывает одно обновление
func (uh *UpdatesHandler) processUpdate(update TelegramUpdate) {
	log.Printf("📨 Получено обновление ID: %d", update.UpdateID)

	// Обработка сообщений
	if update.Message != nil && update.Message.Text != "" {
		chatID := fmt.Sprintf("%d", update.Message.Chat.ID)
		text := strings.TrimSpace(update.Message.Text)

		log.Printf("💬 Сообщение от chat %s: %s", chatID, text)

		if strings.HasPrefix(text, "/") {
			// Обработка команд
			uh.handleCommand(text, chatID)
		} else {
			// Обработка нажатий кнопок меню
			if err := uh.bot.HandleMessage(text, chatID); err != nil {
				log.Printf("❌ Ошибка обработки кнопки меню: %v", err)
			}
		}
	}

	// Обработка callback от inline кнопок
	if update.CallbackQuery != nil && update.CallbackQuery.Data != "" {
		chatID := fmt.Sprintf("%d", update.CallbackQuery.Message.Chat.ID)
		callbackData := update.CallbackQuery.Data

		log.Printf("🔄 Callback от chat %s: %s", chatID, callbackData)

		if err := uh.bot.HandleCallback(callbackData, chatID); err != nil {
			log.Printf("❌ Ошибка обработки callback: %v", err)
		}

		// Отвечаем на callback
		uh.answerCallbackQuery(update.CallbackQuery.ID)
	}
}

// handleCommand обрабатывает команды
func (uh *UpdatesHandler) handleCommand(command, chatID string) {
	log.Printf("⚡ Обработка команды: %s", command)

	switch command {
	case "/start":
		if err := uh.bot.StartCommandHandler(chatID); err != nil {
			log.Printf("❌ Ошибка обработки /start: %v", err)
		}
	case "/status":
		if err := uh.bot.SendMessage("📊 Система работает"); err != nil {
			log.Printf("❌ Ошибка отправки статуса: %v", err)
		}
	case "/menu":
		if err := uh.bot.SendMessage("🔘 Меню активировано"); err != nil {
			log.Printf("❌ Ошибка отправки меню: %v", err)
		}
	case "/test":
		if err := uh.bot.SendTestMessage(); err != nil {
			log.Printf("❌ Ошибка отправки тестового сообщения: %v", err)
		}
	default:
		if err := uh.bot.SendMessage(fmt.Sprintf("❓ Неизвестная команда: %s. Используйте /start", command)); err != nil {
			log.Printf("❌ Ошибка отправки ответа: %v", err)
		}
	}
}

// answerCallbackQuery отвечает на callback запрос
func (uh *UpdatesHandler) answerCallbackQuery(callbackID string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/answerCallbackQuery", uh.config.TelegramBotToken)

	params := map[string]interface{}{
		"callback_query_id": callbackID,
		"text":              "✅",
		"show_alert":        false,
	}

	_, err := uh.httpClient.Post(url, "application/json", toJSONReader(params))
	return err
}

// setWebhook настраивает webhook в Telegram
func (uh *UpdatesHandler) setWebhook() error {
	if !uh.config.HTTPEnabled || uh.config.HTTPPort == 0 {
		return fmt.Errorf("HTTP не включен или порт не указан")
	}

	// URL для webhook (нужен публичный URL)
	webhookURL := fmt.Sprintf("https://your-public-url.com:%d/webhook", uh.config.HTTPPort)

	url := fmt.Sprintf("https://api.telegram.org/bot%s/setWebhook", uh.config.TelegramBotToken)

	params := map[string]interface{}{
		"url":             webhookURL,
		"max_connections": 40,
		"allowed_updates": []string{"message", "callback_query"},
	}

	resp, err := uh.httpClient.Post(url, "application/json", toJSONReader(params))
	if err != nil {
		return fmt.Errorf("failed to set webhook: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Printf("🌐 Webhook установлен: %s", string(body))

	return nil
}

// deleteWebhook удаляет webhook
func (uh *UpdatesHandler) deleteWebhook() error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/deleteWebhook", uh.config.TelegramBotToken)

	params := map[string]interface{}{
		"drop_pending_updates": true,
	}

	resp, err := uh.httpClient.Post(url, "application/json", toJSONReader(params))
	if err != nil {
		return fmt.Errorf("failed to delete webhook: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Printf("🗑️ Webhook удален: %s", string(body))

	return nil
}

// toJSONReader конвертирует map в io.Reader с JSON
func toJSONReader(data interface{}) io.Reader {
	jsonData, _ := json.Marshal(data)
	return strings.NewReader(string(jsonData))
}
