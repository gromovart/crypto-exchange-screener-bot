// internal/delivery/max/bot/handlers/commands/help/handler.go
package help

import (
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
)

type helpHandler struct{ *base.BaseHandler }

func NewHandler() handlers.Handler {
	return &helpHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "max_help_handler",
			Command: "help",
			Type:    handlers.TypeCommand,
		},
	}
}

func (h *helpHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	msg := "📋 *Помощь*\n\n" +
		"Бот отправляет сигналы об изменении цен криптовалют.\n\n" +
		"*Команды:*\n" +
		"• /start — главное меню\n" +
		"• /help — эта справка\n" +
		"• /menu — открыть меню\n\n" +
		"*Настройки:*\n" +
		"• Уведомления — вкл/выкл сигналы роста/падения\n" +
		"• Сигналы — пороги срабатывания\n" +
		"• Периоды — временные интервалы анализа"

	keyboard := kb.Keyboard([][]map[string]string{
		{kb.B(kb.Btn.MainMenu, kb.CbMenuMain)},
	})

	return handlers.HandlerResult{Message: msg, Keyboard: keyboard}, nil
}
