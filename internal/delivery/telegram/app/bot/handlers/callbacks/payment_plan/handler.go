// internal/delivery/telegram/app/bot/handlers/callbacks/payment_plan/handler.go
package payment_plan

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
)

// paymentPlanHandler обработчик выбора платежного плана
type paymentPlanHandler struct {
	*base.BaseHandler
}

// NewHandler создает новый обработчик выбора плана
func NewHandler() handlers.Handler {
	return &paymentPlanHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "payment_plan_handler",
			Command: constants.PaymentConstants.CallbackPaymentPlan,
			Type:    handlers.TypeCallback,
		},
	}
}

// Execute выполняет обработку выбора плана
func (h *paymentPlanHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	// Извлекаем ID плана из callback_data (формат: payment_plan:basic)
	planID := h.extractPlanID(params.Data)
	if planID == "" {
		return handlers.HandlerResult{}, fmt.Errorf("неверный формат callback: %s", params.Data)
	}

	// Получаем информацию о плане
	plan := h.getPlanByID(planID)
	if plan == nil {
		return handlers.HandlerResult{}, fmt.Errorf("план не найден: %s", planID)
	}

	// Создаем сообщение с подтверждением
	message := h.createConfirmationMessage(plan)
	keyboard := h.createConfirmationKeyboard(planID)

	return handlers.HandlerResult{
		Message:  message,
		Keyboard: keyboard,
		Metadata: map[string]interface{}{
			"plan_id":      planID,
			"user_id":      params.User.ID,
			"stars_amount": h.calculateStars(plan.PriceCents),
		},
	}, nil
}

// extractPlanID извлекает ID плана из callback_data
func (h *paymentPlanHandler) extractPlanID(callbackData string) string {
	// Формат: payment_plan:basic
	prefix := constants.PaymentConstants.CallbackPaymentPlan
	if len(callbackData) <= len(prefix) {
		return ""
	}
	return callbackData[len(prefix):]
}

// getPlanByID возвращает план по ID
func (h *paymentPlanHandler) getPlanByID(planID string) *SubscriptionPlan {
	plans := map[string]*SubscriptionPlan{
		"basic": {
			ID:          "basic",
			Name:        "📱 Доступ на 1 месяц",
			Description: "Идеально для начала",
			PriceCents:  1500,
			Features: []string{
				"✅ Неограниченные сигналы",
				"✅ Все виды уведомлений",
			},
		},
		"pro": {
			ID:          "pro",
			Name:        "🚀 Доступ на 3 месяца",
			Description: "Для активных трейдеров",
			PriceCents:  3000,
			Features: []string{
				"✅ Неограниченные сигналы",
				"✅ Все виды уведомлений",
				"✅ Приоритетная поддержка",
			},
		},
		"enterprise": {
			ID:          "enterprise",
			Name:        "🏢 Доступ на 12 месяцев",
			Description: "Максимальные возможности",
			PriceCents:  7500,
			Features: []string{
				"✅ Неограниченные сигналы",
				"✅ Все виды уведомлений",
				"✅ Кастомные настройки",
				"✅ Приоритетная поддержка 24/7",
			},
		},
	}

	return plans[planID]
}

// createConfirmationMessage создает сообщение с подтверждением
func (h *paymentPlanHandler) createConfirmationMessage(plan *SubscriptionPlan) string {
	starsAmount := h.calculateStars(plan.PriceCents)
	usdPrice := float64(plan.PriceCents) / 100

	message := fmt.Sprintf("✅ *Подтверждение выбора*\n\n")
	message += fmt.Sprintf("Вы выбрали план: *%s*\n\n", plan.Name)
	message += fmt.Sprintf("💰 Стоимость: *%d Stars* ($%.2f)\n", starsAmount, usdPrice)
	message += fmt.Sprintf("📋 Описание: %s\n\n", plan.Description)
	message += "🔍 *Включено:*\n"
	for i, feature := range plan.Features {
		message += fmt.Sprintf("%d. %s\n", i+1, feature)
	}
	message += "\nℹ️ *После оплаты:*\n"
	message += "• Подписка активируется автоматически\n"
	message += "• Вы получите уведомление в Telegram\n"
	message += "• Доступ к функциям откроется сразу\n\n"
	message += "Для оплаты нажмите кнопку ниже:"

	return message
}

// createConfirmationKeyboard создает клавиатуру подтверждения
func (h *paymentPlanHandler) createConfirmationKeyboard(planID string) interface{} {
	callbackConfirm := fmt.Sprintf("%s%s",
		constants.PaymentConstants.CallbackPaymentConfirm, planID)

	return map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			{
				{"text": constants.PaymentButtonTexts.PayNow, "callback_data": callbackConfirm},
			},
			{
				{"text": "📜 Условия использования", "callback_data": "/terms"}, // ⭐ ДОБАВИТЬ
				{"text": constants.PaymentButtonTexts.BackToPlans, "callback_data": constants.PaymentConstants.CommandBuy},
			},
			{
				{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackMenuMain},
			},
		},
	}
}

// calculateStars рассчитывает количество Stars с учетом комиссии
func (h *paymentPlanHandler) calculateStars(usdCents int) int {
	return usdCents / 3 // 1500/3 = 500, 3000/3 = 1000, 7500/3 = 2500
}

// Вспомогательный тип
type SubscriptionPlan struct {
	ID          string
	Name        string
	Description string
	PriceCents  int
	Features    []string
}
