// internal/delivery/max/bot/handlers/callbacks/auth_logout/handler.go
package auth_logout

import (
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
)

// Handler — обработчик выхода
// В данной реализации выход означает прекращение уведомлений.
type Handler struct {
	*base.BaseHandler
}

// New создаёт обработчик
func New() handlers.Handler {
	return &Handler{
		BaseHandler: base.New("auth_logout", kb.CbAuthLogout, handlers.TypeCallback),
	}
}

// Execute выполняет обработку
func (h *Handler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	msg := "🚪 *Выход*\n\n" +
		"Для выхода из бота просто перестаньте его использовать.\n\n" +
		"Вы можете отключить все уведомления в разделе Настройки."

	rows := [][]map[string]string{
		{kb.B(kb.Btn.Notifications, kb.CbNotificationsMenu)},
		{kb.B(kb.Btn.MainMenu, kb.CbMenuMain)},
	}

	return handlers.HandlerResult{
		Message:     msg,
		Keyboard:    kb.Keyboard(rows),
		EditMessage: params.MessageID > 0,
	}, nil
}
