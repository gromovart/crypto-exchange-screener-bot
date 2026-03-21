// internal/delivery/max/bot/handlers/callbacks/buy/handler.go
package buy

import (
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
)

// Handler показывает список тарифных планов с ценами
type Handler struct {
	*base.BaseHandler
}

// New создаёт обработчик страницы покупки
func New() handlers.Handler {
	return &Handler{
		BaseHandler: base.New("buy_handler", kb.CbBuy, handlers.TypeCallback),
	}
}

// Execute показывает доступные тарифные планы
func (h *Handler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	msg := "💎 Выберите тарифный план\n\n" +
		"🧪 Тестовый — 10 ₽ (для проверки оплаты)\n" +
		"📱 1 месяц — 1 490 ₽\n" +
		"🚀 3 месяца — 2 490 ₽\n" +
		"🏢 12 месяцев — 5 990 ₽\n\n" +
		"Оплата через Т-Банк (СБП / карта).\n" +
		"После оплаты подписка активируется автоматически."

	rows := [][]map[string]string{
		{kb.B("🧪 Тестовый — 10 ₽", kb.CbPaymentTBankBase+"test")},
		{kb.B("📱 1 месяц — 1 490 ₽", kb.CbPaymentTBankBase+"basic")},
		{kb.B("🚀 3 месяца — 2 490 ₽", kb.CbPaymentTBankBase+"pro")},
		{kb.B("🏢 12 месяцев — 5 990 ₽", kb.CbPaymentTBankBase+"enterprise")},
		kb.BackRow(kb.CbMenuMain),
	}

	return handlers.HandlerResult{
		Message:     msg,
		Keyboard:    kb.Keyboard(rows),
		EditMessage: params.MessageID != "",
	}, nil
}
