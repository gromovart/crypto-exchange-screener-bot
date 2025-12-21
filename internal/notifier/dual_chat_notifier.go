package notifier

import (
	"crypto-exchange-screener-bot/internal/adapters"
	"crypto-exchange-screener-bot/internal/config"
	"crypto-exchange-screener-bot/internal/telegram"
	"crypto-exchange-screener-bot/internal/types"
	"log"
)

// DualChatNotifier отправляет уведомления в два чата
type DualChatNotifier struct {
	mainBot       *telegram.TelegramBot
	monitoringBot *telegram.TelegramBot
	enabled       bool
	stats         map[string]interface{}
}

// NewDualChatNotifier создает нотификатор для двух чатов
func NewDualChatNotifier(cfg *config.Config) *DualChatNotifier {
	// Основной бот (управление)
	mainBot := telegram.NewTelegramBot(cfg)
	if mainBot == nil {
		return nil
	}

	// Бот для мониторинга
	monitoringConfig := *cfg
	monitoringConfig.TelegramChatID = getEnv("MONITORING_CHAT_ID", "")
	if monitoringConfig.TelegramChatID == "" {
		// Используем дефолтный мониторинг чат
		monitoringConfig.TelegramChatID = "-1001234567890" // Замените на ваш
	}

	monitoringBot := telegram.NewTelegramBot(&monitoringConfig)
	if monitoringBot == nil {
		log.Println("⚠️ Не удалось создать бот для мониторинга")
		return &DualChatNotifier{
			mainBot: mainBot,
			enabled: true,
			stats:   make(map[string]interface{}),
		}
	}

	return &DualChatNotifier{
		mainBot:       mainBot,
		monitoringBot: monitoringBot,
		enabled:       true,
		stats: map[string]interface{}{
			"sent_to_main":       0,
			"sent_to_monitoring": 0,
			"errors":             0,
			"type":               "dual_chat",
		},
	}
}

// Send отправляет сигнал в оба чата
func (dcn *DualChatNotifier) Send(signal types.TrendSignal) error {
	if !dcn.enabled {
		return nil
	}

	// Конвертируем в GrowthSignal
	growthSignal := adapters.TrendSignalToGrowthSignal(signal)

	var lastError error
	sentCount := 0

	// Отправляем в мониторинг-чат
	if dcn.monitoringBot != nil {
		if err := dcn.monitoringBot.SendNotification(growthSignal); err != nil {
			log.Printf("❌ Ошибка отправки в мониторинг: %v", err)
			lastError = err
		} else {
			sentCount++
			dcn.stats["sent_to_monitoring"] = dcn.stats["sent_to_monitoring"].(int) + 1
		}
	}

	// Отправляем в основной чат (опционально, можно закомментировать)
	// if dcn.mainBot != nil {
	//     if err := dcn.mainBot.SendNotification(growthSignal); err != nil {
	//         log.Printf("❌ Ошибка отправки в основной чат: %v", err)
	//         lastError = err
	//     } else {
	//         sentCount++
	//         dcn.stats["sent_to_main"] = dcn.stats["sent_to_main"].(int) + 1
	//     }
	// }

	if sentCount == 0 && lastError != nil {
		dcn.stats["errors"] = dcn.stats["errors"].(int) + 1
		return lastError
	}

	return nil
}

// SendControlMessage отправляет сообщение только в основной чат
func (dcn *DualChatNotifier) SendControlMessage(message string) error {
	if !dcn.enabled || dcn.mainBot == nil {
		return nil
	}
	return dcn.mainBot.SendMessage(message)
}

// Name возвращает имя
func (dcn *DualChatNotifier) Name() string {
	return "dual_chat"
}

// IsEnabled возвращает статус
func (dcn *DualChatNotifier) IsEnabled() bool {
	return dcn.enabled
}

// SetEnabled включает/выключает
func (dcn *DualChatNotifier) SetEnabled(enabled bool) {
	dcn.enabled = enabled
}

// GetStats возвращает статистику
func (dcn *DualChatNotifier) GetStats() map[string]interface{} {
	return dcn.stats
}

// getEnv - вспомогательная функция
func getEnv(key, defaultValue string) string {
	// Ваша реализация getEnv
	return defaultValue
}
