// pkg/utils/helpers.go
package utils

import (
	"fmt"
	"time"
)

// FormatDuration форматирует продолжительность в читаемый вид
func FormatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dч %dм", hours, minutes)
	}
	return fmt.Sprintf("%dм", minutes)
}

// FormatPrice форматирует цену с заданной точностью
func FormatPrice(price float64, precision int) string {
	format := fmt.Sprintf("%%.%df", precision)
	return fmt.Sprintf(format, price)
}

// FormatPercent форматирует процентное значение
func FormatPercent(value float64) string {
	if value > 0 {
		return fmt.Sprintf("+%.2f%%", value)
	}
	return fmt.Sprintf("%.2f%%", value)
}

// FormatSignalTime форматирует время для вывода в сигналах
func FormatSignalTime(t time.Time) string {
	return t.Format("2006/01/02 15:04:05")
}

// FormatRelativeTime форматирует время относительно текущего момента
func FormatRelativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Minute {
		return fmt.Sprintf("%d сек. назад", int(diff.Seconds()))
	} else if diff < time.Hour {
		return fmt.Sprintf("%d мин. назад", int(diff.Minutes()))
	} else if diff < 24*time.Hour {
		return fmt.Sprintf("%d ч. назад", int(diff.Hours()))
	}
	return t.Format("2006/01/02 15:04:05")
}
