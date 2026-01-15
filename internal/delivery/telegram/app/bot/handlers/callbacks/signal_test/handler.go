package signal_test

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
)

// signalTestHandler реализация обработчика тестового сигнала
type signalTestHandler struct {
	*base.BaseHandler
}

// NewHandler создает новый обработчик тестового сигнала
func NewHandler() handlers.Handler {
	return &signalTestHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "signal_test_handler",
			Command: constants.CallbackSignalTest,
			Type:    handlers.TypeCallback,
		},
	}
}

// Execute выполняет обработку callback тестового сигнала
func (h *signalTestHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	if params.User == nil {
		return handlers.HandlerResult{}, fmt.Errorf("пользователь не авторизован")
	}

	message := fmt.Sprintf(
		"⚡ *Тестовый сигнал*\n\n"+
			"Проверка работы системы уведомлений.\n\n"+
			"*Что будет проверено:*\n"+
			"• Доставка сообщений\n"+
			"• Форматирование сигналов\n"+
			"• Работа с порогами\n"+
			"• Тихие часы\n\n"+
			"Эта функция появится в следующем обновлении.",
	)

	// Простая клавиатура
	keyboard := map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			{
				{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackSignalsMenu},
			},
		},
	}

	return handlers.HandlerResult{
		Message:  message,
		Keyboard: keyboard,
		Metadata: map[string]interface{}{
			"user_id": params.User.ID,
		},
	}, nil
}
