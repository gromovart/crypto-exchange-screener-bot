// internal/delivery/telegram/services/signal_settings/fall_threshold.go
package signal_settings

import (
	"fmt"

	"crypto-exchange-screener-bot/pkg/logger"
)

// updateFallThreshold обновляет порог падения
func (s *serviceImpl) updateFallThreshold(params SignalSettingsParams) (SignalSettingsResult, error) {
	// Преобразуем значение в float64
	threshold, err := convertToFloat(params.Value)
	if err != nil {
		return SignalSettingsResult{}, fmt.Errorf("неверное значение порога: %w", err)
	}

	// Проверяем допустимость значения
	if threshold < 0.1 || threshold > 50.0 {
		return SignalSettingsResult{}, fmt.Errorf("порог падения должен быть от 0.1%% до 50%%")
	}

	// Обновляем настройки
	err = s.userService.UpdateSettings(params.UserID, map[string]interface{}{
		"min_fall_threshold": threshold,
	})

	if err != nil {
		logger.Error("❌ Ошибка обновления порога падения: %v", err)
		return SignalSettingsResult{}, fmt.Errorf("ошибка обновления настроек: %w", err)
	}

	logger.Info("✅ Порог падения обновлен для пользователя %d: %.1f%%", params.UserID, threshold)

	return SignalSettingsResult{
		Success:      true,
		Message:      fmt.Sprintf("Порог падения установлен: %.1f%%", threshold),
		UpdatedField: "min_fall_threshold",
		NewValue:     threshold,
		UserID:       params.UserID,
	}, nil
}
