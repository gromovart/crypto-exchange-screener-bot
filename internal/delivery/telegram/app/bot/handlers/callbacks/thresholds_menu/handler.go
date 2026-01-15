package thresholds_menu

import (
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
)

// thresholdsMenuHandler реализация обработчика меню порогов
type thresholdsMenuHandler struct {
	*base.BaseHandler
}

// NewHandler создает новый обработчик меню порогов
func NewHandler() handlers.Handler {
	return &thresholdsMenuHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "thresholds_menu_handler",
			Command: constants.CallbackThresholdsMenu,
			Type:    handlers.TypeCallback,
		},
	}
}

// Execute выполняет обработку callback меню порогов
func (h *thresholdsMenuHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	// TODO: Реализовать логику меню порогов
	return handlers.HandlerResult{
		Message: "📊 *Меню порогов*\n\nЭто меню в разработке.",
		Keyboard: map[string]interface{}{
			"inline_keyboard": [][]map[string]string{
				{
					{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackSettingsMain},
				},
			},
		},
	}, nil
}
