// internal/delivery/max/transport/webhook.go
package transport

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	maxpkg "crypto-exchange-screener-bot/internal/delivery/max"
	"crypto-exchange-screener-bot/pkg/logger"
)

// WebhookConfig конфигурация вебхука для MAX
type WebhookConfig struct {
	Domain      string
	Port        int
	Path        string
	SecretToken string
	UseTLS      bool
	TLSCertPath string
	TLSKeyPath  string
	MaxBodySize int64
}

// WebhookServer сервер для обработки webhook запросов от MAX
type WebhookServer struct {
	config  WebhookConfig
	client  *maxpkg.Client
	handler UpdateHandler
	server  *http.Server
}

// NewWebhookServer создает новый сервер webhook для MAX
func NewWebhookServer(cfg WebhookConfig, client *maxpkg.Client, handler UpdateHandler) *WebhookServer {
	return &WebhookServer{
		config:  cfg,
		client:  client,
		handler: handler,
	}
}

// Start запускает сервер webhook с поддержкой TLS
func (ws *WebhookServer) Start() error {
	if ws.client == nil {
		return fmt.Errorf("MAX client не инициализирован")
	}

	// Проверяем наличие сертификатов если используется TLS
	if ws.config.UseTLS {
		if ws.config.TLSCertPath == "" || ws.config.TLSKeyPath == "" {
			return fmt.Errorf("TLS включен, но пути к сертификатам не указаны")
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc(ws.config.Path, ws.handleWebhook)
	mux.HandleFunc("/health", ws.handleHealthCheck)

	addr := fmt.Sprintf(":%d", ws.config.Port)
	ws.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Настраиваем TLS если включено
	if ws.config.UseTLS {
		ws.server.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	logger.Info("🚀 Запуск MAX webhook сервера на %s%s", addr, ws.config.Path)

	go func() {
		var err error
		if ws.config.UseTLS {
			err = ws.server.ListenAndServeTLS(
				ws.config.TLSCertPath,
				ws.config.TLSKeyPath,
			)
		} else {
			err = ws.server.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			logger.Error("❌ MAX webhook server error: %v", err)
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

// handleWebhook обрабатывает входящие webhook запросы от MAX
func (ws *WebhookServer) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Проверяем размер тела
	if r.ContentLength > ws.config.MaxBodySize {
		http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
		return
	}

	// Проверяем секретный токен если настроен
	if ws.config.SecretToken != "" {
		token := r.Header.Get("X-Secret-Token")
		if token != ws.config.SecretToken {
			logger.Warn("⚠️ MAX webhook: неверный секретный токен")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error("❌ Не удалось прочитать тело webhook: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Парсим обновление от MAX
	var update maxpkg.Update
	if err := json.Unmarshal(body, &update); err != nil {
		logger.Error("❌ Не удалось распарсить webhook update: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Логируем входящее обновление
	logger.Info("📩 [MAX WEBHOOK] Получено обновление type=%s", update.UpdateType)
	logger.Debug("   • Message: %v", update.Message != nil)
	logger.Debug("   • Callback: %v", update.Callback != nil)

	// Обрабатываем обновление через хендлер
	if ws.handler != nil {
		go ws.handler(update)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleHealthCheck обрабатывает запросы проверки здоровья
func (ws *WebhookServer) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	if ws.client == nil {
		http.Error(w, "Client not initialized", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status":         "ok",
		"client":         ws.client != nil,
		"time":           time.Now().Format(time.RFC3339),
		"version":        "1.0.0",
		"webhook_mode":   true,
		"webhook_domain": ws.config.Domain,
		"webhook_port":   ws.config.Port,
		"webhook_tls":    ws.config.UseTLS,
	}

	json.NewEncoder(w).Encode(response)
}
