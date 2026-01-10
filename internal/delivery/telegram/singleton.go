// internal/delivery/telegram/singleton.go
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
	botInstance *TelegramBot
	botOnce     sync.Once
	botMutex    sync.RWMutex
)

// GetOrCreateBot создает или возвращает существующий экземпляр бота
func GetOrCreateBot(cfg *config.Config) *TelegramBot {
	return GetOrCreateBotWithAuth(cfg, nil)
}

// GetOrCreateBotWithAuth создает или возвращает существующий экземпляр бота с авторизацией
func GetOrCreateBotWithAuth(cfg *config.Config, userService *users.Service) *TelegramBot {
	if cfg == nil || cfg.TelegramBotToken == "" || cfg.TelegramChatID == "" {
		log.Println("⚠️ Telegram Bot Token или Chat ID не указаны, бот отключен")
		return nil
	}

	botOnce.Do(func() {
		log.Println("🤖 Создание Telegram бота (единственный экземпляр)...")
		botInstance = newTelegramBot(cfg, userService)

		if botInstance != nil {
			log.Printf("✅ Telegram бот создан (Singleton, auth: %v)", userService != nil)
		}
	})

	return botInstance
}

// GetBot возвращает существующий экземпляр бота (без создания)
func GetBot() *TelegramBot {
	botMutex.RLock()
	defer botMutex.RUnlock()
	return botInstance
}

// SetBot устанавливает экземпляр бота (для тестирования)
func SetBot(bot *TelegramBot) {
	botMutex.Lock()
	defer botMutex.Unlock()
	botInstance = bot
}

// ResetBot сбрасывает Singleton (для тестирования)
func ResetBot() {
	botMutex.Lock()
	defer botMutex.Unlock()
	botInstance = nil
	botOnce = sync.Once{}
}

// newTelegramBot создает новый экземпляр бота (внутренняя функция)
func newTelegramBot(cfg *config.Config, userService *users.Service) *TelegramBot {
	// Создаем компоненты
	messageSender := NewMessageSender(cfg)
	notifier := NewNotifier(cfg)
	notifier.SetMessageSender(messageSender)

	// Создаем менеджер меню
	menuManager := NewMenuManager(cfg, messageSender)

	bot := &TelegramBot{
		config:        cfg,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
		baseURL:       fmt.Sprintf("https://api.telegram.org/bot%s/", cfg.TelegramBotToken),
		chatID:        cfg.TelegramChatID,
		notifier:      notifier,
		menuManager:   menuManager,
		messageSender: messageSender,
		startupTime:   time.Now(),
		welcomeSent:   false,
		testMode:      false,
		userService:   userService,
	}

	// Инициализируем авторизацию если userService предоставлен
	if userService != nil {
		if err := bot.initAuth(); err != nil {
			log.Printf("⚠️ Ошибка инициализации авторизации: %v", err)
		}
	}

	// Устанавливаем главное меню
	if err := bot.menuManager.SetupMenu(); err != nil {
		log.Printf("⚠️ Failed to setup menu: %v", err)
	}

	return bot
}
