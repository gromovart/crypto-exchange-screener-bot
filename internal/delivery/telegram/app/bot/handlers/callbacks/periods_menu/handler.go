// internal/delivery/telegram/app/bot/handlers/callbacks/periods_menu/handler.go
package periods_menu

import (
	"fmt"
	"strings"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
	"crypto-exchange-screener-bot/pkg/logger"
	"crypto-exchange-screener-bot/pkg/period"
)

// periodsMenuHandler реализация обработчика меню периодов
type periodsMenuHandler struct {
	*base.BaseHandler
}

// NewHandler создает новый обработчик меню периодов
func NewHandler() handlers.Handler {
	return &periodsMenuHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "periods_menu_handler",
			Command: constants.CallbackPeriodsMenu,
			Type:    handlers.TypeCallback,
		},
	}
}

// Execute выполняет обработку callback меню периодов
func (h *periodsMenuHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {

	logger.Warn("Пользователь ID: %d, Периоды: %v",
		params.User.ID, params.User.PreferredPeriods)
	// Периоды доступны всем
	message := h.createPeriodsMessage(params.User)
	keyboard := h.createPeriodsKeyboard(params.User)

	return handlers.HandlerResult{
		Message:  message,
		Keyboard: keyboard,
		Metadata: map[string]interface{}{
			"user_id": params.User.ID,
		},
	}, nil
}

// createPeriodsMessage создает сообщение для меню периодов
func (h *periodsMenuHandler) createPeriodsMessage(user *models.User) string {
	// Форматируем текущие периоды
	periodsDisplay := h.formatPeriodsForDisplay(user)

	return fmt.Sprintf(
		"⏱️ *Периоды анализа*\n\n"+
			"*Текущие периоды:* %s\n\n"+
			"*Как работают периоды:*\n"+
			"• 1m - сверхкраткосрочные движения\n"+
			"• 5m - краткосрочные движения\n"+
			"• 15m - среднесрочные тренды\n"+
			"• 30m - долгосрочные тенденции\n"+
			"• 1h - анализ по часам\n"+
			"• 4h - внутридневной анализ\n"+
			"• 1d - дневной анализ\n\n"+
			"*Инструкция:*\n"+
			"Нажмите на период, чтобы добавить/удалить его.\n"+
			"✅ - период выбран\n"+
			"⏱️ - период не выбран",
		periodsDisplay,
	)
}

// formatPeriodsForDisplay форматирует периоды для отображения
func (h *periodsMenuHandler) formatPeriodsForDisplay(user *models.User) string {
	if user == nil || len(user.PreferredPeriods) == 0 {
		return "не настроены"
	}

	var periods []string
	for _, p := range user.PreferredPeriods {
		periods = append(periods, period.MinutesToString(p))
	}

	return strings.Join(periods, ", ")
}

// createPeriodsKeyboard создает клавиатуру для меню периодов
func (h *periodsMenuHandler) createPeriodsKeyboard(user *models.User) interface{} {
	// Создаем кнопки с индикаторами выбора
	buttons := [][]map[string]string{
		h.createPeriodButtonRow(user, period.Period1m, constants.CallbackPeriod1m, "1 минута"),
		h.createPeriodButtonRow(user, period.Period5m, constants.CallbackPeriod5m, "5 минут"),
		h.createPeriodButtonRow(user, period.Period15m, constants.CallbackPeriod15m, "15 минут"),
		h.createPeriodButtonRow(user, period.Period30m, constants.CallbackPeriod30m, "30 минут"),
		h.createPeriodButtonRow(user, period.Period1h, constants.CallbackPeriod1h, "1 час"),
		h.createPeriodButtonRow(user, period.Period4h, constants.CallbackPeriod4h, "4 часа"),
	}

	// Добавляем кнопку "1 день" для авторизованных
	if user != nil && user.ID > 0 {
		buttons = append(buttons,
			h.createPeriodButtonRow(user, period.Period1d, constants.CallbackPeriod1d, "1 день"),
		)
	}

	// Кнопка "Назад"
	buttons = append(buttons, []map[string]string{
		{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackMenuMain},
	})

	return map[string]interface{}{
		"inline_keyboard": buttons,
	}
}

// isPeriodSelected проверяет, выбран ли период
func (h *periodsMenuHandler) isPeriodSelected(user *models.User, periodStr string) bool {
	if user == nil || len(user.PreferredPeriods) == 0 {
		return false
	}

	// Преобразуем строку периода в минуты
	periodMinutes, err := period.StringToMinutes(periodStr)
	if err != nil {
		return false
	}

	// Проверяем наличие периода
	for _, p := range user.PreferredPeriods {
		if p == periodMinutes {
			return true
		}
	}
	return false
}

// createPeriodButtonRow создает строку кнопки периода с индикатором
func (h *periodsMenuHandler) createPeriodButtonRow(user *models.User, periodStr, callback, buttonText string) []map[string]string {
	isSelected := h.isPeriodSelected(user, periodStr)

	// Обновленные индикаторы
	var indicator string
	if isSelected {
		indicator = "✅ "
	} else {
		indicator = "⏱️ "
	}

	return []map[string]string{
		{"text": indicator + buttonText, "callback_data": callback},
	}
}
