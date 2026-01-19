package signals_menu

import (
	"fmt"
	"strings"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
)

// signalsMenuHandler реализация обработчика меню сигналов
type signalsMenuHandler struct {
	*base.BaseHandler
}

// NewHandler создает новый обработчик меню сигналов
func NewHandler() handlers.Handler {
	return &signalsMenuHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "signals_menu_handler",
			Command: constants.CallbackSignalsMenu,
			Type:    handlers.TypeCallback,
		},
	}
}

// Execute выполняет обработку callback меню сигналов
func (h *signalsMenuHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	if params.User == nil {
		return handlers.HandlerResult{}, fmt.Errorf("пользователь не авторизован")
	}

	message := h.createSignalsMessage(params.User)
	keyboard := h.createSignalsKeyboard(params.User)

	return handlers.HandlerResult{
		Message:  message,
		Keyboard: keyboard,
		Metadata: map[string]interface{}{
			"user_id": params.User.ID,
		},
	}, nil
}

// createSignalsMessage создает сообщение для меню сигналов
func (h *signalsMenuHandler) createSignalsMessage(user *models.User) string {
	var signalTypes []string

	if user.NotifyGrowth {
		signalTypes = append(signalTypes, constants.SignalButtonTexts.ToggleGrowth)
	}
	if user.NotifyFall {
		signalTypes = append(signalTypes, constants.SignalButtonTexts.ToggleFall)
	}

	signalsStatus := "❌ Нет активных сигналов"
	if len(signalTypes) > 0 {
		signalsStatus = strings.Join(signalTypes, " и ")
	}

	return fmt.Sprintf(
		"%s\n\n"+
			"*Управление сигналами*\n\n"+
			"📊 *Статус отслеживания:*\n"+
			"   • Типы сигналов: %s\n"+
			"   • Минимальный рост: %.1f%%\n"+
			"   • Минимальное падение: %.1f%%\n"+
			"   • Чувствительность: %s\n\n"+
			"⚡ *Последняя активность:*\n"+
			"   • Обнаружено сигналов: %d\n"+
			"   • Последний сигнал: %s\n\n"+
			"Выберите действие:",
		constants.MenuButtonTexts.Signals,
		signalsStatus,
		user.MinGrowthThreshold,
		user.MinFallThreshold,
		h.getSensitivityText(0.5), // TODO: Добавить поле sensitivity в модель User
		0,                         // TODO: Получить реальное количество сигналов
		"недавно",                 // TODO: Получить время последнего сигнала
	)
}

// createSignalsKeyboard создает клавиатуру для меню сигналов
func (h *signalsMenuHandler) createSignalsKeyboard(user *models.User) interface{} {
	// Используем базовые методы для текста переключения
	growthText := h.BaseHandler.GetToggleText(constants.SignalButtonTexts.ToggleGrowth, user.NotifyGrowth)
	fallText := h.BaseHandler.GetToggleText(constants.SignalButtonTexts.ToggleFall, user.NotifyFall)

	keyboard := [][]map[string]string{
		// Настройки типов сигналов
		{
			{"text": growthText, "callback_data": constants.CallbackSignalToggleGrowth},
			{"text": fallText, "callback_data": constants.CallbackSignalToggleFall},
		},
		// Настройки порогов
		{
			{"text": fmt.Sprintf(constants.SignalButtonTexts.ThresholdFormat, constants.DirectionIcons.Up, user.MinGrowthThreshold),
				"callback_data": constants.CallbackSignalSetGrowthThreshold},
			{"text": fmt.Sprintf(constants.SignalButtonTexts.ThresholdFormat, constants.DirectionIcons.Down, user.MinFallThreshold),
				"callback_data": constants.CallbackSignalSetFallThreshold},
		},

		//TODO: рааскомментировать и реализовать эти функции позже
		// Дополнительные настройки
		// {
		// 	{"text": constants.SignalButtonTexts.Sensitivity, "callback_data": constants.CallbackSignalSetSensitivity},
		// 	{"text": constants.SignalButtonTexts.QuietHours, "callback_data": constants.CallbackSignalSetQuietHours},
		// },
		// Действия
		// {
		// 	{"text": constants.SignalButtonTexts.History, "callback_data": constants.CallbackSignalHistory},
		// 	{"text": constants.SignalButtonTexts.TestSignal, "callback_data": constants.CallbackSignalTest},
		// },
		// Навигация
		{
			{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackMenuMain},
			{"text": constants.ButtonTexts.Help, "callback_data": constants.CallbackHelp},
		},
	}

	return map[string]interface{}{
		"inline_keyboard": keyboard,
	}
}

// getSensitivityText возвращает текстовое описание чувствительности
func (h *signalsMenuHandler) getSensitivityText(sensitivity float64) string {
	if sensitivity <= 0.3 {
		return "Низкая"
	} else if sensitivity <= 0.7 {
		return "Средняя"
	} else {
		return "Высокая"
	}
}
