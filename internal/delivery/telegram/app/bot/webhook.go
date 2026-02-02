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

	var update telegram.TelegramUpdate
	if err := json.Unmarshal(body, &update); err != nil {
		log.Printf("❌ Failed to parse webhook update: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Обработка обновления через новую систему
	if err := ws.bot.HandleUpdate(&update); err != nil {
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
