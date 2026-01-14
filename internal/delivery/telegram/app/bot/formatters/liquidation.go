// internal/delivery/telegram/app/bot/formatters/liquidation.go
package formatters

import (
	"fmt"
	"strings"
)

// LiquidationFormatter отвечает за форматирование ликвидаций
type LiquidationFormatter struct {
	numberFormatter *NumberFormatter
}

// NewLiquidationFormatter создает новый форматтер ликвидаций
func NewLiquidationFormatter() *LiquidationFormatter {
	return &LiquidationFormatter{
		numberFormatter: NewNumberFormatter(),
	}
}

// FormatLiquidationBlock форматирует блок ликвидаций
func (f *LiquidationFormatter) FormatLiquidationBlock(
	period string,
	liquidationVolume float64,
	longLiqVolume float64,
	shortLiqVolume float64,
	volume24h float64,
) string {
	if liquidationVolume <= 0 || volume24h <= 0 {
		return ""
	}

	// Рассчитываем проценты
	longPercent := f.numberFormatter.SafeDivide(longLiqVolume, liquidationVolume) * 100
	shortPercent := f.numberFormatter.SafeDivide(shortLiqVolume, liquidationVolume) * 100
	volumePercent := f.numberFormatter.SafeDivide(liquidationVolume, volume24h) * 100

	// Определяем период ликвидаций
	liqPeriod := f.getLiquidationPeriod(period)

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("💥 ЛИКВИДАЦИИ (%s)\n", liqPeriod))

	// Форматируем объем ликвидаций
	liqStr := f.numberFormatter.FormatDollarValue(liquidationVolume)

	// Показываем процент от дневного объема
	if volumePercent > 0 {
		builder.WriteString(fmt.Sprintf("$%s • %.2f%% от объема\n", liqStr, volumePercent))
	} else {
		builder.WriteString(fmt.Sprintf("$%s\n", liqStr))
	}

	// Создаем компактные прогресс-бары (5 символов)
	longBar := f.formatCompactBar(longPercent, "🟢")
	shortBar := f.formatCompactBar(shortPercent, "🔴")

	// Добавляем индикатор дисбаланса
	imbalanceEmoji := ""
	if shortPercent > 60 {
		imbalanceEmoji = " ⚡"
	} else if longPercent > 60 {
		imbalanceEmoji = " ⚡"
	}

	builder.WriteString(fmt.Sprintf("LONG   %3.0f%% %s\n", longPercent, longBar))
	builder.WriteString(fmt.Sprintf("SHORT  %3.0f%% %s%s\n\n", shortPercent, shortBar, imbalanceEmoji))

	return builder.String()
}

// getLiquidationPeriod определяет период ликвидаций
func (f *LiquidationFormatter) getLiquidationPeriod(period string) string {
	if strings.Contains(period, "15") {
		return "15мин"
	} else if strings.Contains(period, "30") {
		return "30мин"
	} else if strings.Contains(period, "1 час") {
		return "1ч"
	}
	return "5мин"
}

// formatCompactBar создает компактный бар из эмодзи (5 символов)
func (f *LiquidationFormatter) formatCompactBar(percentage float64, emoji string) string {
	// Рассчитываем количество заполненных баров (максимум 5)
	bars := int(percentage / 20) // 5 баров по 20% каждый
	if bars > 5 {
		bars = 5
	}
	if bars < 0 {
		bars = 0
	}

	// Строим строку с барами
	var result string
	for i := 0; i < 5; i++ {
		if i < bars {
			result += emoji
		} else {
			result += "▫️"
		}
	}
	return result
}
