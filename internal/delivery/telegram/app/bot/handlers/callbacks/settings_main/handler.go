// internal/delivery/telegram/app/bot/handlers/callbacks/settings_main/handler.go
package settings_main

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
)

// settingsMainHandler реализация обработчика главного меню
type settingsMainHandler struct {
	*base.BaseHandler
}

// NewHandler создает новый обработчик главного меню
func NewHandler() handlers.Handler {
	return &settingsMainHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "settings_main_handler",
			Command: constants.CallbackSettingsMain,
			Type:    handlers.TypeCallback,
		},
	}
}

// Execute выполняет обработку callback главного меню
func (h *settingsMainHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	// Проверяем авторизацию пользователя
	isAuth := params.User != nil && params.User.ID > 0

	// Создаем адаптивное меню
	message := h.createMainMenuMessage(isAuth, params.User)
	keyboard := h.createMainMenuKeyboard(isAuth)

	return handlers.HandlerResult{
		Message:  message,
		Keyboard: keyboard,
		Metadata: map[string]interface{}{
			"is_authenticated": isAuth,
			"user_id":          params.User.ID,
		},
	}, nil
}

// createMainMenuMessage создает сообщение для главного меню
func (h *settingsMainHandler) createMainMenuMessage(isAuth bool, user *models.User) string {
	if isAuth {
		firstName := user.FirstName
		if firstName == "" {
			firstName = "Гость"
		}

		return fmt.Sprintf(
			"%s\n\n"+
				"*Привет, %s!* 👋\n\n"+
				"Выберите раздел для управления ботом:",
			constants.MenuButtonTexts.MainMenu,
			firstName,
		)
	}

	return fmt.Sprintf(
		"%s\n\n"+
			"*Добро пожаловать!* 👋\n\n"+
			"Вы можете использовать основные функции бота.\n"+
			"Для доступа ко всем функциям выполните авторизацию.\n\n"+
			"Выберите раздел:",
		constants.MenuButtonTexts.MainMenu,
	)
}

// createMainMenuKeyboard создает клавиатуру для главного меню
func (h *settingsMainHandler) createMainMenuKeyboard(isAuth bool) interface{} {
	if isAuth {
		// Меню для авторизованных пользователей
		return map[string]interface{}{
			"inline_keyboard": [][]map[string]string{
				{
					{"text": constants.MenuButtonTexts.Profile, "callback_data": constants.CallbackProfileMain},
					{"text": constants.ButtonTexts.Settings, "callback_data": constants.CallbackSettingsMain},
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
				{"text": constants.ButtonTexts.Settings, "callback_data": constants.CallbackSettingsMain},
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
