// internal/delivery/telegram/app/bot/handlers/callbacks/period_select/handler.go
package period_select

import (
	"fmt"
	"strings"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	signal_settings_svc "crypto-exchange-screener-bot/internal/delivery/telegram/services/signal_settings"
	"crypto-exchange-screener-bot/pkg/period"
)

// periodSelectHandler реализация обработчика выбора периода
type periodSelectHandler struct {
	*base.BaseHandler
	service signal_settings_svc.Service
}

// NewHandler создает новый обработчик выбора периода
func NewHandler(service signal_settings_svc.Service) handlers.Handler {
	return &periodSelectHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "period_select_handler",
			Command: "period_select",
			Type:    handlers.TypeCallback,
		},
		service: service,
	}
}

// Execute выполняет обработку callback выбора периода
func (h *periodSelectHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	if params.User == nil {
		return handlers.HandlerResult{}, fmt.Errorf("пользователь не авторизован")
	}

	// Определяем период из callback data (формат: "period_5m")
	periodStr := params.Data
	if periodStr == "" {
		return h.showPeriodsMenu(params)
	}

	// Определяем действие
	var action string
	if strings.HasPrefix(periodStr, "period_manage_") {
		action = strings.TrimPrefix(periodStr, "period_manage_")
		return h.handlePeriodManagement(params, action)
	}

	// Обычный выбор периода (добавление)
	action = "select_period"

	// Подготавливаем параметры для сервиса
	serviceParams := signal_settings_svc.SignalSettingsParams{
		Action: action,
		UserID: params.User.ID,
		ChatID: params.ChatID,
		Value:  periodStr,
	}

	// Вызываем сервис
	result, err := h.service.Exec(serviceParams)
	if err != nil {
		return handlers.HandlerResult{}, fmt.Errorf("ошибка в сервисе настройки сигналов: %w", err)
	}

	// Форматируем периоды для отображения
	periods, _ := result.NewValue.([]int)
	periodsStr := formatPeriodsToString(periods)

	// Создаем сообщение с результатом
	var emoji string
	if actionResult, ok := result.Metadata["action"].(string); ok {
		if actionResult == "added" {
			emoji = "✅"
		} else {
			emoji = "❌"
		}
	} else {
		emoji = "✅"
	}

	message := fmt.Sprintf(
		"%s *Период обновлен*\n\n%s\n\n"+
			"Текущие периоды анализа: %s",
		emoji,
		result.Message,
		periodsStr,
	)

	// Создаем клавиатуру
	keyboard := map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			{
				{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackPeriodsMenu},
			},
		},
	}

	return handlers.HandlerResult{
		Message:  message,
		Keyboard: keyboard,
	}, nil
}

// showPeriodsMenu показывает меню управления периодами
func (h *periodSelectHandler) showPeriodsMenu(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	// Форматируем текущие периоды
	periodsStr := formatPeriodsToString(params.User.PreferredPeriods)

	message := fmt.Sprintf(
		"⏱️ *Управление периодами анализа*\n\n"+
			"Текущие периоды: %s\n\n"+
			"*Выберите действие:*",
		periodsStr,
	)

	keyboard := map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			{
				{"text": "➕ Добавить период", "callback_data": "period_manage_add"},
			},
			{
				{"text": "➖ Удалить период", "callback_data": "period_manage_remove"},
			},
			{
				{"text": "🔄 Сбросить к настройкам по умолчанию", "callback_data": "period_manage_reset"},
			},
			{
				{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackPeriodsMenu},
			},
		},
	}

	return handlers.HandlerResult{
		Message:  message,
		Keyboard: keyboard,
	}, nil
}

// handlePeriodManagement обрабатывает управление периодами
func (h *periodSelectHandler) handlePeriodManagement(params handlers.HandlerParams, action string) (handlers.HandlerResult, error) {
	switch action {
	case "add":
		return h.showAddPeriodMenu(params)
	case "remove":
		return h.showRemovePeriodMenu(params)
	case "reset":
		return h.handleResetPeriods(params)
	default:
		return handlers.HandlerResult{}, fmt.Errorf("неизвестное действие: %s", action)
	}
}

// showAddPeriodMenu показывает меню добавления периода
func (h *periodSelectHandler) showAddPeriodMenu(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	message := "⏱️ *Добавление периода*\n\nВыберите период для добавления:"

	keyboard := map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			{
				{"text": "1 минута", "callback_data": constants.CallbackPeriod1m},
				{"text": "5 минут", "callback_data": constants.CallbackPeriod5m},
			},
			{
				{"text": "15 минут", "callback_data": constants.CallbackPeriod15m},
				{"text": "30 минут", "callback_data": constants.CallbackPeriod30m},
			},
			{
				{"text": "1 час", "callback_data": constants.CallbackPeriod1h},
				{"text": "4 часа", "callback_data": constants.CallbackPeriod4h},
			},
			{
				{"text": "1 день", "callback_data": constants.CallbackPeriod1d},
			},
			{
				{"text": constants.ButtonTexts.Back, "callback_data": "period_select"},
			},
		},
	}

	return handlers.HandlerResult{
		Message:  message,
		Keyboard: keyboard,
	}, nil
}

// showRemovePeriodMenu показывает меню удаления периода
func (h *periodSelectHandler) showRemovePeriodMenu(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	// Создаем кнопки только для текущих периодов пользователя
	var buttons [][]map[string]string

	for _, periodMinutes := range params.User.PreferredPeriods {
		periodStr := period.MinutesToString(periodMinutes)
		callbackData := fmt.Sprintf("period_%s", periodStr)

		buttons = append(buttons, []map[string]string{
			{"text": periodStr, "callback_data": callbackData},
		})
	}

	// Добавляем кнопку "Назад"
	buttons = append(buttons, []map[string]string{
		{"text": constants.ButtonTexts.Back, "callback_data": "period_select"},
	})

	message := "⏱️ *Удаление периода*\n\nВыберите период для удаления:"

	return handlers.HandlerResult{
		Message: message,
		Keyboard: map[string]interface{}{
			"inline_keyboard": buttons,
		},
	}, nil
}

// handleResetPeriods обрабатывает сброс периодов
func (h *periodSelectHandler) handleResetPeriods(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	// Подготавливаем параметры для сервиса
	serviceParams := signal_settings_svc.SignalSettingsParams{
		Action: "reset_periods",
		UserID: params.User.ID,
		ChatID: params.ChatID,
		Value:  nil,
	}

	// Вызываем сервис
	result, err := h.service.Exec(serviceParams)
	if err != nil {
		return handlers.HandlerResult{}, fmt.Errorf("ошибка сброса периодов: %w", err)
	}

	// Создаем сообщение с результатом
	message := fmt.Sprintf("✅ *Периоды сброшены*\n\n%s", result.Message)

	keyboard := map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			{
				{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackPeriodsMenu},
			},
		},
	}

	return handlers.HandlerResult{
		Message:  message,
		Keyboard: keyboard,
	}, nil
}

// Вспомогательная функция для форматирования периодов
func formatPeriodsToString(periods []int) string {
	if len(periods) == 0 {
		return "нет периодов"
	}

	var parts []string
	for _, periodMinutes := range periods {
		parts = append(parts, period.MinutesToString(periodMinutes))
	}
	return strings.Join(parts, ", ")
}
