// internal/delivery/telegram/app/bot/handlers/callbacks/payment_sbp/handler.go
package payment_sbp

import (
	"context"
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	tbank_service "crypto-exchange-screener-bot/internal/delivery/telegram/services/tbank"
	"crypto-exchange-screener-bot/pkg/logger"
)

// Dependencies зависимости обработчика
type Dependencies struct {
	TBankService tbank_service.Service
}

// paymentTBankHandler обработчик оплаты через Т-Банк
type paymentTBankHandler struct {
	*base.BaseHandler
	tbankService tbank_service.Service
}

// NewHandler создаёт обработчик оплаты через Т-Банк
func NewHandler(deps Dependencies) handlers.Handler {
	return &paymentTBankHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "payment_tbank_handler",
			Command: constants.PaymentConstants.CallbackPaymentTBank,
			Type:    handlers.TypeCallback,
		},
		tbankService: deps.TBankService,
	}
}

// Execute выполняет создание платежа и возвращает ссылку на форму оплаты
func (h *paymentTBankHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	if h.tbankService == nil {
		return handlers.HandlerResult{
			Message: "❌ *Оплата через Т-Банк временно недоступна*\n\nПожалуйста, воспользуйтесь оплатой через Telegram Stars.",
			Keyboard: map[string]interface{}{
				"inline_keyboard": [][]map[string]string{
					{{"text": "← К планам", "callback_data": constants.PaymentConstants.CommandBuy}},
					{{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackMenuMain}},
				},
			},
		}, nil
	}

	planID := h.extractPlanID(params.Data)
	if planID == "" {
		return handlers.HandlerResult{}, fmt.Errorf("не удалось извлечь ID плана из callback: %s", params.Data)
	}

	ctx := context.Background()
	result, err := h.tbankService.CreatePayment(ctx, params.User.ID, planID, "", "")
	if err != nil {
		logger.Error("❌ Ошибка создания платежа Т-Банк: план=%s, пользователь=%d, ошибка: %v",
			planID, params.User.ID, err)

		return handlers.HandlerResult{
			Message: "❌ *Ошибка создания платежа*\n\nПожалуйста, попробуйте позже или обратитесь в поддержку.",
			Keyboard: map[string]interface{}{
				"inline_keyboard": [][]map[string]string{
					{{"text": "← К планам", "callback_data": constants.PaymentConstants.CommandBuy}},
					{{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackMenuMain}},
				},
			},
		}, nil
	}

	logger.Info("✅ Платёж Т-Банк создан: OrderId=%s, URL=%s", result.OrderId, result.PaymentURL)

	message := h.buildPaymentMessage(planID, result)
	keyboard := h.buildPaymentKeyboard(planID, result.PaymentURL)

	return handlers.HandlerResult{
		Message:  message,
		Keyboard: keyboard,
		Metadata: map[string]interface{}{
			"plan_id":     planID,
			"order_id":    result.OrderId,
			"payment_url": result.PaymentURL,
			"amount":      result.Amount,
		},
	}, nil
}

// extractPlanID извлекает ID плана из callback_data
func (h *paymentTBankHandler) extractPlanID(data string) string {
	prefix := constants.PaymentConstants.CallbackPaymentTBank
	if len(data) <= len(prefix) {
		return ""
	}
	return data[len(prefix):]
}

// buildPaymentMessage формирует сообщение с инструкцией по оплате
func (h *paymentTBankHandler) buildPaymentMessage(planID string, result *tbank_service.PaymentResult) string {
	planNames := map[string]string{
		"test":       "🧪 Тестовый доступ",
		"basic":      "📱 Доступ на 1 месяц",
		"pro":        "🚀 Доступ на 3 месяца",
		"enterprise": "🏢 Доступ на 12 месяцев",
	}
	planAmounts := map[string]string{
		"test":       "10 ₽",
		"basic":      "1 490 ₽",
		"pro":        "2 490 ₽",
		"enterprise": "5 990 ₽",
	}

	name := planNames[planID]
	amount := planAmounts[planID]

	msg := "💳 *Оплата через Т-Банк*\n\n"
	msg += fmt.Sprintf("📋 Тариф: *%s*\n", name)
	msg += fmt.Sprintf("💰 Сумма: *%s*\n\n", amount)
	msg += "📲 *Как оплатить:*\n"
	msg += "1. Нажмите кнопку «Открыть форму оплаты»\n"
	msg += "2. Выберите способ: СБП, карта или другой\n"
	msg += "3. Подтвердите платёж\n\n"
	msg += "✅ После оплаты подписка активируется автоматически — вы получите уведомление в этом чате.\n\n"
	msg += "❓ Вопросы? Напишите /help"

	return msg
}

// buildPaymentKeyboard формирует клавиатуру для оплаты
func (h *paymentTBankHandler) buildPaymentKeyboard(planID, paymentURL string) interface{} {
	return map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			{
				{"text": "💳 Открыть форму оплаты", "url": paymentURL},
			},
			{
				{"text": "← К планам", "callback_data": constants.PaymentConstants.CommandBuy},
				{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackMenuMain},
			},
		},
	}
}
