// internal/notifier/console_notifier.go
package notifier

import (
	"crypto_exchange_screener_bot/internal/types/analysis"
	"fmt"
	"log"
	"time"
)

// ConsoleNotifier нотификатор для консоли
type ConsoleNotifier struct {
	enabled bool
	compact bool
	stats   map[string]interface{}
}

// NewConsoleNotifier создает консольный нотификатор
func NewConsoleNotifier(compact bool) *ConsoleNotifier {
	return &ConsoleNotifier{
		enabled: true,
		compact: compact,
		stats: map[string]interface{}{
			"sent":           0,
			"last_sent_time": time.Time{},
			"type":           "console",
		},
	}
}

// Send отправляет сигнал в консоль
func (c *ConsoleNotifier) Send(signal analysis.TrendSignal) error {
	if !c.enabled {
		return nil
	}

	var icon, direction string
	if signal.Direction == "growth" {
		icon = "🟢"
		direction = "РОСТ"
	} else {
		icon = "🔴"
		direction = "ПАДЕНИЕ"
	}

	if c.compact {
		log.Printf("%s %s: %s %.2f%% за %d минут (уверенность: %.0f%%)",
			icon, direction, signal.Symbol, signal.ChangePercent,
			signal.PeriodMinutes, signal.Confidence)
	} else {
		fmt.Println("══════════════════════════════════════════════════")
		fmt.Printf("%s %s: %s %.2f%% за %d минут\n",
			icon, direction, signal.Symbol, signal.ChangePercent,
			signal.PeriodMinutes)
		fmt.Printf("   Уверенность: %.0f%% | Точки данных: %d\n",
			signal.Confidence, signal.DataPoints)
		fmt.Printf("🔗 https://www.bybit.com/trade/usdt/%s\n", signal.Symbol)
		fmt.Println("══════════════════════════════════════════════════")
	}

	// Обновляем статистику
	c.stats["sent"] = c.stats["sent"].(int) + 1
	c.stats["last_sent_time"] = time.Now()

	return nil
}

// Name возвращает имя
func (c *ConsoleNotifier) Name() string {
	return "console"
}

// IsEnabled возвращает статус
func (c *ConsoleNotifier) IsEnabled() bool {
	return c.enabled
}

// SetEnabled включает/выключает
func (c *ConsoleNotifier) SetEnabled(enabled bool) {
	c.enabled = enabled
}

// GetStats возвращает статистику
func (c *ConsoleNotifier) GetStats() map[string]interface{} {
	return c.stats
}
