// internal/delivery/telegram/app/bot/handlers/callbacks/payment_plan/handler.go
package payment_plan

import (
	"context"
	"fmt"
	"math"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	currency_client "crypto-exchange-screener-bot/internal/infrastructure/http/currency"
)

// Dependencies зависимости хэндлера
type Dependencies struct {
	IsDev          bool
	TBankEnabled   bool
	CurrencyClient *currency_client.Client
}

// paymentPlanHandler обработчик выбора платежного плана
type paymentPlanHandler struct {
	*base.BaseHandler
	isDev          bool
	tBankEnabled   bool
	currencyClient *currency_client.Client
}

// NewHandler создает новый обработчик выбора плана
func NewHandler(deps ...Dependencies) handlers.Handler {
	isDev := false
	tBankEnabled := false
	var cc *currency_client.Client
	if len(deps) > 0 {
		isDev = deps[0].IsDev
		tBankEnabled = deps[0].TBankEnabled
		cc = deps[0].CurrencyClient
	}
	return &paymentPlanHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "payment_plan_handler",
			Command: constants.PaymentConstants.CallbackPaymentPlan,
			Type:    handlers.TypeCallback,
		},
		isDev:          isDev,
		tBankEnabled:   tBankEnabled,
		currencyClient: cc,
	}
}

// Execute выполняет обработку выбора плана
func (h *paymentPlanHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	planID := h.extractPlanID(params.Data)
	if planID == "" {
		return handlers.HandlerResult{}, fmt.Errorf("неверный формат callback: %s", params.Data)
	}

	plan := h.getPlanByID(planID)
	if plan == nil {
		return handlers.HandlerResult{}, fmt.Errorf("план не найден: %s", planID)
	}

	usdRubRate := h.getRate()
	message := h.createConfirmationMessage(plan, usdRubRate)
	keyboard := h.createConfirmationKeyboard(planID)

	return handlers.HandlerResult{
		Message:  message,
		Keyboard: keyboard,
		Metadata: map[string]interface{}{
			"plan_id":      planID,
			"user_id":      params.User.ID,
			"stars_amount": calculateStars(plan.PriceRub, usdRubRate),
		},
	}, nil
}

func (h *paymentPlanHandler) getRate() float64 {
	if h.currencyClient != nil {
		return h.currencyClient.GetUSDRUB(context.Background())
	}
	return currency_client.FallbackRate
}

func (h *paymentPlanHandler) extractPlanID(callbackData string) string {
	prefix := constants.PaymentConstants.CallbackPaymentPlan
	if len(callbackData) <= len(prefix) {
		return ""
	}
	return callbackData[len(prefix):]
}

func (h *paymentPlanHandler) getPlanByID(planID string) *SubscriptionPlan {
	if planID == "test" && !h.isDev {
		return nil
	}
	plans := map[string]*SubscriptionPlan{
		"test": {
			ID:          "test",
			Name:        "🧪 Тестовый доступ",
			Description: "Для проверки работы платежей",
			PriceRub:    10,
			Features: []string{
				"✅ Проверка оплаты",
				"✅ Доступ на 5 минут",
			},
		},
		"basic": {
			ID:          "basic",
			Name:        "📱 1 месяц",
			Description: "Идеально для начала",
			PriceRub:    1490,
			Features: []string{
				"✅ Все сигналы",
				"✅ Все виды уведомлений",
			},
		},
		"pro": {
			ID:          "pro",
			Name:        "🚀 3 месяца",
			Description: "Для активных трейдеров",
			PriceRub:    2490,
			Features: []string{
				"✅ Все сигналы",
				"✅ Все виды уведомлений",
				"✅ Приоритетная поддержка",
			},
		},
		"enterprise": {
			ID:          "enterprise",
			Name:        "🏢 12 месяцев",
			Description: "Максимальные возможности",
			PriceRub:    5990,
			Features: []string{
				"✅ Все сигналы",
				"✅ Все виды уведомлений",
				"✅ Кастомные настройки",
				"✅ Поддержка 24/7",
			},
		},
	}
	return plans[planID]
}

func (h *paymentPlanHandler) createConfirmationMessage(plan *SubscriptionPlan, usdRubRate float64) string {
	stars := calculateStars(plan.PriceRub, usdRubRate)

	message := "✅ *Подтверждение выбора*\n\n"
	message += fmt.Sprintf("Тариф: *%s*\n", plan.Name)
	message += fmt.Sprintf("💰 Стоимость: *%d ₽*\n", plan.PriceRub)
	message += fmt.Sprintf("⭐ или *%d Stars*\n\n", stars)
	message += fmt.Sprintf("📋 %s\n\n", plan.Description)
	message += "🔍 *Включено:*\n"
	for _, feature := range plan.Features {
		message += feature + "\n"
	}
	message += "\nВыберите способ оплаты:"
	return message
}

func (h *paymentPlanHandler) createConfirmationKeyboard(planID string) interface{} {
	callbackConfirm := fmt.Sprintf("%s%s", constants.PaymentConstants.CallbackPaymentConfirm, planID)
	callbackTBank := fmt.Sprintf("%s%s", constants.PaymentConstants.CallbackPaymentTBank, planID)

	rows := [][]map[string]string{
		{{"text": "⭐ Telegram Stars", "callback_data": callbackConfirm}},
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

	return map[string]interface{}{"inline_keyboard": rows}
}

// calculateStars: ceil((₽ / курс) / $0.013)
func calculateStars(priceRub int, usdRubRate float64) int {
	if usdRubRate <= 0 {
		usdRubRate = currency_client.FallbackRate
	}
	usd := float64(priceRub) / usdRubRate
	return int(math.Ceil(usd / 0.013))
}

// SubscriptionPlan вспомогательный тип
type SubscriptionPlan struct {
	ID          string
	Name        string
	Description string
	PriceRub    int
	Features    []string
}
