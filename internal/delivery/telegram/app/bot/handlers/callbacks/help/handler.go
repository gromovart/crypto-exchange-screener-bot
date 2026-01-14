// internal/delivery/telegram/app/bot/handlers/callbacks/help/handler.go
package help

import (
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	"fmt"
)

// helpHandler реализация обработчика помощи
type helpHandler struct {
	*base.BaseHandler
}

// NewHandler создает новый обработчик помощи
func NewHandler() handlers.Handler {
	return &helpHandler{
		BaseHandler: &base.BaseHandler{ // Изменено на указатель
			Name:    "help_handler",
			Command: constants.CallbackHelp,
			Type:    handlers.TypeCallback,
		},
	}
}

// Execute выполняет обработку callback помощи
func (h *helpHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
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

// createHelpMessage создает сообщение помощи
func (h *helpHandler) createHelpMessage() string {
	return fmt.Sprintf(
		"%s\n\n"+
			"*Основные команды:*\n"+
			"/start - Начало работы\n"+
			"/profile - Ваш профиль\n"+
			"/settings - Настройки профиля\n"+
			"%s - Эта справка\n\n"+
			"*Управление уведомлениями:*\n"+
			"%s - Настройки уведомлений\n"+
			"%s - Настройка порогов\n"+
			"%s - Настройка периодов\n\n"+
			"*Как работает бот:*\n"+
			"1️⃣ Анализирует рынок в реальном времени\n"+
			"2️⃣ Обнаруживает сильные движения цен\n"+
			"3️⃣ Отправляет уведомления при превышении порогов\n"+
			"4️⃣ Считает сигналы по периодам\n\n"+
			"*Настройки по умолчанию:*\n"+
			"📈 Рост: 2.0%%\n"+
			"📉 Падение: 2.0%%\n"+
			"⏱️ Периоды: 5м, 15м, 30м\n"+
			"🔔 Уведомления: включены\n\n"+
			"Используйте команды выше или меню для настройки.",
		constants.ButtonTexts.Help,
		constants.ButtonTexts.Help,
		constants.MenuButtonTexts.Notifications,
		constants.AuthButtonTexts.Thresholds,
		constants.MenuButtonTexts.Periods,
	)
}

// createHelpKeyboard создает клавиатуру для помощи
func (h *helpHandler) createHelpKeyboard() interface{} {
	return map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			{
				{"text": "📚 Полная документация", "url": "https://github.com/your-repo/docs"},
				{"text": "📧 Поддержка", "url": "https://t.me/support_bot"},
			},
			{
				{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackMenuMain},
			},
		},
	}
}
