// internal/delivery/telegram/app/bot/handlers/callbacks/periods_menu/handler.go
package periods_menu

import (
	"fmt"
	"strconv"
	"strings"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
)

// periodsMenuHandler реализация обработчика меню периодов
type periodsMenuHandler struct {
	*base.BaseHandler
}

// NewHandler создает новый обработчик меню периодов
func NewHandler() handlers.Handler {
	return &periodsMenuHandler{
		BaseHandler: &base.BaseHandler{ // Изменено на указатель
			Name:    "periods_menu_handler",
			Command: constants.CallbackPeriodsMenu,
			Type:    handlers.TypeCallback,
		},
	}
}

// Execute выполняет обработку callback меню периодов
func (h *periodsMenuHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
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
	// Преобразуем периоды в строку
	periodsStr := "Не настроены"
	if user != nil && len(user.PreferredPeriods) > 0 {
		var periods []string
		for _, p := range user.PreferredPeriods {
			periods = append(periods, fmt.Sprintf("%dм", p))
		}
		periodsStr = strings.Join(periods, ", ")
	}

	return fmt.Sprintf(
		"%s\n\n"+
			"Текущие периоды: %s\n\n"+
			"Периоды определяют, за какие временные интервалы\n"+
			"бот анализирует движение цены.\n\n"+
			"Выберите периоды для отслеживания:",
		constants.MenuButtonTexts.Periods,
		periodsStr,
	)
}

// createPeriodsKeyboard создает клавиатуру для меню периодов
func (h *periodsMenuHandler) createPeriodsKeyboard(user *models.User) interface{} {
	// Базовые кнопки периодов (доступны всем)
	buttons := [][]map[string]string{
		{
			{"text": constants.PeriodButtonTexts.Period5m, "callback_data": constants.CallbackPeriod5m},
			{"text": constants.PeriodButtonTexts.Period15m, "callback_data": constants.CallbackPeriod15m},
			{"text": constants.PeriodButtonTexts.Period30m, "callback_data": constants.CallbackPeriod30m},
		},
		{
			{"text": constants.PeriodButtonTexts.Period1h, "callback_data": constants.CallbackPeriod1h},
			{"text": constants.PeriodButtonTexts.Period4h, "callback_data": constants.CallbackPeriod4h},
		},
	}

	// Добавляем кнопку "1 день" для авторизованных
	if user != nil && user.ID > 0 {
		buttons = append(buttons, []map[string]string{
			{"text": constants.PeriodButtonTexts.Period1d, "callback_data": constants.CallbackPeriod1d},
		})
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
func (h *periodsMenuHandler) isPeriodSelected(user *models.User, period string) bool {
	if user == nil || len(user.PreferredPeriods) == 0 {
		return false
	}

	// Преобразуем период в число
	periodInt, err := strconv.Atoi(strings.TrimSuffix(period, "m"))
	if err != nil {
		return false
	}

	// Проверяем наличие периода
	for _, p := range user.PreferredPeriods {
		if p == periodInt {
			return true
		}
	}
	return false
}
