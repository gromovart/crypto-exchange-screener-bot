// internal/notifier/telegram_notifier.go
package notifier

import (
	"crypto-exchange-screener-bot/internal/adapters"
	"crypto-exchange-screener-bot/internal/config"
	"crypto-exchange-screener-bot/internal/telegram"
	"crypto-exchange-screener-bot/internal/types"
	"time"
)

// TelegramNotifier нотификатор для Telegram
type TelegramNotifier struct {
	bot     *telegram.TelegramBot
	enabled bool
	stats   map[string]interface{}
}

// NewTelegramNotifier создает Telegram нотификатор
func NewTelegramNotifier(cfg *config.Config) *TelegramNotifier {
	bot := telegram.NewTelegramBot(cfg)
	if bot == nil {
		return nil
	}

	return &TelegramNotifier{
		bot:     bot,
		enabled: true,
		stats: map[string]interface{}{
			"sent":           0,
			"last_sent_time": time.Time{},
			"type":           "telegram",
		},
	}
}

// GetBot возвращает Telegram бота
func (t *TelegramNotifier) GetBot() *telegram.TelegramBot {
	return t.bot
}

// Send отправляет сигнал в Telegram
func (t *TelegramNotifier) Send(signal types.TrendSignal) error {
	if !t.enabled || t.bot == nil {
		return nil
	}

	// Конвертируем TrendSignal в GrowthSignal
	growthSignal := adapters.TrendSignalToGrowthSignal(signal)
	if err := t.bot.SendNotification(growthSignal); err != nil {
		return err
	}

	// Обновляем статистику
	t.stats["sent"] = t.stats["sent"].(int) + 1
	t.stats["last_sent_time"] = time.Now()

	return nil
}

// convertToGrowthSignal конвертирует TrendSignal в GrowthSignal
func convertToGrowthSignal(signal types.TrendSignal) types.GrowthSignal {
	growthPercent := 0.0
	fallPercent := 0.0
	direction := "growth"

	if signal.Direction == "growth" {
		growthPercent = signal.ChangePercent
	} else if signal.Direction == "fall" {
		fallPercent = -signal.ChangePercent // Делаем положительным для Telegram
		direction = "fall"
	} else {
		// Если направление нейтральное, используем изменение цены
		if signal.ChangePercent > 0 {
			growthPercent = signal.ChangePercent
			direction = "growth"
		} else {
			fallPercent = -signal.ChangePercent
			direction = "fall"
		}
	}

	return types.GrowthSignal{
		Symbol:        signal.Symbol,
		Direction:     direction,
		GrowthPercent: growthPercent,
		FallPercent:   fallPercent,
		PeriodMinutes: signal.PeriodMinutes,
		Confidence:    signal.Confidence,
		Timestamp:     signal.Timestamp,
		DataPoints:    signal.DataPoints,
	}
}

// Name возвращает имя
func (t *TelegramNotifier) Name() string {
	return "telegram"
}

// IsEnabled возвращает статус
func (t *TelegramNotifier) IsEnabled() bool {
	return t.enabled
}

// SetEnabled включает/выключает
func (t *TelegramNotifier) SetEnabled(enabled bool) {
	t.enabled = enabled
}

// GetStats возвращает статистику
func (t *TelegramNotifier) GetStats() map[string]interface{} {
	return t.stats
}
