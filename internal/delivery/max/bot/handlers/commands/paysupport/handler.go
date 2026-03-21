// internal/delivery/max/bot/handlers/commands/paysupport/handler.go
package paysupport

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
)

type paysupportHandler struct{ *base.BaseHandler }

func NewHandler() handlers.Handler {
	return &paysupportHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "max_paysupport_handler",
			Command: "paysupport",
			Type:    handlers.TypeCommand,
		},
	}
}

func (h *paysupportHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	userID := 0
	if params.User != nil {
		userID = params.User.ID
	}

	msg := fmt.Sprintf(
		"🛟 *Поддержка*\n\n"+
			"Если у вас возникли вопросы или проблемы, обратитесь следующими способами:\n\n"+
			"📧 *Email:* support@gromovart.ru\n\n"+
			"💬 *Telegram:* @crypto_exchange_screener\n\n"+
			"⚠️ *При обращении укажите ваш ID:* `%d`\n\n"+
			"🕐 Мы обрабатываем запросы в течение 24 часов.",
		userID,
	)

	keyboard := kb.Keyboard([][]map[string]string{
		{kb.BUrl("💬 Написать в Telegram", "https://t.me/artemgrrr")},
		{kb.B(kb.Btn.Help, kb.CbHelp), kb.B(kb.Btn.MainMenu, kb.CbMenuMain)},
	})

	return handlers.HandlerResult{Message: msg, Keyboard: keyboard}, nil
}
