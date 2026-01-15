package signal_history

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
)

// signalHistoryHandler реализация обработчика истории сигналов
type signalHistoryHandler struct {
	*base.BaseHandler
}

// NewHandler создает новый обработчик истории сигналов
func NewHandler() handlers.Handler {
	return &signalHistoryHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "signal_history_handler",
			Command: constants.CallbackSignalHistory,
			Type:    handlers.TypeCallback,
		},
	}
}

// Execute выполняет обработку callback истории сигналов
func (h *signalHistoryHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	if params.User == nil {
		return handlers.HandlerResult{}, fmt.Errorf("пользователь не авторизован")
	}

	message := fmt.Sprintf(
		"📊 *История сигналов*\n\n"+
			"Здесь будет отображаться история полученных сигналов.\n\n"+
			"*Функции в разработке:*\n"+
			"• Просмотр последних сигналов\n"+
			"• Фильтрация по типу и дате\n"+
			"• Статистика эффективности\n"+
			"• Экспорт данных\n\n"+
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
