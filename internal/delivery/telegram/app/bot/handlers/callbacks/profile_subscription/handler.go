package profile_subscription

import (
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
)

// profileSubscriptionHandler реализация обработчика подписки профиля
type profileSubscriptionHandler struct {
	*base.BaseHandler
}

// NewHandler создает новый обработчик подписки профиля
func NewHandler() handlers.Handler {
	return &profileSubscriptionHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "profile_subscription_handler",
			Command: constants.CallbackProfileSubscription,
			Type:    handlers.TypeCallback,
		},
	}
}

// Execute выполняет обработку callback подписки профиля
func (h *profileSubscriptionHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	// TODO: Реализовать логику подписки профиля
	return handlers.HandlerResult{
		Message: "💎 *Подписка профиля*\n\nЭта функция в разработке.",
		Keyboard: map[string]interface{}{
			"inline_keyboard": [][]map[string]string{
				{
					{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackProfileMain},
				},
			},
		},
	}, nil
}
