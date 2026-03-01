// internal/delivery/max/bot/handlers/callbacks/periods_menu/handler.go
package periods_menu

import (
	"fmt"
	"strings"

	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
)

// Handler — обработчик меню периодов
type Handler struct {
	*base.BaseHandler
}

// New создаёт обработчик меню периодов
func New() handlers.Handler {
	return &Handler{
		BaseHandler: base.New("periods_menu", kb.CbPeriodsMenu, handlers.TypeCallback),
	}
}

// Execute выполняет обработку
func (h *Handler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	user := params.User

	activePeriods := map[int]bool{}
	if user != nil {
		for _, p := range user.PreferredPeriods {
			activePeriods[p] = true
		}
	}

	makeBtn := func(label, cb string, mins int) map[string]string {
		prefix := "○ "
		if activePeriods[mins] {
			prefix = "✅ "
		}
		return kb.B(prefix+label, cb)
	}

	// Текущие периоды
	var periodNames []string
	periodMap := map[int]string{1: "1m", 5: "5m", 15: "15m", 30: "30m", 60: "1h", 240: "4h", 1440: "1d"}
	for _, p := range user.PreferredPeriods {
		if name, ok := periodMap[p]; ok {
			periodNames = append(periodNames, name)
		}
	}
	periodsStr := strings.Join(periodNames, ", ")
	if periodsStr == "" {
		periodsStr = "не выбраны"
	}

	msg := fmt.Sprintf(
		"⏱️ *Периоды анализа*\n\n"+
			"Активные: %s\n\n"+
			"Нажмите на период для включения/выключения:",
		periodsStr,
	)

	rows := [][]map[string]string{
		{
			makeBtn(kb.Btn.Period1m, kb.CbPeriod1m, 1),
			makeBtn(kb.Btn.Period5m, kb.CbPeriod5m, 5),
		},
		{
			makeBtn(kb.Btn.Period15m, kb.CbPeriod15m, 15),
			makeBtn(kb.Btn.Period30m, kb.CbPeriod30m, 30),
		},
		{
			makeBtn(kb.Btn.Period1h, kb.CbPeriod1h, 60),
			makeBtn(kb.Btn.Period4h, kb.CbPeriod4h, 240),
		},
		{
			makeBtn(kb.Btn.Period1d, kb.CbPeriod1d, 1440),
		},
		kb.BackRow(kb.CbMenuMain),
	}

	return handlers.HandlerResult{
		Message:     msg,
		Keyboard:    kb.Keyboard(rows),
		EditMessage: params.MessageID > 0,
	}, nil
}
