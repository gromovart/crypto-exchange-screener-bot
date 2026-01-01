// internal/delivery/telegram/singleton.go
package telegram

import (
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
	if cfg == nil || cfg.TelegramBotToken == "" || cfg.TelegramChatID == "" {
		log.Println("⚠️ Telegram Bot Token или Chat ID не указаны, бот отключен")
		return nil
	}

	botOnce.Do(func() {
		log.Println("🤖 Создание Telegram бота (единственный экземпляр)...")
		botInstance = newTelegramBot(cfg)

		if botInstance != nil {
			log.Println("✅ Telegram бот создан (Singleton)")
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
func newTelegramBot(cfg *config.Config) *TelegramBot {
	// Создаем компоненты
	messageSender := NewMessageSender(cfg)
	notifier := NewNotifier(cfg)
	notifier.SetMessageSender(messageSender)

	bot := &TelegramBot{
		config:        cfg,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
		baseURL:       fmt.Sprintf("https://api.telegram.org/bot%s/", cfg.TelegramBotToken),
		chatID:        cfg.TelegramChatID,
		notifier:      notifier,
		menuManager:   NewMenuManager(cfg, messageSender),
		messageSender: messageSender,
		startupTime:   time.Now(),
		welcomeSent:   false,
		testMode:      false,
	}

	// Устанавливаем главное меню
	if err := bot.menuManager.SetupMenu(); err != nil {
		log.Printf("⚠️ Failed to setup menu: %v", err)
	}

	return bot
}
