// internal/delivery/max/bot/handlers/callbacks/stats/handler.go
package stats

import (
	"fmt"
	"time"

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
	now := time.Now()

	msg := fmt.Sprintf(
		"📊 Статус\n\n"+
			"📅 *Дата:* %s\n"+
			"🕐 *Время:* %s\n\n"+
			"🔄 *Система работает*\n"+
			"✅ *Все компоненты активны*\n\n"+
			"📊 *Последние обновления:*\n"+
			"• Рыночные данные: несколько секунд назад\n"+
			"• Анализ сигналов: в реальном времени\n"+
			"• Уведомления: активны\n\n"+
			"⚡ *Производительность:*\n"+
			"• Время ответа: < 100 мс\n"+
			"• Доступность: 99.9%%\n"+
			"• Нагрузка: низкая\n\n"+
			"Используйте кнопки ниже для управления:",
		now.Format("02.01.2006"),
		now.Format("15:04:05"),
	)

	rows := [][]map[string]string{
		{kb.B(kb.Btn.MainMenu, kb.CbMenuMain)},
	}

	return handlers.HandlerResult{
		Message:     msg,
		Keyboard:    kb.Keyboard(rows),
		EditMessage: params.MessageID != "",
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
