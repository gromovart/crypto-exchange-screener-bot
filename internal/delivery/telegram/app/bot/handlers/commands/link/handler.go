// internal/delivery/telegram/app/bot/handlers/commands/link/handler.go
package link

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/core/domain/users"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
)

// linkCommandHandler — обработчик команды /link
// Генерирует одноразовый код для привязки MAX-аккаунта к этому Telegram-аккаунту.
type linkCommandHandler struct {
	*base.BaseHandler
	userService *users.Service
}

// NewHandler создаёт обработчик /link
func NewHandler(userService *users.Service) handlers.Handler {
	return &linkCommandHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "link_command_handler",
			Command: "link",
			Type:    handlers.TypeCommand,
		},
		userService: userService,
	}
}

// Execute генерирует код привязки и отображает его пользователю
func (h *linkCommandHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	if params.User == nil {
		return handlers.HandlerResult{Message: "❌ Пользователь не найден"}, nil
	}

	if h.userService == nil {
		return handlers.HandlerResult{Message: "❌ Сервис недоступен"}, nil
	}

	code, err := h.userService.GenerateLinkCode(params.User.TelegramID)
	if err != nil {
		return handlers.HandlerResult{Message: "❌ Не удалось создать код"}, fmt.Errorf("link: GenerateLinkCode: %w", err)
	}

	msg := fmt.Sprintf(
		"🔗 *Привязка MAX-аккаунта*\n\n"+
			"Ваш код привязки:\n\n"+
			"```\n%s\n```\n\n"+
			"Этот код действителен *15 минут*.\n\n"+
			"📱 *Как использовать:*\n"+
			"1. Откройте бота в MAX мессенджере\n"+
			"2. Нажмите «🔗 Привязать Telegram»\n"+
			"3. Введите код `%s`\n\n"+
			"После привязки ваша подписка Telegram будет доступна в MAX.",
		code, code,
	)

	keyboard := map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			{
				{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackMenuMain},
			},
		},
	}

	return handlers.HandlerResult{
		Message:  msg,
		Keyboard: keyboard,
	}, nil
}
