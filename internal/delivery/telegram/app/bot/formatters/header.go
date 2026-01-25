package formatters

import (
	"strings"
)

// HeaderFormatter отвечает за форматирование заголовка сообщения
type HeaderFormatter struct {
	exchange string
}

// NewHeaderFormatter создает новый форматтер заголовка
func NewHeaderFormatter(exchange string) *HeaderFormatter {
	return &HeaderFormatter{
		exchange: strings.ToUpper(exchange),
	}
}

// GetContractType возвращает тип контракта на основе символа
func (f *HeaderFormatter) GetContractType(symbol string) string {
	symbolUpper := strings.ToUpper(symbol)

	switch {
	case strings.Contains(symbolUpper, "USDT"):
		return "USDT-фьючерс"
	case strings.Contains(symbolUpper, "USD") && !strings.Contains(symbolUpper, "USDT"):
		return "USD-фьючерс"
	case strings.Contains(symbolUpper, "PERP"):
		return "Бессрочный"
	default:
		return "Фьючерс"
	}
}

// ExtractTimeframe извлекает таймфрейм из периода анализа
func (f *HeaderFormatter) ExtractTimeframe(period string) string {
	// Нормализуем входную строку
	period = strings.ToLower(strings.TrimSpace(period))

	switch {
	case strings.HasSuffix(period, "m") || strings.HasSuffix(period, "мин"):
		// Обработка минутных интервалов: "5m", "15m", "30m"
		if strings.Contains(period, "5") {
			return "5мин"
		} else if strings.Contains(period, "15") {
			return "15мин"
		} else if strings.Contains(period, "30") {
			return "30мин"
		} else if strings.Contains(period, "1") {
			return "1мин"
		}
	case strings.HasSuffix(period, "h") || strings.HasSuffix(period, "ч") || strings.Contains(period, "час"):
		// Обработка часовых интервалов: "1h", "4h"
		if strings.Contains(period, "4") {
			return "4ч"
		} else if strings.Contains(period, "1") {
			return "1ч"
		}
	case strings.HasSuffix(period, "d") || strings.HasSuffix(period, "д") || strings.Contains(period, "день"):
		// Обработка дневных интервалов: "1d"
		if strings.Contains(period, "1") {
			return "1д"
		}
	}

	// Дефолтное значение
	return "5мин"
}

// GetIntensityEmoji возвращает эмодзи силы движения на основе процентного изменения
func (f *HeaderFormatter) GetIntensityEmoji(change float64) string {
	switch {
	case change > 5:
		return "🚨" // Очень сильное движение
	case change > 3:
		return "⚡" // Сильное движение
	case change > 1.5:
		return "📈" // Умеренное движение
	default:
		return "" // Слабое движение
	}
}

// GetExchange возвращает название биржи
func (f *HeaderFormatter) GetExchange() string {
	return f.exchange
}
