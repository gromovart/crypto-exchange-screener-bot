// internal/delivery/max/bot/handlers/callbacks/reset_all/handler.go
package reset_all

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/core/domain/users"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
)

// Handler — обработчик полного сброса
type Handler struct {
	*base.BaseHandler
	userService *users.Service
}

// New создаёт обработчик
func New(svc *users.Service) handlers.Handler {
	return &Handler{
		BaseHandler: base.New("reset_all", kb.CbResetAll, handlers.TypeCallback),
		userService: svc,
	}
}

// Execute выполняет полный сброс настроек
func (h *Handler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	user := params.User
	if user == nil {
		return handlers.HandlerResult{Message: "❌ Пользователь не найден"}, nil
	}

	err := h.userService.UpdateSettings(user.ID, map[string]interface{}{
		"notify_growth":         true,
		"notify_fall":           true,
		"min_growth_threshold":  2.0,
		"min_fall_threshold":    2.0,
		"preferred_periods":     []int{5, 15, 30},
		"notifications_enabled": true,
		"notify_continuous":     false,
	})
	if err != nil {
		return handlers.HandlerResult{
			Message:     fmt.Sprintf("❌ Ошибка сброса: %v", err),
			Keyboard:    kb.Keyboard([][]map[string]string{{kb.B(kb.Btn.Back, kb.CbResetMenu)}}),
			EditMessage: params.MessageID > 0,
		}, nil
	}

	msg := "🗑️ *Полный сброс выполнен*\n\n" +
		"✅ Все настройки сброшены к значениям по умолчанию:\n\n" +
		"• Уведомления: включены\n" +
		"• Тип: рост + падение\n" +
		"• Порог роста: 2.0%\n" +
		"• Порог падения: 2.0%\n" +
		"• Периоды: 5m, 15m, 30m"

	rows := [][]map[string]string{
		{kb.B(kb.Btn.MainMenu, kb.CbMenuMain)},
	}

	return handlers.HandlerResult{
		Message:     msg,
		Keyboard:    kb.Keyboard(rows),
		EditMessage: params.MessageID > 0,
	}, nil
}
