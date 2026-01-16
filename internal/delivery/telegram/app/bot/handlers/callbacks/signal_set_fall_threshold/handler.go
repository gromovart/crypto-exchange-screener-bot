// internal/delivery/telegram/app/bot/handlers/callbacks/signal_set_fall_threshold/handler.go
package signal_set_fall_threshold

import (
	"fmt"
	"strconv"
	"strings"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	signal_settings_svc "crypto-exchange-screener-bot/internal/delivery/telegram/services/signal_settings"
)

// signalSetFallThresholdHandler реализация обработчика установки порога падения
type signalSetFallThresholdHandler struct {
	*base.BaseHandler
	service signal_settings_svc.Service
}

// NewHandler создает новый обработчик установки порога падения
func NewHandler(service signal_settings_svc.Service) handlers.Handler {
	return &signalSetFallThresholdHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "signal_set_fall_threshold_handler",
			Command: constants.CallbackSignalSetFallThreshold,
			Type:    handlers.TypeCallback,
		},
		service: service,
	}
}

// Execute выполняет обработку callback установки порога падения
func (h *signalSetFallThresholdHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	if params.User == nil {
		return handlers.HandlerResult{}, fmt.Errorf("пользователь не авторизован")
	}

	// Проверяем, есть ли значение порога в data (формат: "signal_set_fall_threshold:2.0")
	if strings.Contains(params.Data, ":") {
		parts := strings.Split(params.Data, ":")
		if len(parts) == 2 && parts[0] == constants.CallbackSignalSetFallThreshold {
			return h.handleThresholdSelection(params, parts[1])
		}
	}

	// Или старый формат "threshold_fall:2.0" для обратной совместимости
	if strings.HasPrefix(params.Data, "threshold_fall:") {
		thresholdStr := strings.TrimPrefix(params.Data, "threshold_fall:")
		return h.handleThresholdSelection(params, thresholdStr)
	}

	// Иначе показываем меню выбора
	return h.showThresholdMenu(params)
}

// showThresholdMenu показывает меню выбора порога падения
func (h *signalSetFallThresholdHandler) showThresholdMenu(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	// ⚠️ ИСПРАВЛЕНО: Используем MinFallThreshold вместо MinGrowthThreshold
	message := fmt.Sprintf(
		"📉 *Установка порога падения*\n\n"+
			"Текущий порог: *%.1f%%*\n\n"+
			"Выберите новый порог или введите значение вручную.\n"+
			"*Рекомендуемые значения:*\n"+
			"• 1.0%% - высокая чувствительность\n"+
			"• 2.0%% - средняя чувствительность\n"+
			"• 3.0%% - низкая чувствительность\n\n"+
			"*Допустимый диапазон:* 0.1%% - 50.0%%",
		params.User.MinFallThreshold, // ⚠️ ИСПРАВЛЕНО: MinFallThreshold
	)

	// ⚠️ ИСПРАВЛЕНО: Callback-данные должны использовать CallbackSignalSetFallThreshold
	keyboard := map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			{
				{"text": "1.0%", "callback_data": constants.CallbackSignalSetFallThreshold + ":1.0"}, // ⚠️ ИСПРАВЛЕНО
				{"text": "1.5%", "callback_data": constants.CallbackSignalSetFallThreshold + ":1.5"}, // ⚠️ ИСПРАВЛЕНО
				{"text": "2.0%", "callback_data": constants.CallbackSignalSetFallThreshold + ":2.0"}, // ⚠️ ИСПРАВЛЕНО
			},
			{
				{"text": "2.5%", "callback_data": constants.CallbackSignalSetFallThreshold + ":2.5"}, // ⚠️ ИСПРАВЛЕНО
				{"text": "3.0%", "callback_data": constants.CallbackSignalSetFallThreshold + ":3.0"}, // ⚠️ ИСПРАВЛЕНО
				{"text": "5.0%", "callback_data": constants.CallbackSignalSetFallThreshold + ":5.0"}, // ⚠️ ИСПРАВЛЕНО
			},
			{
				{"text": "Ввести вручную", "callback_data": "threshold_fall_custom"}, // ⚠️ ИСПРАВЛЕНО
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
			"user_id":             params.User.ID,
			"current_threshold":   params.User.MinFallThreshold, // ⚠️ ИСПРАВЛЕНО
			"expecting_threshold": true,
			"threshold_type":      "fall", // ⚠️ ИСПРАВЛЕНО
		},
	}, nil
}

// handleThresholdSelection обрабатывает выбор порога падения
func (h *signalSetFallThresholdHandler) handleThresholdSelection(params handlers.HandlerParams, thresholdStr string) (handlers.HandlerResult, error) {
	threshold, err := strconv.ParseFloat(thresholdStr, 64)
	if err != nil {
		return handlers.HandlerResult{}, fmt.Errorf("неверное значение порога: %w", err)
	}

	// Проверяем диапазон
	if threshold < 0.1 || threshold > 50.0 {
		return handlers.HandlerResult{}, fmt.Errorf("порог должен быть от 0.1%% до 50%%")
	}

	// Подготавливаем параметры для сервиса
	serviceParams := signal_settings_svc.SignalSettingsParams{
		Action: "set_fall_threshold",
		UserID: params.User.ID,
		ChatID: params.ChatID,
		Value:  threshold,
	}

	// Вызываем сервис
	result, err := h.service.Exec(serviceParams)
	if err != nil {
		return handlers.HandlerResult{}, fmt.Errorf("ошибка в сервисе настройки сигналов: %w", err)
	}

	// Создаем сообщение с результатом
	message := fmt.Sprintf(
		"✅ *Порог падения обновлен*\n\n%s\n\n"+
			"Теперь вы будете получать уведомления только при падении цены на %.1f%% и более.",
		result.Message,
		threshold,
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
			"new_threshold": threshold,
			"updated_field": result.UpdatedField,
		},
	}, nil
}
