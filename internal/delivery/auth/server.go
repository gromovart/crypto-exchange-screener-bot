// internal/delivery/auth/server.go
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"crypto-exchange-screener-bot/internal/core/domain/users"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
	"crypto-exchange-screener-bot/pkg/logger"
)

// Sender — минимальный интерфейс отправки сообщений (подмножество message_sender.MessageSender)
type Sender interface {
	SendTextMessage(chatID int64, text string, keyboard interface{}) error
}

// TelegramSender — расширенный интерфейс для Telegram с поддержкой message_id
type TelegramSender interface {
	Sender
	SendMenuMessageWithID(chatID int64, text string, keyboard interface{}) (int64, error)
	DeleteMessage(chatID, messageID int64) error
}

// MAXSender — расширенный интерфейс для MAX с поддержкой string message_id
type MAXSender interface {
	Sender
	SendMenuMessageWithID(chatID int64, text string, keyboard interface{}) (string, error)
	DeleteMessage(mid string) error
	// DeleteOTPMessages удаляет из чата недавние сообщения, содержащие только OTP-код (N цифр).
	// Используется после верификации, чтобы очистить сообщения от кнопки «Скопировать код».
	DeleteOTPMessages(chatID int64, otpCode string, extraMid string) error
}

// Server — внутренний HTTP-сервер для OTP-авторизации
// Слушает только на 127.0.0.1, защищён X-Internal-Secret.
type Server struct {
	userService    *users.Service
	sender         Sender
	telegramSender TelegramSender // nil если не Telegram
	maxSender      MAXSender      // nil если не MAX
	store          *OTPStore      // MAX OTP store
	telegramStore  *OTPStore      // Telegram OTP store
	secret         string
	server         *http.Server
	port           int
}

// NewServer создаёт auth-сервер
func NewServer(
	userService *users.Service,
	sender Sender,
	port int,
	secret string,
	otpTTL time.Duration,
) *Server {
	// Проверяем поддерживает ли sender расширенные интерфейсы
	tgSender, _ := sender.(TelegramSender)
	maxSnd, _ := sender.(MAXSender)
	return &Server{
		userService:    userService,
		sender:         sender,
		telegramSender: tgSender,
		maxSender:      maxSnd,
		store:          NewOTPStore(otpTTL),
		telegramStore:  NewOTPStore(10 * time.Minute),
		secret:         secret,
		port:           port,
	}
}

// Start запускает сервер. Неблокирующий.
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/otp", s.withSecret(s.handleOTP))
	mux.HandleFunc("/auth/verify", s.withSecret(s.handleVerify))
	mux.HandleFunc("/auth/telegram/otp", s.withSecret(s.handleTelegramOTP))
	mux.HandleFunc("/auth/telegram/verify", s.withSecret(s.handleTelegramVerify))
	mux.HandleFunc("/health", s.handleHealth)

	addr := fmt.Sprintf("127.0.0.1:%d", s.port)
	s.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	logger.Info("🔐 Auth OTP-сервер запускается на %s", addr)

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("❌ Ошибка auth-сервера: %v", err)
		}
	}()

	time.Sleep(50 * time.Millisecond)
	logger.Info("✅ Auth OTP-сервер запущен (порт: %d)", s.port)
	return nil
}

// SetTelegramSender позволяет инжектировать Telegram sender после создания сервера.
// Вызывается из delivery layer после инициализации обоих ботов.
func (s *Server) SetTelegramSender(ts TelegramSender) {
	s.telegramSender = ts
}

// Stop останавливает сервер
func (s *Server) Stop() error {
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(ctx)
	}
	return nil
}

// ── Middleware ────────────────────────────────────────────────────────────────

// withSecret проверяет заголовок X-Internal-Secret
func (s *Server) withSecret(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Internal-Secret") != s.secret {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next(w, r)
	}
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// POST /auth/otp
// Body: {"max_user_id": 123456}
// Response: {"ok": true, "expires_in": 300}
func (s *Server) handleOTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		MaxUserID int64 `json:"max_user_id"`
	}
	if err := readJSON(r, &req); err != nil || req.MaxUserID == 0 {
		writeError(w, http.StatusBadRequest, "max_user_id обязателен")
		return
	}

	// Ищем пользователя
	user, err := s.userService.GetUserByMaxID(req.MaxUserID)
	if err != nil || user == nil {
		logger.Info("🔐 OTP: пользователь max_user_id=%d не найден", req.MaxUserID)
		writeError(w, http.StatusNotFound, "пользователь не найден")
		return
	}
	if !user.IsActive {
		writeError(w, http.StatusForbidden, "аккаунт деактивирован")
		return
	}
	if !user.IsPremium() && !user.IsAdmin() {
		writeError(w, http.StatusForbidden, "доступ только для пользователей с активной подпиской")
		return
	}

	// Определяем chatID для отправки
	chatID, err := maxChatIDFrom(user)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "нет MAX chat ID у пользователя")
		return
	}

	// Генерируем OTP
	code, err := s.store.Generate(req.MaxUserID)
	if err != nil {
		writeError(w, http.StatusTooManyRequests, err.Error())
		return
	}

	// Отправляем код в MAX
	msg := fmt.Sprintf(
		"🔐 Код для входа в Crypto Analyzer:\n\n*%s*\n\nКод действителен %.0f минут.",
		code,
		s.store.ttl.Minutes(),
	)
	// copy_code отправляет новое сообщение с кодом для удобного копирования.
	// После верификации DeleteOTPMessages удалит и оригинальное, и копию-сообщение.
	maxOTPKeyboard := []interface{}{
		map[string]interface{}{
			"type": "inline_keyboard",
			"payload": map[string]interface{}{
				"buttons": [][]interface{}{
					{
						map[string]string{"type": "callback", "text": "📋 Скопировать код", "payload": "copy_code:" + code},
					},
				},
			},
		},
	}
	var msgIDStr string
	if s.maxSender != nil {
		var sendErr error
		msgIDStr, sendErr = s.maxSender.SendMenuMessageWithID(chatID, msg, maxOTPKeyboard)
		if sendErr != nil {
			logger.Error("❌ OTP: не удалось отправить код user=%d: %v", req.MaxUserID, sendErr)
			writeError(w, http.StatusInternalServerError, "не удалось отправить код")
			return
		}
	} else {
		if sendErr := s.sender.SendTextMessage(chatID, msg, maxOTPKeyboard); sendErr != nil {
			logger.Error("❌ OTP: не удалось отправить код user=%d: %v", req.MaxUserID, sendErr)
			writeError(w, http.StatusInternalServerError, "не удалось отправить код")
			return
		}
	}
	if msgIDStr != "" {
		s.store.SetMessageIDStr(req.MaxUserID, msgIDStr)
	}

	logger.Info("🔐 OTP отправлен: max_user_id=%d chat_id=%d msg_id=%s", req.MaxUserID, chatID, msgIDStr)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":         true,
		"expires_in": int(s.store.ttl.Seconds()),
	})
}

// POST /auth/verify
// Body: {"max_user_id": 123456, "otp": "123456"}
// Response: {"ok": true, "user": {"id":1, "username":"...", "role":"admin"}}
func (s *Server) handleVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		MaxUserID int64  `json:"max_user_id"`
		OTP       string `json:"otp"`
	}
	if err := readJSON(r, &req); err != nil || req.MaxUserID == 0 || req.OTP == "" {
		writeError(w, http.StatusBadRequest, "max_user_id и otp обязательны")
		return
	}

	// Сохраняем msg_id и otpCode до Verify (после успеха запись удаляется из store)
	msgIDStr := s.store.GetMessageIDStr(req.MaxUserID)
	otpCode  := s.store.GetOTPCode(req.MaxUserID)

	ok, err := s.store.Verify(req.MaxUserID, req.OTP)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	if !ok {
		writeError(w, http.StatusUnauthorized, "неверный код")
		return
	}

	// Загружаем пользователя для ответа и получения chatID
	user, _ := s.userService.GetUserByMaxID(req.MaxUserID)

	// Удаляем оригинальное OTP-сообщение и копию от «Скопировать код»
	if s.maxSender != nil && user != nil {
		if chatID, chatErr := maxChatIDFrom(user); chatErr == nil {
			go func() {
				if delErr := s.maxSender.DeleteOTPMessages(chatID, otpCode, msgIDStr); delErr != nil {
					logger.Info("⚠️ DeleteOTPMessages max_user_id=%d: %v", req.MaxUserID, delErr)
				}
			}()
		}
	}

	logger.Info("✅ OTP верифицирован: max_user_id=%d", req.MaxUserID)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok": true,
		"user": map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"role":     user.Role,
		},
	})
}

// POST /auth/telegram/otp
// Body: {"telegram_id": 987654321}
// Response: {"ok": true, "expires_in": 600}
func (s *Server) handleTelegramOTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		TelegramID int64 `json:"telegram_id"`
	}
	if err := readJSON(r, &req); err != nil || req.TelegramID == 0 {
		writeError(w, http.StatusBadRequest, "telegram_id обязателен")
		return
	}

	// Инвалидируем старый код перед генерацией нового
	s.telegramStore.Invalidate(req.TelegramID)

	code, err := s.telegramStore.Generate(req.TelegramID)
	if err != nil {
		writeError(w, http.StatusTooManyRequests, err.Error())
		return
	}

	msg := fmt.Sprintf(
		"🔐 Код для входа в Crypto Analyzer:\n\n`%s`\n\nКод действителен 10 минут.",
		code,
	)
	tgOTPKeyboard := map[string]interface{}{
		"inline_keyboard": [][]map[string]interface{}{
			{
				{"text": "📋 Скопировать код", "copy_text": map[string]string{"text": code}},
			},
		},
	}
	var msgID int64
	if s.telegramSender != nil {
		var sendErr error
		msgID, sendErr = s.telegramSender.SendMenuMessageWithID(req.TelegramID, msg, tgOTPKeyboard)
		if sendErr != nil {
			logger.Error("❌ Telegram OTP: не удалось отправить код telegram_id=%d: %v", req.TelegramID, sendErr)
			writeError(w, http.StatusBadRequest, "не удалось отправить код. Напишите боту /start и повторите")
			return
		}
	} else {
		if sendErr := s.sender.SendTextMessage(req.TelegramID, msg, tgOTPKeyboard); sendErr != nil {
			logger.Error("❌ Telegram OTP: не удалось отправить код telegram_id=%d: %v", req.TelegramID, sendErr)
			writeError(w, http.StatusBadRequest, "не удалось отправить код. Напишите боту /start и повторите")
			return
		}
	}
	if msgID > 0 {
		s.telegramStore.SetMessageID(req.TelegramID, msgID)
	}

	logger.Info("🔐 Telegram OTP отправлен: telegram_id=%d msg_id=%d", req.TelegramID, msgID)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":         true,
		"expires_in": int(s.telegramStore.ttl.Seconds()),
	})
}

// POST /auth/telegram/verify
// Body: {"telegram_id": 987654321, "otp": "123456"}
// Response: {"ok": true, "telegram_id": 987654321, "username": "..."}
func (s *Server) handleTelegramVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		TelegramID int64  `json:"telegram_id"`
		OTP        string `json:"otp"`
	}
	if err := readJSON(r, &req); err != nil || req.TelegramID == 0 || req.OTP == "" {
		writeError(w, http.StatusBadRequest, "telegram_id и otp обязательны")
		return
	}

	// Сохраняем message_id до Verify (после успеха запись удаляется из store)
	msgID := s.telegramStore.GetMessageID(req.TelegramID)

	ok, err := s.telegramStore.Verify(req.TelegramID, req.OTP)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	if !ok {
		writeError(w, http.StatusUnauthorized, "неверный код")
		return
	}

	// Удаляем сообщение с кодом из чата
	if msgID > 0 && s.telegramSender != nil {
		if delErr := s.telegramSender.DeleteMessage(req.TelegramID, msgID); delErr != nil {
			logger.Info("⚠️ не удалось удалить OTP-сообщение telegram_id=%d msg_id=%d: %v", req.TelegramID, msgID, delErr)
		}
	}

	logger.Info("✅ Telegram OTP верифицирован: telegram_id=%d", req.TelegramID)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":          true,
		"telegram_id": req.TelegramID,
		"username":    "", // username не доступен без отдельного запроса к Telegram API
	})
}

// GET /health
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "ok",
		"server": "auth-otp",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func readJSON(r *http.Request, v interface{}) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
	if err != nil {
		return err
	}
	defer r.Body.Close()
	return json.Unmarshal(body, v)
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]interface{}{"ok": false, "error": msg})
}

// maxChatIDFrom возвращает chatID для отправки сообщений в MAX
func maxChatIDFrom(user *models.User) (int64, error) {
	raw := user.GetMaxChatID()
	if raw == "" {
		return 0, fmt.Errorf("MaxChatID пустой")
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id == 0 {
		return 0, fmt.Errorf("невалидный MaxChatID: %s", raw)
	}
	return id, nil
}
