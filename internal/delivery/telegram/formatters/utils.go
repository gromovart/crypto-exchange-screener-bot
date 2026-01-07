// internal/delivery/telegram/formatters/utils.go
package formatters

import (
	"fmt"
	"math"
)

// NumberFormatter отвечает за форматирование чисел
type NumberFormatter struct{}

// NewNumberFormatter создает новый форматтер чисел
func NewNumberFormatter() *NumberFormatter {
	return &NumberFormatter{}
}

// FormatPrice форматирует цену с учетом ее величины
func (f *NumberFormatter) FormatPrice(price float64) string {
	if price <= 0 {
		return "0.00"
	}

	// Адаптивное форматирование в зависимости от величины цены
	switch {
	case price >= 1000:
		return fmt.Sprintf("%.0f", math.Round(price))
	case price >= 100:
		return fmt.Sprintf("%.1f", price)
	case price >= 10:
		return fmt.Sprintf("%.2f", price)
	case price >= 1:
		return fmt.Sprintf("%.3f", price)
	case price >= 0.1:
		return fmt.Sprintf("%.4f", price)
	case price >= 0.01:
		return fmt.Sprintf("%.5f", price)
	case price >= 0.001:
		return fmt.Sprintf("%.6f", price)
	case price >= 0.0001:
		return fmt.Sprintf("%.7f", price)
	default:
		return fmt.Sprintf("%.8f", price)
	}
}

// FormatDollarValue форматирует долларовые значения в читаемый вид (K/M/B)
func (f *NumberFormatter) FormatDollarValue(num float64) string {
	if num <= 0 {
		return "0"
	}

	// Форматируем в миллиарды (B)
	if num >= 1_000_000_000 {
		value := num / 1_000_000_000
		if value < 10 {
			return fmt.Sprintf("%.2fB", value)
		} else if value < 100 {
			return fmt.Sprintf("%.1fB", value)
		} else {
			return fmt.Sprintf("%.0fB", math.Round(value))
		}
	}

	// Форматируем в миллионы (M)
	if num >= 1_000_000 {
		value := num / 1_000_000
		if value < 10 {
			return fmt.Sprintf("%.2fM", value)
		} else if value < 100 {
			return fmt.Sprintf("%.1fM", value)
		} else {
			return fmt.Sprintf("%.0fM", math.Round(value))
		}
	}

	// Форматируем в тысячи (K)
	if num >= 1_000 {
		value := num / 1_000
		if value < 10 {
			return fmt.Sprintf("%.1fK", value)
		} else {
			return fmt.Sprintf("%.0fK", math.Round(value))
		}
	}

	// Меньше 1000 - округляем до целого
	if num >= 1 {
		return fmt.Sprintf("%.0f", math.Round(num))
	}

	// Меньше 1 - показываем с двумя знаками
	return fmt.Sprintf("%.2f", num)
}

// SafeDivide безопасное деление
func (f *NumberFormatter) SafeDivide(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}

// GetSourceIndicator возвращает индикатор источника данных
func GetSourceIndicator(source string) string {
	switch source {
	case "api":
		return " [API]"
	case "storage":
		return " [Хранилище]"
	case "emulated":
		return " [Эмуляция]"
	case "cache":
		return " [Кэш]"
	default:
		return ""
	}
}
