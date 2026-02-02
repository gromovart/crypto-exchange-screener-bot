// internal/delivery/telegram/app/bot/handlers/events/payment/precheckout/interface.go
package pre_checkout

import (
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	"crypto-exchange-screener-bot/internal/delivery/telegram/services/payment"
)

// PaymentPlanHandler интерфейс для обработчиков платежных событий
type PaymentPlanHandler interface {
	handlers.Handler
}

// NewHandler создает новый обработчик pre_checkout_query
func NewHandler(paymentService payment.Service) PaymentPlanHandler {
	return &preCheckoutHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "pre_checkout_handler",
			Command: "pre_checkout_query",
			Type:    handlers.TypeMessage, // Используем TypeMessage для событий
		},
		paymentService: paymentService,
	}
}
