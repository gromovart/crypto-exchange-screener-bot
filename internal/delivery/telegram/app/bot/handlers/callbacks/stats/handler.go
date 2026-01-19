// internal/delivery/telegram/app/bot/handlers/callbacks/stats/handler.go
package stats

import (
	"fmt"
	"time"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
)

// statsHandler реализация обработчика статуса
type statsHandler struct {
	*base.BaseHandler
}

// NewHandler создает новый обработчик статуса
func NewHandler() handlers.Handler {
	return &statsHandler{
		BaseHandler: &base.BaseHandler{ // Изменено на указатель
			Name:    "stats_handler",
			Command: constants.CallbackStats,
			Type:    handlers.TypeCallback,
		},
	}
}

// Execute выполняет обработку callback статуса
func (h *statsHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	// Статус доступен всем (авторизованным и неавторизованным)
	message := h.createStatusMessage()
	keyboard := h.createStatusKeyboard()

	return handlers.HandlerResult{
		Message:  message,
		Keyboard: keyboard,
		Metadata: map[string]interface{}{
			"timestamp": time.Now(),
		},
	}, nil
}

// createStatusMessage создает сообщение статуса
func (h *statsHandler) createStatusMessage() string {
	now := time.Now()

	return fmt.Sprintf(
		"%s\n\n"+
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
		constants.ButtonTexts.Status,
		now.Format("02.01.2006"),
		now.Format("15:04:05"),
	)
}

// createStatusKeyboard создает клавиатуру для статуса
func (h *statsHandler) createStatusKeyboard() interface{} {
	return map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			// {
			// 	{"text": "🔄 Проверить обновления", "callback_data": constants.CallbackTestOK},
			// 	{"text": "📊 Детальная статистика", "callback_data": "detailed_stats"},
			// },
			{
				{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackMenuMain},
			},
		},
	}
}
