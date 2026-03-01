// internal/delivery/max/bot/handlers/callbacks/notify_growth_only/handler.go
package notify_growth_only

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
	signalSvc "crypto-exchange-screener-bot/internal/delivery/telegram/services/signal_settings"
)

// Handler — обработчик: только уведомления о росте
type Handler struct {
	*base.BaseHandler
	service signalSvc.Service
}

// New создаёт обработчик
func New(svc signalSvc.Service) handlers.Handler {
	return &Handler{
		BaseHandler: base.New("notify_growth_only", kb.CbNotifyGrowthOnly, handlers.TypeCallback),
		service:     svc,
	}
}

// Execute выполняет обработку
func (h *Handler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	user := params.User
	if user == nil {
		return handlers.HandlerResult{Message: "❌ Пользователь не найден"}, nil
	}

	// Включаем рост, выключаем падение
	_, err1 := h.service.Exec(signalSvc.SignalSettingsParams{
		Action: "toggle_growth",
		UserID: user.ID,
		Value:  true,
	})
	_, err2 := h.service.Exec(signalSvc.SignalSettingsParams{
		Action: "toggle_fall",
		UserID: user.ID,
		Value:  false,
	})

	if err1 != nil || err2 != nil {
		return handlers.HandlerResult{
			Message:     fmt.Sprintf("❌ Ошибка обновления настроек"),
			Keyboard:    kb.Keyboard([][]map[string]string{{kb.B(kb.Btn.Back, kb.CbNotificationsMenu)}}),
			EditMessage: params.MessageID > 0,
		}, nil
	}

	msg := "🔔 *Уведомления*\n\n📈 Режим: только рост\n\nБудете получать уведомления только при росте цены."

	return handlers.HandlerResult{
		Message:     msg,
		Keyboard:    kb.Keyboard([][]map[string]string{{kb.B(kb.Btn.Back, kb.CbNotificationsMenu)}}),
		EditMessage: params.MessageID > 0,
	}, nil
}
