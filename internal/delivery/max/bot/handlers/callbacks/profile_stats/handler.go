// internal/delivery/max/bot/handlers/callbacks/profile_stats/handler.go
package profile_stats

import (
	"fmt"
	"time"

	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
)

// Handler — обработчик статистики профиля
type Handler struct {
	*base.BaseHandler
}

// New создаёт обработчик
func New() handlers.Handler {
	return &Handler{
		BaseHandler: base.New("profile_stats", kb.CbProfileStats, handlers.TypeCallback),
	}
}

// Execute выполняет обработку
func (h *Handler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	user := params.User
	if user == nil {
		return handlers.HandlerResult{
			Message:     "❌ Пользователь не найден",
			Keyboard:    kb.Keyboard([][]map[string]string{{kb.B(kb.Btn.Back, kb.CbProfileMain)}}),
			EditMessage: params.MessageID != "",
		}, nil
	}

	daysInSystem := int(time.Since(user.CreatedAt).Hours() / 24)

	notifyStr := "❌"
	if user.NotificationsEnabled {
		notifyStr = "✅"
	}

	lastLogin := "—"
	if !user.LastLoginAt.IsZero() {
		lastLogin = user.LastLoginAt.Format("02.01.2006 15:04")
	}

	msg := fmt.Sprintf(
		"📊 Статистика\n\n"+
			"👤 @%s\n"+
			"🆔 ID: %d\n\n"+
			"📈 Сигналов сегодня: %d / %d\n"+
			"🔔 Уведомления: %s\n"+
			"⏰ В системе: %d дней\n"+
			"🕐 Последний вход: %s",
		user.Username,
		user.ID,
		user.SignalsToday,
		user.MaxSignalsPerDay,
		notifyStr,
		daysInSystem,
		lastLogin,
	)

	rows := [][]map[string]string{
		{kb.B(kb.Btn.Back, kb.CbProfileMain)},
	}

	return handlers.HandlerResult{
		Message:     msg,
		Keyboard:    kb.Keyboard(rows),
		EditMessage: params.MessageID != "",
	}, nil
}
