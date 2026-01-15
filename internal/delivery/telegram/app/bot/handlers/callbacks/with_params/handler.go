// internal/delivery/telegram/app/bot/handlers/callbacks/with_params/handler.go
package with_params

import (
	"fmt"
	"strings"
	"time"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	signal_settings_svc "crypto-exchange-screener-bot/internal/delivery/telegram/services/signal_settings"
	"crypto-exchange-screener-bot/pkg/logger"
)

// withParamsHandler реализация обработчика callback-ов с параметрами
type withParamsHandler struct {
	*base.BaseHandler
	signalService signal_settings_svc.Service
}

// NewHandler создает новый обработчик callback-ов с параметрами
func NewHandler(signalService signal_settings_svc.Service) handlers.Handler {
	return &withParamsHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "with_params_handler",
			Command: "with_params", // Общий обработчик для всех callback-ов с параметрами
			Type:    handlers.TypeCallback,
		},
		signalService: signalService,
	}
}

// Execute выполняет обработку callback-ов с параметрами
func (h *withParamsHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {

	if params.User == nil {
		return handlers.HandlerResult{}, fmt.Errorf("пользователь не авторизован")
	}

	// Парсим callback data
	data := params.Data
	var result handlers.HandlerResult
	var err error

	startTime := time.Now()
	defer func() {
		logger.Debug("⏱️ Время выполнения with_params для %s: %v", data, time.Since(startTime))
	}()

	switch {
	case strings.HasPrefix(data, "threshold_growth:"):
		result, err = h.handleThreshold(params, data, "growth")
	case strings.HasPrefix(data, "threshold_fall:"):
		result, err = h.handleThreshold(params, data, "fall")
	case strings.HasPrefix(data, "signal_set_growth_threshold:"):
		result, err = h.handleThreshold(params, data, "growth")
	case strings.HasPrefix(data, "signal_set_fall_threshold:"):
		result, err = h.handleThreshold(params, data, "fall")
	case strings.HasPrefix(data, "sensitivity:"):
		result, err = h.handleSensitivity(params, data)
	case strings.HasPrefix(data, "quiet_hours_"):
		result, err = h.handleQuietHours(params, data)
	default:
		return handlers.HandlerResult{}, fmt.Errorf("неизвестный callback с параметрами: %s", data)
	}

	return result, err
}

// handleThreshold универсальный метод обработки порогов
func (h *withParamsHandler) handleThreshold(params handlers.HandlerParams, data string, thresholdType string) (handlers.HandlerResult, error) {
	startTime := time.Now()
	defer func() {
		logger.Debug("⏱️ Время обработки порога %s: %v", thresholdType, time.Since(startTime))
	}()
	// Извлекаем значение из разных форматов
	var thresholdStr string

	// Пробуем извлечь значение после последнего :
	parts := strings.Split(data, ":")
	if len(parts) >= 2 {
		thresholdStr = parts[len(parts)-1]
	} else {
		return handlers.HandlerResult{}, fmt.Errorf("неверный формат порога: %s", data)
	}

	// Определяем действие для сервиса
	var action string
	if thresholdType == "growth" {
		action = "set_growth_threshold"
	} else if thresholdType == "fall" {
		action = "set_fall_threshold"
	} else {
		return handlers.HandlerResult{}, fmt.Errorf("неизвестный тип порога: %s", thresholdType)
	}

	// Подготавливаем параметры для сервиса
	serviceParams := signal_settings_svc.SignalSettingsParams{
		Action: action,
		UserID: params.User.ID,
		ChatID: params.ChatID,
		Value:  thresholdStr,
	}

	// Вызываем сервис
	result, err := h.signalService.Exec(serviceParams)
	if err != nil {
		return handlers.HandlerResult{}, fmt.Errorf("ошибка в сервисе настройки сигналов: %w", err)
	}

	// Создаем сообщение с результатом
	emoji := "📈"
	if thresholdType == "fall" {
		emoji = "📉"
	}

	message := fmt.Sprintf(
		"%s *Порог %s обновлен*\n\n%s\n\n"+
			"Теперь вы будете получать уведомления только при %s цены на %s%% и более.",
		emoji,
		thresholdType,
		result.Message,
		thresholdType,
		thresholdStr,
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
			"new_threshold": thresholdStr,
			"updated_field": result.UpdatedField,
		},
	}, nil
}

// handleSensitivity обрабатывает установку чувствительности
func (h *withParamsHandler) handleSensitivity(params handlers.HandlerParams, data string) (handlers.HandlerResult, error) {
	parts := strings.Split(data, ":")
	if len(parts) != 2 {
		return handlers.HandlerResult{}, fmt.Errorf("неверный формат чувствительности: %s", data)
	}

	sensitivityLevel := parts[1]

	// TODO: Реализовать после добавления поддержки чувствительности
	message := fmt.Sprintf(
		"🎯 *Чувствительность*\n\n"+
			"Уровень чувствительности: %s\n\n"+
			"Эта функция появится в следующем обновлении.",
		sensitivityLevel,
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
			"user_id":           params.User.ID,
			"sensitivity_level": sensitivityLevel,
		},
	}, nil
}

// handleQuietHours обрабатывает настройку тихих часов
func (h *withParamsHandler) handleQuietHours(params handlers.HandlerParams, data string) (handlers.HandlerResult, error) {
	// TODO: Реализовать настройку тихих часов
	message := "⏱️ *Тихие часы*\n\nНастройка тихих часов в разработке."

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
			"user_id": params.User.ID,
			"action":  data,
		},
	}, nil
}
