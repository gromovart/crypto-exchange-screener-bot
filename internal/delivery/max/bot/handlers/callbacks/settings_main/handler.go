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
	isAuth := user != nil && user.ID > 0

	var msg string
	if isAuth {
		firstName := user.FirstName
		if firstName == "" {
			firstName = "Гость"
		}
		msg = fmt.Sprintf(
			"🏠 Главное меню\n\n"+
				"Привет, %s! 👋\n\n"+
				"Выберите раздел для управления ботом:",
			firstName,
		)
	} else {
		msg = "🏠 Главное меню\n\n" +
			"Добро пожаловать! 👋\n\n" +
			"Вы можете использовать основные функции бота.\n" +
			"Для доступа ко всем функциям выполните авторизацию.\n\n" +
			"Выберите раздел:"
	}

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
