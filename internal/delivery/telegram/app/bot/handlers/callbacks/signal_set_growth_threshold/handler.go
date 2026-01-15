package signal_set_growth_threshold

import (
	"fmt"
	"strconv"
	"strings"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	signal_settings_svc "crypto-exchange-screener-bot/internal/delivery/telegram/services/signal_settings"
)

// signalSetGrowthThresholdHandler реализация обработчика установки порога роста
type signalSetGrowthThresholdHandler struct {
	*base.BaseHandler
	service signal_settings_svc.Service
}

// NewHandler создает новый обработчик установки порога роста
func NewHandler(service signal_settings_svc.Service) handlers.Handler {
	return &signalSetGrowthThresholdHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "signal_set_growth_threshold_handler",
			Command: constants.CallbackSignalSetGrowthThreshold,
			Type:    handlers.TypeCallback,
		},
		service: service,
	}
}

// Execute выполняет обработку callback установки порога роста
func (h *signalSetGrowthThresholdHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	if params.User == nil {
		return handlers.HandlerResult{}, fmt.Errorf("пользователь не авторизован")
	}

	// Проверяем, есть ли значение порога в data (формат: "signal_set_growth_threshold:1.0")
	if strings.Contains(params.Data, ":") {
		parts := strings.Split(params.Data, ":")
		if len(parts) == 2 && parts[0] == constants.CallbackSignalSetGrowthThreshold {
			return h.handleThresholdSelection(params, parts[1])
		}
	}

	// Иначе показываем меню выбора
	return h.showThresholdMenu(params)
}

// showThresholdMenu показывает меню выбора порога
func (h *signalSetGrowthThresholdHandler) showThresholdMenu(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	message := fmt.Sprintf(
		"📈 *Установка порога роста*\n\n"+
			"Текущий порог: *%.1f%%*\n\n"+
			"Выберите новый порог или введите значение вручную.\n"+
			"*Рекомендуемые значения:*\n"+
			"• 1.0%% - высокая чувствительность\n"+
			"• 2.0%% - средняя чувствительность\n"+
			"• 3.0%% - низкая чувствительность\n\n"+
			"*Допустимый диапазон:* 0.1%% - 50.0%%",
		params.User.MinGrowthThreshold,
	)

	// Клавиатура с вариантами порогов
	keyboard := map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			{
				{"text": "1.0%", "callback_data": constants.CallbackSignalSetGrowthThreshold + ":1.0"},
				{"text": "1.5%", "callback_data": constants.CallbackSignalSetGrowthThreshold + ":1.5"},
				{"text": "2.0%", "callback_data": constants.CallbackSignalSetGrowthThreshold + ":2.0"},
			},
			{
				{"text": "2.5%", "callback_data": constants.CallbackSignalSetGrowthThreshold + ":2.5"},
				{"text": "3.0%", "callback_data": constants.CallbackSignalSetGrowthThreshold + ":3.0"},
				{"text": "5.0%", "callback_data": constants.CallbackSignalSetGrowthThreshold + ":5.0"},
			},
			{
				{"text": "Ввести вручную", "callback_data": "threshold_growth_custom"},
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
			"current_threshold":   params.User.MinGrowthThreshold,
			"expecting_threshold": true,
			"threshold_type":      "growth",
		},
	}, nil
}

// handleThresholdSelection обрабатывает выбор порога
func (h *signalSetGrowthThresholdHandler) handleThresholdSelection(params handlers.HandlerParams, thresholdStr string) (handlers.HandlerResult, error) {
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
		Action: "set_growth_threshold",
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
		"✅ *Порог роста обновлен*\n\n%s\n\n"+
			"Теперь вы будете получать уведомления только при росте цены на %.1f%% и более.",
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
