// internal/delivery/telegram/app/bot/formatters/signal.go
package formatters

import (
	"fmt"
	"math"
)

// SignalFormatter отвечает за форматирование сигналов
type SignalFormatter struct {
	numberFormatter *NumberFormatter
}

// NewSignalFormatter создает новый форматтер сигналов
func NewSignalFormatter() *SignalFormatter {
	return &SignalFormatter{
		numberFormatter: NewNumberFormatter(),
	}
}

// FormatSignalBlock форматирует блок сигнала и цены
func (f *SignalFormatter) FormatSignalBlock(
	direction string,
	change float64,
	currentPrice float64,
) string {
	directionIcon := "🟢"
	directionText := "РОСТ"
	changePrefix := "+"

	if direction == "fall" {
		directionIcon = "🔴"
		directionText = "ПАДЕНИЕ"
		changePrefix = "-"
	}

	priceStr := f.numberFormatter.FormatPrice(currentPrice)

	return fmt.Sprintf("%s %s %s%.2f%%\n💰 $%s\n\n",
		directionIcon, directionText, changePrefix, math.Abs(change), priceStr)
}

// GetDirectionInfo возвращает иконку и текст направления
func (f *SignalFormatter) GetDirectionInfo(direction string) (string, string, string) {
	directionIcon := "🟢"
	directionText := "РОСТ"
	changePrefix := "+"

	if direction == "fall" {
		directionIcon = "🔴"
		directionText = "ПАДЕНИЕ"
		changePrefix = "-"
	}

	return directionIcon, directionText, changePrefix
}
