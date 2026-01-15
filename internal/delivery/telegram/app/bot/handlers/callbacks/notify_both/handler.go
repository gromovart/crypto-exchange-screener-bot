package notify_both

import (
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
)

// notifyBothHandler реализация обработчика всех уведомлений
type notifyBothHandler struct {
	*base.BaseHandler
}

// NewHandler создает новый обработчик всех уведомлений
func NewHandler() handlers.Handler {
	return &notifyBothHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "notify_both_handler",
			Command: constants.CallbackNotifyBoth,
			Type:    handlers.TypeCallback,
		},
	}
}

// Execute выполняет обработку callback всех уведомлений
func (h *notifyBothHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	// TODO: Реализовать логику всех уведомлений
	return handlers.HandlerResult{
		Message: "📊 *Все уведомления*\n\nЭта функция в разработке.",
		Keyboard: map[string]interface{}{
			"inline_keyboard": [][]map[string]string{
				{
					{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackNotificationsMenu},
				},
			},
		},
	}, nil
}
