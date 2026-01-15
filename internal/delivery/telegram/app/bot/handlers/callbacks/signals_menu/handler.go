package signals_menu

import (
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
)

// signalsMenuHandler реализация обработчика меню сигналов
type signalsMenuHandler struct {
	*base.BaseHandler
}

// NewHandler создает новый обработчик меню сигналов
func NewHandler() handlers.Handler {
	return &signalsMenuHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "signals_menu_handler",
			Command: constants.CallbackSignalsMenu,
			Type:    handlers.TypeCallback,
		},
	}
}

// Execute выполняет обработку callback меню сигналов
func (h *signalsMenuHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	// TODO: Реализовать логику меню сигналов
	return handlers.HandlerResult{
		Message: "📈 *Меню сигналов*\n\nЭто меню в разработке.",
		Keyboard: map[string]interface{}{
			"inline_keyboard": [][]map[string]string{
				{
					{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackMenuMain},
				},
			},
		},
	}, nil
}
