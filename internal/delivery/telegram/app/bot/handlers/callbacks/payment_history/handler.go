// internal/delivery/telegram/app/bot/handlers/callbacks/payment_history/handler.go
package payment_history

import (
	"context"
	"fmt"
	"strings"

	"crypto-exchange-screener-bot/internal/core/domain/payment"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
)

// Dependencies зависимости хэндлера
type Dependencies struct {
	PaymentCoreService *payment.PaymentService
}

type paymentHistoryHandler struct {
	*base.BaseHandler
	paymentCoreService *payment.PaymentService
}

func NewHandler(deps ...Dependencies) handlers.Handler {
	var svc *payment.PaymentService
	if len(deps) > 0 {
		svc = deps[0].PaymentCoreService
	}
	return &paymentHistoryHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "payment_history_handler",
			Command: constants.PaymentConstants.CallbackPaymentHistory,
			Type:    handlers.TypeCallback,
		},
		paymentCoreService: svc,
	}
}

func (h *paymentHistoryHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	if h.paymentCoreService == nil {
		return handlers.HandlerResult{
			Message:  "⚠️ История платежей временно недоступна.",
			Keyboard: h.backKeyboard(),
		}, nil
	}

	filter := models.NewPaymentFilter()
	filter.UserID = int64(params.User.ID)
	filter.Limit = 20

	payments, err := h.paymentCoreService.GetUserPayments(context.Background(), int64(params.User.ID), filter)
	if err != nil {
		return handlers.HandlerResult{}, fmt.Errorf("ошибка получения истории платежей: %w", err)
	}

	message := h.buildMessage(payments)
	return handlers.HandlerResult{
		Message:  message,
		Keyboard: h.backKeyboard(),
	}, nil
}

func (h *paymentHistoryHandler) buildMessage(payments []*models.Payment) string {
	if len(payments) == 0 {
		return "📋 *История платежей*\n\nПлатежей пока нет."
	}

	var sb strings.Builder
	sb.WriteString("📋 *История платежей*\n\n")

	for i, p := range payments {
		date := p.CreatedAt.Format("02.01.2006")
		status := formatStatus(p.Status)
		provider := formatProvider(p.Provider)
		amount := formatAmount(p)

		sb.WriteString(fmt.Sprintf("%d. %s  %s\n", i+1, provider, status))
		sb.WriteString(fmt.Sprintf("   📅 %s  💰 %s\n", date, amount))
		if p.Description != "" {
			sb.WriteString(fmt.Sprintf("   %s\n", p.Description))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func formatAmount(p *models.Payment) string {
	switch p.Currency {
	case models.CurrencyRUB:
		return fmt.Sprintf("%d ₽", int64(p.Amount*90)) // amount хранится в USD, конвертируем обратно
	case models.CurrencyUSD:
		return fmt.Sprintf("$%.2f", p.Amount)
	default:
		if p.FiatAmount > 0 {
			return fmt.Sprintf("%d ₽", int64(p.FiatAmount)/10)
		}
		return fmt.Sprintf("%.2f %s", p.Amount, p.Currency)
	}
}

func formatStatus(status models.PaymentStatus) string {
	switch status {
	case models.PaymentStatusCompleted:
		return "✅"
	case models.PaymentStatusPending, models.PaymentStatusProcessing:
		return "⏳"
	case models.PaymentStatusFailed:
		return "❌"
	case models.PaymentStatusRefunded:
		return "↩️"
	case models.PaymentStatusCancelled:
		return "🚫"
	default:
		return "❓"
	}
}

func formatProvider(provider string) string {
	switch provider {
	case "tbank":
		return "💳 Т-Банк"
	case "stars", "telegram":
		return "⭐ Telegram Stars"
	default:
		return "💳 " + provider
	}
}

func (h *paymentHistoryHandler) backKeyboard() interface{} {
	return map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			{{"text": constants.ButtonTexts.Back, "callback_data": constants.PaymentConstants.CommandBuy}},
		},
	}
}
