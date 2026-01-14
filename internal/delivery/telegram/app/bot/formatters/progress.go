// internal/delivery/telegram/app/bot/formatters/progress.go
package formatters

import (
	"fmt"
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
	progressBar := f.formatCompactProgressBar(percentage)

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
