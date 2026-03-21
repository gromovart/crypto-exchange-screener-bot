// internal/delivery/telegram/app/bot/handlers/commands/help/handler.go
package help

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
)

// helpCommandHandler реализация обработчика команды /help
type helpCommandHandler struct {
	*base.BaseHandler
}

// NewHandler создает новый обработчик команды /help
func NewHandler() handlers.Handler {
	return &helpCommandHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "help_command_handler",
			Command: "help",
			Type:    handlers.TypeCommand,
		},
	}
}

// Execute выполняет обработку команды /help
func (h *helpCommandHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	message := h.createHelpMessage()
	keyboard := h.createHelpKeyboard()

	return handlers.HandlerResult{
		Message:  message,
		Keyboard: keyboard,
		Metadata: map[string]interface{}{
			"user_id": params.User.ID,
		},
	}, nil
}

// createHelpMessage создает сообщение помощи для команды
func (h *helpCommandHandler) createHelpMessage() string {
	return fmt.Sprintf(
		"📋 *Помощь*\n\n" +
			"*Основные команды:*\n" +
			"/start - Начало работы\n" +
			"/profile - Ваш профиль\n" +
			"/settings - Настройки профиля\n" +
			"/help - Эта справка\n\n" +

			"*Управление уведомлениями:*\n" +
			"/notifications - Настройки уведомлений\n" +
			"/thresholds - Настройка порогов\n" +
			"/periods - Настройка периодов\n\n" +

			"*Подписка и платежи:*\n" +
			"/buy - Купить подписку\n" +
			"/paysupport - Поддержка по платежам\n" +
			"/terms - Условия использования\n\n" +

			"*Как работает бот:*\n" +
			"1️⃣ Анализирует рынок в реальном времени\n" +
			"2️⃣ Обнаруживает сильные движения цен\n" +
			"3️⃣ Отправляет уведомления при превышении порогов\n" +
			"4️⃣ Считает сигналы по периодам\n\n" +

			"*Настройки по умолчанию:*\n" +
			"📈 Рост: 2.0%%\n" +
			"📉 Падение: 2.0%%\n" +
			"⏱️ Периоды: 5м, 15м, 30м\n" +
			"🔔 Уведомления: включены\n\n" +

			"*Поддержка:*\n" +
			"📧 Email: support@gromovart.ru\n" +
			"💬 Telegram: @crypto_exchange_screener\n\n" +

			"Используйте команды выше или меню для настройки.",
	)
}

// createHelpKeyboard создает клавиатуру для помощи
func (h *helpCommandHandler) createHelpKeyboard() interface{} {
	return map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			{
				{"text": constants.ButtonTexts.Documentation, "url": "https://github.com/your-repo/docs"},
				{"text": constants.ButtonTexts.Support, "url": "https://t.me/support_bot"},
			},
			{
				{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackMenuMain},
			},
		},
	}
}
