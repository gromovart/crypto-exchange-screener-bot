package telegram

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// Update представляет обновление от Telegram
type Update struct {
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
		Date     int    `json:"date"`
		Text     string `json:"text"`
		Entities []struct {
			Type   string `json:"type"`
			Offset int    `json:"offset"`
			Length int    `json:"length"`
		} `json:"entities,omitempty"`
	} `json:"message,omitempty"`
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
			Date int    `json:"date"`
			Text string `json:"text"`
		} `json:"message,omitempty"`
		ChatInstance string `json:"chat_instance"`
		Data         string `json:"data"`
	} `json:"callback_query,omitempty"`
}

// WebhookServer - сервер для обработки webhook от Telegram
type WebhookServer struct {
	bot        *TelegramBot
	port       string
	webhookURL string
}

// NewWebhookServer создает новый webhook сервер
func NewWebhookServer(bot *TelegramBot, port, webhookURL string) *WebhookServer {
	return &WebhookServer{
		bot:        bot,
		port:       port,
		webhookURL: webhookURL,
	}
}

// Start запускает webhook сервер
func (ws *WebhookServer) Start() error {
	// Устанавливаем webhook
	if err := ws.setWebhook(); err != nil {
		return fmt.Errorf("failed to set webhook: %w", err)
	}

	// Запускаем HTTP сервер
	http.HandleFunc("/webhook", ws.handleWebhook)
	http.HandleFunc("/health", ws.handleHealth)

	log.Printf("🌐 Telegram webhook сервер запущен на порту %s", ws.port)
	return http.ListenAndServe(":"+ws.port, nil)
}

// setWebhook устанавливает webhook URL
func (ws *WebhookServer) setWebhook() error {
	if ws.webhookURL == "" {
		return nil // Пропускаем если URL не указан
	}

	url := fmt.Sprintf("%ssetWebhook?url=%s/webhook", ws.bot.baseURL, ws.webhookURL)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	log.Printf("✅ Webhook установлен: %s", ws.webhookURL)
	return nil
}

// handleWebhook обрабатывает webhook запросы
func (ws *WebhookServer) handleWebhook(w http.ResponseWriter, r *http.Request) {
	var update Update
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		log.Printf("❌ Ошибка декодирования webhook: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Обрабатываем обновление
	go ws.processUpdate(update)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleHealth проверка здоровья сервера
func (ws *WebhookServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("✅ Telegram bot is healthy"))
}

// processUpdate обрабатывает обновление
func (ws *WebhookServer) processUpdate(update Update) {
	// Обработка callback query (нажатие на кнопку)
	if update.CallbackQuery.Data != "" {
		chatID := update.CallbackQuery.From.ID
		if err := ws.bot.HandleCallback(update.CallbackQuery.Data, chatID); err != nil {
			log.Printf("❌ Ошибка обработки callback: %v", err)
		}
		return
	}

	// Обработка текстовых команд
	if update.Message.Text != "" {
		chatID := update.Message.Chat.ID
		text := update.Message.Text

		switch text {
		case "/start":
			if err := ws.bot.StartCommandHandler(chatID); err != nil {
				log.Printf("❌ Ошибка обработки /start: %v", err)
			}
		case "/status":
			if err := ws.bot.sendStatus(chatID); err != nil {
				log.Printf("❌ Ошибка отправки статуса: %v", err)
			}
		case "/notify_on":
			ws.bot.notifyEnabled = true
			ws.bot.sendMessageWithKeyboardToChat(chatID, "✅ Уведомления включены", nil)
		case "/notify_off":
			ws.bot.notifyEnabled = false
			ws.bot.sendMessageWithKeyboardToChat(chatID, "❌ Уведомления выключены", nil)
		case "/test":
			ws.bot.sendMessageWithKeyboardToChat(chatID, "✅ Тестовое сообщение", nil)
		case "/help":
			helpText := "📋 *Доступные команды:*\n" +
				"/start - Начало работы\n" +
				"/status - Статус системы\n" +
				"/notify_on - Включить уведомления\n" +
				"/notify_off - Выключить уведомления\n" +
				"/test - Тестовое уведомление\n" +
				"/help - Эта справка"
			ws.bot.sendMessageWithKeyboardToChat(chatID, helpText, nil)
		}
	}
}
