// internal/delivery/telegram/multi_chat_bot.go
package telegram

import (
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"crypto-exchange-screener-bot/internal/types"
	"log"
	"os"
	"strings"
	"sync"
)

// MultiChatBot - бот с поддержкой нескольких чатов
type MultiChatBot struct {
	config *config.Config
	bot    *TelegramBot
	mu     sync.RWMutex

	// Списки чатов
	controlChatIDs  []string         // Чаты для управления
	monitoringChats []MonitoringChat // Чаты для мониторинга
}

// MonitoringChat - настройки чата мониторинга
type MonitoringChat struct {
	ChatID       string
	Name         string
	NotifyGrowth bool
	NotifyFall   bool
	Enabled      bool
}

// NewMultiChatBot создает бота с поддержкой нескольких чатов
func NewMultiChatBot(cfg *config.Config) *MultiChatBot {
	if cfg == nil || cfg.TelegramBotToken == "" {
		log.Println("⚠️ Telegram Bot Token не указан или конфиг nil")
		return nil
	}

	// Используем Singleton для основного бота
	mainBot := GetOrCreateBot(cfg)
	if mainBot == nil {
		log.Println("⚠️ Не удалось получить основной Telegram бот")
		return nil
	}

	bot := &MultiChatBot{
		config:          cfg,
		bot:             mainBot,
		controlChatIDs:  []string{cfg.TelegramChatID},
		monitoringChats: []MonitoringChat{},
	}

	// Инициализируем чат мониторинга из конфигурации
	if cfg.MonitoringChatID != "" && cfg.MonitoringEnabled {
		bot.monitoringChats = append(bot.monitoringChats, MonitoringChat{
			ChatID:       cfg.MonitoringChatID,
			Name:         "Monitoring Group",
			NotifyGrowth: cfg.MonitoringNotifyGrowth,
			NotifyFall:   cfg.MonitoringNotifyFall,
			Enabled:      true,
		})
		log.Printf("✅ Добавлен чат мониторинга: %s", cfg.MonitoringChatID)
	}

	return bot
}

// getEnv - вспомогательная функция для получения переменных окружения
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// SendMonitoringNotification отправляет уведомление только в чаты мониторинга
func (mcb *MultiChatBot) SendMonitoringNotification(signal types.GrowthSignal) error {
	if mcb.bot == nil || !mcb.config.TelegramEnabled {
		return nil
	}

	// Создаем копию бота для каждого чата мониторинга
	mcb.mu.RLock()
	monitoringChats := make([]MonitoringChat, len(mcb.monitoringChats))
	copy(monitoringChats, mcb.monitoringChats)
	mcb.mu.RUnlock()

	var lastError error
	sentCount := 0

	for _, chat := range monitoringChats {
		if !chat.Enabled {
			continue
		}

		// Проверяем тип сигнала
		if (signal.Direction == "growth" && !chat.NotifyGrowth) ||
			(signal.Direction == "fall" && !chat.NotifyFall) {
			continue
		}

		// Создаем копию бота с нужным chat_id
		customBot := mcb.createBotForChat(chat.ChatID)
		if customBot == nil {
			continue
		}

		// Отправляем уведомление
		if err := customBot.SendNotification(signal); err != nil {
			log.Printf("❌ Ошибка отправки в чат %s: %v", chat.Name, err)
			lastError = err
		} else {
			sentCount++
			log.Printf("📨 Отправлено в %s: %s %.2f%%",
				chat.Name, signal.Symbol, signal.GrowthPercent)
		}
	}

	if sentCount == 0 && lastError != nil {
		return lastError
	}

	return nil
}

// createBotForChat создает копию бота с другим chat_id
func (mcb *MultiChatBot) createBotForChat(chatID string) *TelegramBot {
	if mcb.bot == nil || chatID == "" {
		return nil
	}

	// Используем специальный конструктор для мониторинг-бота
	return NewTelegramBotWithChatID(mcb.config, chatID)
}

// NewTelegramBotForChat создает бота для конкретного чата
func NewTelegramBotForChat(cfg *config.Config, chatID string) *TelegramBot {
	if cfg.TelegramBotToken == "" || chatID == "" {
		log.Println("⚠️ Telegram Bot Token или Chat ID не указаны")
		return nil
	}

	// Создаем копию конфигурации с новым chat_id
	chatConfig := *cfg
	chatConfig.TelegramChatID = chatID

	// Создаем бота с модифицированной конфигурацией
	bot := NewTelegramBot(&chatConfig)
	if bot == nil {
		return nil
	}

	// Устанавливаем chat_id в messageSender
	if bot.messageSender != nil {
		bot.messageSender.SetChatID(chatID)
	}

	return bot
}

// SendControlNotification отправляет только в чаты управления
func (mcb *MultiChatBot) SendControlNotification(message string) error {
	if mcb.bot == nil || !mcb.config.TelegramEnabled {
		return nil
	}

	var lastError error
	for _, chatID := range mcb.controlChatIDs {
		customBot := mcb.createBotForChat(chatID)
		if customBot == nil {
			continue
		}

		if err := customBot.SendMessage(message); err != nil {
			log.Printf("❌ Ошибка отправки в контрольный чат %s: %v", chatID, err)
			lastError = err
		}
	}
	return lastError
}

// AddMonitoringChat добавляет новый чат для мониторинга
func (mcb *MultiChatBot) AddMonitoringChat(chatID, name string, notifyGrowth, notifyFall bool) {
	mcb.mu.Lock()
	defer mcb.mu.Unlock()

	mcb.monitoringChats = append(mcb.monitoringChats, MonitoringChat{
		ChatID:       chatID,
		Name:         name,
		NotifyGrowth: notifyGrowth,
		NotifyFall:   notifyFall,
		Enabled:      true,
	})

	log.Printf("✅ Добавлен новый чат мониторинга: %s (%s)", name, chatID)
}

// parseExtraChats парсит дополнительные чаты из строки
func (mcb *MultiChatBot) parseExtraChats(chatStr string) {
	// Формат: "chat1:name1:growth:fall,chat2:name2:growth"
	chats := strings.Split(chatStr, ",")

	for _, chat := range chats {
		parts := strings.Split(chat, ":")
		if len(parts) < 2 {
			continue
		}

		chatConfig := MonitoringChat{
			ChatID:       parts[0],
			Name:         parts[1],
			NotifyGrowth: true, // по умолчанию
			NotifyFall:   true, // по умолчанию
			Enabled:      true,
		}

		if len(parts) > 2 {
			chatConfig.NotifyGrowth = strings.Contains(strings.ToLower(parts[2]), "growth")
		}
		if len(parts) > 3 {
			chatConfig.NotifyFall = strings.Contains(strings.ToLower(parts[3]), "fall")
		}

		mcb.monitoringChats = append(mcb.monitoringChats, chatConfig)
		log.Printf("✅ Добавлен чат из конфига: %s", chatConfig.Name)
	}
}

// GetMonitoringStats возвращает статистику по чатам
func (mcb *MultiChatBot) GetMonitoringStats() map[string]interface{} {
	mcb.mu.RLock()
	defer mcb.mu.RUnlock()

	stats := map[string]interface{}{
		"total_monitoring_chats": len(mcb.monitoringChats),
		"total_control_chats":    len(mcb.controlChatIDs),
		"monitoring_chats":       []map[string]interface{}{},
	}

	for _, chat := range mcb.monitoringChats {
		stats["monitoring_chats"] = append(
			stats["monitoring_chats"].([]map[string]interface{}),
			map[string]interface{}{
				"name":          chat.Name,
				"chat_id":       maskChatID(chat.ChatID),
				"notify_growth": chat.NotifyGrowth,
				"notify_fall":   chat.NotifyFall,
				"enabled":       chat.Enabled,
			},
		)
	}

	return stats
}

// maskChatID маскирует chat ID для безопасности
func maskChatID(chatID string) string {
	if len(chatID) <= 4 {
		return "***"
	}
	return chatID[:2] + "***" + chatID[len(chatID)-2:]
}
