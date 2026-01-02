// internal/adapters/notification/system_monitor.go
package notification

import (
	"crypto-exchange-screener-bot/internal/delivery/telegram"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"fmt"
	"log"
	"sync"
	"time"
)

// SystemMonitor - мониторинг ТОЛЬКО системных событий
type SystemMonitor struct {
	telegramBot *telegram.TelegramBot
	enabled     bool
	mu          sync.RWMutex
	stats       map[string]interface{}
}

// NewSystemMonitor создает системный монитор
func NewSystemMonitor(cfg *config.Config) *SystemMonitor {
	if cfg == nil || cfg.Monitoring.ChatID == "" {
		log.Println("⚠️ SystemMonitor: Monitoring chat ID не указан")
		return nil
	}

	// Создаем конфигурацию для мониторинга
	monitorCfg := *cfg
	monitorCfg.TelegramChatID = cfg.Monitoring.ChatID
	monitorCfg.TelegramEnabled = true
	monitorCfg.TelegramNotifyGrowth = false // НЕ отправлять торговые сигналы
	monitorCfg.TelegramNotifyFall = false   // НЕ отправлять торговые сигналы
	monitorCfg.MonitoringEnabled = false    // Отключаем рекурсию

	bot := telegram.NewTelegramBotWithChatID(&monitorCfg, cfg.Monitoring.ChatID)
	if bot == nil {
		log.Println("⚠️ SystemMonitor: Не удалось создать бота мониторинга")
		return nil
	}

	return &SystemMonitor{
		telegramBot: bot,
		enabled:     true,
		stats: map[string]interface{}{
			"system_messages_sent": 0,
			"last_message_time":    time.Time{},
			"errors":               0,
			"type":                 "system_monitor",
		},
	}
}

// SendSystemStatus отправляет статус системы
func (sm *SystemMonitor) SendSystemStatus(status string) error {
	if !sm.enabled || sm.telegramBot == nil {
		return nil
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	message := fmt.Sprintf("🖥️ *Системный мониторинг*\n\n%s", status)

	err := sm.telegramBot.SendMessage(message)
	if err != nil {
		sm.stats["errors"] = sm.stats["errors"].(int) + 1
		log.Printf("❌ SystemMonitor: Ошибка отправки: %v", err)
		return err
	}

	sm.stats["system_messages_sent"] = sm.stats["system_messages_sent"].(int) + 1
	sm.stats["last_message_time"] = time.Now()
	return nil
}

// SendStartupMessage отправляет сообщение о запуске
func (sm *SystemMonitor) SendStartupMessage(appName, version string) error {
	if !sm.enabled || sm.telegramBot == nil {
		return nil
	}

	message := fmt.Sprintf(
		"🚀 *%s запущен*\n"+
			"Версия: %s\n"+
			"Время: %s\n\n"+
			"✅ Система мониторинга активна\n"+
			"📊 Мониторинг системных событий включен",
		appName, version, time.Now().Format("2006-01-02 15:04:05"),
	)

	return sm.SendSystemStatus(message)
}

// SendShutdownMessage отправляет сообщение об остановке
func (sm *SystemMonitor) SendShutdownMessage(appName string) error {
	if !sm.enabled || sm.telegramBot == nil {
		return nil
	}

	message := fmt.Sprintf(
		"🛑 *%s остановлен*\n"+
			"Время: %s\n"+
			"📈 Уведомлений отправлено: %d",
		appName, time.Now().Format("2006-01-02 15:04:05"),
		sm.stats["system_messages_sent"],
	)

	return sm.SendSystemStatus(message)
}

// SendError сообщает об ошибке
func (sm *SystemMonitor) SendError(errorType, details string) error {
	if !sm.enabled || sm.telegramBot == nil {
		return nil
	}

	message := fmt.Sprintf(
		"❌ *Ошибка системы*\n"+
			"Тип: %s\n"+
			"Детали: %s\n"+
			"Время: %s",
		errorType, details, time.Now().Format("15:04:05"),
	)

	return sm.SendSystemStatus(message)
}

// SendHealthCheck отправляет проверку работоспособности
func (sm *SystemMonitor) SendHealthCheck(components map[string]string) error {
	if !sm.enabled || sm.telegramBot == nil {
		return nil
	}

	message := "🩺 *Проверка работоспособности*\n\n"
	for name, status := range components {
		icon := "❌"
		if status == "ok" {
			icon = "✅"
		}
		message += fmt.Sprintf("%s %s: %s\n", icon, name, status)
	}
	message += fmt.Sprintf("\n🕐 %s", time.Now().Format("2006-01-02 15:04:05"))

	return sm.SendSystemStatus(message)
}

// SendStatistics отправляет статистику
func (sm *SystemMonitor) SendStatistics(stats map[string]interface{}) error {
	if !sm.enabled || sm.telegramBot == nil {
		return nil
	}

	message := "📊 *Статистика системы*\n\n"
	for key, value := range stats {
		message += fmt.Sprintf("• %s: %v\n", key, value)
	}
	message += fmt.Sprintf("\n🕐 %s", time.Now().Format("2006-01-02 15:04:05"))

	return sm.SendSystemStatus(message)
}

// GetStats возвращает статистику монитора
func (sm *SystemMonitor) GetStats() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	statsCopy := make(map[string]interface{})
	for k, v := range sm.stats {
		statsCopy[k] = v
	}
	return statsCopy
}

// IsEnabled возвращает статус
func (sm *SystemMonitor) IsEnabled() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.enabled
}

// SetEnabled включает/выключает
func (sm *SystemMonitor) SetEnabled(enabled bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.enabled = enabled
}
