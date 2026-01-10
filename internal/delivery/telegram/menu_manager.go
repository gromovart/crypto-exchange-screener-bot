// internal/delivery/telegram/menu_manager.go
package telegram

import (
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"log"
	"sync"
	"time"
)

// MenuManager - менеджер меню
type MenuManager struct {
	config         *config.Config
	enabled        bool
	mu             sync.RWMutex
	messageSender  *MessageSender
	handlers       *MenuHandlers
	keyboardSystem *KeyboardSystem // ВМЕСТО MenuKeyboards
	menuUtils      *MenuUtils
}

// NewMenuManager создает новый менеджер меню (старый конструктор)
func NewMenuManager(cfg *config.Config, messageSender *MessageSender) *MenuManager {
	// Используем новый конструктор с KeyboardSystem
	keyboardSystem := NewKeyboardSystem(cfg.Exchange)
	menuUtils := NewDefaultMenuUtils()
	handlers := NewMenuHandlersWithUtils(cfg, messageSender, menuUtils)

	return &MenuManager{
		config:         cfg,
		enabled:        true,
		messageSender:  messageSender,
		handlers:       handlers,
		keyboardSystem: keyboardSystem,
		menuUtils:      menuUtils,
	}
}

// NewMenuManagerWithUtils создает менеджер меню с утилитами
func NewMenuManagerWithUtils(cfg *config.Config, messageSender *MessageSender, menuUtils *MenuUtils) *MenuManager {
	// Используем новый конструктор с KeyboardSystem
	keyboardSystem := NewKeyboardSystem(cfg.Exchange)
	handlers := NewMenuHandlersWithUtils(cfg, messageSender, menuUtils)

	return &MenuManager{
		config:         cfg,
		enabled:        true,
		messageSender:  messageSender,
		handlers:       handlers,
		keyboardSystem: keyboardSystem,
		menuUtils:      menuUtils,
	}
}

// NewMenuManagerWithKeyboardSystem создает менеджер меню с KeyboardSystem
func NewMenuManagerWithKeyboardSystem(cfg *config.Config, messageSender *MessageSender, keyboardSystem *KeyboardSystem) *MenuManager {
	menuUtils := NewDefaultMenuUtils()
	handlers := NewMenuHandlersWithUtils(cfg, messageSender, menuUtils)

	return &MenuManager{
		config:         cfg,
		enabled:        true,
		messageSender:  messageSender,
		handlers:       handlers,
		keyboardSystem: keyboardSystem,
		menuUtils:      menuUtils,
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

// SetupMenu устанавливает главное меню
func (mm *MenuManager) SetupMenu() error {
	if !mm.IsEnabled() {
		return nil
	}

	// Используем KeyboardSystem для получения главного меню
	menu := mm.keyboardSystem.GetMainMenu()

	// Добавляем retry логику при отсутствии интернета
	maxRetries := 3
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		err := mm.messageSender.SetReplyKeyboard(mm.messageSender.GetChatID(), menu)
		if err == nil {
			log.Println("✅ Меню успешно установлено")
			return nil
		}

		lastErr = err
		log.Printf("⚠️ Попытка %d/%d установки меню не удалась: %v", i+1, maxRetries, err)

		// Не ждем перед следующей попыткой для последней попытки
		if i < maxRetries-1 {
			time.Sleep(2 * time.Second)
		}
	}

	log.Printf("⚠️ Failed to setup menu after %d retries: %v", maxRetries, lastErr)
	return lastErr
}

// RemoveMenu удаляет меню
func (mm *MenuManager) RemoveMenu() error {
	menu := ReplyKeyboardMarkup{
		RemoveKeyboard: true,
		Selective:      false,
	}

	return mm.messageSender.SetReplyKeyboard(mm.messageSender.GetChatID(), menu)
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

// GetMenuUtils возвращает утилиты меню
func (mm *MenuManager) GetMenuUtils() *MenuUtils {
	return mm.menuUtils
}

// GetKeyboardSystem возвращает систему клавиатур
func (mm *MenuManager) GetKeyboardSystem() *KeyboardSystem {
	return mm.keyboardSystem
}

// SendSettingsMessage отправляет сообщение настроек
func (mm *MenuManager) SendSettingsMessage(chatID string) error {
	// Используем KeyboardSystem для получения меню настроек
	menu := mm.keyboardSystem.GetSettingsMenu()
	mm.messageSender.SetReplyKeyboard(chatID, menu)
	return mm.handlers.SendSettingsInfo(chatID)
}

// SendStatus отправляет статус системы
func (mm *MenuManager) SendStatus(chatID string) error {
	return mm.handlers.SendStatus(chatID)
}

// SendHelp отправляет справку
func (mm *MenuManager) SendHelp(chatID string) error {
	return mm.handlers.SendHelp(chatID)
}

// SendNotificationsMenu отправляет меню уведомлений
func (mm *MenuManager) SendNotificationsMenu(chatID string) error {
	menu := mm.keyboardSystem.GetNotificationsMenu()
	mm.messageSender.SetReplyKeyboard(chatID, menu)
	return mm.handlers.SendNotificationsInfo(chatID)
}

// SendSignalTypesMenu отправляет меню типов сигналов
func (mm *MenuManager) SendSignalTypesMenu(chatID string) error {
	menu := mm.keyboardSystem.GetSignalTypesMenu()
	mm.messageSender.SetReplyKeyboard(chatID, menu)
	return mm.handlers.SendSignalTypesInfo(chatID)
}

// SendPeriodMenu отправляет меню периодов
func (mm *MenuManager) SendPeriodMenu(chatID string) error {
	menu := mm.keyboardSystem.GetPeriodsMenu()
	mm.messageSender.SetReplyKeyboard(chatID, menu)
	return mm.handlers.SendPeriodsInfo(chatID)
}

// SendResetMenu отправляет меню сброса
func (mm *MenuManager) SendResetMenu(chatID string) error {
	menu := mm.keyboardSystem.GetResetMenu()
	mm.messageSender.SetReplyKeyboard(chatID, menu)
	return mm.handlers.SendResetInfo(chatID)
}

// HandleNotifyOn включает уведомления
func (mm *MenuManager) HandleNotifyOn(chatID string) error {
	return mm.handlers.HandleNotifyOn(chatID)
}

// HandleNotifyOff выключает уведомления
func (mm *MenuManager) HandleNotifyOff(chatID string) error {
	return mm.handlers.HandleNotifyOff(chatID)
}

// HandlePeriodChange обрабатывает изменение периода
func (mm *MenuManager) HandlePeriodChange(chatID string, period string) error {
	return mm.handlers.HandlePeriodChange(chatID, period)
}

// HandleResetAllCounters сбрасывает все счетчики
func (mm *MenuManager) HandleResetAllCounters(chatID string) error {
	return mm.handlers.HandleResetAllCounters(chatID)
}

// GetMainMenu возвращает главное меню (для внешнего использования)
func (mm *MenuManager) GetMainMenu() ReplyKeyboardMarkup {
	return mm.keyboardSystem.GetMainMenu()
}

// GetSettingsMenu возвращает меню настроек (для внешнего использования)
func (mm *MenuManager) GetSettingsMenu() ReplyKeyboardMarkup {
	return mm.keyboardSystem.GetSettingsMenu()
}

// CreateNotificationKeyboard создает клавиатуру для уведомлений
func (mm *MenuManager) CreateNotificationKeyboard(symbol string, periodMinutes int) *InlineKeyboardMarkup {
	return mm.keyboardSystem.CreateNotificationKeyboard(symbol, periodMinutes)
}

// CreateEnhancedNotificationKeyboard создает расширенную клавиатуру для уведомлений
func (mm *MenuManager) CreateEnhancedNotificationKeyboard(symbol string, periodMinutes int) *InlineKeyboardMarkup {
	return mm.keyboardSystem.CreateEnhancedNotificationKeyboard(symbol, periodMinutes)
}

// CreateCounterNotificationKeyboard создает клавиатуру для счетчика
func (mm *MenuManager) CreateCounterNotificationKeyboard(symbol string, periodMinutes int) *InlineKeyboardMarkup {
	return mm.keyboardSystem.CreateCounterNotificationKeyboard(symbol, periodMinutes)
}

// ClearKeyboardCache очищает кэш клавиатур
func (mm *MenuManager) ClearKeyboardCache() {
	mm.keyboardSystem.ClearCache()
}

// SetupAuth настраивает авторизацию (алиас для SetupAuthHandlers)
func (mm *MenuManager) SetupAuth(authHandlers *AuthHandlers) {
	// В этой версии MenuManager нет поддержки авторизации
	// Просто логируем вызов для отладки
	// Передаем authHandlers в MenuHandlers
	if mm.handlers != nil {
		mm.handlers.SetAuthHandlers(authHandlers)
		log.Printf("✅ AuthHandlers установлены в MenuHandlers")
	} else {
		log.Printf("⚠️ MenuHandlers не инициализированы, auth не настроена")
	}
}
