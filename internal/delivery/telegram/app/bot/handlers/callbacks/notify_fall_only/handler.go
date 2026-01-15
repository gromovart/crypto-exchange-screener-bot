package notify_fall_only

import (
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
)

// notifyFallOnlyHandler реализация обработчика уведомлений о падении
type notifyFallOnlyHandler struct {
	*base.BaseHandler
}

// NewHandler создает новый обработчик уведомлений о падении
func NewHandler() handlers.Handler {
	return &notifyFallOnlyHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "notify_fall_only_handler",
			Command: constants.CallbackNotifyFallOnly,
			Type:    handlers.TypeCallback,
		},
	}
}

// Execute выполняет обработку callback уведомлений о падении
func (h *notifyFallOnlyHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	// TODO: Реализовать логику уведомлений о падении
	return handlers.HandlerResult{
		Message: "📉 *Уведомления только о падении*\n\nЭта функция в разработке.",
		Keyboard: map[string]interface{}{
			"inline_keyboard": [][]map[string]string{
				{
					{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackNotificationsMenu},
				},
			},
		},
	}, nil
}
