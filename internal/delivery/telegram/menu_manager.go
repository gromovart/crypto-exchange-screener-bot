// internal/delivery/telegram/menu_manager.go
package telegram

import (
	"crypto-exchange-screener-bot/internal/core/domain/users"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
)

// MenuManager управляет меню и обработчиками
type MenuManager struct {
	handlers *MenuHandlers
}

// NewMenuManager создает новый менеджер меню
func NewMenuManager(cfg *config.Config, messageSender *MessageSender) *MenuManager {
	handlers := NewMenuHandlersWithUtils(cfg, messageSender, NewDefaultMenuUtils())
	return &MenuManager{
		handlers: handlers,
	}
}

// NewMenuManagerWithAuth создает менеджер меню с поддержкой авторизации
func NewMenuManagerWithAuth(cfg *config.Config, messageSender *MessageSender, authHandlers *AuthHandlers) *MenuManager {
	handlers := NewMenuHandlersWithUtils(cfg, messageSender, NewDefaultMenuUtils())
	handlers.SetAuthHandlers(authHandlers)
	return &MenuManager{
		handlers: handlers,
	}
}

// NewMenuManagerWithUserServices создает менеджер меню с сервисами пользователей
func NewMenuManagerWithUserServices(cfg *config.Config, messageSender *MessageSender, userService *users.Service, settingsManager *users.SettingsManager) *MenuManager {
	handlers := NewMenuHandlersWithServices(cfg, messageSender, userService, settingsManager)
	return &MenuManager{
		handlers: handlers,
	}
}

// HandleMessage обрабатывает текстовые сообщения
func (mm *MenuManager) HandleMessage(text, chatID string) error {
	return mm.handlers.HandleMessage(text, chatID)
}

// HandleCallback обрабатывает callback от inline кнопок
func (mm *MenuManager) HandleCallback(callbackData, chatID string) error {
	return mm.handlers.HandleCallback(callbackData, chatID)
}

// HandleCommand обрабатывает текстовые команды
func (mm *MenuManager) HandleCommand(cmd, chatID string) error {
	return mm.handlers.HandleCommand(cmd, chatID)
}

// StartCommandHandler обрабатывает команду /start
func (mm *MenuManager) StartCommandHandler(chatID string) error {
	return mm.handlers.StartCommandHandler(chatID)
}

// SendSettingsInfo отправляет информацию о настройках
func (mm *MenuManager) SendSettingsInfo(chatID string) error {
	userID := mm.handlers.getUserIDFromChatID(chatID)
	return mm.handlers.SendSettingsInfo(chatID, userID)
}

// SendStatus отправляет статус системы
func (mm *MenuManager) SendStatus(chatID string) error {
	userID := mm.handlers.getUserIDFromChatID(chatID)
	return mm.handlers.SendStatus(chatID, userID)
}

// SendNotificationsInfo отправляет информацию об уведомлениях
func (mm *MenuManager) SendNotificationsInfo(chatID string) error {
	userID := mm.handlers.getUserIDFromChatID(chatID)
	return mm.handlers.SendNotificationsInfo(chatID, userID)
}

// SendSignalTypesInfo отправляет информацию о типах сигналов
func (mm *MenuManager) SendSignalTypesInfo(chatID string) error {
	userID := mm.handlers.getUserIDFromChatID(chatID)
	return mm.handlers.SendSignalTypesInfo(chatID, userID)
}

// SendPeriodsInfo отправляет информацию о периодах
func (mm *MenuManager) SendPeriodsInfo(chatID string) error {
	userID := mm.handlers.getUserIDFromChatID(chatID)
	return mm.handlers.SendPeriodsInfo(chatID, userID)
}

// SendResetInfo отправляет информацию о сбросе
func (mm *MenuManager) SendResetInfo(chatID string) error {
	userID := mm.handlers.getUserIDFromChatID(chatID)
	return mm.handlers.SendResetInfo(chatID, userID)
}

// HandleNotifyOn включает уведомления
func (mm *MenuManager) HandleNotifyOn(chatID string) error {
	userID := mm.handlers.getUserIDFromChatID(chatID)
	return mm.handlers.HandleNotifyOn(chatID, userID)
}

// HandleNotifyOff выключает уведомления
func (mm *MenuManager) HandleNotifyOff(chatID string) error {
	userID := mm.handlers.getUserIDFromChatID(chatID)
	return mm.handlers.HandleNotifyOff(chatID, userID)
}

// HandlePeriodChange обрабатывает изменение периода
func (mm *MenuManager) HandlePeriodChange(chatID, period string) error {
	userID := mm.handlers.getUserIDFromChatID(chatID)
	return mm.handlers.HandlePeriodChange(chatID, userID, period)
}

// HandleResetAllCounters сбрасывает все счетчики
func (mm *MenuManager) HandleResetAllCounters(chatID string) error {
	return mm.handlers.HandleResetAllCounters(chatID)
}

// SendSymbolSelectionInline отправляет inline меню выбора символа
func (mm *MenuManager) SendSymbolSelectionInline(chatID string) error {
	return mm.handlers.SendSymbolSelectionInline(chatID)
}

// SendHelp отправляет справку
func (mm *MenuManager) SendHelp(chatID string) error {
	return mm.handlers.SendHelp(chatID)
}

// SetUserServices устанавливает сервисы пользователей
func (mm *MenuManager) SetUserServices(userService *users.Service, settingsManager *users.SettingsManager) {
	mm.handlers.SetUserServices(userService, settingsManager)
}

// SetAuthHandlers устанавливает обработчики авторизации
func (mm *MenuManager) SetAuthHandlers(authHandlers *AuthHandlers) {
	mm.handlers.SetAuthHandlers(authHandlers)
}

// GetAuthHandlers возвращает обработчики авторизации
func (mm *MenuManager) GetAuthHandlers() *AuthHandlers {
	return mm.handlers.GetAuthHandlers()
}
