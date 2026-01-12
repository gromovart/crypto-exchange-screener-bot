// internal/delivery/telegram/menu_manager_extended.go
package telegram

// SetupAuth настраивает авторизацию (заглушка для обратной совместимости)
func (mm *MenuManager) SetupAuth(authHandlers *AuthHandlers) {
	mm.SetAuthHandlers(authHandlers)
}

// SetEnabled устанавливает статус enabled (заглушка для обратной совместимости)
func (mm *MenuManager) SetEnabled(enabled bool) {
	// Этот метод был для глобальных настроек, теперь используем персональные
	// Оставляем пустым или логируем
}

// IsEnabled проверяет статус enabled (заглушка для обратной совместимости)
func (mm *MenuManager) IsEnabled() bool {
	// Возвращаем true, так как бот всегда включен
	// Для персональных настроек нужно проверять у конкретного пользователя
	return true
}

// GetKeyboardSystem возвращает систему клавиатур (заглушка для обратной совместимости)
func (mm *MenuManager) GetKeyboardSystem() *KeyboardSystem {
	if mm.handlers != nil {
		return mm.handlers.keyboardSystem
	}
	return nil
}

// SendSettingsMessage отправляет сообщение с настройками (заглушка для обратной совместимости)
func (mm *MenuManager) SendSettingsMessage(chatID string) error {
	return mm.SendSettingsInfo(chatID)
}
