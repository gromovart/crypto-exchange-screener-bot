// internal/delivery/telegram/services/signal_settings/other_actions.go

package signal_settings

import (
	"fmt"
	"strings"

	"crypto-exchange-screener-bot/pkg/logger"
)

// selectPeriod обрабатывает выбор периода (5m, 15m, 30m, 1h, 4h, 1d)
func (s *serviceImpl) selectPeriod(params SignalSettingsParams) (SignalSettingsResult, error) {
	// Получаем строку периода
	period, ok := params.Value.(string)
	if !ok {
		return SignalSettingsResult{}, fmt.Errorf("неверный формат периода")
	}

	// Преобразуем период в минуты
	periodInMinutes, err := convertPeriodToMinutes(period)
	if err != nil {
		return SignalSettingsResult{}, fmt.Errorf("неверный период: %w", err)
	}

	// Получаем текущие настройки пользователя
	user, err := s.userService.GetUserByID(params.UserID)
	if err != nil {
		return SignalSettingsResult{}, fmt.Errorf("ошибка получения пользователя: %w", err)
	}

	// Проверяем, есть ли уже такой период в PreferredPeriods
	var newPeriods []int
	var found bool
	var action string // "added" или "removed"

	for _, p := range user.PreferredPeriods {
		if p == periodInMinutes {
			found = true
			action = "removed"
			// Пропускаем - удаляем период
		} else {
			newPeriods = append(newPeriods, p)
		}
	}

	// Если период не найден - добавляем
	if !found {
		newPeriods = append(user.PreferredPeriods, periodInMinutes)
		action = "added"
	}

	// Нельзя удалить все периоды, должен остаться хотя бы один
	if len(newPeriods) == 0 {
		newPeriods = []int{5} // Минимальный период по умолчанию
		return SignalSettingsResult{
			Success:      true,
			Message:      "⚠️ Нельзя удалить все периоды. Оставлен период 5m.",
			UpdatedField: "preferred_periods",
			NewValue:     newPeriods,
			UserID:       params.UserID,
		}, nil
	}

	// Обновляем настройки
	err = s.userService.UpdateSettings(params.UserID, map[string]interface{}{
		"preferred_periods": newPeriods,
	})

	if err != nil {
		logger.Error("Ошибка обновления периодов: %v", err)
		return SignalSettingsResult{}, fmt.Errorf("ошибка обновления настроек: %w", err)
	}

	// Сообщение в зависимости от действия
	var message string
	if action == "added" {
		message = fmt.Sprintf("✅ Период %s добавлен", period)
	} else {
		message = fmt.Sprintf("❌ Период %s удален", period)
	}

	logger.Info("Период %s для пользователя %d: %s", action, params.UserID, period)

	return SignalSettingsResult{
		Success:      true,
		Message:      message,
		UpdatedField: "preferred_periods",
		NewValue:     newPeriods,
		UserID:       params.UserID,
		Metadata: map[string]interface{}{
			"period":      period,
			"period_min":  periodInMinutes,
			"action":      action,
			"total_count": len(newPeriods),
		},
	}, nil
}

// convertPeriodToMinutes преобразует строку периода в минуты
func convertPeriodToMinutes(period string) (int, error) {
	// Убираем префикс "period_" если есть
	cleanStr := strings.TrimPrefix(period, "period_")

	switch strings.ToLower(cleanStr) {
	case "5m":
		return 5, nil
	case "15m":
		return 15, nil
	case "30m":
		return 30, nil
	case "1h":
		return 60, nil
	case "4h":
		return 240, nil
	case "1d":
		return 1440, nil
	default:
		// Пробуем распарсить число
		cleanStr = strings.ToLower(cleanStr)
		if strings.HasSuffix(cleanStr, "m") {
			numStr := strings.TrimSuffix(cleanStr, "m")
			var num int
			if _, err := fmt.Sscanf(numStr, "%d", &num); err == nil && num > 0 {
				return num, nil
			}
		}
		if strings.HasSuffix(cleanStr, "h") {
			numStr := strings.TrimSuffix(cleanStr, "h")
			var num int
			if _, err := fmt.Sscanf(numStr, "%d", &num); err == nil && num > 0 {
				return num * 60, nil
			}
		}
		return 0, fmt.Errorf("неподдерживаемый период: %s", period)
	}
}

// removePeriod удаляет период из списка
func (s *serviceImpl) removePeriod(params SignalSettingsParams) (SignalSettingsResult, error) {
	period, ok := params.Value.(string)
	if !ok {
		return SignalSettingsResult{}, fmt.Errorf("неверный формат периода")
	}

	periodInMinutes, err := convertPeriodToMinutes(period)
	if err != nil {
		return SignalSettingsResult{}, fmt.Errorf("неверный период: %w", err)
	}

	user, err := s.userService.GetUserByID(params.UserID) // params.UserID уже int
	if err != nil {
		return SignalSettingsResult{}, fmt.Errorf("ошибка получения пользователя: %w", err)
	}

	// Ищем и удаляем период
	var newPeriods []int
	found := false

	for _, p := range user.PreferredPeriods {
		if p != periodInMinutes {
			newPeriods = append(newPeriods, p)
		} else {
			found = true
		}
	}

	if !found {
		return SignalSettingsResult{
			Success:      true,
			Message:      fmt.Sprintf("Период %s не найден в списке", period),
			UpdatedField: "preferred_periods",
			NewValue:     user.PreferredPeriods,
			UserID:       params.UserID,
		}, nil
	}

	// Нельзя удалить все периоды, должен остаться хотя бы один
	if len(newPeriods) == 0 {
		newPeriods = []int{5} // Минимальный период по умолчанию
	}

	err = s.userService.UpdateSettings(params.UserID, map[string]interface{}{
		"preferred_periods": newPeriods,
	})

	if err != nil {
		logger.Error("Ошибка удаления периода: %v", err)
		return SignalSettingsResult{}, fmt.Errorf("ошибка обновления настроек: %w", err)
	}

	logger.Info("Период удален для пользователя %d: %s", params.UserID, period)

	return SignalSettingsResult{
		Success:      true,
		Message:      fmt.Sprintf("✅ Период %s успешно удален", period),
		UpdatedField: "preferred_periods",
		NewValue:     newPeriods,
		UserID:       params.UserID,
	}, nil
}

// resetPeriods сбрасывает периоды к значениям по умолчанию
func (s *serviceImpl) resetPeriods(params SignalSettingsParams) (SignalSettingsResult, error) {
	defaultPeriods := []int{5, 15, 30}

	err := s.userService.UpdateSettings(params.UserID, map[string]interface{}{
		"preferred_periods": defaultPeriods,
	})

	if err != nil {
		logger.Error("Ошибка сброса периодов: %v", err)
		return SignalSettingsResult{}, fmt.Errorf("ошибка обновления настроек: %w", err)
	}

	logger.Info("Периоды сброшены для пользователя %d", params.UserID)

	return SignalSettingsResult{
		Success:      true,
		Message:      "✅ Периоды сброшены к значениям по умолчанию (5m, 15m, 30m)",
		UpdatedField: "preferred_periods",
		NewValue:     defaultPeriods,
		UserID:       params.UserID,
	}, nil
}

// Вспомогательные функции (оставлю их в этом файле, так как они используются здесь)
func formatPeriodsToString(periods []int) string {
	var parts []string
	for _, period := range periods {
		parts = append(parts, formatMinutesToPeriod(period))
	}
	return strings.Join(parts, ", ")
}

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

// updateSensitivity обновляет чувствительность (заглушка)
func (s *serviceImpl) updateSensitivity(params SignalSettingsParams) (SignalSettingsResult, error) {
	return SignalSettingsResult{
		Success:      true,
		Message:      "✅ Чувствительность обновлена",
		UpdatedField: "sensitivity",
		NewValue:     params.Value,
		UserID:       params.UserID,
	}, nil
}
