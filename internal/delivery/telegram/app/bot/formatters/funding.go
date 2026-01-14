// internal/delivery/telegram/app/bot/formatters/funding.go
package formatters

import (
	"fmt"
	"time"
)

// FundingFormatter отвечает за форматирование фандинга
type FundingFormatter struct{}

// NewFundingFormatter создает новый форматтер фандинга
func NewFundingFormatter() *FundingFormatter {
	return &FundingFormatter{}
}

// FormatFundingBlock форматирует блок фандинга
func (f *FundingFormatter) FormatFundingBlock(
	fundingRate float64,
	nextFundingTime time.Time,
) string {
	fundingStr := f.formatFundingWithEmoji(fundingRate)
	timeUntil := f.formatCompactTime(nextFundingTime)

	return fmt.Sprintf("🎯 Фандинг: %s\n⏰ Через: %s", fundingStr, timeUntil)
}

// formatFundingWithEmoji форматирует ставку фандинга с эмодзи
func (f *FundingFormatter) formatFundingWithEmoji(rate float64) string {
	ratePercent := rate * 100

	// Выбираем эмодзи в зависимости от величины ставки фандинга
	var icon string
	switch {
	case ratePercent > 0.015:
		icon = "🟢" // Сильно положительный
	case ratePercent > 0.005:
		icon = "🟡" // Слабо положительный
	case ratePercent > -0.005:
		icon = "⚪" // Нейтральный
	case ratePercent > -0.015:
		icon = "🟠" // Слабо отрицательный
	default:
		icon = "🔴" // Сильно отрицательный
	}

	return fmt.Sprintf("%s %.4f%%", icon, ratePercent)
}

// formatCompactTime форматирует время в компактном читаемом виде
func (f *FundingFormatter) formatCompactTime(nextFundingTime time.Time) string {
	// Если время не задано
	if nextFundingTime.IsZero() {
		return "─"
	}

	now := time.Now()

	// Если время уже прошло
	if nextFundingTime.Before(now) {
		return "сейчас"
	}

	duration := nextFundingTime.Sub(now)

	// Форматируем в зависимости от длительности
	switch {
	case duration.Hours() >= 1:
		hours := int(duration.Hours())
		minutes := int(duration.Minutes()) % 60
		if minutes > 0 {
			return fmt.Sprintf("%dч %dм", hours, minutes)
		}
		return fmt.Sprintf("%dч", hours)
	default:
		minutes := int(duration.Minutes())
		if minutes <= 0 {
			return "скоро!"
		}
		return fmt.Sprintf("%dм", minutes)
	}
}
