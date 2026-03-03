// internal/delivery/max/controller.go
package max

import (
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/formatters/recommendation"
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"math"
	"strings"
	"time"
)

// recommFormatter — пакетный синглтон форматтера рекомендаций
// TODO: перенести пакет recommendation в internal/core или pkg, чтобы убрать
//
//	зависимость delivery/max → delivery/telegram
var recommFormatter = recommendation.NewRecommendationFormatter()

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

// formatSignalText формирует текст сигнала для MAX (аналог FormatCounterSignal для Telegram)
func formatSignalText(data map[string]interface{}) string {
	var b strings.Builder

	symbol := getString(data, "symbol")
	direction := getString(data, "direction")
	change := getFloat64(data, "change_percent")
	price := getFloat64(data, "current_price")
	period := getString(data, "period")
	volume24h := getFloat64(data, "volume_24h")
	oi := getFloat64(data, "open_interest")
	oiChange := getFloat64(data, "oi_change_24h")
	funding := getFloat64(data, "funding_rate")
	rsi := getFloat64(data, "rsi")
	rsiStatus := getString(data, "rsi_status")
	macdSignal := getFloat64(data, "macd_signal")
	macdStatus := getString(data, "macd_status")
	macdDescription := getString(data, "macd_description")
	volDelta := getFloat64(data, "volume_delta")
	volDeltaPct := getFloat64(data, "volume_delta_percent")
	liqTotal := getFloat64(data, "liquidation_volume")
	liqLong := getFloat64(data, "long_liq_volume")
	liqShort := getFloat64(data, "short_liq_volume")

	// S/R зоны
	srSupportPrice := getFloat64(data, "sr_support_price")
	srSupportStrength := getFloat64(data, "sr_support_strength")
	srSupportDistPct := getFloat64(data, "sr_support_dist_pct")
	srSupportHasWall := getBool(data, "sr_support_has_wall")
	srSupportWallUSD := getFloat64(data, "sr_support_wall_usd")
	srResistancePrice := getFloat64(data, "sr_resistance_price")
	srResistanceStrength := getFloat64(data, "sr_resistance_strength")
	srResistanceDistPct := getFloat64(data, "sr_resistance_dist_pct")
	srResistanceHasWall := getBool(data, "sr_resistance_has_wall")
	srResistanceWallUSD := getFloat64(data, "sr_resistance_wall_usd")

	dirIcon := "🟢"
	dirText := "РОСТ"
	changePrefix := "+"
	if direction == "fall" {
		dirIcon = "🔴"
		dirText = "ПАДЕНИЕ"
		changePrefix = "-"
	}

	// 1. Заголовок: направление и процент изменения
	b.WriteString(fmt.Sprintf("%s %s %s%.2f%%\n", dirIcon, dirText, changePrefix, math.Abs(change)))

	// 2. Символ
	b.WriteString(fmt.Sprintf("📛 %s\n\n", symbol))

	// 4. Биржа, период, время
	b.WriteString(fmt.Sprintf("🏷️  BYBIT • %s\n", period))
	b.WriteString(fmt.Sprintf("🕐 %s\n\n", time.Now().Format("15:04:05")))

	// 5. OI с процентным изменением
	if oi > 0 {
		b.WriteString(fmt.Sprintf("📈 OI: %s\n", maxFormatOI(oi, oiChange)))
	}

	// 6. Объём 24ч
	if volume24h > 0 {
		b.WriteString(fmt.Sprintf("📊 Объём 24ч: $%s\n", formatDollarValue(volume24h)))
	}

	// 7. Дельта с цветными эмодзи
	if volDelta != 0 || volDeltaPct != 0 {
		deltaIcon := maxDeltaIcon(volDelta)
		b.WriteString(fmt.Sprintf("📈 Дельта: %s%s (%.1f%%)\n\n",
			deltaIcon, formatDollarValue(math.Abs(volDelta)), volDeltaPct))
	}

	// 8. Технический анализ (RSI + MACD)
	if rsi > 0 || macdSignal != 0 {
		b.WriteString("📊 Тех. анализ:\n")
		if rsi > 0 {
			b.WriteString(maxFormatRSI(rsi, rsiStatus) + "\n")
		}
		if macdSignal != 0 {
			b.WriteString(maxFormatMACD(macdSignal, macdStatus, macdDescription) + "\n")
		}
		b.WriteString("\n")
	}

	// 9. Зоны поддержки/сопротивления
	hasSRSupport := srSupportPrice > 0
	hasSRResistance := srResistancePrice > 0
	if hasSRSupport || hasSRResistance {
		b.WriteString(fmt.Sprintf("📐 Зоны S/R (%s):\n", period))
		if hasSRSupport {
			line := fmt.Sprintf("🟢 Поддержка: $%s ↓%.1f%% | сила: %.0f%%",
				formatPrice(srSupportPrice), srSupportDistPct, srSupportStrength)
			if srSupportHasWall && srSupportWallUSD > 0 {
				line += fmt.Sprintf(" 🧱 $%s", formatDollarValue(srSupportWallUSD))
			}
			b.WriteString(line + "\n")
		}
		if hasSRResistance {
			line := fmt.Sprintf("🔴 Сопротивление: $%s ↑%.1f%% | сила: %.0f%%",
				formatPrice(srResistancePrice), srResistanceDistPct, srResistanceStrength)
			if srResistanceHasWall && srResistanceWallUSD > 0 {
				line += fmt.Sprintf(" 🧱 $%s", formatDollarValue(srResistanceWallUSD))
			}
			b.WriteString(line + "\n")
		}
		b.WriteString("\n")
	}

	// 10. Торговая рекомендация с уровнями
	if tradingRec := recommFormatter.GetTradingRecommendationOnly(
		direction, rsi, macdSignal, volDelta, volDeltaPct,
		liqLong, liqShort, price, change,
	); tradingRec != "" {
		b.WriteString(tradingRec + "\n\n")
	}

	// 11. Фандинг
	if funding != 0 {
		fundIcon := "🟢"
		if funding < 0 {
			fundIcon = "🔴"
		}
		b.WriteString(fmt.Sprintf("🎯 Фандинг: %s %.4f%%\n", fundIcon, funding*100))
	}

	// 12. Ликвидации
	if liqTotal > 0 {
		b.WriteString(fmt.Sprintf("\n💥 Ликвидации: $%s\n", formatDollarValue(liqTotal)))
		if liqLong > 0 || liqShort > 0 {
			b.WriteString(fmt.Sprintf("   LONG: $%s | SHORT: $%s\n",
				formatDollarValue(liqLong), formatDollarValue(liqShort)))
		}
	}

	return strings.TrimSpace(b.String())
}

// ──────────────────────────────────────────────
// Вспомогательные функции
// ──────────────────────────────────────────────

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

func getBool(data map[string]interface{}, key string) bool {
	if v, ok := data[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

// maxDeltaIcon возвращает цветной эмодзи по величине дельты объёмов
func maxDeltaIcon(delta float64) string {
	switch {
	case delta > 100_000:
		return "🟢🔼"
	case delta > 10_000:
		return "🟢"
	case delta > 1_000:
		return "🟡"
	case delta > 0:
		return "⚪"
	case delta < -100_000:
		return "🔴🔽"
	case delta < -10_000:
		return "🔴"
	case delta < -1_000:
		return "🟠"
	case delta < 0:
		return "⚪"
	default:
		return "⚪"
	}
}

// maxFormatOI форматирует открытый интерес с цветными эмодзи изменения.
// Формат аналогичен MetricsFormatter.FormatOIWithChange:
// [valueEmoji]$OI ([changeEmoji][sign]X.X%[strength])
func maxFormatOI(oi float64, change float64) string {
	oiStr := formatDollarValue(oi)
	if change == 0 {
		return fmt.Sprintf("$%s", oiStr)
	}

	absChange := math.Abs(change)

	var valueIcon string
	if change > 0 {
		valueIcon = "🟢"
	} else {
		valueIcon = "🔴"
	}

	changeIcon := "🟢"
	changeSign := "+"
	if change < 0 {
		changeIcon = "🔴"
		changeSign = "-"
	}

	var strength string
	switch {
	case absChange > 5:
		strength = " ⚡"
	case absChange > 2:
		strength = " ↗️"
	}

	return fmt.Sprintf("%s$%s (%s%s%.1f%%%s)", valueIcon, oiStr, changeIcon, changeSign, absChange, strength)
}

// maxFormatRSI форматирует RSI с эмодзи и текстовым статусом
func maxFormatRSI(rsi float64, status string) string {
	var emoji string
	if status == "" {
		// Fallback: вычисляем статус по значению
		switch {
		case rsi >= 70:
			emoji, status = "🔴", "сильная перекупленность"
		case rsi >= 62:
			emoji, status = "🟡", "перекупленность"
		case rsi >= 55:
			emoji, status = "🟢", "бычий настрой"
		case rsi >= 45:
			emoji, status = "⚪", "нейтральный"
		case rsi >= 38:
			emoji, status = "🟠", "медвежий настрой"
		default:
			emoji, status = "🔴", "сильная перепроданность"
		}
	} else {
		// Используем готовый статус из CounterAnalyzer
		switch status {
		case "сильная перекупленность", "перекупленность":
			emoji = "🔴"
		case "бычий настрой":
			emoji = "🟢"
		case "медвежий настрой":
			emoji = "🟠"
		case "сильная перепроданность":
			emoji = "🔴"
		default: // "нейтральный", "недостаточно данных"
			emoji = "⚪"
		}
	}
	return fmt.Sprintf("RSI: %.1f %s (%s)", rsi, emoji, status)
}

// maxFormatMACD форматирует MACD с описанием
func maxFormatMACD(signal float64, status, description string) string {
	if description != "" {
		return fmt.Sprintf("MACD: %s", description)
	}
	if status != "" {
		return fmt.Sprintf("MACD: %s", status)
	}
	// Fallback: вычисляем по значению сигнала
	var emoji, desc string
	switch {
	case signal > 0.1:
		emoji, desc = "🟢", "сильный бычий"
	case signal > 0.01:
		emoji, desc = "🟡", "бычий"
	case signal > -0.01:
		emoji, desc = "⚪", "нейтральный"
	case signal > -0.1:
		emoji, desc = "🟠", "медвежий"
	default:
		emoji, desc = "🔴", "сильный медвежий"
	}
	return fmt.Sprintf("MACD: %s %s", emoji, desc)
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
