package telegram

import (
	"crypto-exchange-screener-bot/internal/core/domain/users"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

var (
	// Singleton экземпляр Telegram бота
	botInstance *TelegramBot
	botOnce     sync.Once

	// Singleton экземпляр для мониторинга
	monitoringBotInstance *TelegramBot
	monitoringBotOnce     sync.Once

	// Список всех созданных ботов для мониторинга
	monitoringBots []*TelegramBot
	monitoringMu   sync.RWMutex

	// Обработчик обновлений для Singleton бота
	updatesHandlerInstance *UpdatesHandler
	updatesHandlerOnce     sync.RWMutex
)

// GetOrCreateBot получает или создает Singleton Telegram бота
func GetOrCreateBot(cfg *config.Config) *TelegramBot {
	botOnce.Do(func() {
		if cfg == nil || cfg.TelegramBotToken == "" {
			log.Println("⚠️ Telegram Bot Token не указан")
			return
		}

		botInstance = createBot(cfg)

		log.Printf("✅ Telegram бот создан (Singleton, auth: %v)", botInstance.HasAuth())
	})

	return botInstance
}

// GetOrCreateBotWithAuth получает или создает Singleton Telegram бота с поддержкой авторизации
func GetOrCreateBotWithAuth(cfg *config.Config, userService *users.Service) *TelegramBot {
	var botCreated bool

	botOnce.Do(func() {
		if cfg == nil || cfg.TelegramBotToken == "" {
			log.Println("⚠️ Telegram Bot Token не указан")
			return
		}

		botInstance = createBotWithAuth(cfg, userService)

		log.Printf("✅ Telegram бот создан с авторизацией (Singleton, auth: %v)", botInstance.HasAuth())
		botCreated = true
	})

	// Если бот уже был создан (без авторизации), обновляем авторизацию
	if !botCreated && botInstance != nil && userService != nil {
		log.Println("🔄 Обновление userService в существующем боте...")

		// Устанавливаем userService и инициализируем авторизацию
		if err := botInstance.SetUserService(userService); err != nil {
			log.Printf("⚠️ Ошибка обновления авторизации: %v", err)
		} else {
			log.Println("✅ Авторизация обновлена в существующем боте")
		}
	}

	// Запускаем обработчик обновлений для Singleton бота
	// (даже если бот был создан ранее)
	startUpdatesHandlerForSingleton()

	return botInstance
}

// GetBot возвращает Singleton экземпляр бота (без создания)
func GetBot() *TelegramBot {
	return botInstance
}

// GetMonitoringBot создает или получает бота для мониторинга
func GetMonitoringBot(cfg *config.Config, chatID string) *TelegramBot {
	monitoringMu.Lock()
	defer monitoringMu.Unlock()

	// Ищем существующий бот для этого chat_id
	for _, bot := range monitoringBots {
		if bot.chatID == chatID {
			return bot
		}
	}

	// Создаем нового бота
	bot := NewTelegramBotWithChatID(cfg, chatID)
	if bot != nil {
		monitoringBots = append(monitoringBots, bot)
		log.Printf("📱 Добавлен бот для мониторинга с chat_id: %s (всего: %d)", chatID, len(monitoringBots))
	}

	return bot
}

// GetMonitoringBotWithAuth создает или получает бота для мониторинга с авторизацией
func GetMonitoringBotWithAuth(cfg *config.Config, chatID string, userService *users.Service) *TelegramBot {
	monitoringMu.Lock()
	defer monitoringMu.Unlock()

	// Ищем существующий бот для этого chat_id
	for _, bot := range monitoringBots {
		if bot.chatID == chatID {
			return bot
		}
	}

	// Создаем нового бота с авторизацией
	bot := NewTelegramBotWithChatIDAndAuth(cfg, chatID, userService)
	if bot != nil {
		monitoringBots = append(monitoringBots, bot)
		log.Printf("📱 Добавлен бот для мониторинга с авторизацией (chat_id: %s, всего: %d)", chatID, len(monitoringBots))
	}

	return bot
}

// GetAllMonitoringBots возвращает всех ботов для мониторинга
func GetAllMonitoringBots() []*TelegramBot {
	monitoringMu.RLock()
	defer monitoringMu.RUnlock()
	return monitoringBots
}

// RemoveMonitoringBot удаляет бота для мониторинга
func RemoveMonitoringBot(chatID string) bool {
	monitoringMu.Lock()
	defer monitoringMu.Unlock()

	for i, bot := range monitoringBots {
		if bot.chatID == chatID {
			// Удаляем бота из списка
			monitoringBots = append(monitoringBots[:i], monitoringBots[i+1:]...)
			log.Printf("🗑️ Удален бот для мониторинга с chat_id: %s (осталось: %d)", chatID, len(monitoringBots))
			return true
		}
	}

	return false
}

// ClearMonitoringBots очищает всех ботов для мониторинга
func ClearMonitoringBots() {
	monitoringMu.Lock()
	defer monitoringMu.Unlock()

	count := len(monitoringBots)
	monitoringBots = nil
	log.Printf("🧹 Очищены все боты для мониторинга (%d штук)", count)
}

// GetMonitoringBotCount возвращает количество ботов для мониторинга
func GetMonitoringBotCount() int {
	monitoringMu.RLock()
	defer monitoringMu.RUnlock()
	return len(monitoringBots)
}

// GetOrCreateUpdatesHandler получает или создает обработчик обновлений для Singleton бота
func GetOrCreateUpdatesHandler(cfg *config.Config) *UpdatesHandler {
	updatesHandlerOnce.Lock()
	defer updatesHandlerOnce.Unlock()

	if updatesHandlerInstance == nil && botInstance != nil {
		log.Println("🔧 Создание UpdatesHandler для Singleton бота...")

		// Создаем обработчик с авторизацией если она настроена
		if botInstance.HasAuth() {
			updatesHandlerInstance = NewUpdatesHandlerWithAuth(cfg, botInstance, botInstance.GetAuthHandlers())
		} else {
			updatesHandlerInstance = NewUpdatesHandler(cfg, botInstance)
		}

		log.Println("✅ UpdatesHandler создан")
	}

	return updatesHandlerInstance
}

// StartUpdatesHandler запускает обработчик обновлений для Singleton бота
func StartUpdatesHandler() error {
	updatesHandlerOnce.Lock()
	defer updatesHandlerOnce.Unlock()

	if updatesHandlerInstance == nil {
		// Получаем или создаем обработчик
		updatesHandler := GetOrCreateUpdatesHandler(botInstance.config)
		if updatesHandler == nil {
			return nil // Если обработчик не может быть создан
		}
		updatesHandlerInstance = updatesHandler
	}

	// Запускаем обработчик в отдельной горутине
	go func() {
		log.Println("🚀 Запуск UpdatesHandler...")
		if err := updatesHandlerInstance.Start(); err != nil {
			log.Printf("❌ Ошибка запуска UpdatesHandler: %v", err)
		} else {
			log.Println("✅ UpdatesHandler запущен")
		}
	}()

	return nil
}

// GetUpdatesHandlerInstance возвращает экземпляр обработчика обновлений
func GetUpdatesHandlerInstance() *UpdatesHandler {
	updatesHandlerOnce.RLock()
	defer updatesHandlerOnce.RUnlock()
	return updatesHandlerInstance
}

// StopUpdatesHandler останавливает обработчик обновлений
func StopUpdatesHandler() error {
	updatesHandlerOnce.Lock()
	defer updatesHandlerOnce.Unlock()

	if updatesHandlerInstance != nil {
		log.Println("🛑 Остановка UpdatesHandler...")
		if err := updatesHandlerInstance.Stop(); err != nil {
			log.Printf("❌ Ошибка остановки UpdatesHandler: %v", err)
			return err
		}
		updatesHandlerInstance = nil
		log.Println("✅ UpdatesHandler остановлен")
	}

	return nil
}

// startUpdatesHandlerForSingleton запускает обработчик обновлений для Singleton бота
func startUpdatesHandlerForSingleton() {
	if botInstance == nil || !botInstance.HasAuth() {
		log.Println("⚠️ Не удалось запустить UpdatesHandler: бот не создан или авторизация не настроена")
		return
	}

	log.Println("🔧 Запуск UpdatesHandler для Singleton бота...")

	// Создаем обработчик с использованием блокировки
	updatesHandlerOnce.Lock()
	defer updatesHandlerOnce.Unlock()

	// Создаем обработчик с авторизацией
	updatesHandlerInstance = NewUpdatesHandlerWithAuth(
		botInstance.config,
		botInstance,
		botInstance.GetAuthHandlers(),
	)

	// НАСТРАИВАЕМ КОМАНДЫ АВТОРИЗАЦИИ
	// Получаем AuthInitializer
	authInitializer := botInstance.GetAuthInitializer()
	if authInitializer != nil && botInstance.GetAuthHandlers() != nil {
		authInitializer.SetupAuthCommands(updatesHandlerInstance, botInstance.GetAuthHandlers())
		log.Println("✅ Команды авторизации настроены для UpdatesHandler")
	} else {
		log.Println("⚠️ Не удалось настроить команды авторизации: AuthInitializer или AuthHandlers недоступны")
	}

	// Запускаем в отдельной горутине
	go func() {
		log.Println("🚀 Запуск UpdatesHandler...")
		if err := updatesHandlerInstance.Start(); err != nil {
			log.Printf("❌ Ошибка запуска UpdatesHandler: %v", err)
		} else {
			log.Println("✅ UpdatesHandler запущен для Singleton бота")
		}
	}()
}

// Вспомогательная функция для создания бота (без авторизации)
func createBot(cfg *config.Config) *TelegramBot {
	messageSender := NewMessageSender(cfg)
	menuUtils := NewMenuUtils(cfg.Exchange)
	notifier := NewNotifier(cfg)
	notifier.SetMessageSender(messageSender)

	// Создаем менеджер меню
	menuManager := NewMenuManagerWithUtils(cfg, messageSender, menuUtils)

	// Создаем buttonBuilder для кнопок
	buttonBuilder := NewButtonURLBuilder(cfg.Exchange)

	bot := &TelegramBot{
		config:        cfg,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
		baseURL:       fmt.Sprintf("https://api.telegram.org/bot%s/", cfg.TelegramBotToken),
		chatID:        cfg.TelegramChatID,
		notifier:      notifier,
		menuManager:   menuManager,
		messageSender: messageSender,
		startupTime:   time.Now(),
		welcomeSent:   false, // Отправлять приветствие!
		testMode:      cfg.MonitoringTestMode || false,
		buttonBuilder: buttonBuilder,
		menuUtils:     menuUtils,
		userService:   nil, // Без авторизации
	}

	return bot
}

// Вспомогательная функция для создания бота с авторизацией
func createBotWithAuth(cfg *config.Config, userService *users.Service) *TelegramBot {
	messageSender := NewMessageSender(cfg)
	menuUtils := NewMenuUtils(cfg.Exchange)
	notifier := NewNotifier(cfg)
	notifier.SetMessageSender(messageSender)

	// Создаем менеджер меню
	menuManager := NewMenuManagerWithUtils(cfg, messageSender, menuUtils)

	// Создаем buttonBuilder для кнопок
	buttonBuilder := NewButtonURLBuilder(cfg.Exchange)

	bot := &TelegramBot{
		config:        cfg,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
		baseURL:       fmt.Sprintf("https://api.telegram.org/bot%s/", cfg.TelegramBotToken),
		chatID:        cfg.TelegramChatID,
		notifier:      notifier,
		menuManager:   menuManager,
		messageSender: messageSender,
		startupTime:   time.Now(),
		welcomeSent:   false, // Отправлять приветствие!
		testMode:      cfg.MonitoringTestMode || false,
		buttonBuilder: buttonBuilder,
		menuUtils:     menuUtils,
		userService:   userService,
	}

	// Инициализируем авторизацию если userService предоставлен
	if userService != nil {
		if err := bot.initAuth(); err != nil {
			log.Printf("⚠️ Ошибка инициализации авторизации: %v", err)
		}
	}
	return bot
}
