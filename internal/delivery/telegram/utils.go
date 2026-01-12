// internal/delivery/telegram/utils.go
package telegram

import (
	"crypto-exchange-screener-bot/internal/infrastructure/config"
)

// getPeriodFromConfig получает период из конфигурации
func getPeriodFromConfig(config *config.Config) string {
	if config.AnalyzerConfigs.CounterAnalyzer.CustomSettings != nil {
		if period, ok := config.AnalyzerConfigs.CounterAnalyzer.CustomSettings["analysis_period"].(string); ok {
			return period
		}
	}
	return "15m"
}

// getNotificationStatus возвращает статус уведомлений
func getNotificationStatus(config *config.Config) string {
	if config.TelegramEnabled {
		return "✅ Включены"
	}
	return "❌ Выключены"
}

// getSignalTypeStatus возвращает статус типа сигнала
func getSignalTypeStatus(enabled bool, signalType string) string {
	if enabled {
		return "✅ Включен"
	}
	return "❌ Выключен"
}

// getSignalTypesStatus возвращает статус типов сигналов
func getSignalTypesStatus(config *config.Config) string {
	if config.TelegramNotifyGrowth && config.TelegramNotifyFall {
		return "Все"
	} else if config.TelegramNotifyGrowth {
		return "Только рост"
	} else if config.TelegramNotifyFall {
		return "Только падение"
	}
	return "Ничего"
}
