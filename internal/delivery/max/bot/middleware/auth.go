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
		userID    int64
		username  string
		firstName string
		lastName  string
		chatID    int64
		text      string
		data      string
		updateID  string
		callbackID string
		messageID  int64
	)

	updateID = strconv.FormatInt(upd.UpdateID, 10)

	switch {
	case upd.Message != nil:
		msg := upd.Message
		userID = msg.From.ID
		username = msg.From.Username
		firstName = msg.From.FirstName
		lastName = msg.From.LastName
		chatID = msg.Chat.ID
		text = msg.Text
		messageID = msg.MessageID
		logger.Info("🔍 MAX Auth: Message от user=%d chat=%d text=%q", userID, chatID, text)

	case upd.CallbackQuery != nil:
		cb := upd.CallbackQuery
		userID = cb.From.ID
		username = cb.From.Username
		firstName = cb.From.FirstName
		lastName = cb.From.LastName
		data = cb.Data
		callbackID = cb.ID
		if cb.Message != nil {
			chatID = cb.Message.Chat.ID
			messageID = cb.Message.MessageID
		} else {
			chatID = userID
		}
		logger.Info("🔍 MAX Auth: Callback от user=%d chat=%d data=%q", userID, chatID, data)

	default:
		return handlers.HandlerParams{}, fmt.Errorf("MAX Auth: неизвестный тип обновления")
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

	// Сохраняем ChatID если не сохранён
	if user.ChatID == "" {
		user.ChatID = strconv.FormatInt(chatID, 10)
		if err := m.userService.UpdateUser(user); err != nil {
			logger.Warn("⚠️ MAX Auth: UpdateUser ChatID: %v", err)
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
