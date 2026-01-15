package signal_toggle_growth

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	signal_settings_svc "crypto-exchange-screener-bot/internal/delivery/telegram/services/signal_settings"
)

// signalToggleGrowthHandler реализация обработчика переключения уведомлений о росте
type signalToggleGrowthHandler struct {
	*base.BaseHandler
	service signal_settings_svc.Service
}

// NewHandler создает новый обработчик переключения уведомлений о росте
func NewHandler(service signal_settings_svc.Service) handlers.Handler {
	return &signalToggleGrowthHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "signal_toggle_growth_handler",
			Command: constants.CallbackSignalToggleGrowth,
			Type:    handlers.TypeCallback,
		},
		service: service,
	}
}

// Execute выполняет обработку callback переключения уведомлений о росте
func (h *signalToggleGrowthHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	if params.User == nil {
		return handlers.HandlerResult{}, fmt.Errorf("пользователь не авторизован")
	}

	// Подготавливаем параметры для сервиса
	serviceParams := signal_settings_svc.SignalSettingsParams{
		Action: "toggle_growth",
		UserID: params.User.ID,
		ChatID: params.ChatID,
		Value:  !params.User.NotifyGrowth, // Переключаем на противоположное
	}

	// Вызываем сервис
	result, err := h.service.Exec(serviceParams)
	if err != nil {
		return handlers.HandlerResult{}, fmt.Errorf("ошибка в сервисе настройки сигналов: %w", err)
	}

	// Создаем сообщение с результатом
	message := fmt.Sprintf(
		"📈 *Настройки сигналов роста*\n\n%s\n\n"+
			"Для изменения других настроек вернитесь в меню сигналов.",
		result.Message,
	)

	// Создаем клавиатуру
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
			"user_id":         params.User.ID,
			"notify_growth":   result.NewValue,
			"updated_field":   result.UpdatedField,
		},
	}, nil
}
