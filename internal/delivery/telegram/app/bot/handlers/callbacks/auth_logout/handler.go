package auth_logout

import (
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
)

// authLogoutHandler реализация обработчика выхода из системы
type authLogoutHandler struct {
	*base.BaseHandler
}

// NewHandler создает новый обработчик выхода из системы
func NewHandler() handlers.Handler {
	return &authLogoutHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "auth_logout_handler",
			Command: constants.CallbackAuthLogout,
			Type:    handlers.TypeCallback,
		},
	}
}

// Execute выполняет обработку callback выхода из системы
func (h *authLogoutHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	// TODO: Реализовать логику выхода из системы
	return handlers.HandlerResult{
		Message: "👋 *Выход из системы*\n\nЭта функция в разработке.",
		Keyboard: map[string]interface{}{
			"inline_keyboard": [][]map[string]string{
				{
					{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackSettingsMain},
				},
			},
		},
	}, nil
}
