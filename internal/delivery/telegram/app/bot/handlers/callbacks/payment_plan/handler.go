// internal/delivery/telegram/app/bot/handlers/callbacks/payment_plan/handler.go
package payment_plan

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
)

// Dependencies зависимости хэндлера
type Dependencies struct {
	IsDev        bool
	TBankEnabled bool
}

// paymentPlanHandler обработчик выбора платежного плана
type paymentPlanHandler struct {
	*base.BaseHandler
	isDev        bool
	tBankEnabled bool
}

// NewHandler создает новый обработчик выбора плана
func NewHandler(deps ...Dependencies) handlers.Handler {
	isDev := false
	tBankEnabled := false
	if len(deps) > 0 {
		isDev = deps[0].IsDev
		tBankEnabled = deps[0].TBankEnabled
	}
	return &paymentPlanHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "payment_plan_handler",
			Command: constants.PaymentConstants.CallbackPaymentPlan,
			Type:    handlers.TypeCallback,
		},
		isDev:        isDev,
		tBankEnabled: tBankEnabled,
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
	// Тестовый план только в dev
	if planID == "test" && !h.isDev {
		return nil
	}
	plans := map[string]*SubscriptionPlan{
		"test": { // ⭐ ТЕСТОВЫЙ ПЛАН
			ID:          "test",
			Name:        "🧪 Тестовый доступ (2⭐)",
			Description: "Для проверки работы платежей",
			PriceCents:  6, // 2 Stars = 6 центов
			Features: []string{
				"✅ Проверка оплаты через Stars",
				"✅ Доступ на 5 минут",
				"✅ Не влияет на основную подписку",
			},
		},
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
	callbackTBank := fmt.Sprintf("%s%s",
		constants.PaymentConstants.CallbackPaymentTBank, planID)

	rows := [][]map[string]string{
		{
			{"text": "⭐ Telegram Stars", "callback_data": callbackConfirm},
		},
	}

	if h.tBankEnabled {
		rows = append(rows, []map[string]string{
			{"text": "💳 Т-Банк (СБП / карта)", "callback_data": callbackTBank},
		})
	}

	rows = append(rows,
		[]map[string]string{
			{"text": constants.PaymentButtonTexts.BackToPlans, "callback_data": constants.PaymentConstants.CommandBuy},
			{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackMenuMain},
		},
	)

	return map[string]interface{}{
		"inline_keyboard": rows,
	}
}

// calculateStars рассчитывает количество Stars с учетом комиссии
func (h *paymentPlanHandler) calculateStars(usdCents int) int {
	if usdCents == 6 { // тестовый план
		return 2
	}
	return usdCents / 3
}

// Вспомогательный тип
type SubscriptionPlan struct {
	ID          string
	Name        string
	Description string
	PriceCents  int
	Features    []string
}
