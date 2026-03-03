// internal/delivery/max/bot/handlers/callbacks/notify_both/handler.go
package notify_both

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
	signalSvc "crypto-exchange-screener-bot/internal/delivery/telegram/services/signal_settings"
)

// Handler — обработчик: все уведомления (рост + падение)
type Handler struct {
	*base.BaseHandler
	service signalSvc.Service
}

// New создаёт обработчик
func New(svc signalSvc.Service) handlers.Handler {
	return &Handler{
		BaseHandler: base.New("notify_both", kb.CbNotifyBoth, handlers.TypeCallback),
		service:     svc,
	}
}

// Execute выполняет обработку
func (h *Handler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	user := params.User
	if user == nil {
		return handlers.HandlerResult{Message: "❌ Пользователь не найден"}, nil
	}

	// Включаем и рост, и падение
	_, err1 := h.service.Exec(signalSvc.SignalSettingsParams{
		Action: "toggle_growth",
		UserID: user.ID,
		Value:  true,
	})
	_, err2 := h.service.Exec(signalSvc.SignalSettingsParams{
		Action: "toggle_fall",
		UserID: user.ID,
		Value:  true,
	})

	if err1 != nil || err2 != nil {
		return handlers.HandlerResult{
			Message:     fmt.Sprintf("❌ Ошибка обновления настроек"),
			Keyboard:    kb.Keyboard([][]map[string]string{{kb.B(kb.Btn.Back, kb.CbNotificationsMenu)}}),
			EditMessage: params.MessageID != "",
		}, nil
	}

	msg := "🔔 *Уведомления*\n\n📊 Режим: все сигналы\n\nБудете получать уведомления о росте и падении цены."

	return handlers.HandlerResult{
		Message:     msg,
		Keyboard:    kb.Keyboard([][]map[string]string{{kb.B(kb.Btn.Back, kb.CbNotificationsMenu)}}),
		EditMessage: params.MessageID != "",
	}, nil
}
