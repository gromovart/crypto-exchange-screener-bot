package reset_menu

import (
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
)

// resetMenuHandler реализация обработчика меню сброса
type resetMenuHandler struct {
	*base.BaseHandler
}

// NewHandler создает новый обработчик меню сброса
func NewHandler() handlers.Handler {
	return &resetMenuHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "reset_menu_handler",
			Command: constants.CallbackResetMenu,
			Type:    handlers.TypeCallback,
		},
	}
}

// Execute выполняет обработку callback меню сброса
func (h *resetMenuHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	// TODO: Реализовать логику меню сброса
	return handlers.HandlerResult{
		Message: "🔄 *Меню сброса*\n\nЭто меню в разработке.",
		Keyboard: map[string]interface{}{
			"inline_keyboard": [][]map[string]string{
				{
					{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackMenuMain},
				},
			},
		},
	}, nil
}
