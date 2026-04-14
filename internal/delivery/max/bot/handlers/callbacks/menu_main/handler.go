// internal/delivery/max/bot/handlers/callbacks/menu_main/handler.go
package menu_main

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
	tradingSession "crypto-exchange-screener-bot/internal/delivery/telegram/services/trading_session"
)

// Handler — обработчик главного меню
type Handler struct {
	*base.BaseHandler
	sessionService tradingSession.Service
}

// New создаёт обработчик главного меню
func New(svc tradingSession.Service) handlers.Handler {
	return &Handler{
		BaseHandler:    base.New("menu_main", kb.CbMenuMain, handlers.TypeCallback),
		sessionService: svc,
	}
}

// Execute выполняет обработку — показывает главное меню
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
		{kb.B(kb.Btn.Status, kb.CbStats)},
		{kb.B(kb.Btn.Notifications, kb.CbNotificationsMenu), kb.B(kb.Btn.Signals, kb.CbSignalsMenu)},
		{kb.B(kb.Btn.Periods, kb.CbPeriodsMenu), kb.B(kb.Btn.Thresholds, kb.CbThresholdsMenu)},
		{kb.B("📋 Мои монеты", kb.CbWatchlistMenu)},
		{kb.B(kb.Btn.Profile, kb.CbProfileMain)},
		{kb.B(kb.Btn.Reset, kb.CbResetMenu), kb.B(kb.Btn.Help, kb.CbHelp)},
	}

	// Кнопка торговой сессии — Start или Stop в зависимости от состояния
	if isAuth && h.sessionService != nil {
		if _, active := h.sessionService.GetActive(user.ID, "max"); active {
			rows = append(rows, []map[string]string{kb.B(kb.Btn.SessionStop, kb.CbSessionStop)})
		} else {
			rows = append(rows, []map[string]string{kb.B(kb.Btn.SessionStart, kb.CbSessionStart)})
		}
	}

	// Кнопка покупки подписки
	rows = append(rows, []map[string]string{kb.B(kb.Btn.Buy, kb.CbBuy)})

	// Кнопка привязки Telegram для MAX-only пользователей
	if user != nil && user.IsMaxOnlyUser() {
		rows = append(rows, []map[string]string{kb.B(kb.Btn.LinkTelegram, kb.CbLinkTelegram)})
	}

	return handlers.HandlerResult{
		Message:     msg,
		Keyboard:    kb.Keyboard(rows),
		EditMessage: params.MessageID != "",
	}, nil
}
