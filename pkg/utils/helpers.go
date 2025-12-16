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

// Формируем сообщение (используем fmt.Sprintf для форматирования строк)
// lines := []string{
// 	"══════════════════════════════════════════════════",
// 	fmt.Sprintf("⚫ %s - %s - %s", message.Exchange, intervalStr, message.Symbol),
// 	fmt.Sprintf("🕐 %s", timeStr), // Добавляем время сигнала
// 	fmt.Sprintf("%s %s: %s", icon, directionStr, changeStr),
// 	fmt.Sprintf("📡 Signal 24h: %d", message.Signal24h),
// 	fmt.Sprintf("🔗 %s", message.SymbolURL),
// 	"══════════════════════════════════════════════════",
// 	"", // Пустая строка для разделения
// 	}

func ParseIntervalToMinutes(interval string) (int, error) {
	switch interval {
	case "1":
		return 1, nil
	case "5":
		return 5, nil
	case "10":
		return 10, nil
	case "15":
		return 15, nil
	case "30":
		return 30, nil
	case "60":
		return 60, nil
	case "120":
		return 120, nil
	case "240":
		return 240, nil
	case "480":
		return 480, nil
	case "720":
		return 720, nil
	case "1440":
		return 1440, nil
	case "10080":
		return 10080, nil
	case "43200":
		return 43200, nil
	default:
		return 0, fmt.Errorf("неизвестный интервал: %s", interval)
	}
}
