// internal/delivery/telegram/app/bot/handlers/events/payment/successful_payment/handler.go
package successful_payment

import (
	"fmt"
	"strconv"
	"strings"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	"crypto-exchange-screener-bot/internal/delivery/telegram/services/payment"
	"crypto-exchange-screener-bot/pkg/logger"
)

// successfulPaymentHandler реализация обработчика successful_payment
type successfulPaymentHandler struct {
	*base.BaseHandler
	paymentService payment.Service
}

// Execute выполняет обработку successful_payment
func (h *successfulPaymentHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	// Извлекаем данные successful_payment из params.Data
	// Формат: successful_payment:{payment_id}:{payload}:{amount}:{currency}:{user_id}:{charge_id}
	paymentData := h.parseSuccessfulPaymentData(params.Data)
	if paymentData.PaymentID == "" {
		return handlers.HandlerResult{}, fmt.Errorf("неверный формат successful_payment данных")
	}

	// Проверяем user_id
	if params.User == nil || params.User.ID == 0 {
		return handlers.HandlerResult{}, fmt.Errorf("пользователь не авторизован")
	}

	// Обработка через payment service
	paymentParams := payment.PaymentParams{
		Action: "successful_payment",
		UserID: params.User.ID,
		ChatID: params.ChatID,
		Data: map[string]interface{}{
			"telegram_payment_charge_id": paymentData.PaymentID,
			"invoice_payload":            paymentData.Payload,
			"total_amount":               paymentData.TotalAmount,
			"currency":                   paymentData.Currency,
			"provider_payment_charge_id": paymentData.ProviderChargeID,
		},
	}

	result, err := h.paymentService.Exec(paymentParams)
	if err != nil {
		logger.Error("❌ Ошибка обработки successful_payment: %v", err)

		// ⭐ Проверяем тип ошибки - активная подписка уже существует
		if strings.Contains(err.Error(), "у пользователя уже есть активная подписка") {
			planName := h.getPlanNameFromPayload(paymentData.Payload)
			return handlers.HandlerResult{
				Message: "✅ *Платеж успешно обработан!*\n\n" +
					fmt.Sprintf("💰 Сумма: *%d Stars*\n", paymentData.TotalAmount) +
					fmt.Sprintf("📋 План: *%s*\n\n", planName) +
					"У вас уже есть активная подписка. Платеж будет зачислен как продление.",
				Keyboard: map[string]interface{}{
					"inline_keyboard": [][]map[string]string{
						{
							{"text": "📊 Мой профиль", "callback_data": constants.CallbackProfileMain},
						},
						{
							{"text": "🔙 В меню", "callback_data": constants.CallbackMenuMain},
						},
					},
				},
			}, nil
		}

		// Другие ошибки
		return handlers.HandlerResult{
			Message: "❌ *Ошибка обработки платежа*\n\n" +
				"Пожалуйста, обратитесь в поддержку.",
			Keyboard: map[string]interface{}{
				"inline_keyboard": [][]map[string]string{
					{
						{"text": "📞 Поддержка", "url": "https://t.me/artemgrrr"},
					},
					{
						{"text": "🔙 В меню", "callback_data": constants.CallbackMenuMain},
					},
				},
			},
		}, nil
	}

	// Формируем сообщение для пользователя при успехе
	message := "✅ *Платеж успешно обработан!*\n\n"
	message += fmt.Sprintf("💰 Сумма: *%d Stars*\n", paymentData.TotalAmount)
	message += fmt.Sprintf("📋 План: *%s*\n", h.getPlanNameFromPayload(paymentData.Payload))
	message += "🎉 Ваша подписка активирована!\n\n"
	message += "Теперь вам доступны все функции выбранного тарифа."

	return handlers.HandlerResult{
		Message: message,
		Keyboard: map[string]interface{}{
			"inline_keyboard": [][]map[string]string{
				{
					{"text": "📊 Мой профиль", "callback_data": constants.CallbackProfileMain},
				},
				{
					{"text": "🔙 В меню", "callback_data": constants.CallbackMenuMain},
				},
			},
		},
		Metadata: map[string]interface{}{
			"payment_id":      paymentData.PaymentID,
			"success":         result.Success,
			"subscription_id": result.SubscriptionID,
			"activated_until": result.ActivatedUntil,
			"stars_amount":    paymentData.TotalAmount,
		},
	}, nil
}

// successfulPaymentData структура для данных successful_payment
type successfulPaymentData struct {
	PaymentID        string
	Payload          string
	TotalAmount      int
	Currency         string
	UserID           int64
	ProviderChargeID string
}

// parseSuccessfulPaymentData парсит данные successful_payment из строки
func (h *successfulPaymentHandler) parseSuccessfulPaymentData(data string) successfulPaymentData {
	// Формат: successful_payment:{payment_id}:{payload}:{amount}:{currency}:{user_id}:{provider_charge_id}
	logger.Debug("📦 Парсинг successful_payment данных: '%s'", data)

	parts := strings.Split(data, ":")
	logger.Debug("📊 Разделено на %d частей: %v", len(parts), parts)

	if len(parts) < 7 || parts[0] != "successful_payment" {
		logger.Error("❌ Неверный формат successful_payment: ожидается 7 частей, получено %d", len(parts))
		return successfulPaymentData{}
	}

	amount, err := strconv.Atoi(parts[3])
	if err != nil {
		logger.Error("❌ Ошибка парсинга amount: %v", err)
		return successfulPaymentData{}
	}

	userID, err := strconv.ParseInt(parts[5], 10, 64)
	if err != nil {
		logger.Error("❌ Ошибка парсинга userID: %v", err)
		return successfulPaymentData{}
	}

	logger.Debug("✅ Успешно распарсено successful_payment:")
	logger.Debug("   • PaymentID: %s", parts[1])
	logger.Debug("   • Payload: %s", parts[2])
	logger.Debug("   • Amount: %d", amount)
	logger.Debug("   • Currency: %s", parts[4])
	logger.Debug("   • UserID: %d", userID)
	logger.Debug("   • ProviderChargeID: %s", parts[6])

	return successfulPaymentData{
		PaymentID:        parts[1],
		Payload:          parts[2],
		TotalAmount:      amount,
		Currency:         parts[4],
		UserID:           userID,
		ProviderChargeID: parts[6],
	}
}

// getPlanNameFromPayload извлекает название плана из payload
func (h *successfulPaymentHandler) getPlanNameFromPayload(payload string) string {
	// Формат: sub_{plan_id}_{user_id}_{nonce}
	parts := strings.Split(payload, "_")
	if len(parts) < 4 || parts[0] != "sub" {
		return "Неизвестный план"
	}

	planID := parts[1]
	plans := map[string]string{
		"basic":      "📱 Basic",
		"pro":        "🚀 Pro",
		"enterprise": "🏢 Enterprise",
	}

	if name, exists := plans[planID]; exists {
		return name
	}
	return "Неизвестный план"
}
