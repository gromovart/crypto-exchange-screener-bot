// internal/delivery/max/bot/handlers/callbacks/menu_main/handler.go
package menu_main

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
)

// Handler — обработчик главного меню
type Handler struct {
	*base.BaseHandler
}

// New создаёт обработчик главного меню
func New() handlers.Handler {
	return &Handler{
		BaseHandler: base.New("menu_main", kb.CbMenuMain, handlers.TypeCallback),
	}
}

// Execute выполняет обработку — показывает главное меню
func (h *Handler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	user := params.User

	greeting := "👋 Главное меню"
	if user != nil && user.FirstName != "" {
		greeting = fmt.Sprintf("👋 Привет, %s!", user.FirstName)
	}

	notifyStatus := "❌"
	if user != nil && user.NotificationsEnabled {
		notifyStatus = "✅"
	}

	msg := fmt.Sprintf(
		"%s\n\n📊 *Crypto Screener Bot*\n\n"+
			"🔔 Уведомления: %s\n\n"+
			"Выберите раздел:",
		greeting, notifyStatus,
	)

	rows := [][]map[string]string{
		{kb.B(kb.Btn.Status, kb.CbStats)},
		{kb.B(kb.Btn.Notifications, kb.CbNotificationsMenu), kb.B(kb.Btn.Signals, kb.CbSignalsMenu)},
		{kb.B(kb.Btn.Periods, kb.CbPeriodsMenu), kb.B(kb.Btn.Thresholds, kb.CbThresholdsMenu)},
		{kb.B(kb.Btn.Profile, kb.CbProfileMain)},
		{kb.B(kb.Btn.Reset, kb.CbResetMenu), kb.B(kb.Btn.Help, kb.CbHelp)},
	}

	return handlers.HandlerResult{
		Message:     msg,
		Keyboard:    kb.Keyboard(rows),
		EditMessage: params.MessageID > 0,
	}, nil
}
