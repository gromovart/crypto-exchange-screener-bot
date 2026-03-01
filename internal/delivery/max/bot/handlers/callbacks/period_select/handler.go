// internal/delivery/max/bot/handlers/callbacks/period_select/handler.go
package period_select

import (
	"fmt"
	"strings"

	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
	signalSvc "crypto-exchange-screener-bot/internal/delivery/telegram/services/signal_settings"
)

// Handler — обработчик выбора периода (вызывается из кнопок period_1m/5m/...)
type Handler struct {
	*base.BaseHandler
	service signalSvc.Service
}

// New создаёт обработчик выбора периода
func New(svc signalSvc.Service) handlers.Handler {
	return &Handler{
		BaseHandler: base.New("period_select", "period_*", handlers.TypeCallback),
		service:     svc,
	}
}

// Execute выполняет обработку
// params.Data — например "period_1m", "period_5m", ...
func (h *Handler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	user := params.User
	if user == nil {
		return handlers.HandlerResult{Message: "❌ Пользователь не найден"}, nil
	}

	// Извлекаем период из Data ("period_5m" -> "5m")
	period := strings.TrimPrefix(params.Data, "period_")

	result, err := h.service.Exec(signalSvc.SignalSettingsParams{
		Action: "select_period",
		UserID: user.ID,
		Value:  period,
	})
	if err != nil {
		return handlers.HandlerResult{
			Message:     fmt.Sprintf("❌ Ошибка: %v", err),
			Keyboard:    kb.Keyboard([][]map[string]string{{kb.B(kb.Btn.Back, kb.CbPeriodsMenu)}}),
			EditMessage: params.MessageID > 0,
		}, nil
	}

	msg := fmt.Sprintf("⏱️ *Периоды*\n\n%s", result.Message)

	return handlers.HandlerResult{
		Message:     msg,
		Keyboard:    kb.Keyboard([][]map[string]string{{kb.B(kb.Btn.Back, kb.CbPeriodsMenu)}}),
		EditMessage: params.MessageID > 0,
	}, nil
}
