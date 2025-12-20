// internal/telegram/webhook.go (исправленная версия)
package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// WebhookServer - сервер для обработки webhook от Telegram
type WebhookServer struct {
	bot        *TelegramBot
	httpServer *http.Server
	mu         sync.RWMutex
	config     struct {
		Port       int
		WebhookURL string
		Secret     string
	}
}

// TelegramUpdate - структура обновления от Telegram
type TelegramUpdate struct {
	UpdateID int `json:"update_id"`
	Message  struct {
		MessageID int `json:"message_id"`
		From      struct {
			ID        int64  `json:"id"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name,omitempty"`
			Username  string `json:"username,omitempty"`
		} `json:"from"`
		Chat struct {
			ID        int64  `json:"id"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name,omitempty"`
			Username  string `json:"username,omitempty"`
			Type      string `json:"type"`
		} `json:"chat"`
		Date    int    `json:"date"`
		Text    string `json:"text"`
		Caption string `json:"caption,omitempty"`
	} `json:"message"`
	CallbackQuery struct {
		ID   string `json:"id"`
		From struct {
			ID        int64  `json:"id"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name,omitempty"`
			Username  string `json:"username,omitempty"`
		} `json:"from"`
		Message struct {
			MessageID int `json:"message_id"`
			Chat      struct {
				ID        int64  `json:"id"`
				FirstName string `json:"first_name"`
				LastName  string `json:"last_name,omitempty"`
				Username  string `json:"username,omitempty"`
				Type      string `json:"type"`
			} `json:"chat"`
		} `json:"message"`
		ChatInstance string `json:"chat_instance"`
		Data         string `json:"data"`
	} `json:"callback_query"`
}

// NewWebhookServer создает новый сервер webhook
func NewWebhookServer(bot *TelegramBot, port int, webhookURL, secret string) *WebhookServer {
	return &WebhookServer{
		bot: bot,
		config: struct {
			Port       int
			WebhookURL string
			Secret     string
		}{
			Port:       port,
			WebhookURL: webhookURL,
			Secret:     secret,
		},
	}
}

// Start запускает сервер webhook
func (ws *WebhookServer) Start() error {
	if ws.bot == nil {
		return fmt.Errorf("telegram bot is not initialized")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/webhook/"+ws.config.Secret, ws.handleWebhook)
	mux.HandleFunc("/health", ws.handleHealthCheck)
	mux.HandleFunc("/", ws.handleDefault)

	ws.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", ws.config.Port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("🌐 Webhook server starting on port %d", ws.config.Port)
	return ws.httpServer.ListenAndServe()
}

// Stop останавливает сервер webhook
func (ws *WebhookServer) Stop() error {
	if ws.httpServer != nil {
		return ws.httpServer.Close()
	}
	return nil
}

// handleWebhook обрабатывает входящие webhook запросы от Telegram
func (ws *WebhookServer) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var update TelegramUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		log.Printf("❌ Failed to decode webhook update: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	log.Printf("📨 Received Telegram update: %+v", update)

	// Обрабатываем сообщения
	if update.Message.Text != "" {
		// Конвертируем int64 chatID в string
		chatID := strconv.FormatInt(update.Message.Chat.ID, 10)
		ws.handleMessage(update.Message.Text, chatID)
	}

	// Обрабатываем callback query
	if update.CallbackQuery.Data != "" {
		// Конвертируем int64 chatID в string
		chatID := strconv.FormatInt(update.CallbackQuery.Message.Chat.ID, 10)
		ws.handleCallbackQuery(update.CallbackQuery.Data, chatID, update.CallbackQuery.ID)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleMessage обрабатывает входящие сообщения
func (ws *WebhookServer) handleMessage(text, chatID string) {
	log.Printf("📝 Handling message from chat %s: %s", chatID, text)

	switch text {
	case "/start":
		if err := ws.bot.StartCommandHandler(chatID); err != nil {
			log.Printf("❌ Failed to handle /start command: %v", err)
		}
	case "/status":
		if err := ws.bot.sendStatus(chatID); err != nil {
			log.Printf("❌ Failed to handle /status command: %v", err)
		}
	case "/notify_on":
		ws.bot.SetNotifyEnabled(true)
		ws.bot.sendMessageWithKeyboardToChat(chatID, "✅ Уведомления включены", nil)
	case "/notify_off":
		ws.bot.SetNotifyEnabled(false)
		ws.bot.sendMessageWithKeyboardToChat(chatID, "❌ Уведомления выключены", nil)
	case "/test":
		message := "📊 *Тестовое сообщение*\n\n" +
			"Это тестовое сообщение для проверки работы бота.\n" +
			"✅ Бот работает корректно!\n" +
			"🕐 Время: " + time.Now().Format("2006-01-02 15:04:05")

		keyboard := &InlineKeyboardMarkup{
			InlineKeyboard: [][]InlineKeyboardButton{
				{
					{Text: "📊 Статус", CallbackData: "status"},
					{Text: "⚙️ Настройки", CallbackData: "config"},
				},
			},
		}

		if err := ws.bot.sendMessageWithKeyboardToChat(chatID, message, keyboard); err != nil {
			log.Printf("❌ Failed to send test message: %v", err)
		}
	case "/help":
		message := "🆘 *Помощь*\n\n" +
			"*Доступные команды:*\n" +
			"/start - Начало работы\n" +
			"/status - Статус системы\n" +
			"/notify_on - Включить уведомления\n" +
			"/notify_off - Выключить уведомления\n" +
			"/test - Тестовое уведомление\n" +
			"/help - Эта справка"

		if err := ws.bot.sendMessageWithKeyboardToChat(chatID, message, nil); err != nil {
			log.Printf("❌ Failed to send help message: %v", err)
		}
	default:
		if err := ws.bot.sendMessageWithKeyboardToChat(chatID, "❓ Неизвестная команда. Используйте /help для списка команд", nil); err != nil {
			log.Printf("❌ Failed to send unknown command message: %v", err)
		}
	}
}

// handleCallbackQuery обрабатывает callback query от inline кнопок
func (ws *WebhookServer) handleCallbackQuery(data, chatID, callbackID string) {
	log.Printf("🔄 Handling callback query from chat %s: %s", chatID, data)

	// Отправляем ответ на callback (чтобы убрать "часики" на кнопке)
	ws.answerCallbackQuery(callbackID)

	// Обрабатываем данные
	if err := ws.bot.HandleCallback(data, chatID); err != nil {
		log.Printf("❌ Failed to handle callback: %v", err)
		ws.bot.sendMessageWithKeyboardToChat(chatID, fmt.Sprintf("❌ Ошибка обработки запроса: %v", err), nil)
	}
}

// answerCallbackQuery отправляет ответ на callback query
func (ws *WebhookServer) answerCallbackQuery(callbackID string) {
	if ws.bot == nil {
		return
	}

	response := struct {
		CallbackQueryID string `json:"callback_query_id"`
		Text            string `json:"text,omitempty"`
		ShowAlert       bool   `json:"show_alert"`
	}{
		CallbackQueryID: callbackID,
		Text:            "✅ Обработано",
		ShowAlert:       false,
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		log.Printf("❌ Failed to marshal callback response: %v", err)
		return
	}

	resp, err := ws.bot.httpClient.Post(
		ws.bot.baseURL+"answerCallbackQuery",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		log.Printf("❌ Failed to answer callback query: %v", err)
		return
	}
	defer resp.Body.Close()
}

// handleHealthCheck обрабатывает запросы проверки здоровья
func (ws *WebhookServer) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	status := map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	}
	json.NewEncoder(w).Encode(status)
}

// handleDefault обрабатывает запросы по умолчанию
func (ws *WebhookServer) handleDefault(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`
		<!DOCTYPE html>
		<html>
		<head>
			<title>Telegram Webhook Server</title>
			<style>
				body { font-family: Arial, sans-serif; margin: 40px; }
				.container { max-width: 800px; margin: 0 auto; }
				h1 { color: #0088cc; }
				.status { background: #f0f9ff; padding: 20px; border-radius: 5px; }
			</style>
		</head>
		<body>
			<div class="container">
				<h1>🤖 Telegram Webhook Server</h1>
				<div class="status">
					<p><strong>Status:</strong> ✅ Running</p>
					<p><strong>Time:</strong> ` + time.Now().Format("2006-01-02 15:04:05") + `</p>
					<p><strong>Endpoint:</strong> /webhook/{secret}</p>
					<p><strong>Health Check:</strong> <a href="/health">/health</a></p>
				</div>
			</div>
		</body>
		</html>
	`))
}
