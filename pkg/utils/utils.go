// pkg/utils/utils.go
package utils

import (
	"fmt"
	"strings"
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

// ParsePeriodToMinutes преобразует строку периода в минуты
func ParsePeriodToMinutes(period string) int {
	period = strings.ToLower(period)

	switch period {
	case "5m", "5 минут", "5 мин":
		return 5
	case "15m", "15 минут", "15 мин":
		return 15
	case "30m", "30 минут", "30 мин":
		return 30
	case "1h", "1 час":
		return 60
	case "4h", "4 часа":
		return 240
	case "1d", "1 день":
		return 1440
	default:
		return 15 // по умолчанию
	}
}

// PeriodToName возвращает человекочитаемое название периода
func PeriodToName(period string) string {
	period = strings.ToLower(period)

	switch period {
	case "5m", "5 минут", "5 мин":
		return "5 минут"
	case "15m", "15 минут", "15 мин":
		return "15 минут"
	case "30m", "30 минут", "30 мин":
		return "30 минут"
	case "1h", "1 час":
		return "1 час"
	case "4h", "4 часа":
		return "4 часа"
	case "1d", "1 день":
		return "1 день"
	default:
		return "15 минут"
	}
}

// IsValidPeriod проверяет валидность периода
func IsValidPeriod(period string) bool {
	validPeriods := map[string]bool{
		"5m":  true,
		"15m": true,
		"30m": true,
		"1h":  true,
		"4h":  true,
		"1d":  true,
	}
	_, exists := validPeriods[strings.ToLower(period)]
	return exists
}
