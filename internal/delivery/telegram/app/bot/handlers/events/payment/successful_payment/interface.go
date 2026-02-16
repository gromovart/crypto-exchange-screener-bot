// internal/delivery/telegram/app/bot/handlers/events/payment/successful_payment/interface.go
package successful_payment

import (
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	"crypto-exchange-screener-bot/internal/delivery/telegram/services/payment"
)

// PaymentPlanHandler интерфейс для обработчиков платежных событий
type PaymentPlanHandler interface {
	handlers.Handler
}

// NewHandler создает новый обработчик successful_payment
func NewHandler(paymentService payment.Service) handlers.Handler {
	return &successfulPaymentHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "successful_payment_handler",
			Command: "successful_payment",
			Type:    handlers.TypeMessage,
		},
		paymentService: paymentService,
	}
}
