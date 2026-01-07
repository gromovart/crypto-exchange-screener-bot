// internal/adapters/notification/telegram_notifier.go
package notification

import (
	"crypto-exchange-screener-bot/internal/delivery/telegram"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"crypto-exchange-screener-bot/internal/types"
	"log"
)

// TelegramNotifierV2 - нотификатор для Telegram (исправленная версия)
type TelegramNotifierV2 struct {
	mainBot       *telegram.TelegramBot // Основной чат
	systemMonitor *SystemMonitor        // Системный мониторинг
	enabled       bool
	stats         map[string]interface{}
}

// NewTelegramNotifierV2 создает новый нотификатор
func NewTelegramNotifierV2(cfg *config.Config) *TelegramNotifierV2 {
	if cfg == nil || !cfg.Telegram.Enabled || cfg.Telegram.ChatID == "" {
		log.Println("⚠️ TelegramNotifierV2: Telegram отключен или ChatID не указан")
		return nil
	}

	// Основной бот для торговых сигналов
	mainBot := telegram.NewTelegramBot(cfg)
	if mainBot == nil {
		log.Println("⚠️ TelegramNotifierV2: Не удалось создать основной бот")
		return nil
	}

	// Системный мониторинг (если настроен)
	var systemMonitor *SystemMonitor
	if cfg.Monitoring.Enabled && cfg.Monitoring.ChatID != "" {
		systemMonitor = NewSystemMonitor(cfg)
		if systemMonitor == nil {
			log.Println("⚠️ TelegramNotifierV2: Не удалось создать системный монитор")
		}
	}

	return &TelegramNotifierV2{
		mainBot:       mainBot,
		systemMonitor: systemMonitor,
		enabled:       true,
		stats: map[string]interface{}{
			"trading_signals_sent": 0,
			"system_messages_sent": 0,
			"errors":               0,
			"type":                 "telegram_v2",
		},
	}
}

// Send отправляет торговый сигнал ТОЛЬКО в основной чат
func (tn *TelegramNotifierV2) Send(signal types.TrendSignal) error {
	// 🔴 ОТКЛЮЧАЕМ отправку торговых сигналов через этот путь
	// Только CounterAnalyzer должен отправлять торговые сигналы через CounterNotifier

	if !tn.enabled || tn.mainBot == nil {
		return nil
	}

	log.Printf("⚠️ TelegramNotifierV2: Торговые сигналы ОТКЛЮЧЕНЫ. Используйте CounterAnalyzer для %s %.2f%%",
		signal.Symbol, signal.ChangePercent)

	// Возвращаем успех, но не отправляем сообщение
	return nil

	/*
		// СТАРЫЙ КОД (КОММЕНТИРУЕМ):
		// Конвертируем в GrowthSignal
		growthSignal := adapters.TrendSignalToGrowthSignal(signal)

		// Отправляем ТОЛЬКО в основной чат
		err := tn.mainBot.SendNotification(growthSignal)
		if err != nil {
			tn.stats["errors"] = tn.stats["errors"].(int) + 1
			log.Printf("❌ TelegramNotifierV2: Ошибка отправки торгового сигнала: %v", err)
			return err
		}

		tn.stats["trading_signals_sent"] = tn.stats["trading_signals_sent"].(int) + 1
		log.Printf("✅ TelegramNotifierV2: Торговый сигнал отправлен в основной чат: %s %.2f%%",
			signal.Symbol, signal.ChangePercent)

		return nil
	*/
}

// SendSystemStatus отправляет системный статус в мониторинг
func (tn *TelegramNotifierV2) SendSystemStatus(status string) error {
	if tn.systemMonitor == nil {
		return nil
	}

	err := tn.systemMonitor.SendSystemStatus(status)
	if err == nil {
		tn.stats["system_messages_sent"] = tn.stats["system_messages_sent"].(int) + 1
	}
	return err
}

// SendStartupMessage отправляет сообщение о запуске
func (tn *TelegramNotifierV2) SendStartupMessage(appName, version string) error {
	if tn.systemMonitor == nil {
		return nil
	}

	return tn.systemMonitor.SendStartupMessage(appName, version)
}

// SendControlMessage отправляет сообщение в основной чат
func (tn *TelegramNotifierV2) SendControlMessage(message string) error {
	if !tn.enabled || tn.mainBot == nil {
		return nil
	}

	return tn.mainBot.SendMessage(message)
}

// SendTestMessage отправляет тестовое сообщение
func (tn *TelegramNotifierV2) SendTestMessage() error {
	if !tn.enabled || tn.mainBot == nil {
		return nil
	}

	return tn.mainBot.SendTestMessage()
}

// GetSystemMonitor возвращает системный монитор
func (tn *TelegramNotifierV2) GetSystemMonitor() *SystemMonitor {
	return tn.systemMonitor
}

// Name возвращает имя
func (tn *TelegramNotifierV2) Name() string {
	return "telegram_v2"
}

// IsEnabled возвращает статус
func (tn *TelegramNotifierV2) IsEnabled() bool {
	return tn.enabled
}

// SetEnabled включает/выключает
func (tn *TelegramNotifierV2) SetEnabled(enabled bool) {
	tn.enabled = enabled
	if tn.systemMonitor != nil {
		tn.systemMonitor.SetEnabled(enabled)
	}
}

// GetStats возвращает статистику
func (tn *TelegramNotifierV2) GetStats() map[string]interface{} {
	statsCopy := make(map[string]interface{})
	for k, v := range tn.stats {
		statsCopy[k] = v
	}

	// Добавляем статистику системного монитора
	if tn.systemMonitor != nil {
		systemStats := tn.systemMonitor.GetStats()
		for k, v := range systemStats {
			statsCopy["system_"+k] = v
		}
	}

	return statsCopy
}
