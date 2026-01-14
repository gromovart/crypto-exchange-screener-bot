// internal/delivery/telegram/app/bot/handlers/start/handler.go
package start

import (
	"fmt"
	"time"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
)

// startHandlerImpl реализация StartHandler
type startHandlerImpl struct {
	*base.BaseHandler
}

// NewHandler создает новый хэндлер команды /start
func NewHandler() handlers.Handler {
	return &startHandlerImpl{
		BaseHandler: &base.BaseHandler{
			Name:    "start_handler",
			Command: "start",
			Type:    handlers.TypeCommand,
		},
	}
}

// Execute выполняет обработку команды /start
func (h *startHandlerImpl) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	// Бизнес-логика команды /start
	message := h.formatWelcomeMessage(params.User)
	keyboard := h.createWelcomeKeyboard()

	return handlers.HandlerResult{
		Message:  message,
		Keyboard: keyboard,
		Metadata: map[string]interface{}{
			"user_id":    params.User.ID,
			"first_name": params.User.FirstName,
			"timestamp":  time.Now(),
		},
	}, nil
}

// formatWelcomeMessage форматирует приветственное сообщение
func (h *startHandlerImpl) formatWelcomeMessage(user *models.User) string {
	firstName := user.FirstName
	if firstName == "" {
		firstName = "Гость"
	}

	username := user.Username
	if username == "" {
		username = "не указан"
	} else {
		username = "@" + username
	}

	return fmt.Sprintf(
		"👋 *Добро пожаловать, %s!*\n\n"+
			"🚀 *Crypto Exchange Screener Bot*\n\n"+
			"✅ Ваш аккаунт: %s\n"+
			"👤 Имя: %s\n"+
			"⭐ Роль: %s\n"+
			"📅 Дата регистрации: %s\n\n"+
			"Бот анализирует рынок криптовалют и отправляет уведомления о сильных движениях.\n\n"+
			"Используйте меню ниже для управления ботом:",
		firstName,
		username,
		firstName,
		h.GetRoleDisplay(user.Role),
		user.CreatedAt.Format("02.01.2006"),
	)
}

// createWelcomeKeyboard создает клавиатуру для приветствия
func (h *startHandlerImpl) createWelcomeKeyboard() interface{} {
	return map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			{
				{"text": constants.MenuButtonTexts.Profile, "callback_data": constants.CallbackProfileMain},
				{"text": constants.ButtonTexts.Settings, "callback_data": constants.CallbackSettingsMain},
			},
			{
				{"text": constants.MenuButtonTexts.Notifications, "callback_data": constants.CallbackNotificationsMenu},
				{"text": constants.ButtonTexts.Help, "callback_data": constants.CallbackHelp},
			},
		},
	}
}
