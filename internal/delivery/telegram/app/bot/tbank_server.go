// internal/delivery/telegram/app/bot/tbank_server.go
package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	tbank_service "crypto-exchange-screener-bot/internal/delivery/telegram/services/tbank"
	"crypto-exchange-screener-bot/pkg/logger"
)

// TBankNotifyServer HTTP-сервер для приёма уведомлений от Т-Банк
// Запускается независимо от режима работы бота (polling/webhook)
type TBankNotifyServer struct {
	service tbank_service.Service
	server  *http.Server
	port    int
}

// NewTBankNotifyServer создаёт сервер уведомлений Т-Банк
func NewTBankNotifyServer(service tbank_service.Service, port int) *TBankNotifyServer {
	return &TBankNotifyServer{
		service: service,
		port:    port,
	}
}

// Start запускает сервер уведомлений
func (s *TBankNotifyServer) Start() error {
	if s.service == nil {
		return fmt.Errorf("TBankService не настроен")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/notify", s.handleNotification)
	mux.HandleFunc("/health", s.handleHealth)

	addr := fmt.Sprintf(":%d", s.port)
	s.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Printf("🏦 Запуск сервера уведомлений Т-Банк на %s/notify", addr)

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("❌ Ошибка сервера уведомлений Т-Банк: %v", err)
		}
	}()

	time.Sleep(50 * time.Millisecond)
	return nil
}

// Stop останавливает сервер
func (s *TBankNotifyServer) Stop() error {
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(ctx)
	}
	return nil
}

// handleNotification обрабатывает входящее уведомление от Т-Банк
func (s *TBankNotifyServer) handleNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		logger.Error("❌ Ошибка чтения тела уведомления Т-Банк: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Парсим JSON в map[string]string для проверки токена
	// (Т-Банк отправляет все поля как строки или числа, приводим к строкам)
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		logger.Error("❌ Ошибка парсинга уведомления Т-Банк: %v, тело: %s", err, string(body))
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Конвертируем в map[string]string для VerifyToken
	params := make(map[string]string, len(raw))
	for k, v := range raw {
		switch val := v.(type) {
		case string:
			params[k] = val
		case bool:
			if val {
				params[k] = "true"
			} else {
				params[k] = "false"
			}
		case float64:
			params[k] = fmt.Sprintf("%v", val)
		case json.Number:
			params[k] = val.String()
		default:
			params[k] = fmt.Sprintf("%v", val)
		}
	}

	logger.Info("📩 Уведомление от Т-Банк: OrderId=%s, Status=%s", params["OrderId"], params["Status"])

	ctx := context.Background()
	if err := s.service.HandleNotification(ctx, params); err != nil {
		logger.Error("❌ Ошибка обработки уведомления Т-Банк: %v", err)
		// Возвращаем 200 OK даже при ошибке обработки,
		// чтобы Т-Банк не повторял запрос с некорректными данными.
		// Для повторяемых ошибок (сеть, БД) возвращаем 500.
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Т-Банк ожидает ответ "OK" с кодом 200
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "OK")
}

// handleHealth проверка работоспособности сервера
func (s *TBankNotifyServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"server": "tbank-notify",
		"time":   time.Now().Format(time.RFC3339),
	})
}
