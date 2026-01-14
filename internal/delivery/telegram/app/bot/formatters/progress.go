// internal/delivery/telegram/app/bot/formatters/progress.go
package formatters

import (
	"fmt"
	"time"
)

// ProgressFormatter отвечает за форматирование прогресс-баров
type ProgressFormatter struct{}

// NewProgressFormatter создает новый форматтер прогресса
func NewProgressFormatter() *ProgressFormatter {
	return &ProgressFormatter{}
}

// FormatProgressBlock форматирует блок прогресса сигналов
func (f *ProgressFormatter) FormatProgressBlock(
	signalCount int,
	maxSignals int,
	period string,
) string {
	percentage := float64(signalCount) / float64(maxSignals) * 100
	progressBar := f.formatConfirmationProgressBar(percentage)

	return fmt.Sprintf("📡 %d/%d %s (%.0f%%)\n🕐 Период: %s\n\n",
		signalCount, maxSignals, progressBar, percentage, period)
}

// formatCompactProgressBar создает компактный прогресс-бар для счетчика сигналов
func (f *ProgressFormatter) formatCompactProgressBar(percentage float64) string {
	// Рассчитываем количество заполненных баров
	bars := int(percentage / 20) // 5 баров по 20% каждый
	if bars > 5 {
		bars = 5
	}
	if bars < 0 {
		bars = 0
	}

	// Строим прогресс-бар с цветами в зависимости от заполнения
	var result string
	for i := 0; i < 5; i++ {
		if i < bars {
			// Цвет баров меняется в зависимости от уровня заполнения
			switch {
			case percentage >= 80:
				result += "🔴" // Высокое заполнение - красный
			case percentage >= 50:
				result += "🟡" // Среднее заполнение - желтый
			default:
				result += "🟢" // Низкое заполнение - зеленый
			}
		} else {
			result += "▫️"
		}
	}
	return result
}

func (f *ProgressFormatter) FormatConfirmationProgress(
	confirmations int,
	required int,
	period string,
	nextAnalysis, nextSignal time.Time,
) string {
	percentage := float64(confirmations) / float64(required) * 100
	progressBar := f.formatCompactProgressBar(percentage)

	timeUntilNextAnalysis := formatTimeUntil(nextAnalysis)
	timeUntilNextSignal := formatTimeUntil(nextSignal)

	return fmt.Sprintf("📡 Подтверждений: %d/%d %s (%.0f%%)\n🕐 Следующий анализ: %s\n⏰ Следующий сигнал: %s",
		confirmations, required, progressBar, percentage,
		timeUntilNextAnalysis, timeUntilNextSignal)
}

// formatTimeUntil форматирует время до события
func formatTimeUntil(t time.Time) string {
	if t.IsZero() {
		return "─"
	}

	now := time.Now()
	if t.Before(now) {
		return "сейчас"
	}

	duration := t.Sub(now)

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

// formatConfirmationProgressBar создает прогресс-бар для подтверждений
// ОБРАТНАЯ логика: больше подтверждений = лучше = зеленый
func (f *ProgressFormatter) formatConfirmationProgressBar(percentage float64) string {
	bars := int(percentage / 20) // 5 баров по 20% каждый
	if bars > 5 {
		bars = 5
	}
	if bars < 0 {
		bars = 0
	}

	var result string
	for i := 0; i < 5; i++ {
		if i < bars {
			// ОБРАТНАЯ логика: 100% = зеленый, 0% = красный
			switch {
			case percentage >= 80:
				result += "🟢" // Много подтверждений - зеленый
			case percentage >= 50:
				result += "🟡" // Среднее количество - желтый
			default:
				result += "🔴" // Мало подтверждений - красный
			}
		} else {
			result += "▫️"
		}
	}
	return result
}
