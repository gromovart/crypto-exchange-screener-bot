// internal/delivery/max/bot/handlers/commands/start/handler.go
package start

import (
	"fmt"
	"time"

	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
	tradingSession "crypto-exchange-screener-bot/internal/delivery/telegram/services/trading_session"
)

type startHandler struct {
	*base.BaseHandler
	sessionService tradingSession.Service
}

func NewHandler(svc tradingSession.Service) handlers.Handler {
	return &startHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "max_start_handler",
			Command: "start",
			Type:    handlers.TypeCommand,
		},
		sessionService: svc,
	}
}

func (h *startHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	isAuth := params.User != nil && params.User.ID > 0

	var firstName string
	if isAuth && params.User.FirstName != "" {
		firstName = params.User.FirstName
	} else {
		firstName = "Гость"
	}

	msg := fmt.Sprintf(
		"👋 *Добро пожаловать, %s!*\n"+
			"🚀 *Crypto Exchange Screener Bot*\n\n"+
			"▫️ Биржа: *Bybit*  •  Обновление: *10-20 сек*\n"+
			"▫️ Символы: фьючерсы USDT\n"+
			"▫️ Сигналы: рост / падение / объёмы / OI\n\n"+
			"━━━ ⚠️ ВАЖНОЕ ПРЕДУПРЕЖДЕНИЕ ━━━\n\n"+
			"▫️ *Рыночные риски* — рынок криптовалют высоко волатилен, торговля связана с риском потери капитала\n\n"+
			"▫️ *Информационный характер* — сигналы не являются руководством к действию (Buy/Sell)\n\n"+
			"▫️ *Ответственность* — все решения о сделках вы принимаете самостоятельно\n\n"+
			"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"+
			"Используйте меню ниже для управления ботом:",
		firstName,
	)

	// Определяем активность торговой сессии и оставшееся время
	var sessionBtnText string
	var sessionBtnCb string
	if isAuth && h.sessionService != nil {
		if session, ok := h.sessionService.GetActive(params.User.ID, "max"); ok {
			remaining := time.Until(session.ExpiresAt)
			sessionBtnText = fmt.Sprintf("🔴 Завершить сессию (%s)", formatRemaining(remaining))
			sessionBtnCb = kb.CbSessionStop
		}
	}
	if sessionBtnText == "" {
		sessionBtnText = kb.Btn.SessionStart
		sessionBtnCb = kb.CbSessionStart
	}

	keyboard := buildMainMenu(isAuth, sessionBtnText, sessionBtnCb)
	return handlers.HandlerResult{Message: msg, Keyboard: keyboard}, nil
}

// formatRemaining форматирует оставшееся время в формате "Xч Yм" или "Yм"
func formatRemaining(d time.Duration) string {
	if d <= 0 {
		return "0м"
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dч %dм", h, m)
	}
	return fmt.Sprintf("%dм", m)
}

const docsURL = "https://teletype.in/@gromovart/pj2UIVlmr55"

func buildMainMenu(isAuth bool, sessionBtnText, sessionBtnCb string) interface{} {
	if isAuth {
		return kb.Keyboard([][]map[string]string{
			{kb.B(kb.Btn.Profile, kb.CbProfileMain), kb.B(kb.Btn.Settings, kb.CbSettingsMain)},
			{kb.B(kb.Btn.Notifications, kb.CbNotificationsMenu), kb.B(kb.Btn.Signals, kb.CbSignalsMenu)},
			{kb.B(kb.Btn.Periods, kb.CbPeriodsMenu), kb.B(kb.Btn.Status, kb.CbStats)},
			{kb.B(kb.Btn.Reset, kb.CbResetMenu), kb.B(kb.Btn.Help, kb.CbHelp)},
			{kb.BUrl("📚 Документация", docsURL)},
			{kb.B(kb.Btn.MainMenu, kb.CbMenuMain)},
			{kb.B(sessionBtnText, sessionBtnCb)},
		})
	}
	return kb.Keyboard([][]map[string]string{
		{kb.B(kb.Btn.Settings, kb.CbSettingsMain), kb.B(kb.Btn.Notifications, kb.CbNotificationsMenu)},
		{kb.B(kb.Btn.Periods, kb.CbPeriodsMenu), kb.B(kb.Btn.Status, kb.CbStats)},
		{kb.B(kb.Btn.Login, kb.CbAuthLogin), kb.B(kb.Btn.Help, kb.CbHelp)},
		{kb.BUrl("📚 Документация", docsURL)},
	})
}
