// internal/delivery/max/bot/middleware/auth.go
package middleware

import (
	"fmt"
	"strconv"
	"time"

	"crypto-exchange-screener-bot/internal/core/domain/users"
	maxpkg "crypto-exchange-screener-bot/internal/delivery/max"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/pkg/logger"
)

// AuthMiddleware — аутентификация пользователей MAX бота
type AuthMiddleware struct {
	userService *users.Service
}

// NewAuthMiddleware создаёт middleware аутентификации
func NewAuthMiddleware(userService *users.Service) *AuthMiddleware {
	return &AuthMiddleware{userService: userService}
}

// ProcessUpdate обрабатывает входящий Update и формирует HandlerParams
func (m *AuthMiddleware) ProcessUpdate(upd maxpkg.Update) (handlers.HandlerParams, error) {
	if m.userService == nil {
		return handlers.HandlerParams{}, fmt.Errorf("MAX Auth: userService недоступен")
	}

	var (
		userID     int64
		username   string
		firstName  string
		lastName   string
		chatID     int64
		text       string
		data       string
		updateID   string
		callbackID string
		messageID  string
	)

	updateID = strconv.FormatInt(upd.Timestamp, 10)

	switch upd.UpdateType {
	case "message_created":
		msg := upd.Message
		if msg == nil {
			return handlers.HandlerParams{}, fmt.Errorf("MAX Auth: message_created без message")
		}
		userID = msg.Sender.UserID
		username = msg.Sender.Username
		firstName = msg.Sender.FirstName
		lastName = msg.Sender.LastName
		chatID = msg.Recipient.ChatID
		if chatID == 0 {
			chatID = userID
		}
		text = msg.Body.Text
		messageID = msg.Body.Mid
		logger.Info("🔍 MAX Auth: message от user=%d chat=%d text=%q", userID, chatID, text)

	case "message_callback":
		cb := upd.Callback
		if cb == nil {
			return handlers.HandlerParams{}, fmt.Errorf("MAX Auth: message_callback без callback")
		}
		userID = cb.User.UserID
		username = cb.User.Username
		firstName = cb.User.FirstName
		lastName = cb.User.LastName
		data = cb.Payload
		callbackID = cb.CallbackID
		if cb.Message != nil {
			chatID = cb.Message.Recipient.ChatID
			messageID = cb.Message.Body.Mid
		}
		if chatID == 0 {
			chatID = userID
		}
		logger.Info("🔍 MAX Auth: callback от user=%d chat=%d payload=%q", userID, chatID, data)

	case "bot_started":
		if upd.User == nil {
			return handlers.HandlerParams{}, fmt.Errorf("MAX Auth: bot_started без user")
		}
		userID = upd.User.UserID
		username = upd.User.Username
		firstName = upd.User.FirstName
		lastName = upd.User.LastName
		chatID = upd.ChatID
		if chatID == 0 {
			chatID = userID
		}
		text = "/start"
		logger.Info("🔍 MAX Auth: bot_started от user=%d chat=%d", userID, chatID)

	default:
		return handlers.HandlerParams{}, fmt.Errorf("MAX Auth: неизвестный update_type: %s", upd.UpdateType)
	}

	// Получаем или создаём пользователя (тот же user store что и у Telegram)
	user, err := m.userService.GetOrCreateUser(userID, username, firstName, lastName)
	if err != nil {
		return handlers.HandlerParams{}, fmt.Errorf("MAX Auth: GetOrCreateUser: %w", err)
	}

	if !user.IsActive {
		return handlers.HandlerParams{}, fmt.Errorf("MAX Auth: аккаунт деактивирован")
	}

	// Обновляем LastLoginAt
	user.LastLoginAt = time.Now()
	if err := m.userService.UpdateUser(user); err != nil {
		logger.Warn("⚠️ MAX Auth: UpdateUser LastLoginAt: %v", err)
	}

	// Сохраняем ChatID если не сохранён (только для message_created/bot_started, где chatID точный)
	if user.ChatID == "" && upd.UpdateType != "message_callback" {
		user.ChatID = strconv.FormatInt(chatID, 10)
		if err := m.userService.UpdateUser(user); err != nil {
			logger.Warn("⚠️ MAX Auth: UpdateUser ChatID: %v", err)
		}
	}

	// Для message_callback: MAX API не возвращает chat_id напрямую, если cb.Message == nil.
	// Используем сохранённый ChatID пользователя (был сохранён при /start или message_created).
	if upd.UpdateType == "message_callback" && chatID == userID && user.ChatID != "" {
		if storedChatID, err := strconv.ParseInt(user.ChatID, 10, 64); err == nil && storedChatID != 0 {
			chatID = storedChatID
			logger.Debug("🔑 MAX Auth: callback chatID из профиля: %d", chatID)
		}
	}

	return handlers.HandlerParams{
		User:       user,
		ChatID:     chatID,
		Text:       text,
		Data:       data,
		UpdateID:   updateID,
		CallbackID: callbackID,
		MessageID:  messageID,
	}, nil
}
