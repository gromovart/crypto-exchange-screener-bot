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
	"crypto-exchange-screener-bot/pkg/logger"
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
		periods = append(periods, formatMinutesToPeriod(p))
	}

	return strings.Join(periods, ", ")
}

// formatMinutesToPeriod форматирует минуты в читаемый период
func formatMinutesToPeriod(minutes int) string {
	switch minutes {
	case 5:
		return "5m"
	case 15:
		return "15m"
	case 30:
		return "30m"
	case 60:
		return "1h"
	case 240:
		return "4h"
	case 1440:
		return "1d"
	default:
		if minutes >= 1440 && minutes%1440 == 0 {
			return fmt.Sprintf("%dd", minutes/1440)
		} else if minutes >= 60 && minutes%60 == 0 {
			return fmt.Sprintf("%dh", minutes/60)
		} else {
			return fmt.Sprintf("%dm", minutes)
		}
	}
}

// createPeriodsKeyboard создает клавиатуру для меню периодов
func (h *periodsMenuHandler) createPeriodsKeyboard(user *models.User) interface{} {
	// Создаем кнопки с индикаторами выбора
	buttons := [][]map[string]string{
		h.createPeriodButtonRow(user, "5m", constants.CallbackPeriod5m, "5 минут"),
		h.createPeriodButtonRow(user, "15m", constants.CallbackPeriod15m, "15 минут"),
		h.createPeriodButtonRow(user, "30m", constants.CallbackPeriod30m, "30 минут"),
		h.createPeriodButtonRow(user, "1h", constants.CallbackPeriod1h, "1 час"),
		h.createPeriodButtonRow(user, "4h", constants.CallbackPeriod4h, "4 часа"),
	}

	// Добавляем кнопку "1 день" для авторизованных
	if user != nil && user.ID > 0 {
		buttons = append(buttons,
			h.createPeriodButtonRow(user, "1d", constants.CallbackPeriod1d, "1 день"),
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

// createPeriodButtonRow создает строку кнопки периода с индикатором
func (h *periodsMenuHandler) createPeriodButtonRow(user *models.User, periodStr, callback, buttonText string) []map[string]string {
	isSelected := false
	if user != nil {
		periodMinutes := convertPeriodStrToMinutes(periodStr)
		for _, p := range user.PreferredPeriods {
			if p == periodMinutes {
				isSelected = true
				break
			}
		}
	}

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

// convertPeriodStrToMinutes конвертирует строку периода в минуты
func convertPeriodStrToMinutes(periodStr string) int {
	switch periodStr {
	case "5m":
		return 5
	case "15m":
		return 15
	case "30m":
		return 30
	case "1h":
		return 60
	case "4h":
		return 240
	case "1d":
		return 1440
	default:
		// Пробуем распарсить
		if strings.HasSuffix(periodStr, "m") {
			numStr := strings.TrimSuffix(periodStr, "m")
			num, _ := strconv.Atoi(numStr)
			return num
		}
		if strings.HasSuffix(periodStr, "h") {
			numStr := strings.TrimSuffix(periodStr, "h")
			num, _ := strconv.Atoi(numStr)
			return num * 60
		}
		if strings.HasSuffix(periodStr, "d") {
			numStr := strings.TrimSuffix(periodStr, "d")
			num, _ := strconv.Atoi(numStr)
			return num * 1440
		}
		return 0
	}
}
