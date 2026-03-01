// internal/delivery/max/bot/handlers/callbacks/with_params/handler.go
package with_params

import (
	"strings"

	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
)

// Handler — fallback обработчик для callback с параметрами
// Используется когда callback_data содержит ":" (параметры)
type Handler struct {
	*base.BaseHandler
}

// New создаёт обработчик
func New() handlers.Handler {
	return &Handler{
		BaseHandler: base.New("with_params", kb.CbWithParams, handlers.TypeCallback),
	}
}

// Execute выполняет обработку
// params.Data может быть "base_action:value"
func (h *Handler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	// Пытаемся разобрать базовое действие
	data := params.Data
	if strings.Contains(data, ":") {
		parts := strings.SplitN(data, ":", 2)
		baseAction := parts[0]
		_ = parts[1] // value

		switch baseAction {
		case kb.CbSignalSetGrowthThreshold:
			// Перенаправляем к обработчику порога роста
			// Возвращаем сообщение — роутер должен был перехватить раньше
			return handlers.HandlerResult{
				Message:     "📈 Настройка порога роста",
				Keyboard:    kb.Keyboard([][]map[string]string{{kb.B(kb.Btn.Back, kb.CbSignalsMenu)}}),
				EditMessage: params.MessageID > 0,
			}, nil

		case kb.CbSignalSetFallThreshold:
			return handlers.HandlerResult{
				Message:     "📉 Настройка порога падения",
				Keyboard:    kb.Keyboard([][]map[string]string{{kb.B(kb.Btn.Back, kb.CbSignalsMenu)}}),
				EditMessage: params.MessageID > 0,
			}, nil
		}
	}

	// Неизвестный callback
	return handlers.HandlerResult{
		Message:     "❓ Неизвестная команда",
		Keyboard:    kb.Keyboard([][]map[string]string{{kb.B(kb.Btn.MainMenu, kb.CbMenuMain)}}),
		EditMessage: params.MessageID > 0,
	}, nil
}
