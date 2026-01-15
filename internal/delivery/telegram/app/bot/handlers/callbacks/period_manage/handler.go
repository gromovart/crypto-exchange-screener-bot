package period_manage

import (
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
)

// periodManageHandler реализация обработчика управления периодами
type periodManageHandler struct {
	*base.BaseHandler
}

// NewHandler создает новый обработчик управления периодами
func NewHandler() handlers.Handler {
	return &periodManageHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "period_manage_handler",
			Command: constants.CallbackPeriodManage,
			Type:    handlers.TypeCallback,
		},
	}
}

// Execute выполняет обработку callback управления периодами
func (h *periodManageHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	// TODO: Реализовать логику управления периодами
	return handlers.HandlerResult{
		Message: "⏱️ *Управление периодами*\n\nЭта функция в разработке.",
		Keyboard: map[string]interface{}{
			"inline_keyboard": [][]map[string]string{
				{
					{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackPeriodsMenu},
				},
			},
		},
	}, nil
}
