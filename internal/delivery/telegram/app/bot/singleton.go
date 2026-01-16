package bot

import (
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"crypto-exchange-screener-bot/pkg/logger"
	"log"
	"sync"
)

var (
	// globalBot - синглтон экземпляр Telegram бота для обратной совместимости
	globalBot     *TelegramBot
	globalBotOnce sync.Once
	globalBotMu   sync.RWMutex
)

// GetOrCreateBot создает или возвращает синглтон Telegram бота
// для обратной совместимости со старым кодом
func GetOrCreateBot(cfg *config.Config) *TelegramBot {
	return GetOrCreateBotWithDeps(cfg, nil)
}

// GetOrCreateBotWithDeps создает или возвращает синглтон Telegram бота с зависимостями
func GetOrCreateBotWithDeps(cfg *config.Config, deps *Dependencies) *TelegramBot {
	globalBotOnce.Do(func() {
		if cfg == nil || cfg.TelegramBotToken == "" {
			log.Println("⚠️ Telegram Bot Token не указан")
			return
		}

		// Создаем бота с зависимостями или без них
		if deps != nil {
			globalBot = NewTelegramBot(cfg, deps)
			logger.Info("✅ Telegram бот создан с зависимостями (Singleton)")
		} else {
			// Создаем минимальные зависимости для совместимости
			globalBot = NewTelegramBot(cfg, &Dependencies{})
			logger.Info("✅ Telegram бот создан без зависимостей (Singleton)")
		}
	})

	return globalBot
}

// GetBot возвращает существующий экземпляр бота (без создания нового)
func GetBot() *TelegramBot {
	globalBotMu.RLock()
	defer globalBotMu.RUnlock()
	return globalBot
}

// SetBot устанавливает бота вручную (для тестов)
func SetBot(bot *TelegramBot) {
	globalBotMu.Lock()
	defer globalBotMu.Unlock()
	globalBot = bot
}

// ResetBot сбрасывает синглтон бота (для тестов)
func ResetBot() {
	globalBotMu.Lock()
	defer globalBotMu.Unlock()
	globalBot = nil
	globalBotOnce = sync.Once{}
}
