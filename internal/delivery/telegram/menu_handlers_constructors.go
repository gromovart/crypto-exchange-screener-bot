// internal/delivery/telegram/menu_handlers_constructors.go
package telegram

import (
	"crypto-exchange-screener-bot/internal/infrastructure/config"
)

// NewMenuHandlersWithUtils создает обработчики меню с утилиты
func NewMenuHandlersWithUtils(cfg *config.Config, messageSender *MessageSender, menuUtils *MenuUtils) *MenuHandlers {
	keyboardSystem := NewKeyboardSystem(cfg.Exchange)

	return &MenuHandlers{
		config:          cfg,
		messageSender:   messageSender,
		keyboardSystem:  keyboardSystem,
		menuUtils:       menuUtils,
		authHandlers:    nil,
		settingsManager: nil,
		userService:     nil,
		userMapping:     NewUserMappingService(nil),
	}
}
