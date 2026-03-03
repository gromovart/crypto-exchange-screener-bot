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

		// Приоритет: chatID из сообщения → chatID из update → 0 (резолвим после GetOrCreateUser)
		if cb.Message != nil && cb.Message.Recipient.ChatID != 0 {
			chatID = cb.Message.Recipient.ChatID
		} else if upd.ChatID != 0 {
			chatID = upd.ChatID
		}
		// chatID == 0 → резолвим из user.ChatID в БД после GetOrCreateUser

		if cb.Message != nil {
			messageID = cb.Message.Body.Mid
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

	// Получаем или создаём пользователя по MAX user ID
	user, err := m.userService.GetOrCreateUserByMaxID(userID, username, firstName, lastName)
	if err != nil {
		return handlers.HandlerParams{}, fmt.Errorf("MAX Auth: GetOrCreateUserByMaxID: %w", err)
	}

	if !user.IsActive {
		return handlers.HandlerParams{}, fmt.Errorf("MAX Auth: аккаунт деактивирован")
	}

	// Если chatID не определён из update — берём MaxChatID (MAX dialog ID) из профиля.
	// ВАЖНО: нельзя использовать user.ChatID — после привязки TG это Telegram chat ID.
	if chatID == 0 && user.MaxChatID != "" {
		if stored, err := strconv.ParseInt(user.MaxChatID, 10, 64); err == nil && stored != 0 {
			chatID = stored
			logger.Info("🔍 MAX Auth: chatID=%d взят из MaxChatID для user=%d", chatID, userID)
		}
	}
	// Последний резерв — userID (вероятно не сработает, но лучше, чем 0)
	if chatID == 0 {
		chatID = userID
		logger.Warn("⚠️ MAX Auth: chatID не определён, используем userID=%d как fallback", userID)
	}

	// Обновляем LastLoginAt
	user.LastLoginAt = time.Now()

	// Сохраняем/обновляем chatID
	if chatID != 0 {
		chatIDStr := strconv.FormatInt(chatID, 10)
		// MaxChatID — всегда обновляем для MAX-пользователей
		if user.MaxChatID != chatIDStr {
			user.MaxChatID = chatIDStr
		}
		// ChatID обновляем только для MAX-only пользователей (без привязанного TG)
		// Для привязанных пользователей ChatID = TG chat ID, менять нельзя
		if user.IsMaxOnlyUser() {
			if chatID != userID {
				storedChatID, _ := strconv.ParseInt(user.ChatID, 10, 64)
				if storedChatID != chatID {
					user.ChatID = chatIDStr
				}
			} else if user.ChatID == "" && upd.UpdateType != "message_callback" {
				user.ChatID = chatIDStr
			}
		}
	}

	if err := m.userService.UpdateUser(user); err != nil {
		logger.Warn("⚠️ MAX Auth: UpdateUser: %v", err)
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
