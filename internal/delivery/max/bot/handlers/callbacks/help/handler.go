// internal/delivery/max/bot/handlers/callbacks/help/handler.go
package help

import (
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	cmdHelp "crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/commands/help"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
)

// Handler — обработчик помощи (callback)
type Handler struct {
	*base.BaseHandler
}

// New создаёт обработчик помощи
func New() handlers.Handler {
	return &Handler{
		BaseHandler: base.New("help_cb", kb.CbHelp, handlers.TypeCallback),
	}
}

// Execute выполняет обработку
func (h *Handler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	rows := [][]map[string]string{
		{kb.BUrl("📚 Документация", "https://teletype.in/@gromovart/pj2UIVlmr55")},
		{kb.B(kb.Btn.MainMenu, kb.CbMenuMain)},
	}
	return handlers.HandlerResult{
		Message:     cmdHelp.HelpText(),
		Keyboard:    kb.Keyboard(rows),
		EditMessage: params.MessageID != "",
	}, nil
}
