// internal/delivery/telegram/app/bot/webhook.go
package bot

import (
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/middlewares"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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

	// Преобразуем в формат middlewares.TelegramUpdate
	middlewareUpdate := createMiddlewareUpdate(update)

	// Обработка обновления через новую систему
	if err := ws.bot.HandleUpdate(middlewareUpdate); err != nil {
		log.Printf("❌ Failed to handle update: %v", err)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// createMiddlewareUpdate создает middlewares.TelegramUpdate из нашего TelegramUpdate
func createMiddlewareUpdate(update TelegramUpdate) *middlewares.TelegramUpdate {
	middlewareUpdate := &middlewares.TelegramUpdate{
		UpdateID: int(update.UpdateID),
	}

	if update.Message != nil {
		// Создаем структуру Message как ожидает middlewares
		middlewareUpdate.Message = &struct {
			MessageID int `json:"message_id"`
			From      *struct {
				ID        int64  `json:"id"`
				Username  string `json:"username"`
				FirstName string `json:"first_name"`
				LastName  string `json:"last_name"`
			} `json:"from"`
			Chat *struct {
				ID int64 `json:"id"`
			} `json:"chat"`
			Text string `json:"text"`
		}{
			MessageID: int(update.Message.MessageID),
			From: &struct {
				ID        int64  `json:"id"`
				Username  string `json:"username"`
				FirstName string `json:"first_name"`
				LastName  string `json:"last_name"`
			}{
				ID:        update.Message.From.ID,
				Username:  update.Message.From.Username,
				FirstName: update.Message.From.FirstName,
				LastName:  update.Message.From.LastName,
			},
			Chat: &struct {
				ID int64 `json:"id"`
			}{
				ID: update.Message.Chat.ID,
			},
			Text: update.Message.Text,
		}
	}

	if update.CallbackQuery != nil {
		// Создаем структуру CallbackQuery как ожидает middlewares
		middlewareUpdate.CallbackQuery = &struct {
			ID      string `json:"id"`
			From    *struct {
				ID        int64  `json:"id"`
				Username  string `json:"username"`
				FirstName string `json:"first_name"`
				LastName  string `json:"last_name"`
			} `json:"from"`
			Message *struct {
				MessageID int `json:"message_id"`
				Chat      *struct {
					ID int64 `json:"id"`
				} `json:"chat"`
			} `json:"message"`
			Data string `json:"data"`
		}{
			ID: update.CallbackQuery.ID,
			From: &struct {
				ID        int64  `json:"id"`
				Username  string `json:"username"`
				FirstName string `json:"first_name"`
				LastName  string `json:"last_name"`
			}{
				ID:        update.CallbackQuery.From.ID,
				Username:  update.CallbackQuery.From.Username,
				FirstName: update.CallbackQuery.From.FirstName,
				LastName:  update.CallbackQuery.From.LastName,
			},
			Data: update.CallbackQuery.Data,
		}

		if update.CallbackQuery.Message != nil {
			middlewareUpdate.CallbackQuery.Message = &struct {
				MessageID int `json:"message_id"`
				Chat      *struct {
					ID int64 `json:"id"`
				} `json:"chat"`
			}{
				Chat: &struct {
					ID int64 `json:"id"`
				}{
					ID: update.CallbackQuery.Message.Chat.ID,
				},
			}
		}
	}

	return middlewareUpdate
}

// answerCallbackQuery отправляет ответ на callback запрос
func (ws *WebhookServer) answerCallbackQuery(callbackID string) error {
	// Упрощенная реализация
	return nil
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
