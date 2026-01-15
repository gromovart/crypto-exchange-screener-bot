package signal_set_quiet_hours

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
)

// signalSetQuietHoursHandler реализация обработчика настройки тихих часов
type signalSetQuietHoursHandler struct {
	*base.BaseHandler
}

// NewHandler создает новый обработчик настройки тихих часов
func NewHandler() handlers.Handler {
	return &signalSetQuietHoursHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "signal_set_quiet_hours_handler",
			Command: constants.CallbackSignalSetQuietHours,
			Type:    handlers.TypeCallback,
		},
	}
}

// Execute выполняет обработку callback настройки тихих часов
func (h *signalSetQuietHoursHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	if params.User == nil {
		return handlers.HandlerResult{}, fmt.Errorf("пользователь не авторизован")
	}

	message := fmt.Sprintf(
		"⏱️ *Настройка тихих часов*\n\n"+
			"Текущие тихие часы: *%02d:00 - %02d:00*\n\n"+
			"В тихие часы уведомления не отправляются.\n\n"+
			"*Выберите действие:*",
		params.User.QuietHoursStart,
		params.User.QuietHoursEnd,
	)

	// Клавиатура для настройки тихих часов
	keyboard := map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			{
				{"text": "🕐 Установить начало", "callback_data": "quiet_hours_set_start"},
				{"text": "🕙 Установить конец", "callback_data": "quiet_hours_set_end"},
			},
			{
				{"text": "✅ Включить тихие часы", "callback_data": "quiet_hours_enable"},
				{"text": "❌ Выключить тихие часы", "callback_data": "quiet_hours_disable"},
			},
			{
				{"text": "🔄 Сбросить к умолчанию", "callback_data": "quiet_hours_reset"},
			},
			{
				{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackSignalsMenu},
			},
		},
	}

	return handlers.HandlerResult{
		Message:  message,
		Keyboard: keyboard,
		Metadata: map[string]interface{}{
			"user_id":          params.User.ID,
			"quiet_hours_start": params.User.QuietHoursStart,
			"quiet_hours_end":   params.User.QuietHoursEnd,
		},
	}, nil
}
