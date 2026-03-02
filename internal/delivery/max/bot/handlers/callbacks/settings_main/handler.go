// internal/delivery/max/bot/handlers/callbacks/settings_main/handler.go
package settings_main

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
)

// Handler — обработчик меню настроек
type Handler struct {
	*base.BaseHandler
}

// New создаёт обработчик меню настроек
func New() handlers.Handler {
	return &Handler{
		BaseHandler: base.New("settings_main", kb.CbSettingsMain, handlers.TypeCallback),
	}
}

// Execute выполняет обработку
func (h *Handler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	user := params.User

	notifyStatus := "❌ Выключены"
	growthStatus := "❌"
	fallStatus := "❌"

	if user != nil {
		if user.NotificationsEnabled {
			notifyStatus = "✅ Включены"
		}
		if user.NotifyGrowth {
			growthStatus = "✅"
		}
		if user.NotifyFall {
			fallStatus = "✅"
		}
	}

	msg := fmt.Sprintf(
		"⚙️ *Настройки*\n\n"+
			"🔔 Уведомления: %s\n"+
			"📈 Рост: %s | 📉 Падение: %s\n\n"+
			"Выберите раздел для настройки:",
		notifyStatus, growthStatus, fallStatus,
	)

	rows := [][]map[string]string{
		{kb.B(kb.Btn.Notifications, kb.CbNotificationsMenu)},
		{kb.B(kb.Btn.Signals, kb.CbSignalsMenu)},
		{kb.B(kb.Btn.Periods, kb.CbPeriodsMenu)},
		{kb.B(kb.Btn.Thresholds, kb.CbThresholdsMenu)},
		{kb.B(kb.Btn.Reset, kb.CbResetMenu)},
		kb.BackRow(kb.CbMenuMain),
	}

	return handlers.HandlerResult{
		Message:     msg,
		Keyboard:    kb.Keyboard(rows),
		EditMessage: params.MessageID != "",
	}, nil
}
