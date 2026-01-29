// internal/delivery/telegram/app/bot/handlers/callbacks/payment_confirm/handler.go
package payment_confirm

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
)

// paymentConfirmHandler обработчик подтверждения платежа
type paymentConfirmHandler struct {
	*base.BaseHandler
}

// NewHandler создает новый обработчик подтверждения платежа
func NewHandler() handlers.Handler {
	return &paymentConfirmHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "payment_confirm_handler",
			Command: constants.PaymentConstants.CallbackPaymentConfirm,
			Type:    handlers.TypeCallback,
		},
	}
}

// Execute выполняет обработку подтверждения платежа
func (h *paymentConfirmHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	// Извлекаем ID плана из callback_data (формат: payment_confirm:basic)
	planID := h.extractPlanID(params.Data)
	if planID == "" {
		return handlers.HandlerResult{}, fmt.Errorf("неверный формат callback: %s", params.Data)
	}

	// Получаем информацию о плане
	plan := h.getPlanByID(planID)
	if plan == nil {
		return handlers.HandlerResult{}, fmt.Errorf("план не найден: %s", planID)
	}

	// Создаем инвойс и ссылку для оплаты
	invoiceLink := h.createInvoiceLink(params.User.ID, plan)

	// Сообщение с инструкцией по оплате
	message := h.createPaymentMessage(plan, invoiceLink)
	keyboard := h.createPaymentKeyboard(planID, invoiceLink)

	return handlers.HandlerResult{
		Message:  message,
		Keyboard: keyboard,
		Metadata: map[string]interface{}{
			"plan_id":      planID,
			"user_id":      params.User.ID,
			"invoice_link": invoiceLink,
			"stars_amount": h.calculateStars(plan.PriceCents),
		},
	}, nil
}

// extractPlanID извлекает ID плана из callback_data
func (h *paymentConfirmHandler) extractPlanID(callbackData string) string {
	prefix := constants.PaymentConstants.CallbackPaymentConfirm
	if len(callbackData) <= len(prefix) {
		return ""
	}
	return callbackData[len(prefix):]
}

// getPlanByID возвращает план по ID
func (h *paymentConfirmHandler) getPlanByID(planID string) *SubscriptionPlan {
	plans := map[string]*SubscriptionPlan{
		"basic": {
			ID:         "basic",
			Name:       "📱 Basic",
			PriceCents: 299,
		},
		"pro": {
			ID:         "pro",
			Name:       "🚀 Pro",
			PriceCents: 999,
		},
		"enterprise": {
			ID:         "enterprise",
			Name:       "🏢 Enterprise",
			PriceCents: 2499,
		},
	}
	return plans[planID]
}

// createInvoiceLink создает ссылку для оплаты (заглушка)
func (h *paymentConfirmHandler) createInvoiceLink(userID int, plan *SubscriptionPlan) string {
	// TODO: Интеграция с Telegram Stars API
	// Временная заглушка
	return fmt.Sprintf("https://t.me/bot?start=pay_%d_%s", userID, plan.ID)
}

// createPaymentMessage создает сообщение с инструкцией по оплате
func (h *paymentConfirmHandler) createPaymentMessage(plan *SubscriptionPlan, invoiceLink string) string {
	starsAmount := h.calculateStars(plan.PriceCents)
	usdPrice := float64(plan.PriceCents) / 100

	message := "💳 *Оплата через Telegram Stars*\n\n"
	message += fmt.Sprintf("План: *%s*\n", plan.Name)
	message += fmt.Sprintf("Сумма: *%d Stars* ($%.2f)\n\n", starsAmount, usdPrice)

	message += "📋 *Как оплатить:*\n"
	message += "1. Убедитесь, что у вас есть Stars в @wallet\n"
	message += "2. Нажмите кнопку '💳 Оплатить сейчас'\n"
	message += "3. Подтвердите платеж в открывшемся окне\n"
	message += "4. После успешной оплаты вы получите уведомление\n\n"

	message += "🔄 *После оплаты:*\n"
	message += "• Подписка активируется автоматически\n"
	message += "• Вы получите подтверждение в этот чат\n"
	message += "• Все функции плана будут доступны сразу\n\n"

	message += "❓ *Проблемы с оплатой?*\n"
	message += "Напишите в поддержку через /help"

	return message
}

// createPaymentKeyboard создает клавиатуру для оплаты
func (h *paymentConfirmHandler) createPaymentKeyboard(planID, invoiceLink string) interface{} {
	return map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			{
				{"text": "💳 Оплатить сейчас", "url": invoiceLink},
			},
			{
				{"text": "🔄 Проверить статус", "callback_data": fmt.Sprintf("%s%s",
					constants.PaymentConstants.CallbackPaymentCheck, planID)},
			},
			{
				{"text": constants.PaymentButtonTexts.BackToPlans, "callback_data": constants.PaymentConstants.CommandBuy},
				{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackMenuMain},
			},
		},
	}
}

// calculateStars рассчитывает количество Stars с учетом комиссии
func (h *paymentConfirmHandler) calculateStars(usdCents int) int {
	baseStars := usdCents / 100
	if baseStars < 1 {
		baseStars = 1
	}
	commission := baseStars / 20 // 5%
	if commission < 1 {
		commission = 1
	}
	return baseStars + commission
}

// Вспомогательный тип
type SubscriptionPlan struct {
	ID         string
	Name       string
	PriceCents int
}
