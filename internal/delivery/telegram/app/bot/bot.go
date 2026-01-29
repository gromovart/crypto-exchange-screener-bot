// internal/delivery/telegram/app/bot/bot.go
package bot

import (
	"crypto-exchange-screener-bot/internal/core/domain/users"
	"crypto-exchange-screener-bot/internal/delivery/telegram"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/router"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/message_sender"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/middlewares"
	telegram_http "crypto-exchange-screener-bot/internal/delivery/telegram/app/http_client"
	"crypto-exchange-screener-bot/internal/delivery/telegram/services/notifications_toggle"
	signal_settings_service "crypto-exchange-screener-bot/internal/delivery/telegram/services/signal_settings"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"sync"
	"time"
)

// TelegramBot - бот для отправки уведомлений в Telegram
type TelegramBot struct {
	config *config.Config

	// HTTP клиенты
	telegramClient *telegram_http.TelegramClient
	pollingClient  *telegram_http.PollingClient

	// MessageSender для отправки сообщений
	messageSender message_sender.MessageSender

	// Новая система хэндлеров
	handlerFactory *handlers.HandlerFactory
	router         router.Router
	authMiddleware *middlewares.AuthMiddleware

	// Режимы работы
	pollingHandler *PollingClient
	webhookServer  *WebhookServer
	mu             sync.RWMutex
	startupTime    time.Time
	currentMode    string // "polling" или "webhook"
}

// Dependencies зависимости для TelegramBot
type Dependencies struct {
	UserService *users.Service
}

// NewTelegramBot создает новый экземпляр TelegramBot
func NewTelegramBot(config *config.Config, deps *Dependencies) *TelegramBot {
	// Создаем MessageSender
	ms := message_sender.NewMessageSender(config)

	// Создаем HTTP клиенты
	baseURL := "https://api.telegram.org/bot" + config.TelegramBotToken + "/"
	telegramClient := telegram_http.NewTelegramClient(baseURL)
	pollingClient := telegram_http.NewPollingClient(baseURL)

	// Создаем middleware аутентификации
	authMiddleware := middlewares.NewAuthMiddleware(deps.UserService)

	// Создаем фабрику хэндлеров
	handlerFactory := handlers.NewHandlerFactory()

	// Создаем сервис для переключения уведомлений
	notificationsToggleService := notifications_toggle.NewService(deps.UserService)

	// Создаем сервис настройки сигналов
	signalSettingsService := signal_settings_service.NewServiceWithDependencies(deps.UserService)

	// Инициализируем фабрику с сервисом
	InitHandlerFactory(handlerFactory, notificationsToggleService, signalSettingsService, config)

	// Регистрируем все хэндлеры
	router := handlerFactory.RegisterAllHandlers()

	bot := &TelegramBot{
		config:         config,
		telegramClient: telegramClient,
		pollingClient:  pollingClient,
		messageSender:  ms,
		handlerFactory: handlerFactory,
		router:         router,
		authMiddleware: authMiddleware,
		startupTime:    time.Now(),
	}

	// Определяем текущий режим работы
	bot.currentMode = "polling"
	if config.IsWebhookMode() {
		bot.currentMode = "webhook"
	}

	logger.Info("🤖 TelegramBot создан (режим: %s)", bot.currentMode)

	// Создаем обработчики для выбранного режима
	if bot.currentMode == "polling" {
		bot.pollingHandler = NewPollingClient(bot)
		logger.Info("🔄 PollingHandler создан")
	} else {
		bot.webhookServer = NewWebhookServer(config, bot)
		logger.Info("🌐 WebhookServer создан")
	}

	// Устанавливаем меню команд Telegram
	if err := bot.SetMyCommands(); err != nil {
		logger.Warn("Не удалось установить меню команд: %v", err)
		logger.Info("Бот будет работать, но меню команд в Telegram может не отображаться")
	}

	return bot
}

// Start запускает бота в выбранном режиме
func (b *TelegramBot) Start() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	logger.Info("🚀 Запуск Telegram бота (режим: %s)", b.currentMode)

	if b.currentMode == "polling" {
		return b.startPolling()
	} else {
		return b.startWebhook()
	}
}

// Stop останавливает бота
func (b *TelegramBot) Stop() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	logger.Info("🛑 Остановка Telegram бота (режим: %s)", b.currentMode)

	if b.currentMode == "polling" {
		return b.stopPolling()
	} else {
		return b.stopWebhook()
	}
}

// startPolling запускает polling режим
func (b *TelegramBot) startPolling() error {
	if b.pollingHandler == nil {
		return fmt.Errorf("polling handler не инициализирован")
	}

	logger.Info("🔄 Запуск polling режима...")
	return b.pollingHandler.Start()
}

// stopPolling останавливает polling режим
func (b *TelegramBot) stopPolling() error {
	if b.pollingHandler == nil {
		return nil
	}

	logger.Info("🛑 Остановка polling режима...")
	return b.pollingHandler.Stop()
}

// startWebhook запускает webhook режим
func (b *TelegramBot) startWebhook() error {
	if b.webhookServer == nil {
		return fmt.Errorf("webhook server не инициализирован")
	}

	logger.Info("🌐 Запуск webhook режима на порту %d...", b.config.HTTPPort)
	return b.webhookServer.Start()
}

// stopWebhook останавливает webhook режим
func (b *TelegramBot) stopWebhook() error {
	if b.webhookServer == nil {
		return nil
	}

	logger.Info("🛑 Остановка webhook режима...")
	return b.webhookServer.Stop()
}

// HandleUpdate обрабатывает обновление от Telegram (новая система)
func (b *TelegramBot) HandleUpdate(update *middlewares.TelegramUpdate) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Обрабатываем обновление через auth middleware
	handlerParams, err := b.authMiddleware.ProcessUpdate(update)
	if err != nil {
		// Если ошибка авторизации, отправляем сообщение об ошибке
		return b.sendAuthError(handlerParams.ChatID, err.Error())
	}

	// Определяем команду/callback для обработки
	var command string
	if update.Message != nil && update.Message.Text != "" {
		command = update.Message.Text
	} else if update.CallbackQuery != nil {
		command = update.CallbackQuery.Data
	} else {
		return nil // Игнорируем другие типы обновлений
	}

	// Обрабатываем команду через роутер
	result, err := b.router.Handle(command, convertToRouterParams(handlerParams))
	if err != nil {
		return b.messageSender.SendTextMessage(handlerParams.ChatID, "Ошибка: "+err.Error(), nil)
	}

	// Отправляем результат пользователю
	return b.messageSender.SendTextMessage(handlerParams.ChatID, result.Message, result.Keyboard)
}

// GetPollingClient возвращает polling клиент для polling.go
func (b *TelegramBot) GetPollingClient() *telegram_http.PollingClient {
	return b.pollingClient
}

// GetTelegramClient возвращает telegram клиент
func (b *TelegramBot) GetTelegramClient() *telegram_http.TelegramClient {
	return b.telegramClient
}

// SendTextMessage отправляет текстовое сообщение (для интерфейса TelegramBotClient)
func (b *TelegramBot) SendTextMessage(chatID int64, text string, keyboard interface{}) error {
	return b.messageSender.SendTextMessage(chatID, text, keyboard)
}

// GetMessageSender возвращает MessageSender для использования другими компонентами
func (b *TelegramBot) GetMessageSender() message_sender.MessageSender {
	return b.messageSender
}

// GetConfig возвращает конфигурацию
func (b *TelegramBot) GetConfig() *config.Config {
	return b.config
}

// IsRunning проверяет работает ли бот
func (b *TelegramBot) IsRunning() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.currentMode == "polling" {
		return b.pollingHandler != nil && b.pollingHandler.running
	} else {
		return b.webhookServer != nil
	}
}

// IsPolling проверяет работает ли бот в polling режиме
func (b *TelegramBot) IsPolling() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.currentMode == "polling" && b.pollingHandler != nil && b.pollingHandler.running
}

// IsWebhook проверяет работает ли бот в webhook режиме
func (b *TelegramBot) IsWebhook() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.currentMode == "webhook" && b.webhookServer != nil
}

// GetCurrentMode возвращает текущий режим работы
func (b *TelegramBot) GetCurrentMode() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.currentMode
}

// StartPolling запускает polling режим (для обратной совместимости с transport)
func (b *TelegramBot) StartPolling() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.currentMode != "polling" {
		return fmt.Errorf("бот работает в режиме %s, нельзя запустить polling", b.currentMode)
	}

	return b.startPolling()
}

// StopPolling останавливает polling режим (для обратной совместимости с transport)
func (b *TelegramBot) StopPolling() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.currentMode != "polling" {
		return nil // Если не polling режим, просто игнорируем
	}

	return b.stopPolling()
}

// Вспомогательные методы

// convertToRouterParams конвертирует HandlerParams в router.HandlerParams
func convertToRouterParams(params middlewares.HandlerParams) router.HandlerParams {
	return router.HandlerParams{
		User:     params.User,
		ChatID:   params.ChatID,
		Text:     params.Text,
		Data:     params.Data,
		UpdateID: params.UpdateID,
	}
}

// sendAuthError отправляет сообщение об ошибке авторизации
func (b *TelegramBot) sendAuthError(chatID int64, message string) error {
	errorMessage := "🔐 *Ошибка авторизации*\n\n" + message

	// Создаем инлайн клавиатуру для авторизации
	keyboard := telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{
				{Text: "🔑 Войти", CallbackData: "auth_login"},
			},
		},
	}

	return b.messageSender.SendTextMessage(chatID, errorMessage, keyboard)
}

// GetHandlerFactory возвращает фабрику хэндлеров
func (b *TelegramBot) GetHandlerFactory() *handlers.HandlerFactory {
	return b.handlerFactory
}

// GetRouter возвращает роутер
func (b *TelegramBot) GetRouter() router.Router {
	return b.router
}

// GetAuthMiddleware возвращает middleware аутентификации
func (b *TelegramBot) GetAuthMiddleware() *middlewares.AuthMiddleware {
	return b.authMiddleware
}

// SetMyCommands устанавливает меню команд в Telegram
func (b *TelegramBot) SetMyCommands() error {
	logger.Info("Установка меню команд в Telegram API")

	// Список команд для меню (используем константы)
	commands := []telegram.BotCommand{
		{Command: "/start", Description: constants.CommandDescriptions.Start},
		{Command: "/help", Description: constants.CommandDescriptions.Help},
		{Command: "/profile", Description: constants.CommandDescriptions.Profile},
		{Command: "/settings", Description: constants.CommandDescriptions.Settings},
		{Command: "/notifications", Description: constants.CommandDescriptions.Notifications},
		{Command: "/periods", Description: constants.CommandDescriptions.Periods},
		{Command: "/thresholds", Description: constants.CommandDescriptions.Thresholds},
		{Command: "/commands", Description: constants.CommandDescriptions.Commands},
		{Command: "/stats", Description: constants.CommandDescriptions.Stats},
	}

	logger.Debug("Подготовлено %d команд для отправки", len(commands))

	// Устанавливаем команды
	if err := b.telegramClient.SetMyCommands(commands); err != nil {
		logger.Error("Ошибка установки меню команд: %v", err)
		return fmt.Errorf("ошибка настройки меню команд: %v", err)
	}

	logger.Info("Меню команд успешно отправлено в Telegram API")

	// Логируем список команд только на уровне debug
	for _, cmd := range commands {
		logger.Debug("   • %s - %s", cmd.Command, cmd.Description)
	}

	return nil
}
