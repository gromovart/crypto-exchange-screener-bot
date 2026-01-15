package reset_settings

import (
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
)

// resetSettingsHandler реализация обработчика сброса настроек
type resetSettingsHandler struct {
	*base.BaseHandler
}

// NewHandler создает новый обработчик сброса настроек
func NewHandler() handlers.Handler {
	return &resetSettingsHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "reset_settings_handler",
			Command: constants.CallbackResetSettings,
			Type:    handlers.TypeCallback,
		},
	}
}

// Execute выполняет обработку callback сброса настроек
func (h *resetSettingsHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	// TODO: Реализовать логику сброса настроек
	return handlers.HandlerResult{
		Message: "⚙️ *Сброс настроек*\n\nЭта функция в разработке.",
		Keyboard: map[string]interface{}{
			"inline_keyboard": [][]map[string]string{
				{
					{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackSettingsMain},
				},
			},
		},
	}, nil
}
