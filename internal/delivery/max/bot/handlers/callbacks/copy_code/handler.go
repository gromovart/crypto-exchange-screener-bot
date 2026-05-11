// internal/delivery/max/bot/handlers/callbacks/copy_code/handler.go
package copy_code

import (
	"strings"
	"time"

	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
)

// Handler отправляет значение из callback-данных отдельным сообщением
// для удобного копирования. Callback data формат: "copy_code:{value}"
type Handler struct {
	*base.BaseHandler
}

// New создаёт обработчик copy_code
func New() handlers.Handler {
	return &Handler{
		BaseHandler: base.New("copy_code", "copy_code", handlers.TypeCallback),
	}
}

// Execute разбирает callback data и отправляет значение отдельным сообщением
func (h *Handler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	data := params.Data

	// Извлекаем значение из "copy_code:{value}"
	value := ""
	if strings.HasPrefix(data, "copy_code:") {
		value = strings.TrimPrefix(data, "copy_code:")
	}

	if value == "" {
		return handlers.HandlerResult{
			Message:     "❌ Нет данных для копирования",
			Keyboard:    kb.Keyboard([][]map[string]string{{kb.B(kb.Btn.MainMenu, kb.CbMenuMain)}}),
			EditMessage: false,
		}, nil
	}

	// Отправляем значение новым сообщением (не редактируем старое)
	// Пользователь может удержать и скопировать; сообщение удалится автоматически через 60 сек
	return handlers.HandlerResult{
		Message:         value,
		Keyboard:        nil,
		EditMessage:     false,
		AutoDeleteAfter: 60 * time.Second,
	}, nil
}
