// /internal/delivery/telegram/app/bot/handlers/callbacks/signal_toggle_fall/handler.go
package signal_toggle_fall

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	signal_settings_svc "crypto-exchange-screener-bot/internal/delivery/telegram/services/signal_settings"
)

// signalToggleFallHandler реализация обработчика переключения уведомлений о падении
type signalToggleFallHandler struct {
	*base.BaseHandler
	service signal_settings_svc.Service
}

// NewHandler создает новый обработчик переключения уведомлений о падении
func NewHandler(service signal_settings_svc.Service) handlers.Handler {
	return &signalToggleFallHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "signal_toggle_fall_handler",
			Command: constants.CallbackSignalToggleFall,
			Type:    handlers.TypeCallback,
		},
		service: service,
	}
}

// Execute выполняет обработку callback переключения уведомлений о падении
func (h *signalToggleFallHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	if params.User == nil {
		return handlers.HandlerResult{}, fmt.Errorf("пользователь не авторизован")
	}

	// Подготавливаем параметры для сервиса
	serviceParams := signal_settings_svc.SignalSettingsParams{
		Action: "toggle_fall",
		UserID: params.User.ID,
		ChatID: params.ChatID,
		Value:  !params.User.NotifyFall, // Переключаем на противоположное
	}

	// Вызываем сервис
	result, err := h.service.Exec(serviceParams)
	if err != nil {
		return handlers.HandlerResult{}, fmt.Errorf("ошибка в сервисе настройки сигналов: %w", err)
	}

	// Создаем сообщение с результатом
	message := fmt.Sprintf(
		"📉 *Настройки сигналов падения*\n\n%s\n\n"+
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
			"user_id":       params.User.ID,
			"notify_fall":   result.NewValue,
			"updated_field": result.UpdatedField,
		},
	}, nil
}
