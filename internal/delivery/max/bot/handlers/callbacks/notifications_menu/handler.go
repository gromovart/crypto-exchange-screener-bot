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

	enabled := false
	growthOnly := false
	fallOnly := false

	if user != nil {
		enabled = user.NotificationsEnabled
		growthOnly = user.NotifyGrowth && !user.NotifyFall
		fallOnly = !user.NotifyGrowth && user.NotifyFall
	}

	enabledStr := "❌ Выключены"
	if enabled {
		enabledStr = "✅ Включены"
	}

	typeStr := "📊 Все сигналы"
	if growthOnly {
		typeStr = "📈 Только рост"
	} else if fallOnly {
		typeStr = "📉 Только падение"
	}

	msg := fmt.Sprintf(
		"🔔 *Уведомления*\n\n"+
			"Статус: %s\n"+
			"Тип: %s\n\n"+
			"Управление уведомлениями:",
		enabledStr, typeStr,
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
		EditMessage: params.MessageID > 0,
	}, nil
}
