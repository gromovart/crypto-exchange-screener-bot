// internal/delivery/max/bot/handlers/callbacks/thresholds_menu/handler.go
package thresholds_menu

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
)

// Handler — обработчик меню порогов
type Handler struct {
	*base.BaseHandler
}

// New создаёт обработчик
func New() handlers.Handler {
	return &Handler{
		BaseHandler: base.New("thresholds_menu", kb.CbThresholdsMenu, handlers.TypeCallback),
	}
}

// Execute выполняет обработку
func (h *Handler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	user := params.User

	growthThreshold := 2.0
	fallThreshold := 2.0

	if user != nil {
		if user.MinGrowthThreshold > 0 {
			growthThreshold = user.MinGrowthThreshold
		}
		if user.MinFallThreshold > 0 {
			fallThreshold = user.MinFallThreshold
		}
	}

	msg := fmt.Sprintf(
		"🎯 *Пороги сигналов*\n\n"+
			"📈 Порог роста: %.1f%%\n"+
			"📉 Порог падения: %.1f%%\n\n"+
			"Сигнал отправляется когда цена изменяется на указанный %% за выбранный период.",
		growthThreshold, fallThreshold,
	)

	growthBtn := fmt.Sprintf(kb.Btn.ThresholdFormat, "📈", growthThreshold)
	fallBtn := fmt.Sprintf(kb.Btn.ThresholdFormat, "📉", fallThreshold)

	rows := [][]map[string]string{
		{
			kb.B(growthBtn, kb.CbSignalSetGrowthThreshold),
			kb.B(fallBtn, kb.CbSignalSetFallThreshold),
		},
		kb.BackRow(kb.CbMenuMain),
	}

	return handlers.HandlerResult{
		Message:     msg,
		Keyboard:    kb.Keyboard(rows),
		EditMessage: params.MessageID > 0,
	}, nil
}
