// internal/delivery/telegram/app/bot/formatters/technical.go
package formatters

import (
	"fmt"
)

// TechnicalFormatter отвечает за форматирование технического анализа
type TechnicalFormatter struct{}

// NewTechnicalFormatter создает новый форматтер тех. анализа
func NewTechnicalFormatter() *TechnicalFormatter {
	return &TechnicalFormatter{}
}

// FormatRSI форматирует RSI с описанием состояния
func (f *TechnicalFormatter) FormatRSI(rsi float64) string {
	var emoji, description string

	// Определяем зону RSI
	switch {
	case rsi >= 70:
		emoji = "🔴"
		description = "сильная перекупленность"
	case rsi >= 62:
		emoji = "🟡"
		description = "перекупленность"
	case rsi >= 55:
		emoji = "🟢"
		description = "бычий настрой"
	case rsi >= 45:
		emoji = "⚪"
		description = "нейтральный"
	case rsi >= 38:
		emoji = "🟠"
		description = "медвежий настрой"
	default:
		emoji = "🔴"
		description = "сильная перепроданность"
	}

	return fmt.Sprintf("RSI: %.1f %s (%s)", rsi, emoji, description)
}

// FormatRSIWithStatus форматирует RSI с готовым статусом (для реальных данных из CounterAnalyzer)
func (f *TechnicalFormatter) FormatRSIWithStatus(rsi float64, rsiStatus string) string {
	var emoji string

	// Определяем эмодзи по статусу
	switch rsiStatus {
	case "сильная перекупленность", "перекупленность":
		emoji = "🔴"
	case "бычий настрой":
		emoji = "🟢"
	case "медвежий настрой":
		emoji = "🟠"
	case "сильная перепроданность":
		emoji = "🔴"
	default: // "нейтральный", "недостаточно данных"
		emoji = "⚪"
	}

	return fmt.Sprintf("RSI: %.1f %s (%s)", rsi, emoji, rsiStatus)
}

// FormatMACD форматирует MACD с описанием сигнала
func (f *TechnicalFormatter) FormatMACD(macdSignal float64) string {
	var emoji, description string

	// Определяем силу MACD сигнала
	switch {
	case macdSignal > 0.1:
		emoji = "🟢"
		description = "сильный бычий"
	case macdSignal > 0.01:
		emoji = "🟡"
		description = "бычий"
	case macdSignal > -0.01:
		emoji = "⚪"
		description = "нейтральный"
	case macdSignal > -0.1:
		emoji = "🟠"
		description = "медвежий"
	default:
		emoji = "🔴"
		description = "сильный медвежий"
	}

	return fmt.Sprintf("MACD: %s %s", emoji, description)
}

// FormatMACDWithDescription форматирует MACD с готовым описанием (для реальных данных из CounterAnalyzer)
func (f *TechnicalFormatter) FormatMACDWithDescription(macdDescription string) string {
	return fmt.Sprintf("MACD: %s", macdDescription)
}
