// internal/delivery/telegram/menu_manager.go
package telegram

import (
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"log"
	"sync"
)

// MenuManager - менеджер меню
type MenuManager struct {
	config        *config.Config
	enabled       bool
	mu            sync.RWMutex
	messageSender *MessageSender
	handlers      *MenuHandlers
	keyboards     *MenuKeyboards
}

// NewMenuManager создает новый менеджер меню
func NewMenuManager(cfg *config.Config, messageSender *MessageSender) *MenuManager {
	handlers := NewMenuHandlers(cfg, messageSender)
	keyboards := NewMenuKeyboards()

	return &MenuManager{
		config:        cfg,
		enabled:       true,
		messageSender: messageSender,
		handlers:      handlers,
		keyboards:     keyboards,
	}
}

// SetEnabled включает/выключает меню
func (mm *MenuManager) SetEnabled(enabled bool) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	mm.enabled = enabled

	if enabled {
		mm.SetupMenu()
	} else {
		mm.RemoveMenu()
	}
}

// IsEnabled возвращает статус меню
func (mm *MenuManager) IsEnabled() bool {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	return mm.enabled
}

// SetupMenu устанавливает главное меню (2 ряда)
func (mm *MenuManager) SetupMenu() error {
	if !mm.IsEnabled() {
		return nil
	}

	menu := mm.keyboards.GetMainMenu()
	return mm.messageSender.SetReplyKeyboard(menu)
}

// RemoveMenu удаляет меню
func (mm *MenuManager) RemoveMenu() error {
	menu := ReplyKeyboardMarkup{
		RemoveKeyboard: true,
		Selective:      false,
	}

	return mm.messageSender.SetReplyKeyboard(menu)
}

// StartCommandHandler обрабатывает команду /start
func (mm *MenuManager) StartCommandHandler(chatID string) error {
	return mm.handlers.StartCommandHandler(chatID)
}

// HandleMessage обрабатывает текстовые сообщения
func (mm *MenuManager) HandleMessage(text, chatID string) error {
	log.Printf("📝 Handling menu message from chat %s: %s", chatID, text)
	return mm.handlers.HandleMessage(text, chatID)
}

// HandleCallback обрабатывает callback от inline кнопок
func (mm *MenuManager) HandleCallback(callbackData string, chatID string) error {
	log.Printf("🔄 Handling callback: %s for chat %s", callbackData, chatID)
	return mm.handlers.HandleCallback(callbackData, chatID)
}

// Делегирующие методы для совместимости
func (mm *MenuManager) SendSettingsMessage(chatID string) error {
	mm.messageSender.SetReplyKeyboard(mm.keyboards.GetSettingsMenu())
	return mm.handlers.SendSettingsInfo(chatID)
}

func (mm *MenuManager) SendStatus(chatID string) error {
	return mm.handlers.SendStatus(chatID)
}

func (mm *MenuManager) SendHelp(chatID string) error {
	return mm.handlers.SendHelp(chatID)
}

func (mm *MenuManager) SendNotificationsMenu(chatID string) error {
	mm.messageSender.SetReplyKeyboard(mm.keyboards.GetNotificationsMenu())
	return mm.handlers.SendNotificationsInfo(chatID)
}

func (mm *MenuManager) SendSignalTypesMenu(chatID string) error {
	mm.messageSender.SetReplyKeyboard(mm.keyboards.GetSignalTypesMenu())
	return mm.handlers.SendSignalTypesInfo(chatID)
}

func (mm *MenuManager) SendPeriodMenu(chatID string) error {
	mm.messageSender.SetReplyKeyboard(mm.keyboards.GetPeriodsMenu())
	return mm.handlers.SendPeriodsInfo(chatID)
}

func (mm *MenuManager) SendResetMenu(chatID string) error {
	mm.messageSender.SetReplyKeyboard(mm.keyboards.GetResetMenu())
	return mm.handlers.SendResetInfo(chatID)
}

func (mm *MenuManager) HandleNotifyOn(chatID string) error {
	return mm.handlers.HandleNotifyOn(chatID)
}

func (mm *MenuManager) HandleNotifyOff(chatID string) error {
	return mm.handlers.HandleNotifyOff(chatID)
}

func (mm *MenuManager) HandlePeriodChange(chatID string, period string) error {
	return mm.handlers.HandlePeriodChange(chatID, period)
}

func (mm *MenuManager) HandleResetAllCounters(chatID string) error {
	return mm.handlers.HandleResetAllCounters(chatID)
}
