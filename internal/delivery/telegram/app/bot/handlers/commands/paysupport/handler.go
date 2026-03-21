// internal/delivery/telegram/app/bot/handlers/commands/paysupport/handler.go
package paysupport

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
)

// paysupportCommandHandler реализация обработчика команды /paysupport
type paysupportCommandHandler struct {
	*base.BaseHandler
}

// NewHandler создает новый обработчик команды /paysupport
func NewHandler() handlers.Handler {
	return &paysupportCommandHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "paysupport_command_handler",
			Command: "paysupport",
			Type:    handlers.TypeCommand,
		},
	}
}

// Execute выполняет обработку команды /paysupport
func (h *paysupportCommandHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	message := fmt.Sprintf(
		"🛟 *Поддержка по платежам*\n\n"+
			"Если у вас возникли вопросы или проблемы с оплатой, вы можете обратиться следующими способами:\n\n"+
			"📧 *Email:* `support@gromovart.ru`\n\n"+
			"💬 *Telegram:* @crypto_exchange_screener\n\n"+
			"⚠️ *Важно:*\n"+
			"• При обращении укажите ваш ID: `%d`\n"+
			"• Если у вас есть чек об оплате (receipt), прикрепите его скриншот\n"+
			"• Мы обрабатываем запросы в течение 24 часов\n\n"+
			"📋 *Частые вопросы:*\n"+
			"• `/buy` — проверить статус подписки\n"+
			"• `/terms` — условия использования и возврата\n\n"+
			"С уважением, команда Crypto Exchange Screener Bot",
		params.User.ID,
	)

	// Убираем URL кнопки для email, оставляем только Telegram
	keyboard := map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			{
				{"text": "💬 Написать в Telegram", "url": "https://t.me/artemgrrr"},
			},
			{
				{"text": "🏠 Главное меню", "callback_data": constants.CallbackMenuMain},
				{"text": "📋 Помощь", "callback_data": constants.CallbackHelp},
			},
		},
	}

	return handlers.HandlerResult{
		Message:  message,
		Keyboard: keyboard,
		Metadata: map[string]interface{}{
			"user_id": params.User.ID,
			"command": "paysupport",
		},
	}, nil
}
