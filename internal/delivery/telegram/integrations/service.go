// internal/delivery/telegram/integrations/service.go
package integrations

import (
	"fmt"
	"log"
	"sync"
	"time"

	"crypto-exchange-screener-bot/internal/core/domain/subscription"
	"crypto-exchange-screener-bot/internal/core/domain/users"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/buttons"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/formatters"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/message_sender"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/middlewares"
	counterctrl "crypto-exchange-screener-bot/internal/delivery/telegram/controllers/counter"
	countersvc "crypto-exchange-screener-bot/internal/delivery/telegram/services/counter"
	profilesvc "crypto-exchange-screener-bot/internal/delivery/telegram/services/profile"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
)

// PackageStats статистика пакета
type PackageStats struct {
	ProfileRequests   int64  `json:"profile_requests"`
	CounterSignals    int64  `json:"counter_signals"`
	RegularSignals    int64  `json:"regular_signals"`
	NotificationsSent int64  `json:"notifications_sent"`
	Errors            int64  `json:"errors"`
	StartTime         string `json:"start_time"`
}

// telegramPackageServiceImpl реализация TelegramPackageService
type telegramPackageServiceImpl struct {
	config              *config.Config
	userService         *users.Service
	subscriptionService *subscription.Service
	eventBus            types.EventBus

	// Внутренние компоненты
	botClient      TelegramBotClient
	messageSender  message_sender.MessageSender
	profileService profilesvc.Service
	counterService countersvc.Service

	// Контроллеры
	counterController counterctrl.Controller

	// Управление
	mu                sync.RWMutex
	isRunning         bool
	eventBusConnected bool

	// Статистика
	stats PackageStats
}

// NewTelegramPackageService создает новый главный сервис Telegram пакета
func NewTelegramPackageService(
	config *config.Config,
	userService *users.Service,
	subscriptionService *subscription.Service,
	eventBus types.EventBus,
	botClient TelegramBotClient,
) (TelegramPackageService, error) {

	logger.Info("🤖 Creating Telegram package service...")

	// 1. Проверяем обязательные зависимости
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}
	if userService == nil {
		return nil, fmt.Errorf("userService is required")
	}
	if subscriptionService == nil {
		return nil, fmt.Errorf("subscriptionService is required")
	}
	if eventBus == nil {
		return nil, fmt.Errorf("eventBus is required")
	}
	if botClient == nil {
		return nil, fmt.Errorf("botClient is required")
	}

	// 2. ИСПОЛЬЗУЕМ MessageSender из botClient вместо создания нового
	messageSender := botClient.GetMessageSender()
	if messageSender == nil {
		// Fallback: создаем реальный MessageSender только если у botClient его нет
		if config.TelegramEnabled && config.TelegramBotToken != "" {
			messageSender = message_sender.NewMessageSender(config)
			logger.Warn("⚠️ Created MessageSender as fallback (botClient didn't provide one)")
		} else {
			// Используем stub
			messageSender = &stubMessageSender{}
			log.Println("⚠️ Using stub message sender (Telegram disabled or no token)")
		}
	} else {
		logger.Info("✅ Using MessageSender from botClient")
	}

	// 3. СОЗДАЕМ BUTTON BUILDER
	buttonBuilder := buttons.NewButtonBuilder()
	logger.Info("🛠️ ButtonBuilder created")

	// 4. Создаем провайдер форматтеров
	formatterProvider := formatters.NewFormatterProvider("BYBIT") // Можно брать из конфига

	// 5. Создаем внутренние сервисы
	profileService := profilesvc.NewService(userService, subscriptionService)
	counterService := countersvc.NewService(userService, formatterProvider, messageSender, buttonBuilder)

	// 6. Создаем контроллеры
	counterController := counterctrl.NewController(counterService)

	service := &telegramPackageServiceImpl{
		config:              config,
		userService:         userService,
		subscriptionService: subscriptionService,
		eventBus:            eventBus,
		botClient:           botClient,
		messageSender:       messageSender,
		profileService:      profileService,
		counterService:      counterService,
		counterController:   counterController,
		isRunning:           false,
		eventBusConnected:   false,
		stats: PackageStats{
			StartTime: time.Now().Format(time.RFC3339),
		},
	}

	logger.Info("✅ Telegram package service created")
	return service, nil
}

// GetUserProfile возвращает профиль пользователя
func (s *telegramPackageServiceImpl) GetUserProfile(userID int64) (*ProfileData, error) {
	s.mu.Lock()
	s.stats.ProfileRequests++
	s.mu.Unlock()

	log.Printf("📊 Getting profile for user %d", userID)

	result, err := s.profileService.Exec(profilesvc.ProfileParams{
		UserID: userID,
		Action: "get",
	})

	if err != nil {
		s.mu.Lock()
		s.stats.Errors++
		s.mu.Unlock()
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}

	profileResult, ok := result.(profilesvc.ProfileResult)
	if !ok {
		return nil, fmt.Errorf("invalid profile result type")
	}

	if !profileResult.Success {
		return nil, fmt.Errorf("profile service returned error")
	}

	return &ProfileData{
		User:    profileResult.Data,
		Message: profileResult.Message,
	}, nil
}

// HandleCounterSignal обрабатывает событие счетчика
func (s *telegramPackageServiceImpl) HandleCounterSignal(event types.Event) error {
	s.mu.Lock()
	s.stats.CounterSignals++
	s.mu.Unlock()

	logger.Debug("🔢 Handling counter signal: %s", event.Type)
	return s.counterController.HandleEvent(event)
}

// HandleRegularSignal обрабатывает регулярное событие сигнала
func (s *telegramPackageServiceImpl) HandleRegularSignal(event types.Event) error {
	s.mu.Lock()
	s.stats.RegularSignals++
	s.mu.Unlock()

	log.Printf("📡 Handling regular signal: %s", event.Type)
	return nil
}

// SendUserNotification отправляет уведомление пользователю
func (s *telegramPackageServiceImpl) SendUserNotification(userID int64, message string) error {
	s.mu.Lock()
	s.stats.NotificationsSent++
	s.mu.Unlock()

	log.Printf("📨 Sending notification to user %d", userID)

	user, err := s.userService.GetUserByID(int(userID))
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return fmt.Errorf("user not found")
	}

	chatID := user.TelegramID
	return s.messageSender.SendTextMessage(chatID, message, nil)
}

// GetPackageStats возвращает статистику пакета
func (s *telegramPackageServiceImpl) GetPackageStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"profile_requests":    s.stats.ProfileRequests,
		"counter_signals":     s.stats.CounterSignals,
		"regular_signals":     s.stats.RegularSignals,
		"notifications_sent":  s.stats.NotificationsSent,
		"errors":              s.stats.Errors,
		"start_time":          s.stats.StartTime,
		"is_running":          s.isRunning,
		"event_bus_connected": s.eventBusConnected,
		"services": map[string]bool{
			"profile_service": s.profileService != nil,
			"counter_service": s.counterService != nil,
			"bot_client":      s.botClient != nil,
			"message_sender":  s.messageSender != nil,
		},
	}
}

// GetHealthStatus возвращает статус здоровья сервиса
func (s *telegramPackageServiceImpl) GetHealthStatus() HealthStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	servicesStatus := make(map[string]string)

	checkService := func(name string, service interface{}) {
		if service != nil {
			servicesStatus[name] = "healthy"
		} else {
			servicesStatus[name] = "unhealthy"
		}
	}

	checkService("profile_service", s.profileService)
	checkService("counter_service", s.counterService)
	checkService("bot_client", s.botClient)
	checkService("message_sender", s.messageSender)

	overallStatus := "healthy"
	for _, status := range servicesStatus {
		if status == "unhealthy" {
			overallStatus = "degraded"
			break
		}
	}

	return HealthStatus{
		Status:   overallStatus,
		Services: servicesStatus,
		EventBus: EventBusStatus{
			Connected:    s.eventBusConnected,
			Subscribers:  1,
			EventsSent:   s.stats.CounterSignals + s.stats.RegularSignals,
			EventsFailed: s.stats.Errors,
		},
		LastUpdated: time.Now().Format(time.RFC3339),
	}
}

// Start запускает сервис
func (s *telegramPackageServiceImpl) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("service already running")
	}

	log.Println("🚀 Starting Telegram package service...")
	s.eventBusConnected = true
	s.isRunning = true
	log.Println("✅ Telegram package service started")
	return nil
}

// Stop останавливает сервис
func (s *telegramPackageServiceImpl) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return nil
	}

	log.Println("🛑 Stopping Telegram package service...")
	s.eventBusConnected = false
	s.isRunning = false
	log.Println("✅ Telegram package service stopped")
	return nil
}

// IsRunning проверяет работает ли сервис
func (s *telegramPackageServiceImpl) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isRunning
}

// maskToken маскирует токен для безопасного логирования
func maskToken(token string) string {
	if len(token) < 10 {
		return "***"
	}
	return token[:6] + "..." + token[len(token)-4:]
}

// stubMessageSender заглушка для MessageSender
type stubMessageSender struct{}

func (s *stubMessageSender) SendTextMessage(chatID int64, text string, keyboard interface{}) error {
	log.Printf("[STUB] Send message to %d: %s", chatID, text[:min(50, len(text))])
	return nil
}

func (s *stubMessageSender) SendMessageWithKeyboard(chatID int64, text string, keyboard interface{}) error {
	log.Printf("[STUB] Send message with keyboard to %d: %s", chatID, text[:min(50, len(text))])
	return nil
}

func (s *stubMessageSender) EditMessageText(chatID, messageID int64, text string, keyboard interface{}) error {
	log.Printf("[STUB] Edit message %d in chat %d: %s", messageID, chatID, text[:min(50, len(text))])
	return nil
}

func (s *stubMessageSender) DeleteMessage(chatID, messageID int64) error {
	log.Printf("[STUB] Delete message %d in chat %d", messageID, chatID)
	return nil
}

func (s *stubMessageSender) AnswerCallback(callbackID, text string, showAlert bool) error {
	log.Printf("[STUB] Answer callback %s: %s (showAlert: %v)", callbackID, text, showAlert)
	return nil
}

func (s *stubMessageSender) SetChatID(chatID int64) {
	log.Printf("[STUB] Set chat ID: %d", chatID)
}

func (s *stubMessageSender) GetChatID() int64 {
	return 0
}

func (s *stubMessageSender) SetTestMode(enabled bool) {
	log.Printf("[STUB] Set test mode: %v", enabled)
}

func (s *stubMessageSender) IsTestMode() bool {
	return false
}

// stubTelegramBotClient заглушка для TelegramBotClient
type stubTelegramBotClient struct {
	config *config.Config
}

func (s *stubTelegramBotClient) SendTextMessage(chatID int64, text string, keyboard interface{}) error {
	log.Printf("[STUB BOT] Send message to %d: %s", chatID, text[:min(50, len(text))])
	return nil
}

func (s *stubTelegramBotClient) GetMessageSender() message_sender.MessageSender {
	// Возвращаем nil, чтобы использовался реальный MessageSender из конфигурации
	return nil
}

func (s *stubTelegramBotClient) HandleUpdate(update *middlewares.TelegramUpdate) error {
	log.Printf("[STUB BOT] Handle update")
	return nil
}

func (s *stubTelegramBotClient) IsRunning() bool {
	return true
}

func (s *stubTelegramBotClient) GetConfig() *config.Config {
	return s.config
}

// NewTelegramPackageServiceWithDefaults создает сервис с ботом по умолчанию
func NewTelegramPackageServiceWithDefaults(
	config *config.Config,
	userService *users.Service,
	subscriptionService *subscription.Service,
	eventBus types.EventBus,
) (TelegramPackageService, error) {

	// Используем существующий TelegramBot из синглтона
	existingBot := bot.GetBot()
	if existingBot == nil {
		// Если бот еще не создан, создаем stub
		logger.Warn("⚠️ TelegramBot not available, using stub")
		botClient := &stubTelegramBotClient{
			config: config,
		}

		return NewTelegramPackageService(
			config,
			userService,
			subscriptionService,
			eventBus,
			botClient,
		)
	}

	// Используем существующий бот
	botClient := existingBot // *bot.TelegramBot уже реализует TelegramBotClient

	return NewTelegramPackageService(
		config,
		userService,
		subscriptionService,
		eventBus,
		botClient,
	)
}
