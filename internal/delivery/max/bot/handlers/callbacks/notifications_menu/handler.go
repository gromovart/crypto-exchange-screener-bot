// internal/delivery/max/bot/handlers/callbacks/notifications_menu/handler.go
package notifications_menu

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
)

// Handler — обработчик меню уведомлений
type Handler struct {
	*base.BaseHandler
}

// New создаёт обработчик меню уведомлений
func New() handlers.Handler {
	return &Handler{
		BaseHandler: base.New("notifications_menu", kb.CbNotificationsMenu, handlers.TypeCallback),
	}
}

// Execute выполняет обработку
func (h *Handler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	user := params.User

	notifyGrowthText := h.GetToggleText("📈 Рост", false)
	notifyFallText := h.GetToggleText("📉 Падение", false)
	enabledStr := h.GetBoolDisplay(false)

	if user != nil {
		notifyGrowthText = h.GetToggleText("📈 Рост", user.NotifyGrowth)
		notifyFallText = h.GetToggleText("📉 Падение", user.NotifyFall)
		enabledStr = h.GetBoolDisplay(user.NotificationsEnabled)
	}

	msg := fmt.Sprintf(
		"🔔 Уведомления\n\n"+
			"Текущие настройки:\n\n"+
			"🔊 Общие уведомления: %s\n"+
			"%s\n"+
			"%s\n\n"+
			"Выберите настройку для изменения:",
		enabledStr,
		notifyGrowthText,
		notifyFallText,
	)

	rows := [][]map[string]string{
		{kb.B(kb.Btn.NotifyToggleAll, kb.CbNotifyToggleAll)},
		{kb.B(kb.Btn.NotifyGrowthOnly, kb.CbNotifyGrowthOnly), kb.B(kb.Btn.NotifyFallOnly, kb.CbNotifyFallOnly)},
		{kb.B(kb.Btn.NotifyBoth, kb.CbNotifyBoth)},
		kb.BackRow(kb.CbMenuMain),
	}

	return handlers.HandlerResult{
		Message:     msg,
		Keyboard:    kb.Keyboard(rows),
		EditMessage: params.MessageID != "",
	}, nil
}
