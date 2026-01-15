// internal/delivery/telegram/services/notifications_toggle/service.go
package notifications_toggle

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/core/domain/users"
	"crypto-exchange-screener-bot/pkg/logger"
)

// serviceImpl реализация Service
type serviceImpl struct {
	userService *users.Service
}

// NewService создает новый сервис переключения уведомлений
func NewService(userService *users.Service) Service {
	return &serviceImpl{
		userService: userService,
	}
}

func (s *serviceImpl) Exec(params NotificationsToggleResultParams) (NotificationsToggleResult, error) {
	// Извлекаем данные
	data, ok := params.Data.(map[string]interface{})
	if !ok {
		return NotificationsToggleResult{}, fmt.Errorf("неверный формат данных")
	}

	userID, ok := data["user_id"].(int)
	if !ok {
		return NotificationsToggleResult{}, fmt.Errorf("неверный user_id")
	}

	newState, ok := data["new_state"].(bool)
	if !ok {
		return NotificationsToggleResult{}, fmt.Errorf("неверный new_state")
	}

	logger.Debug("🔄 Переключение уведомлений для пользователя %d: %v", userID, newState)

	// Обновляем настройки пользователя через метод UpdateSettings
	err := s.userService.UpdateSettings(userID, map[string]interface{}{
		"notifications_enabled": newState,
	})

	if err != nil {
		logger.Error("❌ Ошибка обновления уведомлений: %v", err)
		return NotificationsToggleResult{}, fmt.Errorf("ошибка обновления настроек: %w", err)
	}

	logger.Info("✅ Уведомления переключены для пользователя %d: %v", userID, newState)

	return NotificationsToggleResult{
		Processed: true,
		Message:   fmt.Sprintf("Уведомления %s", getStatusText(newState)),
		SentTo:    1,
	}, nil
}

func getStatusText(enabled bool) string {
	if enabled {
		return "включены ✅"
	}
	return "выключены ❌"
}
