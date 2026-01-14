// internal/delivery/telegram/app/bot/handlers/commands/thresholds/handler.go
package thresholds

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
)

// thresholdsCommandHandler реализация обработчика команды /thresholds
type thresholdsCommandHandler struct {
	*base.BaseHandler
}

// NewHandler создает новый обработчик команды /thresholds
func NewHandler() handlers.Handler {
	return &thresholdsCommandHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "thresholds_command_handler",
			Command: "thresholds",
			Type:    handlers.TypeCommand,
		},
	}
}

// Execute выполняет обработку команды /thresholds
func (h *thresholdsCommandHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	if params.User == nil {
		return handlers.HandlerResult{}, fmt.Errorf("пользователь не авторизован")
	}

	message := h.createThresholdsMessage(params.User)
	keyboard := h.createThresholdsKeyboard(params.User)

	return handlers.HandlerResult{
		Message:  message,
		Keyboard: keyboard,
		Metadata: map[string]interface{}{
			"user_id": params.User.ID,
		},
	}, nil
}

// createThresholdsMessage создает сообщение для команды /thresholds
func (h *thresholdsCommandHandler) createThresholdsMessage(user *models.User) string {
	return fmt.Sprintf(
		"%s\n\n"+
			"Текущие пороги:\n\n"+
			"📈 Минимальный рост: %.2f%%\n"+
			"📉 Минимальное падение: %.2f%%\n\n"+
			"Порог определяет, насколько сильным должно быть движение,\n"+
			"чтобы бот отправил вам уведомление.\n\n"+
			"Рекомендуемые значения: 2.0%% - 5.0%%\n\n"+
			"Выберите порог для изменения:",
		constants.AuthButtonTexts.Thresholds,
		user.MinGrowthThreshold,
		user.MinFallThreshold,
	)
}

// createThresholdsKeyboard создает клавиатуру для команды /thresholds
func (h *thresholdsCommandHandler) createThresholdsKeyboard(user *models.User) interface{} {
	return map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			{
				{"text": fmt.Sprintf("📈 Рост: %.2f%%", user.MinGrowthThreshold),
					"callback_data": constants.CallbackThresholdGrowth},
			},
			{
				{"text": fmt.Sprintf("📉 Падение: %.2f%%", user.MinFallThreshold),
					"callback_data": constants.CallbackThresholdFall},
			},
			{
				{"text": "2.0% (по умолчанию)", "callback_data": "threshold_2"},
				{"text": "3.0% (средний)", "callback_data": "threshold_3"},
			},
			{
				{"text": "5.0% (строгий)", "callback_data": "threshold_5"},
				{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackMenuMain},
			},
		},
	}
}
