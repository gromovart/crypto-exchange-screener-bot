// internal/delivery/telegram/app/bot/webhook.go
package bot

import (
	"crypto-exchange-screener-bot/internal/delivery/telegram"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"crypto-exchange-screener-bot/pkg/logger"
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

// Start запускает сервер webhook с поддержкой TLS
func (ws *WebhookServer) Start() error {
	if ws.bot == nil {
		return fmt.Errorf("telegram bot not initialized")
	}

	// Проверяем наличие сертификатов если используется TLS
	if ws.config.Webhook.UseTLS {
		if ws.config.Webhook.TLSCertPath == "" || ws.config.Webhook.TLSKeyPath == "" {
			return fmt.Errorf("TLS включен но пути к сертификатам не указаны")
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc(ws.config.Webhook.Path, ws.handleWebhook)
	mux.HandleFunc("/health", ws.handleHealthCheck)

	addr := fmt.Sprintf(":%d", ws.config.Webhook.Port)
	ws.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Настраиваем TLS если включено
	if ws.config.Webhook.UseTLS {
		ws.server.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	log.Printf("🚀 Starting Telegram webhook server on %s%s", addr, ws.config.Webhook.Path)

	go func() {
		var err error
		if ws.config.Webhook.UseTLS {
			err = ws.server.ListenAndServeTLS(
				ws.config.Webhook.TLSCertPath,
				ws.config.Webhook.TLSKeyPath,
			)
		} else {
			err = ws.server.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			log.Printf("❌ Webhook server error: %v", err)
		}
	}()

	// Проверяем что сервер запустился
	time.Sleep(100 * time.Millisecond)
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

	// Проверяем размер тела
	if r.ContentLength > ws.config.Webhook.MaxBodySize {
		http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("❌ Failed to read webhook body: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// ⭐ ТОЧНО КАК В POLLING.GO - парсим обновление
	var updateData struct {
		UpdateID int `json:"update_id"`
		Message  *struct {
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
			Text              string `json:"text"`
			SuccessfulPayment *struct {
				Currency                string `json:"currency"`
				TotalAmount             int    `json:"total_amount"`
				InvoicePayload          string `json:"invoice_payload"`
				TelegramPaymentChargeID string `json:"telegram_payment_charge_id"`
				ProviderPaymentChargeID string `json:"provider_payment_charge_id"`
			} `json:"successful_payment"`
		} `json:"message"`
		CallbackQuery *struct {
			ID   string `json:"id"`
			From *struct {
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
		} `json:"callback_query"`
		PreCheckoutQuery *struct {
			ID   string `json:"id"`
			From *struct {
				ID        int64  `json:"id"`
				Username  string `json:"username"`
				FirstName string `json:"first_name"`
				LastName  string `json:"last_name"`
			} `json:"from"`
			Currency       string `json:"currency"`
			TotalAmount    int    `json:"total_amount"`
			InvoicePayload string `json:"invoice_payload"`
		} `json:"pre_checkout_query"`
	}

	if err := json.Unmarshal(body, &updateData); err != nil {
		log.Printf("❌ Failed to parse webhook update: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// ⭐ ЛОГИРУЕМ КАК В POLLING.GO
	logger.Warn("📩 [WEBHOOK] Получено обновление ID=%d", updateData.UpdateID)
	logger.Warn("   • Message: %v", updateData.Message != nil)
	logger.Warn("   • Callback: %v", updateData.CallbackQuery != nil)
	logger.Warn("   • PreCheckout: %v", updateData.PreCheckoutQuery != nil)

	// Если есть successful_payment, логируем детали
	if updateData.Message != nil && updateData.Message.SuccessfulPayment != nil {
		logger.Warn("💰💰💰 [WEBHOOK] SUCCESSFUL PAYMENT DETECTED!")
		logger.Warn("   • From ID: %d", updateData.Message.From.ID)
		logger.Warn("   • Amount: %d %s", updateData.Message.SuccessfulPayment.TotalAmount, updateData.Message.SuccessfulPayment.Currency)
		logger.Warn("   • Payload: %s", updateData.Message.SuccessfulPayment.InvoicePayload)
		logger.Warn("   • TelegramChargeID: %s", updateData.Message.SuccessfulPayment.TelegramPaymentChargeID)
		logger.Warn("   • ProviderChargeID: %s", updateData.Message.SuccessfulPayment.ProviderPaymentChargeID)
	}

	// ⭐ КОНВЕРТИРУЕМ В TELEGRAM.TELEGRAMUPDATE КАК В POLLING.GO
	middlewareUpdate := &telegram.TelegramUpdate{
		UpdateID: updateData.UpdateID,
	}

	// Обработка Message
	if updateData.Message != nil {
		msg := &telegram.Message{
			MessageID: int64(updateData.Message.MessageID),
			Text:      updateData.Message.Text,
		}

		if updateData.Message.From != nil {
			msg.From = telegram.User{
				ID:        updateData.Message.From.ID,
				Username:  updateData.Message.From.Username,
				FirstName: updateData.Message.From.FirstName,
				LastName:  updateData.Message.From.LastName,
			}
		}

		if updateData.Message.Chat != nil {
			msg.Chat = telegram.Chat{
				ID: updateData.Message.Chat.ID,
			}
		}

		if updateData.Message.SuccessfulPayment != nil {
			msg.SuccessfulPayment = &telegram.SuccessfulPayment{
				Currency:                updateData.Message.SuccessfulPayment.Currency,
				TotalAmount:             updateData.Message.SuccessfulPayment.TotalAmount,
				InvoicePayload:          updateData.Message.SuccessfulPayment.InvoicePayload,
				TelegramPaymentChargeID: updateData.Message.SuccessfulPayment.TelegramPaymentChargeID,
				ProviderPaymentChargeID: updateData.Message.SuccessfulPayment.ProviderPaymentChargeID,
			}
		}

		middlewareUpdate.Message = msg
	}

	// Обработка CallbackQuery
	if updateData.CallbackQuery != nil {
		callback := &telegram.CallbackQueryStruct{
			ID:   updateData.CallbackQuery.ID,
			Data: updateData.CallbackQuery.Data,
			From: &struct {
				ID        int64  `json:"id"`
				Username  string `json:"username"`
				FirstName string `json:"first_name"`
				LastName  string `json:"last_name"`
			}{
				ID:        updateData.CallbackQuery.From.ID,
				Username:  updateData.CallbackQuery.From.Username,
				FirstName: updateData.CallbackQuery.From.FirstName,
				LastName:  updateData.CallbackQuery.From.LastName,
			},
		}

		if updateData.CallbackQuery.Message != nil {
			callback.Message = &struct {
				MessageID int `json:"message_id"`
				Chat      *struct {
					ID int64 `json:"id"`
				} `json:"chat"`
			}{
				MessageID: updateData.CallbackQuery.Message.MessageID,
			}

			if updateData.CallbackQuery.Message.Chat != nil {
				callback.Message.Chat = &struct {
					ID int64 `json:"id"`
				}{
					ID: updateData.CallbackQuery.Message.Chat.ID,
				}
			}
		}

		middlewareUpdate.CallbackQuery = callback
	}

	// Обработка PreCheckoutQuery
	if updateData.PreCheckoutQuery != nil {
		preCheckout := &telegram.PreCheckoutQuery{
			ID:             updateData.PreCheckoutQuery.ID,
			Currency:       updateData.PreCheckoutQuery.Currency,
			TotalAmount:    updateData.PreCheckoutQuery.TotalAmount,
			InvoicePayload: updateData.PreCheckoutQuery.InvoicePayload,
			From: &struct {
				ID        int64  `json:"id"`
				Username  string `json:"username"`
				FirstName string `json:"first_name"`
				LastName  string `json:"last_name"`
			}{
				ID:        updateData.PreCheckoutQuery.From.ID,
				Username:  updateData.PreCheckoutQuery.From.Username,
				FirstName: updateData.PreCheckoutQuery.From.FirstName,
				LastName:  updateData.PreCheckoutQuery.From.LastName,
			},
		}

		middlewareUpdate.PreCheckoutQuery = preCheckout
	}

	// Обработка обновления через бота
	if err := ws.bot.HandleUpdate(middlewareUpdate); err != nil {
		log.Printf("❌ Failed to handle update: %v", err)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleHealthCheck обрабатывает запросы проверки здоровья
func (ws *WebhookServer) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	if ws.bot == nil {
		http.Error(w, "Bot not initialized", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status":         "ok",
		"bot":            ws.bot != nil,
		"time":           time.Now().Format(time.RFC3339),
		"version":        "1.0.0",
		"webhook_mode":   true,
		"webhook_domain": ws.config.Webhook.Domain,
		"webhook_port":   ws.config.Webhook.Port,
		"webhook_tls":    ws.config.Webhook.UseTLS,
	}

	json.NewEncoder(w).Encode(response)
}
