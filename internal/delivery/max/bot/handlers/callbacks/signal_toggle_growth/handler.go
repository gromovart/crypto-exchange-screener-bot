// internal/delivery/max/bot/handlers/callbacks/signal_toggle_growth/handler.go
package signal_toggle_growth

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
	signalSvc "crypto-exchange-screener-bot/internal/delivery/telegram/services/signal_settings"
)

// Handler — обработчик переключения сигналов роста
type Handler struct {
	*base.BaseHandler
	service signalSvc.Service
}

// New создаёт обработчик
func New(svc signalSvc.Service) handlers.Handler {
	return &Handler{
		BaseHandler: base.New("signal_toggle_growth", kb.CbSignalToggleGrowth, handlers.TypeCallback),
		service:     svc,
	}
}

// Execute выполняет обработку
func (h *Handler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	user := params.User
	if user == nil {
		return handlers.HandlerResult{Message: "❌ Пользователь не найден"}, nil
	}

	result, err := h.service.Exec(signalSvc.SignalSettingsParams{
		Action: "toggle_growth",
		UserID: user.ID,
	})
	if err != nil {
		return handlers.HandlerResult{
			Message:     fmt.Sprintf("❌ Ошибка: %v", err),
			Keyboard:    kb.Keyboard([][]map[string]string{{kb.B(kb.Btn.Back, kb.CbSignalsMenu)}}),
			EditMessage: params.MessageID != "",
		}, nil
	}

	msg := fmt.Sprintf("📈 *Сигналы роста*\n\n%s", result.Message)

	return handlers.HandlerResult{
		Message:     msg,
		Keyboard:    kb.Keyboard([][]map[string]string{{kb.B(kb.Btn.Back, kb.CbSignalsMenu)}}),
		EditMessage: params.MessageID != "",
	}, nil
}
