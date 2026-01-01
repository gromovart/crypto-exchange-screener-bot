// internal/delivery/telegram/webhook.go
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

// WebhookServer - сервер для обработки webhook запросов от Telegram
type WebhookServer struct {
	config *config.Config
	bot    *TelegramBot
	server *http.Server
}

// NewWebhookServer создает новый сервер webhook
func NewWebhookServer(cfg *config.Config, bot *TelegramBot) *WebhookServer {
	return &WebhookServer{
		config: cfg,
		bot:    bot,
	}
}

// TelegramUpdate - структура для обновлений от Telegram
type TelegramUpdate struct {
	UpdateID      int64          `json:"update_id"`
	Message       *Message       `json:"message,omitempty"`
	CallbackQuery *CallbackQuery `json:"callback_query,omitempty"`
}

// Message - сообщение от пользователя
type Message struct {
	MessageID int64  `json:"message_id"`
	From      User   `json:"from"`
	Chat      Chat   `json:"chat"`
	Text      string `json:"text"`
	Date      int64  `json:"date"`
}

// CallbackQuery - callback от inline кнопки
type CallbackQuery struct {
	ID           string   `json:"id"`
	From         User     `json:"from"`
	Message      *Message `json:"message"`
	ChatInstance string   `json:"chat_instance"`
	Data         string   `json:"data"`
}

// User - пользователь Telegram
type User struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

// Chat - чат Telegram
type Chat struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title,omitempty"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}

// Start запускает сервер webhook
func (ws *WebhookServer) Start() error {
	if ws.bot == nil {
		return fmt.Errorf("telegram bot not initialized")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", ws.handleWebhook)
	mux.HandleFunc("/health", ws.handleHealthCheck)

	ws.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", ws.config.HTTPPort),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Printf("🚀 Starting Telegram webhook server on port %d", ws.config.HTTPPort)

	go func() {
		if err := ws.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("❌ Webhook server error: %v", err)
		}
	}()

	return nil
}

// Stop останавливает сервер webhook
func (ws *WebhookServer) Stop() error {
	if ws.server != nil {
		return ws.server.Close()
	}
	return nil
}

// handleWebhook обрабатывает входящие webhook запросы
func (ws *WebhookServer) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("❌ Failed to read webhook body: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var update TelegramUpdate
	if err := json.Unmarshal(body, &update); err != nil {
		log.Printf("❌ Failed to parse webhook update: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Обработка обновления
	ws.handleUpdate(update)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleUpdate обрабатывает обновление от Telegram
func (ws *WebhookServer) handleUpdate(update TelegramUpdate) {
	// Обработка сообщений (нажатия кнопок меню)
	if update.Message != nil && update.Message.Text != "" {
		chatID := fmt.Sprintf("%d", update.Message.Chat.ID)
		text := strings.TrimSpace(update.Message.Text)

		log.Printf("📨 Received message from chat %s: %s", chatID, text)

		// Обработка команд и нажатий кнопок меню
		if strings.HasPrefix(text, "/") {
			// Обработка текстовых команд
			ws.handleCommand(text, chatID, update.Message.From.Username)
		} else {
			// Обработка нажатий кнопок меню
			if err := ws.bot.HandleMessage(text, chatID); err != nil {
				log.Printf("❌ Failed to handle menu button: %v", err)
			}
		}
	}

	// Обработка callback от inline кнопок
	if update.CallbackQuery != nil && update.CallbackQuery.Data != "" {
		chatID := fmt.Sprintf("%d", update.CallbackQuery.Message.Chat.ID)
		callbackData := update.CallbackQuery.Data

		log.Printf("🔄 Processing callback: %s from chat %s", callbackData, chatID)

		if err := ws.bot.HandleCallback(callbackData, chatID); err != nil {
			log.Printf("❌ Failed to handle callback: %v", err)
		}

		// Отправляем подтверждение
		ws.answerCallbackQuery(update.CallbackQuery.ID)
	}
}

// handleCommand обрабатывает команды
func (ws *WebhookServer) handleCommand(command, chatID, username string) {
	log.Printf("⚡ Processing command: %s from @%s", command, username)

	switch command {
	case "/start":
		if err := ws.bot.StartCommandHandler(chatID); err != nil {
			log.Printf("❌ Failed to handle /start: %v", err)
		}
	case "/status":
		if err := ws.bot.menuManager.SendStatus(chatID); err != nil {
			log.Printf("❌ Failed to send status: %v", err)
		}
	case "/help":
		if err := ws.bot.menuManager.SendHelp(chatID); err != nil {
			log.Printf("❌ Failed to send help: %v", err)
		}
	case "/notify_on":
		if err := ws.bot.menuManager.HandleNotifyOn(chatID); err != nil {
			log.Printf("❌ Failed to enable notifications: %v", err)
		}
	case "/notify_off":
		if err := ws.bot.menuManager.HandleNotifyOff(chatID); err != nil {
			log.Printf("❌ Failed to disable notifications: %v", err)
		}
	case "/settings":
		if err := ws.bot.menuManager.SendSettingsMessage(chatID); err != nil {
			log.Printf("❌ Failed to send settings: %v", err)
		}
	case "/test":
		if err := ws.bot.SendTestMessage(); err != nil {
			log.Printf("❌ Failed to send test message: %v", err)
		}
	default:
		if err := ws.bot.SendMessage(fmt.Sprintf("❓ Неизвестная команда: %s. Используйте /help", command)); err != nil {
			log.Printf("❌ Failed to send unknown command message: %v", err)
		}
	}
}

// answerCallbackQuery отправляет ответ на callback запрос
func (ws *WebhookServer) answerCallbackQuery(callbackID string) error {
	if ws.bot == nil || ws.bot.messageSender == nil {
		return fmt.Errorf("bot or message sender not initialized")
	}

	answer := struct {
		CallbackQueryID string `json:"callback_query_id"`
		Text            string `json:"text,omitempty"`
		ShowAlert       bool   `json:"show_alert,omitempty"`
	}{
		CallbackQueryID: callbackID,
		Text:            "✅",
		ShowAlert:       false,
	}

	return ws.bot.messageSender.SendTelegramRequest("answerCallbackQuery", answer)
}

// handleHealthCheck обрабатывает запросы проверки здоровья
func (ws *WebhookServer) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	if ws.bot == nil {
		http.Error(w, "Bot not initialized", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status":  "ok",
		"bot":     ws.bot != nil,
		"time":    time.Now().Format(time.RFC3339),
		"version": "1.0.0",
	}

	json.NewEncoder(w).Encode(response)
}
