// internal/delivery/max/bot/handlers/callbacks/signal_set_growth_threshold/handler.go
package signal_set_growth_threshold

import (
	"fmt"
	"strconv"
	"strings"

	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
	signalSvc "crypto-exchange-screener-bot/internal/delivery/telegram/services/signal_settings"
)

// Handler — обработчик установки порога роста
type Handler struct {
	*base.BaseHandler
	service signalSvc.Service
}

// New создаёт обработчик
func New(svc signalSvc.Service) handlers.Handler {
	return &Handler{
		BaseHandler: base.New("signal_set_growth_threshold", kb.CbSignalSetGrowthThreshold, handlers.TypeCallback),
		service:     svc,
	}
}

// Execute выполняет обработку
// Если Data содержит значение (with_params), устанавливает порог.
// Иначе показывает кнопки с вариантами.
func (h *Handler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	user := params.User
	if user == nil {
		return handlers.HandlerResult{Message: "❌ Пользователь не найден"}, nil
	}

	// Проверяем, есть ли значение в data (формат: "signal_set_growth_threshold:2.5")
	if strings.Contains(params.Data, ":") {
		parts := strings.SplitN(params.Data, ":", 2)
		if len(parts) == 2 {
			val, err := strconv.ParseFloat(parts[1], 64)
			if err == nil {
				result, err := h.service.Exec(signalSvc.SignalSettingsParams{
					Action: "set_growth_threshold",
					UserID: user.ID,
					Value:  val,
				})
				if err != nil {
					return handlers.HandlerResult{
						Message:     fmt.Sprintf("❌ Ошибка: %v", err),
						Keyboard:    kb.Keyboard([][]map[string]string{{kb.B(kb.Btn.Back, kb.CbSignalsMenu)}}),
						EditMessage: params.MessageID != "",
					}, nil
				}
				return handlers.HandlerResult{
					Message:     fmt.Sprintf("📈 *Порог роста*\n\n%s", result.Message),
					Keyboard:    kb.Keyboard([][]map[string]string{{kb.B(kb.Btn.Back, kb.CbSignalsMenu)}}),
					EditMessage: params.MessageID != "",
				}, nil
			}
		}
	}

	// Показываем варианты порогов
	current := user.MinGrowthThreshold
	msg := fmt.Sprintf(
		"📈 *Порог роста*\n\nТекущий порог: %.1f%%\n\nВыберите новое значение:",
		current,
	)

	thresholds := []float64{0.5, 1.0, 1.5, 2.0, 3.0, 5.0, 7.0, 10.0}
	var rows [][]map[string]string

	var row []map[string]string
	for i, t := range thresholds {
		marker := ""
		if t == current {
			marker = "✅ "
		}
		btn := kb.B(fmt.Sprintf("%s%.1f%%", marker, t),
			fmt.Sprintf("%s:%.1f", kb.CbSignalSetGrowthThreshold, t))
		row = append(row, btn)
		if len(row) == 2 || i == len(thresholds)-1 {
			rows = append(rows, row)
			row = nil
		}
	}
	rows = append(rows, kb.BackRow(kb.CbSignalsMenu))

	return handlers.HandlerResult{
		Message:     msg,
		Keyboard:    kb.Keyboard(rows),
		EditMessage: params.MessageID != "",
	}, nil
}
