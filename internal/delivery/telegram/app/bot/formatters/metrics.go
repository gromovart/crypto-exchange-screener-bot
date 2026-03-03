// internal/delivery/telegram/app/bot/formatters/metrics.go
package formatters

import (
	"fmt"
	"math"
)

// MetricsFormatter отвечает за форматирование рыночных метрик
type MetricsFormatter struct {
	numberFormatter *NumberFormatter
}

// NewMetricsFormatter создает новый форматтер метрик
func NewMetricsFormatter() *MetricsFormatter {
	return &MetricsFormatter{
		numberFormatter: NewNumberFormatter(),
	}
}

// FormatOIWithChange форматирует открытый интерес с процентным изменением.
// Формат аналогичен FormatVolumeDelta: [valueEmoji]$OI ([changeEmoji][sign]X.X%[strength])
// valueEmoji — направление изменения OI: 🟢 рост, 🔴 падение
// strength — ⚡ сильное (>5%), ↗️ умеренное (>2%), пусто для слабого
func (f *MetricsFormatter) FormatOIWithChange(oi float64, change float64) string {
	if oi <= 0 {
		return "─"
	}

	oiStr := f.numberFormatter.FormatDollarValue(oi)

	if change == 0 {
		return fmt.Sprintf("$%s", oiStr)
	}

	absChange := math.Abs(change)

	// Эмодзи перед значением OI — отражает направление изменения
	var valueIcon string
	if change > 0 {
		valueIcon = "🟢"
	} else {
		valueIcon = "🔴"
	}

	// Эмодзи и знак для процентного изменения
	changeIcon := "🟢"
	changeSign := "+"
	if change < 0 {
		changeIcon = "🔴"
		changeSign = "-"
	}

	// Индикатор силы изменения (как у дельты)
	var strength string
	switch {
	case absChange > 5:
		strength = " ⚡"
	case absChange > 2:
		strength = " ↗️"
	}

	return fmt.Sprintf("%s$%s (%s%s%.1f%%%s)", valueIcon, oiStr, changeIcon, changeSign, absChange, strength)
}

// FormatVolumeDelta форматирует дельту объемов с процентом изменения
func (f *MetricsFormatter) FormatVolumeDelta(delta float64, deltaPercent float64, direction string) string {
	// Если данных нет - возвращаем прочерк
	if delta == 0 && deltaPercent == 0 {
		return "─"
	}

	// Определяем знак и цвет дельты
	var deltaIcon string
	deltaFormatted := math.Abs(delta)

	// Определяем дельту
	switch {
	case delta > 100000: // Значительная положительная дельта (>100K)
		deltaIcon = "🟢🔼" // Сильные покупки
	case delta > 10000: // Умеренная положительная дельта (>10K)
		deltaIcon = "🟢" // Покупки преобладают
	case delta > 1000: // Небольшая положительная дельта (>1K)
		deltaIcon = "🟡" // Слабые покупки
	case delta > 0: // Положительная но маленькая
		deltaIcon = "⚪" // Нейтрально
	case delta < -100000: // Значительная отрицательная дельта (<-100K)
		deltaIcon = "🔴🔽" // Сильные продажи
	case delta < -10000: // Умеренная отрицательная дельта (<-10K)
		deltaIcon = "🔴" // Продажи преобладают
	case delta < -1000: // Небольшая отрицательная дельта (<-1K)
		deltaIcon = "🟠" // Слабые продажи
	case delta < 0: // Отрицательная но маленькая
		deltaIcon = "⚪" // Нейтрально
	default:
		deltaIcon = "⚪" // Нулевая дельта
	}

	// Форматируем значение дельты
	deltaStr := f.numberFormatter.FormatDollarValue(deltaFormatted)

	// Если есть процент изменения, добавляем его с проверкой согласованности
	if deltaPercent != 0 {
		percentIcon := "🟢"
		percentPrefix := "+"

		if deltaPercent < 0 {
			percentIcon = "🔴"
			percentPrefix = "-"
		}

		// Проверяем согласованность знаков
		deltaSignPositive := delta > 0
		deltaPercentSignPositive := deltaPercent > 0

		if deltaSignPositive == deltaPercentSignPositive {
			// Согласованные знаки
			strength := math.Min(math.Abs(deltaPercent)/10, 1.0)

			switch {
			case strength > 0.7:
				// Сильная согласованность
				return fmt.Sprintf("%s%s (%s%s%.1f%% ⚡)",
					deltaIcon, deltaStr, percentIcon, percentPrefix, math.Abs(deltaPercent))
			case strength > 0.4:
				// Средняя согласованность
				return fmt.Sprintf("%s%s (%s%s%.1f%% ↗️)",
					deltaIcon, deltaStr, percentIcon, percentPrefix, math.Abs(deltaPercent))
			default:
				// Слабая согласованность
				return fmt.Sprintf("%s%s (%s%s%.1f%%)",
					deltaIcon, deltaStr, percentIcon, percentPrefix, math.Abs(deltaPercent))
			}
		} else {
			// Противоречивые знаки
			contradictionStrength := math.Min(math.Abs(deltaPercent)/10, 1.0)

			switch {
			case contradictionStrength > 0.7:
				// Сильное противоречие
				return fmt.Sprintf("%s%s (🔄 %s%.1f%% ⚠️)",
					deltaIcon, deltaStr, percentPrefix, math.Abs(deltaPercent))
			case contradictionStrength > 0.4:
				// Среднее противоречие
				return fmt.Sprintf("%s%s (⚠️ %s%.1f%%)",
					deltaIcon, deltaStr, percentPrefix, math.Abs(deltaPercent))
			default:
				// Слабое противоречие
				return fmt.Sprintf("%s%s (%s%.1f%%)",
					deltaIcon, deltaStr, percentPrefix, math.Abs(deltaPercent))
			}
		}
	}

	return fmt.Sprintf("%s%s", deltaIcon, deltaStr)
}
