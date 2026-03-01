// internal/delivery/max/bot/handlers/callbacks/reset_menu/handler.go
package reset_menu

import (
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
)

// Handler — обработчик меню сброса
type Handler struct {
	*base.BaseHandler
}

// New создаёт обработчик
func New() handlers.Handler {
	return &Handler{
		BaseHandler: base.New("reset_menu", kb.CbResetMenu, handlers.TypeCallback),
	}
}

// Execute выполняет обработку
func (h *Handler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	msg := "🔄 *Сброс настроек*\n\n" +
		"⚠️ Выберите что сбросить:\n\n" +
		"• *Настройки* — сбрасывает пороги, периоды и типы уведомлений\n" +
		"• *Всё* — полный сброс всех настроек"

	rows := [][]map[string]string{
		{kb.B(kb.Btn.ResetSettings, kb.CbResetSettings)},
		{kb.B(kb.Btn.ResetAll, kb.CbResetAll)},
		kb.BackRow(kb.CbMenuMain),
	}

	return handlers.HandlerResult{
		Message:     msg,
		Keyboard:    kb.Keyboard(rows),
		EditMessage: params.MessageID > 0,
	}, nil
}
