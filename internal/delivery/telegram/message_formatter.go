// internal/delivery/telegram/message_formatter.go
package telegram

import (
	"fmt"
	"log"
	"math"
	"strings"
	"time"
)

// MarketMessageFormatter форматирует сообщения с рыночными данными
type MarketMessageFormatter struct {
	exchange string
}

// NewMarketMessageFormatter создает новый форматтер
func NewMarketMessageFormatter(exchange string) *MarketMessageFormatter {
	return &MarketMessageFormatter{
		exchange: strings.ToUpper(exchange),
	}
}

// FormatCounterMessage форматирует сообщение счетчика с полными данными
func (f *MarketMessageFormatter) FormatCounterMessage(
	symbol string,
	direction string,
	change float64,
	signalCount int,
	maxSignals int,
	currentPrice float64,
	volume24h float64,
	openInterest float64,
	oiChange24h float64,
	fundingRate float64,
	averageFunding float64,
	nextFundingTime time.Time,
	period string,
) string {
	// Отладочный лог
	log.Printf("🔍 MarketMessageFormatter.FormatCounterMessage для %s:", symbol)
	log.Printf("   openInterest = %.1f", openInterest)
	log.Printf("   oiChange24h = %.1f%%", oiChange24h)
	log.Printf("   currentPrice = %.5f", currentPrice)
	log.Printf("   volume24h = %.2f", volume24h)
	log.Printf("   fundingRate = %.6f", fundingRate)

	var builder strings.Builder

	// Заголовок
	builder.WriteString(fmt.Sprintf("⚫ %s - 1мин - %s\n", f.exchange, symbol))
	builder.WriteString(fmt.Sprintf("🕐 %s\n", time.Now().Format("2006/01/02 15:04:05")))

	// Информация о символе
	f.addSymbolInfo(&builder, symbol, currentPrice)

	// Направление и изменение
	directionIcon := "🟢"
	changePrefix := "+"
	directionText := "РОСТ"
	if direction == "fall" {
		directionIcon = "🔴"
		changePrefix = "-"
		directionText = "ПАДЕНИЕ"
	}

	builder.WriteString(fmt.Sprintf("\n%s %s: %s%.2f%%\n",
		directionIcon,
		directionText,
		changePrefix,
		change))

	// Цена с адаптивным форматированием
	builder.WriteString(fmt.Sprintf("💰 Цена: $%s\n", f.formatPrice(currentPrice)))

	// Объем с проверкой правдоподобности
	builder.WriteString(fmt.Sprintf("📊 Объем 24ч: $%s\n", f.formatDollarValue(volume24h)))

	// Открытый интерес с улучшенным форматированием
	oiText := f.formatOpenInterest(openInterest, oiChange24h)
	builder.WriteString(fmt.Sprintf("📈 Открытый интерес: %s\n", oiText))

	// Фандинг с улучшенным расчетом времени
	builder.WriteString("🎯 Фандинг: ")
	fundingStr := f.formatFunding(fundingRate, "тек.")

	// Добавляем средний фандинг если он отличается от текущего
	if averageFunding != 0 && math.Abs(fundingRate-averageFunding) > 0.0001 {
		fundingStr += fmt.Sprintf(" / %s", f.formatFunding(averageFunding, "ср."))
	}
	builder.WriteString(fundingStr)

	// Время до следующего фандинга
	timeUntilFunding := f.formatTimeUntilFunding(nextFundingTime)
	if timeUntilFunding != "" {
		builder.WriteString(fmt.Sprintf(" (через %s)", timeUntilFunding))
	}
	builder.WriteString("\n")

	// Счетчик сигналов с прогресс-баром
	percentage := float64(signalCount) / float64(maxSignals) * 100
	builder.WriteString(fmt.Sprintf("📡 Сигналов: %d/%d", signalCount, maxSignals))

	// Прогресс-бар
	progressBar := f.formatProgressBar(percentage)
	if progressBar != "" {
		builder.WriteString(fmt.Sprintf(" %s", progressBar))
	}

	// Процент заполнения с индикаторами
	if percentage >= 25 {
		builder.WriteString(fmt.Sprintf(" (%.0f%% заполнено)", percentage))

		// Добавляем эмодзи-индикаторы
		if percentage >= 80 {
			builder.WriteString(" 🚨")
		} else if percentage >= 50 {
			builder.WriteString(" ⚠️")
		}
	}

	builder.WriteString(fmt.Sprintf("\n⏱️  Период: %s", period))

	// Добавляем рекомендации по времени (УДАЛЕНО дублирование в конце функции)
	f.addTimeRecommendation(&builder, period, signalCount, maxSignals)

	return builder.String() // УДАЛЕНО: Дублирующий код предупреждений
}

// addSymbolInfo добавляет информацию о символе
func (f *MarketMessageFormatter) addSymbolInfo(builder *strings.Builder, symbol string, price float64) {
	// Определяем тип контракта
	if strings.Contains(symbol, "USDT") {
		builder.WriteString("💎 USDT-фьючерс\n")
	} else if strings.Contains(symbol, "USD") {
		builder.WriteString("💵 USD-фьючерс\n")
	} else if strings.Contains(symbol, "PERP") {
		builder.WriteString("📈 Бессрочный контракт\n")
	}

	// Оцениваем волатильность
	volatility := f.estimateVolatility(price)
	if volatility > 0 {
		volatilityIcon := "📊"
		if volatility > 10 {
			volatilityIcon = "📈"
		} else if volatility < 2 {
			volatilityIcon = "📉"
		}
		builder.WriteString(fmt.Sprintf("%s Волатильность: ~%.1f%%\n",
			volatilityIcon, volatility))
	}
}

// estimateVolatility оценивает волатильность на основе цены
func (f *MarketMessageFormatter) estimateVolatility(price float64) float64 {
	// Простая эвристика: чем дешевле монета, тем выше волатильность
	if price < 0.001 {
		return 15.0
	} else if price < 0.01 {
		return 8.0
	} else if price < 0.1 {
		return 5.0
	} else if price < 1 {
		return 3.0
	}
	return 2.0
}

// formatProgressBar создает прогресс-бар
func (f *MarketMessageFormatter) formatProgressBar(percentage float64) string {
	if percentage < 10 {
		return "▫️▫️▫️▫️▫️"
	} else if percentage < 30 {
		return "🟩▫️▫️▫️▫️"
	} else if percentage < 50 {
		return "🟩🟩▫️▫️▫️"
	} else if percentage < 70 {
		return "🟩🟩🟩▫️▫️"
	} else if percentage < 90 {
		return "🟩🟩🟩🟩▫️"
	} else {
		return "🟩🟩🟩🟩🟩"
	}
}

// addTimeRecommendation добавляет рекомендации по времени
func (f *MarketMessageFormatter) addTimeRecommendation(builder *strings.Builder, period string, signalCount int, maxSignals int) {
	percentage := float64(signalCount) / float64(maxSignals) * 100

	switch period {
	case "5 минут":
		if signalCount >= 4 {
			builder.WriteString("\n⏰ Ожидайте скорого сброса счетчика")
		}
	case "15 минут":
		if signalCount >= 12 {
			builder.WriteString("\n⏰ Почти достигнут лимит сигналов")
		}
	case "30 минут":
		if signalCount >= 25 {
			builder.WriteString("\n⏰ Высокая активность")
		}
	case "1 час":
		if signalCount >= 50 {
			builder.WriteString("\n⏰ Интенсивное движение")
		}
	case "4 часа":
		if signalCount >= 200 {
			builder.WriteString("\n⏰ Активная торговая сессия")
		}
	}

	// Общие рекомендации (ЕДИНСТВЕННОЕ место для этих предупреждений)
	if percentage >= 80 {
		builder.WriteString("\n🚨 Внимание: счетчик скоро сбросится")
	} else if percentage >= 60 {
		builder.WriteString("\n⚠️  Повышенная активность")
	}
}

// formatPrice форматирует цену с учетом ее величины
func (f *MarketMessageFormatter) formatPrice(price float64) string {
	if price >= 100 {
		return fmt.Sprintf("%.2f", price)
	} else if price >= 1 {
		return fmt.Sprintf("%.4f", price)
	} else if price >= 0.1 {
		return fmt.Sprintf("%.5f", price)
	} else if price >= 0.01 {
		return fmt.Sprintf("%.6f", price)
	} else if price >= 0.001 {
		return fmt.Sprintf("%.7f", price)
	} else {
		return fmt.Sprintf("%.8f", price)
	}
}

// formatDollarValue форматирует долларовые значения в читаемый вид
func (f *MarketMessageFormatter) formatDollarValue(num float64) string {
	if num == 0 {
		return "0"
	}

	if num < 0 {
		return "ошибка"
	}

	// Форматируем в M (миллионы) или K (тысячи)
	if num >= 1_000_000_000 {
		return fmt.Sprintf("%.1fB", num/1_000_000_000)
	} else if num >= 1_000_000 {
		// Для миллионов показываем один знак после запятой
		value := num / 1_000_000
		if value < 10 {
			// Для значений меньше 10 миллионов показываем один знак после запятой
			return fmt.Sprintf("%.1fM", value)
		} else {
			// Для значений больше 10 миллионов показываем без десятичных знаков
			return fmt.Sprintf("%.0fM", math.Round(value))
		}
	} else if num >= 1_000 {
		// Для тысяч показываем без десятичных знаков
		return fmt.Sprintf("%.0fK", math.Round(num/1_000))
	} else if num >= 1 {
		return fmt.Sprintf("%.0f", math.Round(num))
	} else {
		return fmt.Sprintf("%.2f", num)
	}
}

// formatOpenInterest форматирует открытый интерес
func (f *MarketMessageFormatter) formatOpenInterest(oi float64, oiChange24h float64) string {
	if oi < 0 {
		return "ошибка"
	}

	// Если OI = 0, показываем другое сообщение
	if oi == 0 {
		return "⏳ обновление"
	}

	// Форматируем число в $XX.XM/K/B формат
	oiStr := f.formatDollarValue(oi)

	// Добавляем изменение если есть
	if oiChange24h != 0 {
		changeIcon := "🟢"
		changePrefix := "+"

		if oiChange24h < 0 {
			changeIcon = "🔴"
			changePrefix = "-"
		}

		// Используем абсолютное значение для отображения
		changeValue := math.Abs(oiChange24h)

		// Форматируем с одним знаком после запятой
		return fmt.Sprintf("$%s (%s%s%.1f%%)",
			oiStr,
			changeIcon,
			changePrefix,
			changeValue)
	}

	// Если нет данных об изменении, просто показываем значение
	return fmt.Sprintf("$%s", oiStr)
}

// formatFunding форматирует ставку фандинга
func (f *MarketMessageFormatter) formatFunding(rate float64, label string) string {
	ratePercent := rate * 100
	rateStr := fmt.Sprintf("%.4f%%", math.Abs(ratePercent))

	// Улучшенная цветовая логика
	var icon string
	if ratePercent > 0.015 {
		icon = "🟢" // Сильно положительный
	} else if ratePercent > 0.005 {
		icon = "🟡" // Слабо положительный
	} else if ratePercent > -0.005 {
		icon = "⚪" // Нейтральный
	} else if ratePercent > -0.015 {
		icon = "🟠" // Слабо отрицательный
	} else {
		icon = "🔴" // Сильно отрицательный
	}

	if label != "" {
		return fmt.Sprintf("%s %s %s", icon, label, rateStr)
	}
	return fmt.Sprintf("%s %s", icon, rateStr)
}

// formatTimeUntilFunding форматирует время до следующего фандинга
func (f *MarketMessageFormatter) formatTimeUntilFunding(nextFundingTime time.Time) string {
	if nextFundingTime.IsZero() {
		return ""
	}

	now := time.Now()
	if nextFundingTime.Before(now) {
		return "сейчас"
	}

	duration := nextFundingTime.Sub(now)

	// Более человекочитаемый формат
	if duration.Hours() >= 2 {
		hours := int(duration.Hours())
		minutes := int(duration.Minutes()) % 60
		return fmt.Sprintf("%dч %dм", hours, minutes)
	} else if duration.Minutes() >= 1 {
		minutes := int(duration.Minutes())
		return fmt.Sprintf("%dм", minutes)
	} else {
		seconds := int(duration.Seconds())
		if seconds <= 10 {
			return "скоро!"
		}
		return fmt.Sprintf("%dс", seconds)
	}
}

// calculateNextFundingTime рассчитывает следующее время фандинга
func (f *MarketMessageFormatter) calculateNextFundingTime() time.Time {
	now := time.Now().UTC()

	// Фандинг в 00:00, 08:00, 16:00 UTC
	hour := now.Hour()

	// Определяем следующий час фандинга
	var nextHour int
	switch {
	case hour < 8:
		nextHour = 8
	case hour < 16:
		nextHour = 16
	default:
		// Завтра в 00:00
		nextHour = 0
		now = now.Add(24 * time.Hour)
	}

	// Создаем время
	return time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		nextHour,
		0, 0, 0,
		time.UTC,
	)
}

// getDirectionText возвращает текст направления (сохранен для обратной совместимости)
func (f *MarketMessageFormatter) getDirectionText(direction string) string {
	switch direction {
	case "growth":
		return "РОСТ"
	case "fall":
		return "ПАДЕНИЕ"
	default:
		return direction
	}
}

// formatValue форматирует числовые значения (сохранен для обратной совместимости)
func (f *MarketMessageFormatter) formatValue(value float64) string {
	return f.formatDollarValue(value)
}

// formatVolume форматирует объем (сохранен для обратной совместимости)
func (f *MarketMessageFormatter) formatVolume(volume float64) string {
	return f.formatDollarValue(volume)
}
