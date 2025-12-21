package notifier

import (
	"crypto-exchange-screener-bot/internal/adapters"
	"crypto-exchange-screener-bot/internal/config"
	"crypto-exchange-screener-bot/internal/telegram"
	"crypto-exchange-screener-bot/internal/types"
	"log"
)

// EnhancedTelegramNotifier - улучшенный нотификатор для работы с несколькими чатами
type EnhancedTelegramNotifier struct {
	multiChatBot *telegram.MultiChatBot
	enabled      bool
	stats        map[string]interface{}
}

// NewEnhancedTelegramNotifier создает улучшенный нотификатор
func NewEnhancedTelegramNotifier(cfg *config.Config) *EnhancedTelegramNotifier {
	multiChatBot := telegram.NewMultiChatBot(cfg)
	if multiChatBot == nil {
		return nil
	}

	return &EnhancedTelegramNotifier{
		multiChatBot: multiChatBot,
		enabled:      true,
		stats: map[string]interface{}{
			"sent_to_control":    0,
			"sent_to_monitoring": 0,
			"errors":             0,
			"type":               "enhanced_telegram",
		},
	}
}

// Send отправляет сигнал в соответствующие чаты
func (etn *EnhancedTelegramNotifier) Send(signal types.TrendSignal) error {
	if !etn.enabled || etn.multiChatBot == nil {
		return nil
	}

	// Конвертируем в GrowthSignal
	growthSignal := adapters.TrendSignalToGrowthSignal(signal)

	// Отправляем только в чаты мониторинга
	err := etn.multiChatBot.SendMonitoringNotification(growthSignal)
	if err != nil {
		etn.stats["errors"] = etn.stats["errors"].(int) + 1
		return err
	}

	// Увеличиваем счетчик
	if signal.Direction == "growth" {
		etn.stats["sent_to_monitoring"] = etn.stats["sent_to_monitoring"].(int) + 1
	} else {
		etn.stats["sent_to_monitoring"] = etn.stats["sent_to_monitoring"].(int) + 1
	}

	log.Printf("📊 Отправлено уведомление: %s %.2f%% в %d чатов",
		signal.Symbol, signal.ChangePercent,
		len(etn.multiChatBot.GetMonitoringStats()["monitoring_chats"].([]map[string]interface{})))

	return nil
}

// SendControlMessage отправляет сообщение только в контрольные чаты
func (etn *EnhancedTelegramNotifier) SendControlMessage(message string) error {
	return etn.multiChatBot.SendControlNotification(message)
}

// GetStats возвращает статистику
func (etn *EnhancedTelegramNotifier) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})
	for k, v := range etn.stats {
		stats[k] = v
	}

	// Добавляем статистику чатов
	chatStats := etn.multiChatBot.GetMonitoringStats()
	for k, v := range chatStats {
		stats[k] = v
	}

	return stats
}

// Name, IsEnabled, SetEnabled - как в обычном нотификаторе
func (etn *EnhancedTelegramNotifier) Name() string            { return "enhanced_telegram" }
func (etn *EnhancedTelegramNotifier) IsEnabled() bool         { return etn.enabled }
func (etn *EnhancedTelegramNotifier) SetEnabled(enabled bool) { etn.enabled = enabled }
