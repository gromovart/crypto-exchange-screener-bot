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
	menuUtils     *MenuUtils // ДОБАВЛЕНО
}

// NewMenuManager создает новый менеджер меню (старый конструктор)
func NewMenuManager(cfg *config.Config, messageSender *MessageSender) *MenuManager {
	// Используем старый конструктор для обратной совместимости
	handlers := NewMenuHandlers(cfg, messageSender)
	keyboards := NewMenuKeyboards()

	// Создаем menuUtils по умолчанию
	menuUtils := NewDefaultMenuUtils()

	return &MenuManager{
		config:        cfg,
		enabled:       true,
		messageSender: messageSender,
		handlers:      handlers,
		keyboards:     keyboards,
		menuUtils:     menuUtils, // ДОБАВЛЕНО
	}
}

// NewMenuManagerWithUtils создает менеджер меню с утилитами
func NewMenuManagerWithUtils(cfg *config.Config, messageSender *MessageSender, menuUtils *MenuUtils) *MenuManager {
	// Используем новый конструктор с утилитами
	handlers := NewMenuHandlersWithUtils(cfg, messageSender, menuUtils)
	keyboards := NewMenuKeyboards()

	return &MenuManager{
		config:        cfg,
		enabled:       true,
		messageSender: messageSender,
		handlers:      handlers,
		keyboards:     keyboards,
		menuUtils:     menuUtils, // ДОБАВЛЕНО
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

	// Используем menuUtils для создания меню, если доступно
	var menu ReplyKeyboardMarkup
	if mm.menuUtils != nil {
		menu = mm.menuUtils.FormatCompactMenu()
	} else {
		// Fallback на старый метод
		menu = mm.keyboards.GetMainMenu()
	}

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

// GetMenuUtils возвращает утилиты меню (ДОБАВЛЕН МЕТОД)
func (mm *MenuManager) GetMenuUtils() *MenuUtils {
	return mm.menuUtils
}

// SendSettingsMessage отправляет сообщение настроек
func (mm *MenuManager) SendSettingsMessage(chatID string) error {
	// Используем menuUtils для создания меню настроек
	var menu ReplyKeyboardMarkup
	if mm.menuUtils != nil {
		menu = mm.menuUtils.FormatSettingsMenu()
	} else {
		// Fallback на старый метод
		menu = mm.keyboards.GetSettingsMenu()
	}

	mm.messageSender.SetReplyKeyboard(menu)
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
