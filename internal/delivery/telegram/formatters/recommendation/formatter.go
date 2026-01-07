// internal/delivery/telegram/formatters/recommendation/formatter.go
package recommendation

import (
	"fmt"
	"strings"
)

// Formatter форматирует вывод рекомендаций
type Formatter struct{}

// NewFormatter создает новый форматтер вывода
func NewFormatter() *Formatter {
	return &Formatter{}
}

// FormatResult форматирует результат рекомендаций
func (f *Formatter) FormatResult(
	primarySignal string,
	recommendations []string,
	strength string,
) string {
	if primarySignal == "" || len(recommendations) == 0 {
		return ""
	}

	var result strings.Builder
	result.WriteString(primarySignal + "\n")

	for i, rec := range recommendations {
		icon := f.getRecommendationIcon(rec)
		cleanRec := rec

		if icon != "" && strings.HasPrefix(cleanRec, icon+" ") {
			cleanRec = strings.TrimPrefix(cleanRec, icon+" ")
		}

		result.WriteString(fmt.Sprintf("%d. %s%s\n", i+1,
			func() string {
				if icon != "" {
					return icon + " "
				}
				return ""
			}(),
			cleanRec))
	}

	// Добавляем итоговую оценку
	result.WriteString(fmt.Sprintf("\n🎯 ИТОГ: %s движение с %s дельтой объемов",
		strength,
		f.getDeltaStrengthDescription(strength)))

	return strings.TrimSpace(result.String())
}

// getRecommendationIcon возвращает иконку для рекомендации
func (f *Formatter) getRecommendationIcon(rec string) string {
	lowerRec := strings.ToLower(rec)

	switch {
	case strings.Contains(lowerRec, "дельта покупок"):
		return "📈"
	case strings.Contains(lowerRec, "дельта продаж"):
		return "📉"
	case strings.Contains(lowerRec, "long"):
		return "📈"
	case strings.Contains(lowerRec, "short"):
		return "📉"
	case strings.Contains(lowerRec, "рост"):
		return "📈"
	case strings.Contains(lowerRec, "падение"):
		return "📉"
	case strings.Contains(lowerRec, "бычий"):
		return "📈"
	case strings.Contains(lowerRec, "медвежий"):
		return "📉"
	case strings.Contains(lowerRec, "покуп"):
		return "📈"
	case strings.Contains(lowerRec, "продаж"):
		return "📉"
	case strings.Contains(lowerRec, "⚠️"):
		return "⚠️"
	case strings.Contains(lowerRec, "🔄"):
		return "🔄"
	case strings.Contains(lowerRec, "💥"):
		return "💥"
	case strings.Contains(lowerRec, "✅"):
		return "✅"
	case strings.Contains(lowerRec, "🟡"):
		return "🟡"
	case strings.Contains(lowerRec, "rsi"):
		return "📊"
	case strings.Contains(lowerRec, "macd"):
		return "📈"
	default:
		if len(rec) > 0 {
			firstRune := []rune(rec)[0]
			if (firstRune >= 0x1F600 && firstRune <= 0x1F64F) ||
				(firstRune >= 0x1F300 && firstRune <= 0x1F5FF) ||
				(firstRune >= 0x1F680 && firstRune <= 0x1F6FF) {
				return ""
			}
		}
		return "•"
	}
}

// getDeltaStrengthDescription возвращает описание силы дельты
func (f *Formatter) getDeltaStrengthDescription(strength string) string {
	switch strength {
	case "сильное":
		return "сильной"
	case "умеренное":
		return "умеренной"
	default:
		return "слабой"
	}
}

// GetSourceIndicator возвращает индикатор источника данных
func (f *Formatter) GetSourceIndicator(source string) string {
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
