package notify_growth_only

import (
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
)

// notifyGrowthOnlyHandler реализация обработчика уведомлений о росте
type notifyGrowthOnlyHandler struct {
	*base.BaseHandler
}

// NewHandler создает новый обработчик уведомлений о росте
func NewHandler() handlers.Handler {
	return &notifyGrowthOnlyHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "notify_growth_only_handler",
			Command: constants.CallbackNotifyGrowthOnly,
			Type:    handlers.TypeCallback,
		},
	}
}

// Execute выполняет обработку callback уведомлений о росте
func (h *notifyGrowthOnlyHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	// TODO: Реализовать логику уведомлений о росте
	return handlers.HandlerResult{
		Message: "📈 *Уведомления только о росте*\n\nЭта функция в разработке.",
		Keyboard: map[string]interface{}{
			"inline_keyboard": [][]map[string]string{
				{
					{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackNotificationsMenu},
				},
			},
		},
	}, nil
}
