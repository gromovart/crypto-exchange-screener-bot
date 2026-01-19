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

	// Polling handler
	pollingHandler *PollingClient

	mu          sync.RWMutex
	startupTime time.Time
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
	InitHandlerFactory(handlerFactory, notificationsToggleService, signalSettingsService)

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

	// Создаем polling handler
	bot.pollingHandler = NewPollingClient(bot)

	// Устанавливаем меню команд Telegram
	if err := bot.SetMyCommands(); err != nil {
		logger.Warn("Не удалось установить меню команд: %v", err)
		logger.Info("Бот будет работать, но меню команд в Telegram может не отображаться")
	}

	return bot
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

// Для обратной совместимости со старым webhook.go

// HandleMessage обрабатывает текстовое сообщение (старый метод)
func (b *TelegramBot) HandleMessage(text, chatID string) error {
	// TODO: Реализовать через новую систему
	return nil
}

// HandleCallback обрабатывает callback (старый метод)
func (b *TelegramBot) HandleCallback(callbackData, chatID string) error {
	// TODO: Реализовать через новую систему
	return nil
}

// StartCommandHandler обработчик команды /start (старый метод)
func (b *TelegramBot) StartCommandHandler(chatID string) error {
	// TODO: Реализовать через новую систему
	return nil
}

// SendTestMessage отправляет тестовое сообщение (старый метод)
func (b *TelegramBot) SendTestMessage() error {
	// TODO: Реализовать через новую систему
	return nil
}

// SendMessage отправляет сообщение (старый метод)
func (b *TelegramBot) SendMessage(text string) error {
	return b.messageSender.SendTextMessage(b.messageSender.GetChatID(), text, nil)
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

// IsRunning проверяет работает ли бот (для интерфейса TelegramBotClient)
func (b *TelegramBot) IsRunning() bool {
	return b.pollingHandler != nil && b.pollingHandler.running
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

// Добавляю методы для polling:
func (b *TelegramBot) StartPolling() error {
	return b.pollingHandler.Start()
}

func (b *TelegramBot) StopPolling() error {
	return b.pollingHandler.Stop()
}

func (b *TelegramBot) IsPolling() bool {
	return b.pollingHandler != nil && b.pollingHandler.running
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
