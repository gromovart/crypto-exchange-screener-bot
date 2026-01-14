// internal/delivery/telegram/app/bot/handlers/commands/settings/handler.go
package settings

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
)

// settingsCommandHandler реализация обработчика команды /settings
type settingsCommandHandler struct {
	*base.BaseHandler
}

// NewHandler создает новый обработчик команды /settings
func NewHandler() handlers.Handler {
	return &settingsCommandHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "settings_command_handler",
			Command: "settings",
			Type:    handlers.TypeCommand,
		},
	}
}

// Execute выполняет обработку команды /settings
func (h *settingsCommandHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	// Проверяем авторизацию пользователя
	isAuth := params.User != nil && params.User.ID > 0

	// Создаем адаптивное меню
	message := h.createSettingsMessage(isAuth, params.User)
	keyboard := h.createSettingsKeyboard(isAuth)

	return handlers.HandlerResult{
		Message:  message,
		Keyboard: keyboard,
		Metadata: map[string]interface{}{
			"is_authenticated": isAuth,
			"user_id":          params.User.ID,
		},
	}, nil
}

// createSettingsMessage создает сообщение для команды /settings
func (h *settingsCommandHandler) createSettingsMessage(isAuth bool, user *models.User) string {
	if isAuth {
		firstName := user.FirstName
		if firstName == "" {
			firstName = "Гость"
		}

		return fmt.Sprintf(
			"%s\n\n"+
				"*Привет, %s!* 👋\n\n"+
				"Выберите раздел для управления ботом:",
			constants.AuthButtonTexts.Settings,
			firstName,
		)
	}

	return fmt.Sprintf(
		"%s\n\n"+
			"*Добро пожаловать!* 👋\n\n"+
			"Вы можете использовать основные функции бота.\n"+
			"Для доступа ко всем функциям выполните авторизацию.\n\n"+
			"Выберите раздел:",
		constants.AuthButtonTexts.Settings,
	)
}

// createSettingsKeyboard создает клавиатуру для команды /settings
func (h *settingsCommandHandler) createSettingsKeyboard(isAuth bool) interface{} {
	if isAuth {
		// Меню для авторизованных пользователей
		return map[string]interface{}{
			"inline_keyboard": [][]map[string]string{
				{
					{"text": constants.MenuButtonTexts.Profile, "callback_data": constants.CallbackProfileMain},
					{"text": constants.AuthButtonTexts.Settings, "callback_data": constants.CallbackSettingsMain},
				},
				{
					{"text": constants.MenuButtonTexts.Notifications, "callback_data": constants.CallbackNotificationsMenu},
					{"text": constants.MenuButtonTexts.Signals, "callback_data": constants.CallbackSignalsMenu},
				},
				{
					{"text": constants.MenuButtonTexts.Periods, "callback_data": constants.CallbackPeriodsMenu},
					{"text": constants.ButtonTexts.Status, "callback_data": constants.CallbackStats},
				},
				{
					{"text": constants.MenuButtonTexts.Reset, "callback_data": constants.CallbackResetMenu},
					{"text": constants.ButtonTexts.Help, "callback_data": constants.CallbackHelp},
				},
			},
		}
	}

	// Меню для неавторизованных пользователей
	return map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			{
				{"text": constants.AuthButtonTexts.Settings, "callback_data": constants.CallbackSettingsMain},
				{"text": constants.MenuButtonTexts.Notifications, "callback_data": constants.CallbackNotificationsMenu},
			},
			{
				{"text": constants.MenuButtonTexts.Periods, "callback_data": constants.CallbackPeriodsMenu},
				{"text": constants.ButtonTexts.Status, "callback_data": constants.CallbackStats},
			},
			{
				{"text": constants.AuthButtonTexts.Login, "callback_data": constants.CallbackAuthLogin},
				{"text": constants.ButtonTexts.Help, "callback_data": constants.CallbackHelp},
			},
		},
	}
}
