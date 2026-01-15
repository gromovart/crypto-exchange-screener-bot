package signal_settings

import (
	"fmt"

	"crypto-exchange-screener-bot/pkg/logger"
)

// updateGrowthThreshold обновляет порог роста
func (s *serviceImpl) updateGrowthThreshold(params SignalSettingsParams) (SignalSettingsResult, error) {
	// Преобразуем значение в float64
	threshold, err := convertToFloat(params.Value)
	if err != nil {
		return SignalSettingsResult{}, fmt.Errorf("неверное значение порога: %w", err)
	}

	// Проверяем допустимость значения
	if threshold < 0.1 || threshold > 50.0 {
		return SignalSettingsResult{}, fmt.Errorf("порог роста должен быть от 0.1%% до 50%%")
	}

	// Обновляем настройки
	err = s.userService.UpdateSettings(params.UserID, map[string]interface{}{
		"min_growth_threshold": threshold,
	})

	if err != nil {
		logger.Error("❌ Ошибка обновления порога роста: %v", err)
		return SignalSettingsResult{}, fmt.Errorf("ошибка обновления настроек: %w", err)
	}

	logger.Info("✅ Порог роста обновлен для пользователя %d: %.1f%%", params.UserID, threshold)

	return SignalSettingsResult{
		Success:      true,
		Message:      fmt.Sprintf("Порог роста установлен: %.1f%%", threshold),
		UpdatedField: "min_growth_threshold",
		NewValue:     threshold,
		UserID:       params.UserID,
	}, nil
}
