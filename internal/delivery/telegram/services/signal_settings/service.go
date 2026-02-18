// internal/delivery/telegram/services/signal_settings/service.go
package signal_settings

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/core/domain/users"
)

// serviceImpl реализация Service
type serviceImpl struct {
	userService *users.Service
}

// NewService создает новый сервис настройки сигналов
func NewService(userService *users.Service) Service {
	return &serviceImpl{
		userService: userService,
	}
}

// Exec выполняет операции с настройками сигналов
func (s *serviceImpl) Exec(params SignalSettingsParams) (SignalSettingsResult, error) {
	switch params.Action {
	case "toggle_growth":
		return s.toggleGrowthSignal(params)
	case "toggle_fall":
		return s.toggleFallSignal(params)
	case "set_growth_threshold":
		return s.updateGrowthThreshold(params)
	case "set_fall_threshold":
		return s.updateFallThreshold(params)
	case "set_sensitivity":
		return s.updateSensitivity(params)
	case "select_period":
		return s.selectPeriod(params)
	case "remove_period":
		return s.removePeriod(params)
	case "reset_periods":
		return s.resetPeriods(params)
	default:
		return SignalSettingsResult{}, fmt.Errorf("неподдерживаемое действие: %s", params.Action)
	}
}
