// internal/utils/formatter.go
package utils

import (
	"fmt"
	"time"
)

// MarketDataFormatter форматирует рыночные данные для отображения
type MarketDataFormatter struct{}

// FormatPrice форматирует цену
func (f *MarketDataFormatter) FormatPrice(price float64) string {
	if price >= 1000 {
		return fmt.Sprintf("$%.2f", price)
	} else if price >= 1 {
		return fmt.Sprintf("$%.4f", price)
	} else {
		return fmt.Sprintf("$%.6f", price)
	}
}

// FormatVolume форматирует объем
func (f *MarketDataFormatter) FormatVolume(volume float64) string {
	if volume >= 1_000_000_000 {
		return fmt.Sprintf("%.2fB", volume/1_000_000_000)
	} else if volume >= 1_000_000 {
		return fmt.Sprintf("%.2fM", volume/1_000_000)
	} else if volume >= 1_000 {
		return fmt.Sprintf("%.1fK", volume/1_000)
	}
	return fmt.Sprintf("%.0f", volume)
}

// FormatOI форматирует открытый интерес
func (f *MarketDataFormatter) FormatOI(oi float64) string {
	return f.FormatVolume(oi) // Используем ту же логику
}

// FormatFunding форматирует ставку фандинга
func (f *MarketDataFormatter) FormatFunding(rate float64) (string, string) {
	ratePercent := rate * 100
	rateStr := fmt.Sprintf("%.4f%%", ratePercent)

	var emoji string
	if ratePercent > 0.015 {
		emoji = "🟢" // Сильно положительный
	} else if ratePercent > 0.005 {
		emoji = "🟡" // Слабо положительный
	} else if ratePercent > -0.005 {
		emoji = "⚪" // Нейтральный
	} else if ratePercent > -0.015 {
		emoji = "🟠" // Слабо отрицательный
	} else {
		emoji = "🔴" // Сильно отрицательный
	}

	return emoji, rateStr
}

// FormatChange форматирует изменение с эмодзи
func (f *MarketDataFormatter) FormatChange(change float64) (string, string) {
	changeStr := fmt.Sprintf("%.2f%%", change)

	var emoji string
	if change > 0 {
		emoji = "🟢"
		changeStr = "+" + changeStr
	} else if change < 0 {
		emoji = "🔴"
	} else {
		emoji = "⚪"
	}

	return emoji, changeStr
}

// FormatTimeLeft форматирует оставшееся время
func (f *MarketDataFormatter) FormatTimeLeft(t time.Time) string {
	duration := time.Until(t)
	if duration <= 0 {
		return "сейчас"
	}

	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dч %dм", hours, minutes)
	}
	return fmt.Sprintf("%dм", minutes)
}
