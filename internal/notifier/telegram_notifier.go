package notifier

import (
	"crypto-exchange-screener-bot/internal/adapters"
	"crypto-exchange-screener-bot/internal/config"
	"crypto-exchange-screener-bot/internal/telegram"
	"crypto-exchange-screener-bot/internal/types"
	"log"
	"time"
)

// TelegramNotifier нотификатор для Telegram
type TelegramNotifier struct {
	bot     *telegram.TelegramBot
	enabled bool
	stats   map[string]interface{}
}

// NewTelegramNotifier создает Telegram нотификатор с переданным ботом
func NewTelegramNotifier(cfg *config.Config, bot *telegram.TelegramBot) *TelegramNotifier {
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

	// ПРОВЕРЯЕМ ТЕСТОВЫЙ РЕЖИМ ПЕРЕД ОТПРАВКОЙ
	if t.bot.IsTestMode() {
		// В тестовом режиме логируем, но не отправляем
		log.Printf("🧪 Test mode - Skip Telegram notification for %s: %.2f%%",
			signal.Symbol, signal.ChangePercent)
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
