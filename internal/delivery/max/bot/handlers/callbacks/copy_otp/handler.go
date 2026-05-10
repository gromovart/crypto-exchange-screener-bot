// internal/delivery/max/bot/handlers/callbacks/copy_otp/handler.go
package copy_otp

import (
	"fmt"
	"strings"

	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
)

// Handler редактирует OTP-сообщение на месте, чтобы показать код для копирования.
// Использует EditMessage: true — оригинальное сообщение сохраняет тот же mid,
// поэтому оно корректно удаляется после успешной верификации.
// Callback data формат: "copy_otp:{code}"
type Handler struct {
	*base.BaseHandler
}

// New создаёт обработчик copy_otp
func New() handlers.Handler {
	return &Handler{
		BaseHandler: base.New("copy_otp", kb.CbCopyOTP, handlers.TypeCallback),
	}
}

// Execute редактирует OTP-сообщение, отображая код крупно для удобного копирования
func (h *Handler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	data := params.Data

	// Извлекаем код из "copy_otp:{code}"
	code := ""
	if strings.HasPrefix(data, "copy_otp:") {
		code = strings.TrimPrefix(data, "copy_otp:")
	}

	if code == "" {
		return handlers.HandlerResult{
			Message:     "❌ Код не найден",
			Keyboard:    kb.Keyboard([][]map[string]string{{kb.B(kb.Btn.MainMenu, kb.CbMenuMain)}}),
			EditMessage: true,
		}, nil
	}

	msg := fmt.Sprintf(
		"🔐 Код для входа:\n\n%s\n\nСкопируйте код и введите на сайте.\nСообщение будет удалено автоматически.",
		code,
	)

	return handlers.HandlerResult{
		Message:     msg,
		Keyboard:    nil, // убираем кнопку — код уже показан
		EditMessage: true, // редактируем оригинальное OTP-сообщение (mid сохраняется)
	}, nil
}
