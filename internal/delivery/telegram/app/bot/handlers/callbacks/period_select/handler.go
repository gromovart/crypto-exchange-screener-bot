package period_select

import (
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
)

// periodSelectHandler реализация обработчика выбора периода
type periodSelectHandler struct {
	*base.BaseHandler
}

// NewHandler создает новый обработчик выбора периода
func NewHandler() handlers.Handler {
	return &periodSelectHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "period_select_handler",
			Command: "period_select",
			Type:    handlers.TypeCallback,
		},
	}
}

// Execute выполняет обработку callback выбора периода
func (h *periodSelectHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	// TODO: Реализовать логику выбора периода
	return handlers.HandlerResult{
		Message: "⏱️ *Выбор периода*\n\nЭта функция в разработке.",
		Keyboard: map[string]interface{}{
			"inline_keyboard": [][]map[string]string{
				{
					{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackPeriodsMenu},
				},
			},
		},
	}, nil
}
