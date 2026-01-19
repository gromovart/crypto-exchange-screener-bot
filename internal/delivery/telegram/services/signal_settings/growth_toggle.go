// internal/delivery/telegram/services/signal_settings/growth_toggle.go
package signal_settings

import (
	"fmt"

	"crypto-exchange-screener-bot/pkg/logger"
)

// toggleGrowthSignal переключает уведомления о росте
func (s *serviceImpl) toggleGrowthSignal(params SignalSettingsParams) (SignalSettingsResult, error) {
	// Получаем текущие настройки пользователя
	user, err := s.userService.GetUserByID(params.UserID)
	if err != nil {
		return SignalSettingsResult{}, fmt.Errorf("ошибка получения пользователя: %w", err)
	}

	// Определяем новое значение
	newValue := !user.NotifyGrowth
	if params.Value != nil {
		if val, ok := params.Value.(bool); ok {
			newValue = val
		}
	}

	// Обновляем настройки
	err = s.userService.UpdateSettings(params.UserID, map[string]interface{}{
		"notify_growth": newValue,
	})

	if err != nil {
		logger.Error("❌ Ошибка обновления настроек роста: %v", err)
		return SignalSettingsResult{}, fmt.Errorf("ошибка обновления настроек: %w", err)
	}

	logger.Info("✅ Настройки роста обновлены для пользователя %d: %v", params.UserID, newValue)

	return SignalSettingsResult{
		Success:      true,
		Message:      fmt.Sprintf("Уведомления о росте %s", getToggleText(newValue)),
		UpdatedField: "notify_growth",
		NewValue:     newValue,
		UserID:       params.UserID,
	}, nil
}
