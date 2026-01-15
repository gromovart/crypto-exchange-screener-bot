package signal_set_sensitivity

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
)

// signalSetSensitivityHandler реализация обработчика настройки чувствительности
type signalSetSensitivityHandler struct {
	*base.BaseHandler
}

// NewHandler создает новый обработчик настройки чувствительности
func NewHandler() handlers.Handler {
	return &signalSetSensitivityHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "signal_set_sensitivity_handler",
			Command: constants.CallbackSignalSetSensitivity,
			Type:    handlers.TypeCallback,
		},
	}
}

// Execute выполняет обработку callback настройки чувствительности
func (h *signalSetSensitivityHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	if params.User == nil {
		return handlers.HandlerResult{}, fmt.Errorf("пользователь не авторизован")
	}

	// TODO: Когда добавится поле чувствительности в модель User, показывать текущее значение

	message := fmt.Sprintf(
		"🎯 *Настройка чувствительности*\n\n"+
			"Чувствительность определяет, насколько агрессивно система будет реагировать на изменения цены.\n\n"+
			"*Уровни чувствительности:*\n"+
			"• 🔴 Низкая - только сильные сигналы, меньше ложных срабатываний\n"+
			"• 🟡 Средняя - баланс между качеством и количеством сигналов\n"+
			"• 🟢 Высокая - больше сигналов, но возможны ложные срабатывания\n\n"+
			"Выберите уровень чувствительности:",
	)

	// Клавиатура с вариантами чувствительности
	keyboard := map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			{
				{"text": "🔴 Низкая", "callback_data": "sensitivity:low"},
				{"text": "🟡 Средняя", "callback_data": "sensitivity:medium"},
				{"text": "🟢 Высокая", "callback_data": "sensitivity:high"},
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
			"expecting_sensitivity": true,
		},
	}, nil
}
