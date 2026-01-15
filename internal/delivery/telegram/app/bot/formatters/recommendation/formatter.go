// internal/delivery/telegram/app/bot/formatters/recommendation/formatter.go
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

// FormatResult форматирует результат рекомендаций с торговыми уровнями
func (f *Formatter) FormatResult(
	primarySignal string,
	recommendations []string,
	strength string,
	tradingRecommendation string,
) string {
	if primarySignal == "" || len(recommendations) == 0 {
		return ""
	}

	var result strings.Builder

	// Заголовок блока
	result.WriteString("🎯 РЕКОМЕНДАЦИЯ:\n")

	// Направление тренда с переносом строки
	result.WriteString("📌 Направление:\n")
	result.WriteString(fmt.Sprintf("%s\n\n", primarySignal))

	// Заголовок анализа
	result.WriteString("📊 Анализ сигналов:\n")

	// Рекомендации
	// Форматируем строки с фиксированной шириной для номеров
	for i, rec := range recommendations {
		cleanText := f.getCleanTextWithoutIcons(rec)
		icon := f.getRecommendationIcon(rec)

		// Используем фиксированную ширину: 3 символа для номера
		numberStr := fmt.Sprintf("%2d.", i+1) // " 1.", "10." и т.д.

		// Форматируем строку с двумя табами
		if icon != "" && cleanText != "" {
			result.WriteString(fmt.Sprintf("%s\t%s %s\n", numberStr, icon, cleanText))
		} else if icon != "" {
			result.WriteString(fmt.Sprintf("%s\t%s\n", numberStr, icon))
		} else {
			result.WriteString(fmt.Sprintf("%s\t%s\n", numberStr, cleanText))
		}
	}

	result.WriteString("\n")

	// Добавляем торговую рекомендацию с уровнями (если есть)
	if tradingRecommendation != "" {
		result.WriteString(tradingRecommendation)
		result.WriteString("\n\n")
	}

	// Итоговая строка (заключение)
	result.WriteString(fmt.Sprintf("🎯 ЗАКЛЮЧЕНИЕ: %s движение с %s дельтой объемов",
		strength,
		f.getDeltaStrengthDescription(strength)))

	return strings.TrimSpace(result.String())
}

// FormatResultLegacy сохраняет старый формат для обратной совместимости
func (f *Formatter) FormatResultLegacy(
	primarySignal string,
	recommendations []string,
	strength string,
) string {
	if primarySignal == "" || len(recommendations) == 0 {
		return ""
	}

	var result strings.Builder

	// Направление тренда
	result.WriteString("📌 Направление:\n")
	result.WriteString(fmt.Sprintf("%s\n\n", primarySignal))

	// Заголовок анализа
	result.WriteString("📊 Анализ сигналов:\n")

	// Рекомендации
	for i, rec := range recommendations {
		cleanText := f.getCleanTextWithoutIcons(rec)
		icon := f.getRecommendationIcon(rec)

		// Форматируем строку с двумя табами
		if icon != "" && cleanText != "" {
			result.WriteString(fmt.Sprintf("%d.\t\t%s %s\n", i+1, icon, cleanText))
		} else if icon != "" {
			result.WriteString(fmt.Sprintf("%d.\t\t%s\n", i+1, icon))
		} else {
			result.WriteString(fmt.Sprintf("%d.\t\t%s\n", i+1, cleanText))
		}
	}

	// Итоговая строка (устаревшая)
	result.WriteString(fmt.Sprintf("\n🎯 Итог: %s движение с %s дельтой объемов",
		strength,
		f.getDeltaStrengthDescription(strength)))

	return strings.TrimSpace(result.String())
}

// getCleanTextWithoutIcons возвращает текст без иконок в начале
func (f *Formatter) getCleanTextWithoutIcons(rec string) string {
	cleanRec := strings.TrimSpace(rec)

	// Список всех возможных иконок
	allIcons := []string{"📊", "📈", "📉", "💥", "✅", "⚠️", "🔄", "🟡", "🎯"}

	// Удаляем иконки из начала строки
	for {
		changed := false
		for _, possibleIcon := range allIcons {
			// Проверяем, начинается ли строка с иконки (с пробелом или без)
			if strings.HasPrefix(cleanRec, possibleIcon+" ") {
				cleanRec = strings.TrimPrefix(cleanRec, possibleIcon+" ")
				changed = true
				break
			}
			if strings.HasPrefix(cleanRec, possibleIcon) {
				cleanRec = strings.TrimPrefix(cleanRec, possibleIcon)
				changed = true
				break
			}
		}
		if !changed {
			break
		}
	}

	return strings.TrimSpace(cleanRec)
}

// getRecommendationIcon возвращает иконку для рекомендации
func (f *Formatter) getRecommendationIcon(rec string) string {
	lowerRec := strings.ToLower(rec)

	// Сначала проверяем сложные случаи с приоритетом

	// 1. Дельта продаж ПРИ РОСТЕ или покупок при падении
	if strings.Contains(lowerRec, "дельта продаж при росте") ||
		strings.Contains(lowerRec, "дельта покупок при падении") {
		return "⚠️"
	}

	// 2. Осторожность с LONG/SHORT
	if strings.Contains(lowerRec, "осторожность с long") ||
		strings.Contains(lowerRec, "осторожность с short") {
		return "⚠️"
	}

	// 3. Возможна коррекция
	if strings.Contains(lowerRec, "возможна коррекция") {
		return "⚠️"
	}

	// 4. Возможен разворот
	if strings.Contains(lowerRec, "возможен разворот") {
		return "🔄"
	}

	// 5. Возможен отскок
	if strings.Contains(lowerRec, "возможен отскок") {
		return "💥"
	}

	// 6. Обычные случаи
	if strings.Contains(lowerRec, "дельта покупок") {
		return "📈"
	}

	if strings.Contains(lowerRec, "дельта продаж") {
		return "📉"
	}

	if strings.Contains(lowerRec, "rsi") {
		return "📊"
	}

	if strings.Contains(lowerRec, "macd") {
		if strings.Contains(lowerRec, "медвежий") || strings.Contains(lowerRec, "слабый медвежий") {
			return "📉"
		}
		return "📈" // бычий или нейтральный
	}

	if strings.Contains(lowerRec, "объемы подтверждают") {
		return "✅"
	}

	if strings.Contains(lowerRec, "объемы слабо подтверждают") {
		return "🟡"
	}

	if strings.Contains(lowerRec, "объемы противоречат") {
		return "⚠️"
	}

	if strings.Contains(lowerRec, "ликвидации") {
		return "💥"
	}

	if strings.Contains(lowerRec, "противоречие") {
		return "🔄"
	}

	if strings.Contains(lowerRec, "волатильность") {
		return "💥"
	}

	return ""
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
