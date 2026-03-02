// internal/delivery/max/controller.go
package max

import (
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"math"
	"strings"
	"time"
)

// Controller EventSubscriber — отправляет сигналы в MAX мессенджер
type Controller struct {
	client *Client
	chatID int64
}

// NewController создаёт контроллер доставки в MAX
func NewController(client *Client, chatID int64) *Controller {
	return &Controller{
		client: client,
		chatID: chatID,
	}
}

// GetName возвращает имя контроллера
func (c *Controller) GetName() string {
	return "max_counter_controller"
}

// GetSubscribedEvents возвращает список подписанных событий
func (c *Controller) GetSubscribedEvents() []types.EventType {
	return []types.EventType{
		types.EventCounterSignalDetected,
	}
}

// HandleEvent обрабатывает событие сигнала
func (c *Controller) HandleEvent(event types.Event) error {
	// Пропускаем если chatID не настроен (широковещание отключено)
	if c.chatID == 0 {
		return nil
	}

	dataMap, ok := event.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("max controller: неверный формат данных события")
	}

	text := formatSignalText(dataMap)

	if err := c.client.SendMessage(c.chatID, text); err != nil {
		logger.Warn("⚠️ MAX: ошибка отправки сообщения: %v", err)
		return fmt.Errorf("max: send message: %w", err)
	}

	logger.Debug("✅ MAX: сигнал %s отправлен", getString(dataMap, "symbol"))
	return nil
}

// formatSignalText формирует текст сигнала для MAX
func formatSignalText(data map[string]interface{}) string {
	var b strings.Builder

	symbol := getString(data, "symbol")
	direction := getString(data, "direction")
	change := getFloat64(data, "change_percent")
	price := getFloat64(data, "current_price")
	period := getString(data, "period")
	volume24h := getFloat64(data, "volume_24h")
	oi := getFloat64(data, "open_interest")
	funding := getFloat64(data, "funding_rate")
	rsi := getFloat64(data, "rsi")
	volDelta := getFloat64(data, "volume_delta")
	volDeltaPct := getFloat64(data, "volume_delta_percent")
	liqTotal := getFloat64(data, "liquidation_volume")
	liqLong := getFloat64(data, "long_liq_volume")
	liqShort := getFloat64(data, "short_liq_volume")

	dirIcon := "🟢"
	dirText := "РОСТ"
	changePrefix := "+"
	if direction == "fall" {
		dirIcon = "🔴"
		dirText = "ПАДЕНИЕ"
		changePrefix = "-"
	}

	// Заголовок
	b.WriteString(fmt.Sprintf("%s %s %s%.2f%%\n", dirIcon, dirText, changePrefix, math.Abs(change)))
	b.WriteString(fmt.Sprintf("📛 %s\n", symbol))
	b.WriteString(fmt.Sprintf("🏷️  BYBIT • %s\n", period))
	b.WriteString(fmt.Sprintf("🕐 %s\n\n", time.Now().Format("15:04:05")))

	// Цена
	b.WriteString(fmt.Sprintf("💰 $%s\n", formatPrice(price)))

	// Объём и OI
	if volume24h > 0 {
		b.WriteString(fmt.Sprintf("📊 Объём 24ч: $%s\n", formatDollarValue(volume24h)))
	}
	if oi > 0 {
		b.WriteString(fmt.Sprintf("📈 OI: $%s\n", formatDollarValue(oi)))
	}

	// Дельта
	if volDelta != 0 {
		deltaSign := "+"
		if volDelta < 0 {
			deltaSign = ""
		}
		b.WriteString(fmt.Sprintf("📈 Дельта: %s%s (%.1f%%)\n", deltaSign, formatDollarValue(volDelta), volDeltaPct))
	}

	// Технический анализ
	if rsi > 0 {
		b.WriteString(fmt.Sprintf("\n📊 RSI: %.1f\n", rsi))
	}

	// Фандинг
	if funding != 0 {
		fundIcon := "🟢"
		if funding < 0 {
			fundIcon = "🔴"
		}
		b.WriteString(fmt.Sprintf("🎯 Фандинг: %s %.4f%%\n", fundIcon, funding*100))
	}

	// Ликвидации
	if liqTotal > 0 {
		b.WriteString(fmt.Sprintf("\n💥 Ликвидации: $%s\n", formatDollarValue(liqTotal)))
		if liqLong > 0 || liqShort > 0 {
			b.WriteString(fmt.Sprintf("   LONG: $%s | SHORT: $%s\n", formatDollarValue(liqLong), formatDollarValue(liqShort)))
		}
	}

	return strings.TrimSpace(b.String())
}

// --- вспомогательные ---

func getString(data map[string]interface{}, key string) string {
	if v, ok := data[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getFloat64(data map[string]interface{}, key string) float64 {
	if v, ok := data[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return 0
}

func formatPrice(price float64) string {
	if price >= 1000 {
		return fmt.Sprintf("%.2f", price)
	} else if price >= 1 {
		return fmt.Sprintf("%.4f", price)
	} else if price >= 0.001 {
		return fmt.Sprintf("%.6f", price)
	}
	return fmt.Sprintf("%.8f", price)
}

func formatDollarValue(v float64) string {
	abs := math.Abs(v)
	switch {
	case abs >= 1_000_000_000:
		return fmt.Sprintf("%.1fB", v/1_000_000_000)
	case abs >= 1_000_000:
		return fmt.Sprintf("%.1fM", v/1_000_000)
	case abs >= 1_000:
		return fmt.Sprintf("%.1fK", v/1_000)
	default:
		return fmt.Sprintf("%.0f", v)
	}
}
