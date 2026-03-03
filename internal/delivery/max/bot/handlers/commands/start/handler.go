// internal/delivery/max/bot/handlers/commands/start/handler.go
package start

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
)

type startHandler struct {
	*base.BaseHandler
}

func NewHandler() handlers.Handler {
	return &startHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "max_start_handler",
			Command: "start",
			Type:    handlers.TypeCommand,
		},
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

	keyboard := buildMainMenu(isAuth)
	return handlers.HandlerResult{Message: msg, Keyboard: keyboard}, nil
}

const docsURL = "https://teletype.in/@gromovart/pj2UIVlmr55"

func buildMainMenu(isAuth bool) interface{} {
	if isAuth {
		return kb.Keyboard([][]map[string]string{
			{kb.B(kb.Btn.Profile, kb.CbProfileMain), kb.B(kb.Btn.Settings, kb.CbSettingsMain)},
			{kb.B(kb.Btn.Notifications, kb.CbNotificationsMenu), kb.B(kb.Btn.Signals, kb.CbSignalsMenu)},
			{kb.B(kb.Btn.Periods, kb.CbPeriodsMenu), kb.B(kb.Btn.Status, kb.CbStats)},
			{kb.B(kb.Btn.Reset, kb.CbResetMenu), kb.B(kb.Btn.Help, kb.CbHelp)},
			{kb.BUrl("📚 Документация", docsURL)},
		})
	}
	return kb.Keyboard([][]map[string]string{
		{kb.B(kb.Btn.Settings, kb.CbSettingsMain), kb.B(kb.Btn.Notifications, kb.CbNotificationsMenu)},
		{kb.B(kb.Btn.Periods, kb.CbPeriodsMenu), kb.B(kb.Btn.Status, kb.CbStats)},
		{kb.B(kb.Btn.Login, kb.CbAuthLogin), kb.B(kb.Btn.Help, kb.CbHelp)},
		{kb.BUrl("📚 Документация", docsURL)},
	})
}
