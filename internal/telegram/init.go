package telegram

import (
	"crypto-exchange-screener-bot/internal/config"
	"log"
)

// InitBot инициализирует Telegram бота один раз
// Используется в тестовых сценариях, где DataManager не используется
func InitBot(cfg *config.Config) *TelegramBot {
	log.Println("🤖 Инициализация Telegram бота...")

	if cfg.TelegramBotToken == "" || cfg.TelegramChatID == "" {
		log.Println("⚠️ Telegram Bot Token или Chat ID не указаны, бот отключен")
		return nil
	}

	return NewTelegramBot(cfg)
}

// SetTestModeForBot устанавливает тестовый режим для бота
// Это глобальная функция для удобства
func SetTestModeForBot(bot *TelegramBot, enabled bool) {
	if bot != nil {
		bot.SetTestMode(enabled)
		if enabled {
			log.Println("🧪 Telegram бот переключен в тестовый режим")
		} else {
			log.Println("🚀 Telegram бот переключен в рабочий режим")
		}
	}
}
