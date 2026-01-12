// internal/delivery/telegram/menu_handlers.go
package telegram

import (
	"crypto-exchange-screener-bot/internal/core/domain/users"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"fmt"
	"log"
)

// MenuHandlers - обработчики меню
type MenuHandlers struct {
	config          *config.Config
	messageSender   *MessageSender
	keyboardSystem  *KeyboardSystem
	menuUtils       *MenuUtils
	authHandlers    *AuthHandlers
	settingsManager *users.SettingsManager
	userService     *users.Service
	userMapping     *UserMappingService
}

// NewMenuHandlers создает новые обработчики меню
func NewMenuHandlers(cfg *config.Config, messageSender *MessageSender) *MenuHandlers {
	menuUtils := NewDefaultMenuUtils()
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

// NewMenuHandlersWithAuth создает обработчики меню с поддержкой авторизации
func NewMenuHandlersWithAuth(cfg *config.Config, messageSender *MessageSender, authHandlers *AuthHandlers) *MenuHandlers {
	menuUtils := NewDefaultMenuUtils()
	keyboardSystem := NewKeyboardSystem(cfg.Exchange)

	return &MenuHandlers{
		config:          cfg,
		messageSender:   messageSender,
		keyboardSystem:  keyboardSystem,
		menuUtils:       menuUtils,
		authHandlers:    authHandlers,
		settingsManager: nil,
		userService:     nil,
		userMapping:     NewUserMappingService(nil),
	}
}

// NewMenuHandlersWithServices создает обработчики меню с сервисами пользователей
func NewMenuHandlersWithServices(cfg *config.Config, messageSender *MessageSender, userService *users.Service, settingsManager *users.SettingsManager) *MenuHandlers {
	menuUtils := NewDefaultMenuUtils()
	keyboardSystem := NewKeyboardSystem(cfg.Exchange)

	return &MenuHandlers{
		config:          cfg,
		messageSender:   messageSender,
		keyboardSystem:  keyboardSystem,
		menuUtils:       menuUtils,
		authHandlers:    nil,
		settingsManager: settingsManager,
		userService:     userService,
		userMapping:     NewUserMappingService(userService),
	}
}

// SetUserServices устанавливает сервисы пользователей
func (mh *MenuHandlers) SetUserServices(userService *users.Service, settingsManager *users.SettingsManager) {
	mh.userService = userService
	mh.settingsManager = settingsManager
	mh.userMapping = NewUserMappingService(userService)
	log.Printf("✅ User services установлены в MenuHandlers")
}

// SetAuthHandlers устанавливает обработчики авторизации
func (mh *MenuHandlers) SetAuthHandlers(authHandlers *AuthHandlers) {
	mh.authHandlers = authHandlers
}

// GetAuthHandlers возвращает обработчики авторизации
func (mh *MenuHandlers) GetAuthHandlers() *AuthHandlers {
	return mh.authHandlers
}

// InvalidateUserCache инвалидирует кэш пользователя
func (mh *MenuHandlers) InvalidateUserCache(chatID string) {
	mh.userMapping.InvalidateCache(chatID)
}

// getUserIDFromChatID получает userID из chatID
func (mh *MenuHandlers) getUserIDFromChatID(chatID string) int {
	return mh.userMapping.GetUserID(chatID)
}

// hasUserServices проверяет, доступны ли сервисы пользователей
func (mh *MenuHandlers) hasUserServices() bool {
	return mh.userService != nil && mh.settingsManager != nil
}

// HandleMessage обрабатывает текстовые сообщения из меню
func (mh *MenuHandlers) HandleMessage(text, chatID string) error {
	log.Printf("🔍 HandleMessage вызван: text='%s', chatID='%s'", text, chatID)

	// Получаем userID для работы с настройками
	userID := mh.getUserIDFromChatID(chatID)

	switch text {
	case "⚙️ Настройки":
		return mh.handleSettings(chatID, userID)
	case "📊 Статус":
		return mh.SendStatus(chatID, userID)
	case "🔔 Уведомления":
		return mh.handleNotifications(chatID, userID)
	case "✅ Включить":
		return mh.HandleNotifyOn(chatID, userID)
	case "❌ Выключить":
		return mh.HandleNotifyOff(chatID, userID)
	case "📈 Сигналы":
		return mh.handleSignals(chatID, userID)
	case "📈 Только рост":
		return mh.handleGrowthOnly(chatID, userID)
	case "📉 Только падение":
		return mh.handleFallOnly(chatID, userID)
	case "📊 Все сигналы":
		return mh.handleAllSignals(chatID, userID)
	case "⏱️ Периоды":
		return mh.handlePeriods(chatID, userID)
	case "⏱️ 5мин", "⏱️ 5 мин":
		return mh.HandlePeriodChange(chatID, userID, "5m")
	case "⏱️ 15мин", "⏱️ 15 мин":
		return mh.HandlePeriodChange(chatID, userID, "15m")
	case "⏱️ 30мин", "⏱️ 30 мин":
		return mh.HandlePeriodChange(chatID, userID, "30m")
	case "⏱️ 1 час":
		return mh.HandlePeriodChange(chatID, userID, "1h")
	case "⏱️ 4 часа":
		return mh.HandlePeriodChange(chatID, userID, "4h")
	case "🔄 Сбросить":
		return mh.handleReset(chatID, userID)
	case "🔄 Все счетчики":
		return mh.HandleResetAllCounters(chatID)
	case "📋 Помощь":
		return mh.SendHelp(chatID)
	case "🔙 Назад", "🔙 Главное меню":
		return mh.handleBack(chatID)
	default:
		return mh.handleDefault(text, chatID)
	}
}

// HandleCallback обрабатывает callback от inline кнопок
func (mh *MenuHandlers) HandleCallback(callbackData string, chatID string) error {
	log.Printf("🔄 Handling callback: %s for chat %s", callbackData, chatID)

	// Получаем userID для работы с настройками
	userID := mh.getUserIDFromChatID(chatID)

	// Проверяем, относится ли callback к авторизации
	if mh.isAuthCallback(callbackData) {
		return mh.handleAuthCallback(callbackData, chatID)
	}

	// Используем menuUtils для парсинга callback данных
	action, params := mh.menuUtils.ParseCallbackData(callbackData)

	// Обрабатываем основные callback действия
	switch action {
	case "menu":
		if len(params) > 0 {
			return mh.handleMenuCallback(params[0], chatID, userID)
		}
	case "period":
		if len(params) > 0 {
			return mh.HandlePeriodChange(chatID, userID, params[0])
		}
	case "reset":
		if len(params) > 0 {
			return mh.handleResetCallback(params[0], callbackData, chatID, userID)
		}
	case "notify":
		if len(params) > 0 {
			return mh.handleNotifyCallback(params[0], chatID, userID)
		}
	case CallbackStats:
		return mh.SendStatus(chatID, userID)
	case CallbackSettingsMain: // Изменено с CallbackSettings
		return mh.handleCallbackSettings(chatID, userID)
	case CallbackNotifyToggle: // Изменено с CallbackSettingsNotifyToggle
		return mh.handleNotifyToggle(chatID, userID)
	case CallbackSignalsMenu: // Изменено с CallbackSettingsSignalType
		return mh.handleSignalTypeCallback(chatID, userID)
	case CallbackNotifyGrowthOnly: // Изменено с CallbackTrackGrowthOnly
		return mh.handleTrackGrowthOnly(chatID, userID)
	case CallbackNotifyFallOnly: // Изменено с CallbackTrackFallOnly
		return mh.handleTrackFallOnly(chatID, userID)
	case CallbackNotifyBoth: // Изменено с CallbackTrackBoth
		return mh.handleTrackBoth(chatID, userID)
	case CallbackPeriodSelect: // Изменено с CallbackSettingsChangePeriod
		return mh.handleChangePeriodCallback(chatID, userID)
	case CallbackPeriod5m:
		return mh.HandlePeriodChange(chatID, userID, "5m")
	case CallbackPeriod15m:
		return mh.HandlePeriodChange(chatID, userID, "15m")
	case CallbackPeriod30m:
		return mh.HandlePeriodChange(chatID, userID, "30m")
	case CallbackPeriod1h:
		return mh.HandlePeriodChange(chatID, userID, "1h")
	case CallbackPeriod4h:
		return mh.HandlePeriodChange(chatID, userID, "4h")
	case CallbackPeriod1d:
		return mh.HandlePeriodChange(chatID, userID, "1d")
	case CallbackMenuBack: // Изменено с CallbackSettingsBack
		return mh.handleSettingsBack(chatID, userID)
	case CallbackMenuMain: // Изменено с CallbackSettingsBackToMain
		return mh.handleBackToMain(chatID)
	case CallbackResetCounters: // Изменено с CallbackSettingsResetCounter
		return mh.handleResetCounterCallback(chatID, userID)
	case CallbackResetAll:
		return mh.HandleResetAllCounters(chatID)
	case CallbackResetBySymbol:
		return mh.SendSymbolSelectionInline(chatID)
	case "help":
		return mh.SendHelp(chatID)
	case "chart":
		return mh.handleChartCallback(chatID)
	case "test_ok":
		return mh.handleTestOK(chatID)
	case "test_cancel":
		return mh.handleTestCancel(chatID)
	case "toggle_test_mode":
		return mh.handleToggleTestMode(chatID)
	}

	return fmt.Errorf("unknown callback data: %s", callbackData)
}
