// internal/delivery/telegram/app/bot/bot.go
package bot

import (
	"crypto-exchange-screener-bot/internal/core/domain/subscription"
	"crypto-exchange-screener-bot/internal/core/domain/users"
	"crypto-exchange-screener-bot/internal/delivery/telegram"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/router"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/message_sender"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/middlewares"
	telegram_http "crypto-exchange-screener-bot/internal/delivery/telegram/app/http_client"
	services_factory "crypto-exchange-screener-bot/internal/delivery/telegram/services/factory"
	"crypto-exchange-screener-bot/internal/delivery/telegram/services/notifications_toggle"
	payment_service "crypto-exchange-screener-bot/internal/delivery/telegram/services/payment"
	profile_service "crypto-exchange-screener-bot/internal/delivery/telegram/services/profile"
	signal_settings_service "crypto-exchange-screener-bot/internal/delivery/telegram/services/signal_settings"
	tbank_service "crypto-exchange-screener-bot/internal/delivery/telegram/services/tbank"
	trading_session "crypto-exchange-screener-bot/internal/delivery/telegram/services/trading_session"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	currency_client "crypto-exchange-screener-bot/internal/infrastructure/http/currency"
	tbank_client "crypto-exchange-screener-bot/internal/infrastructure/http/tbank"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Dependencies зависимости для TelegramBot
type Dependencies struct {
	ServiceFactory *services_factory.ServiceFactory
}

// TelegramBot - бот для отправки уведомлений в Telegram
type TelegramBot struct {
	config *config.Config

	// HTTP клиенты
	telegramClient *telegram_http.TelegramClient
	pollingClient  *telegram_http.PollingClient
	starsClient    *telegram_http.StarsClient

	// MessageSender для отправки сообщений
	messageSender message_sender.MessageSender

	// Роутер для обработки команд
	router                 router.Router
	authMiddleware         *middlewares.AuthMiddleware
	subscriptionMiddleware *middlewares.SubscriptionMiddleware

	// Режимы работы
	pollingHandler *PollingClient
	webhookServer  *WebhookServer
	tbankServer    *TBankNotifyServer
	mu             sync.RWMutex
	startupTime    time.Time
	currentMode    string // "polling" или "webhook"
}

// NewTelegramBot создает новый экземпляр TelegramBot
func NewTelegramBot(config *config.Config, deps *Dependencies) *TelegramBot {
	// Создаем MessageSender
	ms := message_sender.NewMessageSender(config)

	// Создаем HTTP клиенты
	baseURL := "https://api.telegram.org/bot" + config.TelegramBotToken + "/"
	telegramClient := telegram_http.NewTelegramClient(baseURL)
	pollingClient := telegram_http.NewPollingClient(baseURL)

	// Создаем StarsClient для работы с платежами Telegram Stars
	// Для цифровых товаров provider_token может быть пустой строкой ""
	starsClient := telegram_http.NewStarsClient(baseURL, "")
	logger.Info("✅ StarsClient создан для работы с Telegram Stars API")

	// Получаем UserService из ServiceFactory
	var userService *users.Service
	if deps.ServiceFactory != nil {
		userService = deps.ServiceFactory.GetUserService()
	} else {
		logger.Error("❌ ServiceFactory не предоставлена в зависимостях")
		userService = nil
	}

	// Получаем SubscriptionService из ServiceFactory
	var subscriptionService *subscription.Service
	if deps.ServiceFactory != nil {
		subscriptionService = deps.ServiceFactory.GetSubscriptionService()
	} else {
		logger.Warn("⚠️ ServiceFactory не предоставлена, SubscriptionService будет nil")
	}

	// Создаем middleware аутентификации
	authMiddleware := middlewares.NewAuthMiddleware(
		userService,
		subscriptionService,
		deps.ServiceFactory.GetSubscriptionRepository(),
	)

	// Создаем middleware подписки
	subscriptionMiddleware := middlewares.NewSubscriptionMiddleware(subscriptionService)

	// Создаем фабрику хэндлеров и роутер
	handlerFactory := handlers.NewHandlerFactory()

	// Создаем сервисы
	var profileSvc profile_service.Service
	if deps.ServiceFactory != nil {
		profileSvc = deps.ServiceFactory.CreateProfileService()
		logger.Info("✅ ProfileService создан через фабрику")
	} else {
		logger.Error("❌ ServiceFactory не предоставлена, ProfileService не может быть создан")
		profileSvc = nil
	}

	notificationsToggleService := notifications_toggle.NewService(userService)
	signalSettingsService := signal_settings_service.NewServiceWithDependencies(userService)

	// Получаем TradingSessionService из ServiceFactory (единственный экземпляр)
	var tradingSessionService trading_session.Service
	if deps.ServiceFactory != nil {
		tradingSessionService = deps.ServiceFactory.GetTradingSessionService()
	}
	if tradingSessionService == nil {
		// Fallback: создаём если фабрика недоступна
		tradingSessionService = trading_session.NewService(userService, ms)
	}

	// Получаем PaymentService из ServiceFactory
	var paymentService payment_service.Service
	if deps.ServiceFactory != nil {
		paymentService = deps.ServiceFactory.CreatePaymentService()
	} else {
		logger.Warn("⚠️ ServiceFactory не предоставлена, PaymentService будет nil")
		paymentService = nil
	}

	// Создаем CurrencyClient для актуального курса USD/RUB от ЦБ РФ
	currencyClient := currency_client.NewClient()
	logger.Info("💱 CurrencyClient создан (резервный курс: %.0f ₽/$)", currency_client.FallbackRate)

	// Создаем TBankService если Т-Банк настроен
	var tbankSvc tbank_service.Service
	if config.TBank.Enabled && config.TBank.TerminalKey != "" && config.TBank.Password != "" {
		tbankHTTPClient := tbank_client.NewClient(config.TBank.TerminalKey, config.TBank.Password)
		tbankSvc = tbank_service.NewService(tbank_service.Dependencies{
			TBankClient:         tbankHTTPClient,
			SubscriptionService: subscriptionService,
			UserService:         userService,
			MessageSender:       ms,
			Password:            config.TBank.Password,
			NotifyURL:           config.TBank.NotifyURL,
			SuccessURL:          config.TBank.SuccessURL,
			FailURL:             config.TBank.FailURL,
		})
		logger.Info("✅ TBankService создан (терминал: %s)", config.TBank.TerminalKey)
	} else {
		logger.Info("ℹ️ Т-Банк не настроен (TBANK_ENABLED=false или отсутствуют ключи)")
	}

	// Создаем структуру сервисов
	services := &Services{
		signalSettingsService:      signalSettingsService,
		notificationsToggleService: notificationsToggleService,
		paymentService:             paymentService,
		profileService:             profileSvc,
		starsClient:                starsClient,
		tradingSessionService:      tradingSessionService,
		userService:                userService,
		tbankService:               tbankSvc,
		currencyClient:             currencyClient,
		subscriptionService:        subscriptionService,
	}

	// Инициализируем фабрику с сервисами
	InitHandlerFactory(handlerFactory, config, services, subscriptionMiddleware)

	// Регистрируем все хэндлеры
	router := handlerFactory.RegisterAllHandlers()

	bot := &TelegramBot{
		config:                 config,
		telegramClient:         telegramClient,
		pollingClient:          pollingClient,
		starsClient:            starsClient,
		messageSender:          ms,
		router:                 router,
		authMiddleware:         authMiddleware,
		subscriptionMiddleware: subscriptionMiddleware,
		startupTime:            time.Now(),
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

	// Создаем сервер уведомлений Т-Банк если сервис настроен
	if tbankSvc != nil {
		bot.tbankServer = NewTBankNotifyServer(tbankSvc, config.TBank.NotifyPort)
		logger.Info("🏦 TBankNotifyServer создан (порт: %d)", config.TBank.NotifyPort)
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

	// Запускаем сервер уведомлений Т-Банк (независимо от режима)
	b.startTBankServer()

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

	b.stopTBankServer()

	if b.currentMode == "polling" {
		return b.stopPolling()
	} else {
		return b.stopWebhook()
	}
}

// startTBankServer запускает сервер уведомлений Т-Банк если настроен
func (b *TelegramBot) startTBankServer() {
	if b.tbankServer != nil {
		if err := b.tbankServer.Start(); err != nil {
			logger.Error("❌ Ошибка запуска TBankNotifyServer: %v", err)
		} else {
			logger.Info("✅ TBankNotifyServer запущен")
		}
	}
}

// stopTBankServer останавливает сервер уведомлений Т-Банк
func (b *TelegramBot) stopTBankServer() {
	if b.tbankServer != nil {
		if err := b.tbankServer.Stop(); err != nil {
			logger.Warn("⚠️ Ошибка остановки TBankNotifyServer: %v", err)
		}
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
	if b.webhookServer != nil {
		return b.webhookServer.Stop()
	}
	return nil
}

// HandleUpdate обрабатывает обновление от Telegram
func (b *TelegramBot) HandleUpdate(update *telegram.TelegramUpdate) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// СПЕЦИАЛЬНАЯ ОБРАБОТКА ДЛЯ PRE-CHECKOUT QUERY (ЭВЕНТ)
	if update.PreCheckoutQuery != nil {
		logger.Info("💰 Получен PreCheckoutQuery эвент: ID=%s, пользователь=%d, сумма=%d %s, payload=%s",
			update.PreCheckoutQuery.ID,
			update.PreCheckoutQuery.From.ID,
			update.PreCheckoutQuery.TotalAmount,
			update.PreCheckoutQuery.Currency,
			update.PreCheckoutQuery.InvoicePayload)

		// Обрабатываем через auth middleware
		handlerParams, err := b.authMiddleware.ProcessUpdate(update)
		if err != nil {
			logger.Error("❌ Ошибка аутентификации pre_checkout_query: %v", err)
			return b.starsClient.AnswerPreCheckoutQuery(update.PreCheckoutQuery.ID, false, "Ошибка авторизации")
		}

		// Вызываем обработчик pre_checkout_query
		result, err := b.router.Handle("pre_checkout_query", convertToRouterParams(handlerParams))
		if err != nil {
			logger.Error("❌ Ошибка обработки pre_checkout_query: %v", err)
			return b.starsClient.AnswerPreCheckoutQuery(update.PreCheckoutQuery.ID, false, "Внутренняя ошибка сервера")
		}

		// Отправляем ответ через StarsClient
		if result.Metadata != nil {
			if params, ok := result.Metadata["telegram_params"].(map[string]interface{}); ok {
				queryID, _ := params["pre_checkout_query_id"].(string)
				ok, _ := params["ok"].(bool)
				errorMessage, _ := params["error_message"].(string)
				return b.starsClient.AnswerPreCheckoutQuery(queryID, ok, errorMessage)
			}
		}
		return b.starsClient.AnswerPreCheckoutQuery(update.PreCheckoutQuery.ID, true, "")
	}

	// СПЕЦИАЛЬНАЯ ОБРАБОТКА ДЛЯ SUCCESSFUL PAYMENT (ЭВЕНТ)
	if update.Message != nil && update.Message.SuccessfulPayment != nil {
		logger.Warn("💰💰💰 [SUCCESSFUL PAYMENT] ПОЛУЧЕН В BOT!")
		logger.Warn("   • From: %d", update.Message.From.ID)
		logger.Warn("   • Amount: %d %s", update.Message.SuccessfulPayment.TotalAmount, update.Message.SuccessfulPayment.Currency)
		logger.Warn("   • Payload: %s", update.Message.SuccessfulPayment.InvoicePayload)
		logger.Warn("   • TelegramChargeID: %s", update.Message.SuccessfulPayment.TelegramPaymentChargeID)
		logger.Warn("   • ProviderChargeID: %s", update.Message.SuccessfulPayment.ProviderPaymentChargeID)

		// Обрабатываем через auth middleware
		handlerParams, err := b.authMiddleware.ProcessUpdate(update)
		if err != nil {
			logger.Error("❌ Ошибка аутентификации successful_payment: %v", err)
			return b.messageSender.SendTextMessage(handlerParams.ChatID,
				"❌ Ошибка авторизации. Пожалуйста, попробуйте позже.", nil)
		}

		// Вызываем обработчик successful_payment
		logger.Warn("🔄 Вызов роутера для successful_payment")
		result, err := b.router.Handle("successful_payment", convertToRouterParams(handlerParams))
		if err != nil {
			logger.Error("❌ Ошибка обработки successful_payment: %v", err)
			return b.messageSender.SendTextMessage(handlerParams.ChatID,
				"❌ Ошибка обработки платежа. Пожалуйста, обратитесь в поддержку.", nil)
		}

		// Отправляем сообщение пользователю
		logger.Warn("✅ Отправка подтверждения пользователю")
		return b.messageSender.SendTextMessage(handlerParams.ChatID, result.Message, result.Keyboard)
	}

	// Обычная обработка сообщений и callback-ов
	handlerParams, err := b.authMiddleware.ProcessUpdate(update)
	if err != nil {
		return b.sendAuthError(handlerParams.ChatID, err.Error())
	}

	var command string
	if update.Message != nil && update.Message.Text != "" {
		command = update.Message.Text
	} else if update.CallbackQuery != nil {
		command = update.CallbackQuery.Data
	} else {
		return nil
	}

	result, err := b.router.Handle(command, convertToRouterParams(handlerParams))
	if err != nil {
		errText := strings.ReplaceAll(err.Error(), "_", "\\_")
		return b.messageSender.SendTextMessage(handlerParams.ChatID, "Ошибка: "+errText, nil)
	}

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

	b.startTBankServer()

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
func convertToRouterParams(params handlers.HandlerParams) router.HandlerParams {
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

// GetRouter возвращает роутер
func (b *TelegramBot) GetRouter() router.Router {
	return b.router
}

// GetAuthMiddleware возвращает middleware аутентификации
func (b *TelegramBot) GetAuthMiddleware() *middlewares.AuthMiddleware {
	return b.authMiddleware
}

// GetSubscriptionMiddleware возвращает middleware подписки
func (b *TelegramBot) GetSubscriptionMiddleware() *middlewares.SubscriptionMiddleware {
	return b.subscriptionMiddleware
}

// SetMyCommands устанавливает меню команд в Telegram
func (b *TelegramBot) SetMyCommands() error {
	logger.Info("Установка меню команд в Telegram API")

	// Список команд для меню (используем константы)
	commands := []telegram.BotCommand{
		{Command: "/start", Description: constants.CommandDescriptions.Start},
		{Command: "/help", Description: constants.CommandDescriptions.Help},
		{Command: "/buy", Description: constants.CommandDescriptions.Buy},
		{Command: "/paysupport", Description: constants.CommandDescriptions.PaySupport},
		{Command: "/terms", Description: constants.CommandDescriptions.Terms},
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
