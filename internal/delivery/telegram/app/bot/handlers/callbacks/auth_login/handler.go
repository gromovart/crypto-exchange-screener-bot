package auth_login

import (
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
)

// authLoginHandler реализация обработчика авторизации
type authLoginHandler struct {
	*base.BaseHandler
}

// NewHandler создает новый обработчик авторизации
func NewHandler() handlers.Handler {
	return &authLoginHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "auth_login_handler",
			Command: constants.CallbackAuthLogin,
			Type:    handlers.TypeCallback,
		},
	}
}

// Execute выполняет обработку callback авторизации
func (h *authLoginHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	// TODO: Реализовать логику авторизации
	return handlers.HandlerResult{
		Message: "🔑 *Авторизация*\n\nЭта функция в разработке.",
		Keyboard: map[string]interface{}{
			"inline_keyboard": [][]map[string]string{
				{
					{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackSettingsMain},
				},
			},
		},
	}, nil
}
