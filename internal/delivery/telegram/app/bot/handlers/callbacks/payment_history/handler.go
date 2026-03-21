// internal/delivery/telegram/app/bot/handlers/callbacks/payment_history/handler.go
package payment_history

import (
	"context"
	"fmt"
	"strings"

	"crypto-exchange-screener-bot/internal/core/domain/subscription"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
)

// Dependencies зависимости хэндлера
type Dependencies struct {
	SubscriptionService *subscription.Service
}

type paymentHistoryHandler struct {
	*base.BaseHandler
	subscriptionService *subscription.Service
}

func NewHandler(deps ...Dependencies) handlers.Handler {
	var svc *subscription.Service
	if len(deps) > 0 {
		svc = deps[0].SubscriptionService
	}
	return &paymentHistoryHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "payment_history_handler",
			Command: constants.PaymentConstants.CallbackPaymentHistory,
			Type:    handlers.TypeCallback,
		},
		subscriptionService: svc,
	}
}

func (h *paymentHistoryHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	if h.subscriptionService == nil {
		return handlers.HandlerResult{
			Message:  "⚠️ История платежей временно недоступна.",
			Keyboard: h.backKeyboard(),
		}, nil
	}

	subs, err := h.subscriptionService.GetRepository().GetAllByUserID(context.Background(), params.User.ID)
	if err != nil {
		return handlers.HandlerResult{}, fmt.Errorf("ошибка получения истории: %w", err)
	}

	message := h.buildMessage(subs)
	return handlers.HandlerResult{
		Message:  message,
		Keyboard: h.backKeyboard(),
	}, nil
}

func (h *paymentHistoryHandler) buildMessage(subs []*models.UserSubscription) string {
	if len(subs) == 0 {
		return "📋 *История платежей*\n\nПлатежей пока нет."
	}

	var sb strings.Builder
	sb.WriteString("📋 *История платежей*\n\n")

	for i, sub := range subs {
		planName := sub.PlanName
		if planName == "" {
			planName = sub.PlanCode
		}
		if planName == "" {
			planName = "Неизвестный план"
		}

		status := h.formatStatus(sub.Status)
		date := sub.CreatedAt.Format("02.01.2006")

		sb.WriteString(fmt.Sprintf("%d. *%s*\n", i+1, planName))
		sb.WriteString(fmt.Sprintf("   📅 %s  %s\n", date, status))

		if amount := formatAmount(sub.Metadata); amount != "" {
			sb.WriteString(fmt.Sprintf("   💰 %s\n", amount))
		}

		if sub.CurrentPeriodEnd != nil {
			sb.WriteString(fmt.Sprintf("   ⏳ до %s\n", sub.CurrentPeriodEnd.Format("02.01.2006")))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func formatAmount(meta map[string]interface{}) string {
	if meta == nil {
		return ""
	}
	// Сумма от Т-Банк (в рублях)
	if rub, ok := meta["amount_rub"]; ok {
		switch v := rub.(type) {
		case float64:
			return fmt.Sprintf("%d ₽", int64(v))
		case int64:
			return fmt.Sprintf("%d ₽", v)
		case int:
			return fmt.Sprintf("%d ₽", v)
		}
	}
	// Fallback: копейки → рубли
	if kopecks, ok := meta["amount_kopecks"]; ok {
		switch v := kopecks.(type) {
		case float64:
			return fmt.Sprintf("%d ₽", int64(v)/100)
		case int64:
			return fmt.Sprintf("%d ₽", v/100)
		}
	}
	return ""
}

func (h *paymentHistoryHandler) formatStatus(status string) string {
	switch status {
	case models.StatusActive:
		return "✅ Активна"
	case models.StatusTrialing:
		return "🔄 Пробная"
	case models.StatusCanceled:
		return "❌ Отменена"
	case models.StatusExpired:
		return "⌛ Истекла"
	case models.StatusPastDue:
		return "⚠️ Просрочена"
	default:
		return status
	}
}

func (h *paymentHistoryHandler) backKeyboard() interface{} {
	return map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			{{"text": constants.ButtonTexts.Back, "callback_data": constants.PaymentConstants.CommandBuy}},
		},
	}
}
