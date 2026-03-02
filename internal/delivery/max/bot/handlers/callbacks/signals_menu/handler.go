// internal/delivery/max/bot/handlers/callbacks/signals_menu/handler.go
package signals_menu

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
)

// Handler — обработчик меню сигналов
type Handler struct {
	*base.BaseHandler
}

// New создаёт обработчик меню сигналов
func New() handlers.Handler {
	return &Handler{
		BaseHandler: base.New("signals_menu", kb.CbSignalsMenu, handlers.TypeCallback),
	}
}

// Execute выполняет обработку
func (h *Handler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	user := params.User

	growthStr := "❌"
	fallStr := "❌"
	growthThreshold := 2.0
	fallThreshold := 2.0

	if user != nil {
		if user.NotifyGrowth {
			growthStr = "✅"
		}
		if user.NotifyFall {
			fallStr = "✅"
		}
		if user.MinGrowthThreshold > 0 {
			growthThreshold = user.MinGrowthThreshold
		}
		if user.MinFallThreshold > 0 {
			fallThreshold = user.MinFallThreshold
		}
	}

	growthBtn := fmt.Sprintf(kb.Btn.ThresholdFormat, "📈", growthThreshold)
	fallBtn := fmt.Sprintf(kb.Btn.ThresholdFormat, "📉", fallThreshold)

	msg := fmt.Sprintf(
		"📈 *Настройки сигналов*\n\n"+
			"Рост: %s (порог: %.1f%%)\n"+
			"Падение: %s (порог: %.1f%%)\n\n"+
			"Выберите действие:",
		growthStr, growthThreshold, fallStr, fallThreshold,
	)

	rows := [][]map[string]string{
		{
			kb.B(kb.Btn.SignalToggleGrowth+" "+growthStr, kb.CbSignalToggleGrowth),
			kb.B(kb.Btn.SignalToggleFall+" "+fallStr, kb.CbSignalToggleFall),
		},
		{
			kb.B(growthBtn, kb.CbSignalSetGrowthThreshold),
			kb.B(fallBtn, kb.CbSignalSetFallThreshold),
		},
		kb.BackRow(kb.CbMenuMain),
	}

	return handlers.HandlerResult{
		Message:     msg,
		Keyboard:    kb.Keyboard(rows),
		EditMessage: params.MessageID != "",
	}, nil
}
