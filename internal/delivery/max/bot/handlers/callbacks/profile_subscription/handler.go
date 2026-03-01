// internal/delivery/max/bot/handlers/callbacks/profile_subscription/handler.go
package profile_subscription

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
)

// Handler — обработчик информации о подписке
type Handler struct {
	*base.BaseHandler
}

// New создаёт обработчик
func New() handlers.Handler {
	return &Handler{
		BaseHandler: base.New("profile_subscription", kb.CbProfileSubscription, handlers.TypeCallback),
	}
}

// Execute выполняет обработку
func (h *Handler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	user := params.User

	tier := "Free"
	if user != nil && user.SubscriptionTier != "" {
		tier = user.SubscriptionTier
	}

	var msg string
	switch tier {
	case "premium", "pro":
		msg = fmt.Sprintf(
			"💎 *Подписка*\n\n"+
				"Тариф: *%s*\n"+
				"Статус: ✅ Активна\n\n"+
				"Доступные функции:\n"+
				"• Неограниченные сигналы\n"+
				"• Все таймфреймы\n"+
				"• Приоритетные уведомления",
			tier,
		)
	default:
		msg = "💎 *Подписка*\n\n" +
			"Тариф: *Free*\n" +
			"Статус: Бесплатный план\n\n" +
			"Ограничения:\n" +
			"• До 10 сигналов в день\n" +
			"• Базовые таймфреймы\n\n" +
			"Для расширенного доступа обратитесь к администратору."
	}

	rows := [][]map[string]string{
		{kb.B(kb.Btn.Back, kb.CbProfileMain)},
	}

	return handlers.HandlerResult{
		Message:     msg,
		Keyboard:    kb.Keyboard(rows),
		EditMessage: params.MessageID > 0,
	}, nil
}
