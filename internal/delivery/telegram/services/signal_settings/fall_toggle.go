package signal_settings

import (
	"fmt"

	"crypto-exchange-screener-bot/pkg/logger"
)

// toggleFallSignal переключает уведомления о падении
func (s *serviceImpl) toggleFallSignal(params SignalSettingsParams) (SignalSettingsResult, error) {
	// Получаем текущие настройки пользователя
	user, err := s.userService.GetUserByID(params.UserID)
	if err != nil {
		return SignalSettingsResult{}, fmt.Errorf("ошибка получения пользователя: %w", err)
	}

	// Определяем новое значение
	newValue := !user.NotifyFall
	if params.Value != nil {
		if val, ok := params.Value.(bool); ok {
			newValue = val
		}
	}

	// Обновляем настройки
	err = s.userService.UpdateSettings(params.UserID, map[string]interface{}{
		"notify_fall": newValue,
	})

	if err != nil {
		logger.Error("❌ Ошибка обновления настроек падения: %v", err)
		return SignalSettingsResult{}, fmt.Errorf("ошибка обновления настроек: %w", err)
	}

	logger.Info("✅ Настройки падения обновлены для пользователя %d: %v", params.UserID, newValue)

	return SignalSettingsResult{
		Success:      true,
		Message:      fmt.Sprintf("Уведомления о падении %s", getToggleText(newValue)),
		UpdatedField: "notify_fall",
		NewValue:     newValue,
		UserID:       params.UserID,
	}, nil
}
