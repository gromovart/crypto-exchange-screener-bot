// internal/delivery/max/bot/handlers/callbacks/payment_tbank/handler.go
package payment_tbank

import (
	"context"
	"fmt"
	"strings"

	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
	tbank_service "crypto-exchange-screener-bot/internal/delivery/telegram/services/tbank"
	"crypto-exchange-screener-bot/pkg/logger"
)

var planNames = map[string]string{
	"test":       "Тестовый доступ",
	"basic":      "Доступ на 1 месяц",
	"pro":        "Доступ на 3 месяца",
	"enterprise": "Доступ на 12 месяцев",
}

var planAmounts = map[string]string{
	"test":       "10 ₽",
	"basic":      "1 490 ₽",
	"pro":        "2 490 ₽",
	"enterprise": "5 990 ₽",
}

// Handler обрабатывает создание платежа через Т-Банк
type Handler struct {
	*base.BaseHandler
	tbankService tbank_service.Service
	successURL   string // MAX-specific redirect после успешной оплаты
	failURL      string // MAX-specific redirect после неудачной оплаты
}

// New создаёт обработчик платежа через Т-Банк.
// successURL/failURL — URL редиректа после оплаты (пустая строка = использовать дефолт сервиса).
func New(svc tbank_service.Service, successURL, failURL string) handlers.Handler {
	return &Handler{
		BaseHandler:  base.New("payment_tbank_handler", kb.CbPaymentTBankWildcard, handlers.TypeCallback),
		tbankService: svc,
		successURL:   successURL,
		failURL:      failURL,
	}
}

// Execute создаёт платёж через Т-Банк и возвращает ссылку на форму оплаты
func (h *Handler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	if h.tbankService == nil {
		return handlers.HandlerResult{
			Message:     "❌ Оплата через Т-Банк временно недоступна.\n\nПопробуйте позже.",
			Keyboard:    kb.Keyboard([][]map[string]string{kb.BackRow(kb.CbBuy)}),
			EditMessage: params.MessageID != "",
		}, nil
	}

	planID := h.extractPlanID(params.Data)
	if planID == "" {
		return handlers.HandlerResult{}, fmt.Errorf("не удалось извлечь planID из: %s", params.Data)
	}

	result, err := h.tbankService.CreatePayment(context.Background(), params.User.ID, planID, h.successURL, h.failURL)
	if err != nil {
		logger.Error("❌ MAX payment_tbank: план=%s, user=%d: %v", planID, params.User.ID, err)
		return handlers.HandlerResult{
			Message:     "❌ Ошибка создания платежа. Попробуйте позже или обратитесь в поддержку.",
			Keyboard:    kb.Keyboard([][]map[string]string{kb.BackRow(kb.CbBuy)}),
			EditMessage: params.MessageID != "",
		}, nil
	}

	logger.Info("✅ MAX payment_tbank создан: OrderId=%s, plan=%s, user=%d", result.OrderId, planID, params.User.ID)

	name := planNames[planID]
	amount := planAmounts[planID]

	msg := "💳 Оплата через Т-Банк\n\n" +
		fmt.Sprintf("Тариф: %s\n", name) +
		fmt.Sprintf("Сумма: %s\n\n", amount) +
		"Как оплатить:\n" +
		"1. Нажмите кнопку «Открыть форму оплаты»\n" +
		"2. Выберите способ: СБП, карта или другой\n" +
		"3. Подтвердите платёж\n\n" +
		"После оплаты подписка активируется автоматически."

	rows := [][]map[string]string{
		{kb.BUrl("💳 Открыть форму оплаты", result.PaymentURL)},
		{kb.B("← К тарифам", kb.CbBuy), kb.B(kb.Btn.MainMenu, kb.CbMenuMain)},
	}

	return handlers.HandlerResult{
		Message:     msg,
		Keyboard:    kb.Keyboard(rows),
		EditMessage: params.MessageID != "",
	}, nil
}

// extractPlanID извлекает planID из callback data (payment_tbank_basic → basic)
func (h *Handler) extractPlanID(data string) string {
	return strings.TrimPrefix(data, kb.CbPaymentTBankBase)
}
