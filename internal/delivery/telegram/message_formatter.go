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
	log.Printf("   openInterest параметр = %.0f", openInterest)
	log.Printf("   oiChange24h = %.1f%%", oiChange24h)
	log.Printf("   currentPrice = %.4f", currentPrice)
	log.Printf("   volume24h = %.2f", volume24h)
	log.Printf("   fundingRate = %f", fundingRate)

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
	builder.WriteString(fmt.Sprintf("📊 Объем 24ч: %s\n",
		f.formatVolumeWithVerification(volume24h, currentPrice, symbol)))

	// Открытый интерес с улучшенным форматированием
	oiText := f.formatLargeNumber(openInterest)
	if oiChange24h != 0 {
		changeIcon := "🟢"
		if oiChange24h < 0 {
			changeIcon = "🔴"
		}
		oiText = fmt.Sprintf("%s (%s%.1f%%)", oiText, changeIcon, oiChange24h)
	}
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
		builder.WriteString(fmt.Sprintf(" (%s)", timeUntilFunding))
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

	// Добавляем рекомендации по времени
	f.addTimeRecommendation(&builder, period, signalCount, maxSignals)

	return builder.String()
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

	// Общие рекомендации
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

// formatVolumeWithVerification форматирует объем с проверкой правдоподобности
func (f *MarketMessageFormatter) formatVolumeWithVerification(volume float64, price float64, symbol string) string {
	// Если объем менее $1000, показываем специальный формат
	if volume < 1000 {
		return fmt.Sprintf("$%.0f", volume)
	}

	// Проверяем на явно нереалистичные значения
	// Если цена < $0.1 и объем > $10M - это подозрительно
	if price < 0.1 && volume > 10_000_000 {
		log.Printf("⚠️ Подозрительный объем для %s: цена=$%s, объем=$%.0f",
			symbol, f.formatPrice(price), volume)

		// Пробуем скорректировать - возможно, это объем в монетах, а не в USD
		volumeInUSD := volume * price

		// Если результат более реалистичен, используем его
		if volumeInUSD < 10_000_000 && volumeInUSD > 100 {
			return fmt.Sprintf("$%s", f.formatVolume(volumeInUSD))
		}

		// Если все еще подозрительно, показываем без K/M/B
		if volumeInUSD > 10_000_000 {
			return fmt.Sprintf("$%.0f", volumeInUSD)
		}

		// Иначе показываем "N/A"
		return "N/A"
	}

	return fmt.Sprintf("$%s", f.formatVolume(volume))
}

// formatVolume форматирует объем
func (f *MarketMessageFormatter) formatVolume(volume float64) string {
	if volume >= 1_000_000_000 {
		return fmt.Sprintf("%.2fB", volume/1_000_000_000)
	} else if volume >= 1_000_000 {
		return fmt.Sprintf("%.2fM", volume/1_000_000)
	} else if volume >= 1_000 {
		return fmt.Sprintf("%.1fK", volume/1_000)
	}
	return fmt.Sprintf("%.0f", volume)
}

// formatOpenInterest форматирует открытый интерес
func (f *MarketMessageFormatter) formatOpenInterest(oi float64, oiChange24h float64) string {
	if oi <= 0 {
		// Если OI недоступен, показываем причину
		// Проверяем тип символа
		return "🔍 не поддерживается" // Или "⏳ ожидание данных"
	}

	oiStr := f.formatLargeNumber(oi)

	if oiChange24h != 0 {
		changeIcon := "🟢"
		if oiChange24h < 0 {
			changeIcon = "🔴"
		}
		return fmt.Sprintf("%s (%s%.1f%%)", oiStr, changeIcon, oiChange24h)
	}

	return oiStr
}

// formatLargeNumber форматирует большие числа в читаемый вид
func (f *MarketMessageFormatter) formatLargeNumber(num float64) string {

	// Измените условие для 0:
	if num == 0 {
		return "$0" // ⚠️ Изменено с "недоступно" на "$0"
	}

	if num < 0 {
		return "ошибка" // Отрицательные значения невозможны для OI
	}
	
	if num >= 1_000_000_000_000 {
		return fmt.Sprintf("$%.2fT", num/1_000_000_000_000)
	} else if num >= 1_000_000_000 {
		return fmt.Sprintf("$%.2fB", num/1_000_000_000)
	} else if num >= 1_000_000 {
		return fmt.Sprintf("$%.1fM", num/1_000_000)
	} else if num >= 1_000 {
		return fmt.Sprintf("$%.1fK", num/1_000)
	} else if num >= 1 {
		return fmt.Sprintf("$%.0f", num)
	} else if num > 0 {
		return fmt.Sprintf("$%.2f", num)
	}
	return "недоступно"
}

// formatFunding форматирует ставку фандинга
func (f *MarketMessageFormatter) formatFunding(rate float64, label string) string {
	ratePercent := rate * 100
	rateStr := fmt.Sprintf("%.4f%%", math.Abs(ratePercent))

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
		// Если время не задано, рассчитываем следующее
		nextFundingTime = f.calculateNextFundingTime()
	}

	now := time.Now()
	if nextFundingTime.Before(now) {
		// Если время в прошлом, рассчитываем следующее
		nextFundingTime = f.calculateNextFundingTime()
	}

	duration := nextFundingTime.Sub(now)
	if duration <= 0 {
		return ""
	}

	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60

	if hours > 0 {
		return fmt.Sprintf("через %dч %dм", hours, minutes)
	} else if minutes > 0 {
		return fmt.Sprintf("через %dм", minutes)
	}

	return "скоро"
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
	return f.formatVolume(value)
}
