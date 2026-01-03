// internal/delivery/telegram/message_formatter.go
package telegram

import (
	"fmt"
	"log"
	"math"
	"strings"
	"time"
)

// ==================== ТИПЫ И КОНСТРУКТОР ====================

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

// ==================== ОСНОВНЫЕ МЕТОДЫ ФОРМАТИРОВАНИЯ ====================

// FormatCounterMessage форматирует сообщение счетчика с полными данными
// (совместимость со старым кодом)
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
	liquidationVolume float64,
	longLiqVolume float64,
	shortLiqVolume float64,
) string {
	// Отладочный лог
	log.Printf("🔍 MarketMessageFormatter.FormatCounterMessage для %s:", symbol)
	log.Printf("   openInterest = %.1f", openInterest)
	log.Printf("   oiChange24h = %.1f%%", oiChange24h)
	log.Printf("   currentPrice = %.5f", currentPrice)
	log.Printf("   volume24h = %.2f", volume24h)
	log.Printf("   fundingRate = %.6f", fundingRate)
	log.Printf("   liquidationVolume = %.2f", liquidationVolume)
	log.Printf("   longLiqVolume = %.2f", longLiqVolume)
	log.Printf("   shortLiqVolume = %.2f", shortLiqVolume)

	// Временно используем нулевые значения для дельты и индикаторов
	// TODO: Получить реальные данные из анализатора
	volumeDelta := 0.0
	volumeDeltaPercent := 0.0
	rsi := 0.0
	macdSignal := 0.0

	return f.FormatMessage(
		symbol,
		direction,
		change,
		signalCount,
		maxSignals,
		currentPrice,
		volume24h,
		openInterest,
		oiChange24h,
		fundingRate,
		averageFunding,
		nextFundingTime,
		period,
		liquidationVolume,
		longLiqVolume,
		shortLiqVolume,
		volumeDelta,
		volumeDeltaPercent,
		rsi,
		macdSignal,
	)
}

// FormatMessage создает сообщение в чистом формате без рамки
func (f *MarketMessageFormatter) FormatMessage(
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
	liquidationVolume float64,
	longLiqVolume float64,
	shortLiqVolume float64,
	volumeDelta float64, // Дельта объемов в USD
	volumeDeltaPercent float64, // Изменение дельты в процентах
	rsi float64, // Индикатор RSI (0 если недоступен)
	macdSignal float64, // Сигнал MACD (0 если недоступен)
) string {
	var builder strings.Builder

	// ==================== БЛОК 1: ЗАГОЛОВОК ====================
	timeframe := f.extractTimeframe(period)
	contractType := f.getContractType(symbol)

	builder.WriteString(fmt.Sprintf("🏷️  %s • %s\n", f.exchange, timeframe))
	builder.WriteString(fmt.Sprintf("📛 %s\n", symbol))
	builder.WriteString(fmt.Sprintf("📄 %s\n", contractType))
	builder.WriteString(fmt.Sprintf("🕐 %s\n\n", time.Now().Format("15:04:05")))

	// ==================== БЛОК 2: СИГНАЛ И ЦЕНА ====================
	directionIcon := "🟢"
	directionText := "РОСТ"
	changePrefix := "+"

	if direction == "fall" {
		directionIcon = "🔴"
		directionText = "ПАДЕНИЕ"
		changePrefix = "-"
	}

	// Добавляем индикатор силы движения
	intensityEmoji := f.getIntensityEmoji(math.Abs(change))

	builder.WriteString(fmt.Sprintf("%s %s %s%.2f%% %s\n",
		directionIcon, directionText, changePrefix, math.Abs(change), intensityEmoji))
	builder.WriteString(fmt.Sprintf("💰 $%s\n\n", f.formatPrice(currentPrice)))

	// ==================== БЛОК 3: РЫНОЧНЫЕ МЕТРИКИ ====================
	// Открытый интерес
	oiStr := f.formatOIWithChange(openInterest, oiChange24h)
	builder.WriteString(fmt.Sprintf("📈 OI: %s\n", oiStr))

	// Объем 24ч
	volumeStr := f.formatDollarValue(volume24h)
	builder.WriteString(fmt.Sprintf("📊 Объем 24ч: $%s\n", volumeStr))

	// Дельта объемов с изменением
	if volumeDelta != 0 || volumeDeltaPercent != 0 {
		deltaStr := f.formatVolumeDelta(volumeDelta, volumeDeltaPercent, direction)
		builder.WriteString(fmt.Sprintf("📈 Дельта: %s\n\n", deltaStr))
	} else {
		builder.WriteString("\n")
	}

	// ==================== БЛОК 4: ТЕХНИЧЕСКИЙ АНАЛИЗ ====================
	if rsi > 0 || macdSignal != 0 {
		builder.WriteString(fmt.Sprintf("📊 Тех. анализ:\n"))

		// RSI
		if rsi > 0 {
			rsiStr := f.formatRSI(rsi)
			builder.WriteString(fmt.Sprintf("%s\n", rsiStr))
		}

		// MACD
		if macdSignal != 0 {
			macdStr := f.formatMACD(macdSignal)
			builder.WriteString(fmt.Sprintf("%s\n", macdStr))
		}

		builder.WriteString("\n")
	}

	// ==================== БЛОК 5: ЛИКВИДАЦИИ ====================
	if liquidationVolume > 0 && volume24h > 0 {
		// Рассчитываем проценты
		var longPercent, shortPercent, volumePercent float64
		if liquidationVolume > 0 {
			longPercent = (longLiqVolume / liquidationVolume) * 100
			shortPercent = (shortLiqVolume / liquidationVolume) * 100
		}
		if volume24h > 0 {
			volumePercent = (liquidationVolume / volume24h) * 100
		}

		// Определяем период ликвидаций из анализа
		liqPeriod := "5мин"
		if strings.Contains(period, "15") {
			liqPeriod = "15мин"
		} else if strings.Contains(period, "30") {
			liqPeriod = "30мин"
		} else if strings.Contains(period, "1 час") {
			liqPeriod = "1ч"
		}

		builder.WriteString(fmt.Sprintf("💥 ЛИКВИДАЦИИ (%s)\n", liqPeriod))

		// Форматируем объем ликвидаций
		liqStr := f.formatDollarValue(liquidationVolume)

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
	}

	// ==================== БЛОК 6: ПРОГРЕСС И ПЕРИОД ====================
	// Прогресс сигналов
	percentage := float64(signalCount) / float64(maxSignals) * 100
	progressBar := f.formatCompactProgressBar(percentage)

	builder.WriteString(fmt.Sprintf("📡 %d/%d %s (%.0f%%)\n",
		signalCount, maxSignals, progressBar, percentage))

	// Период анализа
	builder.WriteString(fmt.Sprintf("🕐 Период: %s\n\n", period))

	// ==================== БЛОК 7: РЕКОМЕНДАЦИИ ПО ТОРГОВЛЕ ====================
	recommendation := f.getTradingRecommendation(direction, rsi, macdSignal, volumeDelta, longLiqVolume, shortLiqVolume)
	if recommendation != "" {
		builder.WriteString(fmt.Sprintf("🎯 Рекомендация:\n%s\n\n", recommendation))
	}

	// ==================== БЛОК 8: ФАНДИНГ ====================
	// Форматируем фандинг
	fundingStr := f.formatFundingWithEmoji(fundingRate)

	// Время до следующего фандинга
	timeUntil := f.formatCompactTime(nextFundingTime)

	builder.WriteString(fmt.Sprintf("🎯 Фандинг: %s\n", fundingStr))
	builder.WriteString(fmt.Sprintf("⏰ Через: %s", timeUntil))

	return builder.String()
}

// ==================== МЕТОДЫ ТЕХНИЧЕСКОГО АНАЛИЗА ====================

// formatRSI форматирует RSI с описанием состояния
func (f *MarketMessageFormatter) formatRSI(rsi float64) string {
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

// formatMACD форматирует MACD с описанием сигнала
func (f *MarketMessageFormatter) formatMACD(macdSignal float64) string {
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

// getTradingRecommendation возвращает рекомендации по торговле на основе всех индикаторов
func (f *MarketMessageFormatter) getTradingRecommendation(
	direction string,
	rsi float64,
	macdSignal float64,
	volumeDelta float64,
	longLiqVolume float64,
	shortLiqVolume float64,
) string {
	var recommendations []string

	// Анализ RSI - определяем зоны перекупленности/перепроданности
	if rsi >= 70 {
		recommendations = append(recommendations, "RSI в зоне перекупленности - осторожность с LONG")
	} else if rsi <= 30 {
		recommendations = append(recommendations, "RSI в зоне перепроданности - осторожность с SHORT")
	}

	// Анализ MACD - определяем тренд
	if macdSignal > 0.05 {
		recommendations = append(recommendations, "MACD бычий - рассмотреть LONG")
	} else if macdSignal < -0.05 {
		recommendations = append(recommendations, "MACD медвежий - рассмотреть SHORT")
	}

	// Анализ дельты объемов - определяем настроения
	if volumeDelta > 0 {
		if direction == "growth" {
			recommendations = append(recommendations, "Дельта подтверждает рост - LONG приоритет")
		} else {
			recommendations = append(recommendations, "Дельта противоречит падению - возможен разворот")
		}
	} else if volumeDelta < 0 {
		if direction == "fall" {
			recommendations = append(recommendations, "Дельта подтверждает падение - SHORT приоритет")
		} else {
			recommendations = append(recommendations, "Дельта противоречит росту - возможна коррекция")
		}
	}

	// Анализ ликвидаций - определяем давление на рынок
	if longLiqVolume > shortLiqVolume*1.5 {
		recommendations = append(recommendations, "Много LONG ликвидаций - возможен отскок вверх")
	} else if shortLiqVolume > longLiqVolume*1.5 {
		recommendations = append(recommendations, "Много SHORT ликвидаций - возможен отскок вниз")
	}

	// Если рекомендаций нет - возвращаем пустую строку
	if len(recommendations) == 0 {
		return ""
	}

	// Определяем общий сигнал на основе всех рекомендаций
	var primarySignal string
	if len(recommendations) >= 2 {
		bullishCount := 0
		bearishCount := 0

		for _, rec := range recommendations {
			if strings.Contains(rec, "LONG") || strings.Contains(rec, "рост") || strings.Contains(rec, "бычий") {
				bullishCount++
			} else if strings.Contains(rec, "SHORT") || strings.Contains(rec, "падение") || strings.Contains(rec, "медвежий") {
				bearishCount++
			}
		}

		switch {
		case bullishCount > bearishCount:
			primarySignal = "🟢 Преобладают бычьи сигналы"
		case bearishCount > bullishCount:
			primarySignal = "🔴 Преобладают медвежьи сигналы"
		default:
			primarySignal = "⚪ Смешанные сигналы"
		}
	} else {
		primarySignal = "📊 Одиночный сигнал"
	}

	// Формируем итоговое сообщение с нумерованными рекомендациями
	result := primarySignal + "\n"
	for i, rec := range recommendations {
		result += fmt.Sprintf("%d. %s\n", i+1, rec)
	}

	return strings.TrimSpace(result)
}

// ==================== МЕТОДЫ ФОРМАТИРОВАНИЯ ДЕЛЬТЫ ОБЪЕМОВ ====================

// formatVolumeDelta форматирует дельту объемов с процентом изменения
func (f *MarketMessageFormatter) formatVolumeDelta(delta float64, deltaPercent float64, direction string) string {
	// Если данных нет - возвращаем прочерк
	if delta == 0 && deltaPercent == 0 {
		return "─"
	}

	// Определяем знак и цвет дельты
	var deltaIcon string
	deltaFormatted := math.Abs(delta)

	switch {
	case delta > 0:
		deltaIcon = "🟢" // Положительная дельта - покупки преобладают
	case delta < 0:
		deltaIcon = "🔴" // Отрицательная дельта - продажи преобладают
	default:
		deltaIcon = "⚪" // Нулевая дельта
	}

	// Форматируем значение дельты
	deltaStr := f.formatDollarValue(deltaFormatted)

	// Если есть процент изменения, добавляем его с проверкой согласованности
	if deltaPercent != 0 {
		percentIcon := "🟢"
		percentPrefix := "+"

		if deltaPercent < 0 {
			percentIcon = "🔴"
			percentPrefix = "-"
		}

		// Проверяем согласованность знаков дельты и процента изменения
		if (delta > 0 && deltaPercent > 0) || (delta < 0 && deltaPercent < 0) {
			// Согласованные знаки - покупатели/продавцы усиливают давление
			return fmt.Sprintf("%s%s (%s%s%.1f%%)",
				deltaIcon, deltaStr, percentIcon, percentPrefix, math.Abs(deltaPercent))
		} else {
			// Противоречивые знаки - возможен разворот
			return fmt.Sprintf("%s%s (⚠️ %s%.1f%%)",
				deltaIcon, deltaStr, percentPrefix, math.Abs(deltaPercent))
		}
	}

	return fmt.Sprintf("%s%s", deltaIcon, deltaStr)
}

// ==================== ВСПОМОГАТЕЛЬНЫЕ МЕТОДЫ ФОРМАТИРОВАНИЯ ====================

// getContractType возвращает тип контракта на основе символа
func (f *MarketMessageFormatter) getContractType(symbol string) string {
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

// extractTimeframe извлекает таймфрейм из периода анализа
func (f *MarketMessageFormatter) extractTimeframe(period string) string {
	switch {
	case strings.Contains(period, "5"):
		return "5мин"
	case strings.Contains(period, "15"):
		return "15мин"
	case strings.Contains(period, "30"):
		return "30мин"
	case strings.Contains(period, "1 час"):
		return "1ч"
	case strings.Contains(period, "4"):
		return "4ч"
	case strings.Contains(period, "1 день"):
		return "1д"
	default:
		return "1мин"
	}
}

// getIntensityEmoji возвращает эмодзи силы движения на основе процентного изменения
func (f *MarketMessageFormatter) getIntensityEmoji(change float64) string {
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

// formatOIWithChange форматирует открытый интерес с процентным изменением
func (f *MarketMessageFormatter) formatOIWithChange(oi float64, change float64) string {
	// Если OI недоступен
	if oi <= 0 {
		return "─"
	}

	oiStr := f.formatDollarValue(oi)

	// Если есть изменение, добавляем его с цветным индикатором
	if change != 0 {
		changeIcon := "🟢"
		if change < 0 {
			changeIcon = "🔴"
		}
		return fmt.Sprintf("$%s (%s%+.1f%%)", oiStr, changeIcon, math.Abs(change))
	}

	return fmt.Sprintf("$%s", oiStr)
}

// formatCompactBar создает компактный бар из эмодзи (5 символов)
func (f *MarketMessageFormatter) formatCompactBar(percentage float64, emoji string) string {
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

// formatCompactProgressBar создает компактный прогресс-бар для счетчика сигналов
func (f *MarketMessageFormatter) formatCompactProgressBar(percentage float64) string {
	// Рассчитываем количество заполненных баров
	bars := int(percentage / 20) // 5 баров по 20% каждый
	if bars > 5 {
		bars = 5
	}
	if bars < 0 {
		bars = 0
	}

	// Строим прогресс-бар с цветами в зависимости от заполнения
	var result string
	for i := 0; i < 5; i++ {
		if i < bars {
			// Цвет баров меняется в зависимости от уровня заполнения
			switch {
			case percentage >= 80:
				result += "🔴" // Высокое заполнение - красный
			case percentage >= 50:
				result += "🟡" // Среднее заполнение - желтый
			default:
				result += "🟢" // Низкое заполнение - зеленый
			}
		} else {
			result += "▫️"
		}
	}
	return result
}

// formatFundingWithEmoji форматирует ставку фандинга с эмодзи
func (f *MarketMessageFormatter) formatFundingWithEmoji(rate float64) string {
	ratePercent := rate * 100

	// Выбираем эмодзи в зависимости от величины ставки фандинга
	var icon string
	switch {
	case ratePercent > 0.015:
		icon = "🟢" // Сильно положительный
	case ratePercent > 0.005:
		icon = "🟡" // Слабо положительный
	case ratePercent > -0.005:
		icon = "⚪" // Нейтральный
	case ratePercent > -0.015:
		icon = "🟠" // Слабо отрицательный
	default:
		icon = "🔴" // Сильно отрицательный
	}

	return fmt.Sprintf("%s %.4f%%", icon, ratePercent)
}

// formatCompactTime форматирует время в компактном читаемом виде
func (f *MarketMessageFormatter) formatCompactTime(nextFundingTime time.Time) string {
	// Если время не задано
	if nextFundingTime.IsZero() {
		return "─"
	}

	now := time.Now()

	// Если время уже прошло
	if nextFundingTime.Before(now) {
		return "сейчас"
	}

	duration := nextFundingTime.Sub(now)

	// Форматируем в зависимости от длительности
	switch {
	case duration.Hours() >= 1:
		hours := int(duration.Hours())
		minutes := int(duration.Minutes()) % 60
		if minutes > 0 {
			return fmt.Sprintf("%dч %dм", hours, minutes)
		}
		return fmt.Sprintf("%dч", hours)
	default:
		minutes := int(duration.Minutes())
		if minutes <= 0 {
			return "скоро!"
		}
		return fmt.Sprintf("%dм", minutes)
	}
}

// ==================== МЕТОДЫ ФОРМАТИРОВАНИЯ ЧИСЕЛ ====================

// formatPrice форматирует цену с учетом ее величины
func (f *MarketMessageFormatter) formatPrice(price float64) string {
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

// formatDollarValue форматирует долларовые значения в читаемый вид (K/M/B)
func (f *MarketMessageFormatter) formatDollarValue(num float64) string {
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
