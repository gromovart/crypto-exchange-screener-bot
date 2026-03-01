// internal/delivery/max/bot/handlers/callbacks/stats/handler.go
package stats

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
)

// Handler — обработчик статуса бота
type Handler struct {
	*base.BaseHandler
}

// New создаёт обработчик статуса
func New() handlers.Handler {
	return &Handler{
		BaseHandler: base.New("stats", kb.CbStats, handlers.TypeCallback),
	}
}

// Execute выполняет обработку
func (h *Handler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	user := params.User

	notifyStatus := "❌ Выключены"
	growthStr := "❌"
	fallStr := "❌"
	growthThreshold := 2.0
	fallThreshold := 2.0
	periodsStr := "не заданы"
	signalsToday := 0
	maxSignals := 0
	tier := "Free"

	if user != nil {
		if user.NotificationsEnabled {
			notifyStatus = "✅ Включены"
		}
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
		if len(user.PreferredPeriods) > 0 {
			var parts []string
			for _, p := range user.PreferredPeriods {
				parts = append(parts, formatPeriod(p))
			}
			periodsStr = joinStrings(parts, ", ")
		}
		signalsToday = user.SignalsToday
		maxSignals = user.MaxSignalsPerDay
		if user.SubscriptionTier != "" {
			tier = user.SubscriptionTier
		}
	}

	msg := fmt.Sprintf(
		"📊 *Статус бота*\n\n"+
			"🔔 Уведомления: %s\n"+
			"📈 Рост: %s (порог: %.1f%%)\n"+
			"📉 Падение: %s (порог: %.1f%%)\n"+
			"⏱️ Периоды: %s\n\n"+
			"📧 Сигналов сегодня: %d / %d\n"+
			"💎 Подписка: %s",
		notifyStatus,
		growthStr, growthThreshold,
		fallStr, fallThreshold,
		periodsStr,
		signalsToday, maxSignals,
		tier,
	)

	rows := [][]map[string]string{
		{kb.B(kb.Btn.Settings, kb.CbSettingsMain)},
		{kb.B(kb.Btn.MainMenu, kb.CbMenuMain)},
	}

	return handlers.HandlerResult{
		Message:     msg,
		Keyboard:    kb.Keyboard(rows),
		EditMessage: params.MessageID > 0,
	}, nil
}

func formatPeriod(mins int) string {
	switch mins {
	case 1:
		return "1m"
	case 5:
		return "5m"
	case 15:
		return "15m"
	case 30:
		return "30m"
	case 60:
		return "1h"
	case 240:
		return "4h"
	case 1440:
		return "1d"
	default:
		return fmt.Sprintf("%dm", mins)
	}
}

func joinStrings(ss []string, sep string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
