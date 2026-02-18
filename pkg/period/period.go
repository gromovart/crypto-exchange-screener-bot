// pkg/period/period.go
package period

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// StringToMinutes конвертирует строковый период в минуты
func StringToMinutes(period string) (int, error) {
	period = strings.ToLower(strings.TrimSpace(period))

	switch period {
	case "1m":
		return Minutes1, nil
	case "5m":
		return Minutes5, nil
	case "15m":
		return Minutes15, nil
	case "30m":
		return Minutes30, nil
	case "1h":
		return Minutes60, nil
	case "4h":
		return Minutes240, nil
	case "1d":
		return Minutes1440, nil
	default:
		// Пробуем распарсить как число минут
		if strings.HasSuffix(period, "m") {
			minutesStr := strings.TrimSuffix(period, "m")
			minutes, err := strconv.Atoi(minutesStr)
			if err == nil && minutes > 0 {
				return minutes, nil
			}
		}
		return 0, fmt.Errorf("неизвестный формат периода: %s", period)
	}
}

// MinutesToString конвертирует минуты в строковый период
func MinutesToString(minutes int) string {
	switch minutes {
	case Minutes1:
		return Period1m
	case Minutes5:
		return Period5m
	case Minutes15:
		return Period15m
	case Minutes30:
		return Period30m
	case Minutes60:
		return Period1h
	case Minutes240:
		return Period4h
	case Minutes1440:
		return Period1d
	default:
		// Для пользовательских периодов
		return fmt.Sprintf("%dm", minutes)
	}
}

// ✅ НОВАЯ УНИВЕРСАЛЬНАЯ ФУНКЦИЯ
// PeriodToDuration конвертирует строковый период в time.Duration (без ошибки)
func PeriodToDuration(period string) time.Duration {
	switch period {
	case "1m":
		return 1 * time.Minute
	case "5m":
		return 5 * time.Minute
	case "15m":
		return 15 * time.Minute
	case "30m":
		return 30 * time.Minute
	case "1h":
		return 1 * time.Hour
	case "4h":
		return 4 * time.Hour
	case "1d":
		return 24 * time.Hour
	default:
		// Пробуем распарсить как число минут
		if minutes, err := StringToMinutes(period); err == nil {
			return MinutesToDuration(minutes)
		}
		// Если не удалось распарсить, возвращаем дефолт
		return 15 * time.Minute
	}
}

// StringToDuration конвертирует строковый период в time.Duration с проверкой ошибки
func StringToDuration(period string) (time.Duration, error) {
	if !IsValidPeriod(period) {
		return 0, fmt.Errorf("неизвестный период: %s", period)
	}
	return PeriodToDuration(period), nil
}

// MinutesToDuration конвертирует минуты в time.Duration
func MinutesToDuration(minutes int) time.Duration {
	return time.Duration(minutes) * time.Minute
}

// DurationToMinutes конвертирует time.Duration в минуты
func DurationToMinutes(duration time.Duration) int {
	return int(duration.Minutes())
}

// DurationToString конвертирует time.Duration в строковый период
func DurationToString(duration time.Duration) string {
	minutes := DurationToMinutes(duration)
	return MinutesToString(minutes)
}

// IsValidPeriod проверяет, является ли период валидным
func IsValidPeriod(period string) bool {
	_, err := StringToMinutes(period)
	return err == nil
}

// IsValidMinutes проверяет, являются ли минуты валидным периодом
func IsValidMinutes(minutes int) bool {
	// Проверяем, что минуты положительные и не превышают разумный предел
	return minutes > 0 && minutes <= Minutes1440*30 // до 30 дней
}

// IsStandardPeriod проверяет, является ли период стандартным
func IsStandardPeriod(period string) bool {
	for _, stdPeriod := range AllPeriods {
		if period == stdPeriod {
			return true
		}
	}
	return false
}

// IsStandardMinutes проверяет, являются ли минуты стандартным периодом
func IsStandardMinutes(minutes int) bool {
	for _, stdMinutes := range AllPeriodsMinutes {
		if minutes == stdMinutes {
			return true
		}
	}
	return false
}

// GetStandardPeriods возвращает все стандартные периоды
func GetStandardPeriods() []string {
	return AllPeriods
}

// GetStandardMinutes возвращает все стандартные периоды в минутах
func GetStandardMinutes() []int {
	return AllPeriodsMinutes
}

// ParseUserPeriods парсит периоды пользователя из разных форматов
func ParseUserPeriods(periods []interface{}) ([]int, error) {
	result := make([]int, 0, len(periods))

	for _, p := range periods {
		switch v := p.(type) {
		case int:
			if v > 0 {
				result = append(result, v)
			}
		case int64:
			if v > 0 {
				result = append(result, int(v))
			}
		case string:
			minutes, err := StringToMinutes(v)
			if err == nil && minutes > 0 {
				result = append(result, minutes)
			}
		case float64:
			if v > 0 {
				result = append(result, int(v))
			}
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("нет валидных периодов")
	}

	return result, nil
}

// FormatPeriodsForDisplay форматирует периоды для отображения
func FormatPeriodsForDisplay(minutes []int) []string {
	result := make([]string, len(minutes))
	for i, m := range minutes {
		result[i] = FormatPeriodForDisplay(m)
	}
	return result
}

// FormatPeriodForDisplay форматирует период для отображения
func FormatPeriodForDisplay(minutes int) string {
	switch minutes {
	case Minutes1:
		return "1 минута"
	case Minutes5:
		return "5 минут"
	case Minutes15:
		return "15 минут"
	case Minutes30:
		return "30 минут"
	case Minutes60:
		return "1 час"
	case Minutes240:
		return "4 часа"
	case Minutes1440:
		return "1 день"
	default:
		if minutes < 60 {
			return fmt.Sprintf("%d минут", minutes)
		} else if minutes < 1440 {
			hours := minutes / 60
			if hours == 1 {
				return "1 час"
			} else if hours >= 2 && hours <= 4 {
				return fmt.Sprintf("%d часа", hours)
			} else {
				return fmt.Sprintf("%d часов", hours)
			}
		} else {
			days := minutes / 1440
			if days == 1 {
				return "1 день"
			} else if days >= 2 && days <= 4 {
				return fmt.Sprintf("%d дня", days)
			} else {
				return fmt.Sprintf("%d дней", days)
			}
		}
	}
}

// GetMaxPeriod возвращает максимальный период из списка минут
func GetMaxPeriod(minutes []int) int {
	if len(minutes) == 0 {
		return Minutes5 // По умолчанию 5 минут
	}

	max := minutes[0]
	for _, m := range minutes[1:] {
		if m > max {
			max = m
		}
	}
	return max
}

// GetMinPeriod возвращает минимальный период из списка минут
func GetMinPeriod(minutes []int) int {
	if len(minutes) == 0 {
		return Minutes5 // По умолчанию 5 минут
	}

	min := minutes[0]
	for _, m := range minutes[1:] {
		if m < min {
			min = m
		}
	}
	return min
}

// ClampPeriod ограничивает период разумными пределами
func ClampPeriod(minutes int, minLimit, maxLimit int) int {
	if minutes < minLimit {
		return minLimit
	}
	if minutes > maxLimit {
		return maxLimit
	}
	return minutes
}

// ClampPeriodStandard ограничивает период стандартными пределами (1м - 1день)
func ClampPeriodStandard(minutes int) int {
	return ClampPeriod(minutes, Minutes1, Minutes1440)
}

// ConvertToStandardPeriod конвертирует произвольный период в ближайший стандартный
func ConvertToStandardPeriod(minutes int) int {
	standardPeriods := []int{Minutes1, Minutes5, Minutes15, Minutes30, Minutes60, Minutes240, Minutes1440}

	for _, std := range standardPeriods {
		if minutes <= std {
			return std
		}
	}
	return Minutes1440
}

var DefaultDuration = MinutesToDuration(DefaultMinutes)

// Дефолтные значения
const (
	DefaultPeriod    = Period15m
	DefaultPeriodStr = Period15m
	DefaultMinutes   = Minutes5
)
