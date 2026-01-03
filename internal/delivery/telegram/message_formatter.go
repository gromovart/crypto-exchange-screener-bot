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
	)
}

// FormatCleanDashboardMessage создает сообщение в чистом формате без рамки
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

	// Объем
	volumeStr := f.formatDollarValue(volume24h)
	builder.WriteString(fmt.Sprintf("📊 Объем: $%s\n\n", volumeStr))

	// ==================== БЛОК 4: ЛИКВИДАЦИИ ====================
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

	// ==================== БЛОК 5: ПРОГРЕСС И ПЕРИОД ====================
	// Прогресс сигналов
	percentage := float64(signalCount) / float64(maxSignals) * 100
	progressBar := f.formatCompactProgressBar(percentage)

	builder.WriteString(fmt.Sprintf("📡 %d/%d %s (%.0f%%)\n",
		signalCount, maxSignals, progressBar, percentage))

	// Период анализа
	builder.WriteString(fmt.Sprintf("🕐 Период: %s\n\n", period))

	// ==================== БЛОК 6: ФАНДИНГ ====================
	// Форматируем фандинг
	fundingStr := f.formatFundingWithEmoji(fundingRate)

	// Время до следующего фандинга
	timeUntil := f.formatCompactTime(nextFundingTime)

	builder.WriteString(fmt.Sprintf("🎯 Фандинг: %s\n", fundingStr))
	builder.WriteString(fmt.Sprintf("⏰ Через: %s", timeUntil))

	return builder.String()
}

// ==================== ВСПОМОГАТЕЛЬНЫЕ МЕТОДЫ ====================

// getContractType возвращает тип контракта
func (f *MarketMessageFormatter) getContractType(symbol string) string {
	symbolUpper := strings.ToUpper(symbol)
	if strings.Contains(symbolUpper, "USDT") {
		return "USDT-фьючерс"
	} else if strings.Contains(symbolUpper, "USD") && !strings.Contains(symbolUpper, "USDT") {
		return "USD-фьючерс"
	} else if strings.Contains(symbolUpper, "PERP") {
		return "Бессрочный"
	}
	return "Фьючерс"
}

// extractTimeframe извлекает таймфрейм из периода
func (f *MarketMessageFormatter) extractTimeframe(period string) string {
	if strings.Contains(period, "5") {
		return "5мин"
	} else if strings.Contains(period, "15") {
		return "15мин"
	} else if strings.Contains(period, "30") {
		return "30мин"
	} else if strings.Contains(period, "1 час") {
		return "1ч"
	} else if strings.Contains(period, "4") {
		return "4ч"
	} else if strings.Contains(period, "1 день") {
		return "1д"
	}
	return "1мин"
}

// getIntensityEmoji возвращает эмодзи силы движения
func (f *MarketMessageFormatter) getIntensityEmoji(change float64) string {
	if change > 5 {
		return "🚨"
	} else if change > 3 {
		return "⚡"
	} else if change > 1.5 {
		return "📈"
	}
	return ""
}

// formatOIWithChange форматирует OI с изменением
func (f *MarketMessageFormatter) formatOIWithChange(oi float64, change float64) string {
	if oi <= 0 {
		return "─"
	}

	oiStr := f.formatDollarValue(oi)

	if change != 0 {
		changeIcon := "🟢"
		if change < 0 {
			changeIcon = "🔴"
		}
		return fmt.Sprintf("$%s (%s%+.1f%%)", oiStr, changeIcon, math.Abs(change))
	}

	return fmt.Sprintf("$%s", oiStr)
}

// formatCompactBar создает компактный бар (5 символов)
func (f *MarketMessageFormatter) formatCompactBar(percentage float64, emoji string) string {
	bars := int(percentage / 20) // 5 баров по 20% каждый
	if bars > 5 {
		bars = 5
	}
	if bars < 0 {
		bars = 0
	}

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

// formatCompactProgressBar создает компактный прогресс-бар (5 символов)
func (f *MarketMessageFormatter) formatCompactProgressBar(percentage float64) string {
	bars := int(percentage / 20) // 5 баров по 20% каждый
	if bars > 5 {
		bars = 5
	}
	if bars < 0 {
		bars = 0
	}

	var result string
	for i := 0; i < 5; i++ {
		if i < bars {
			// Цвет баров в зависимости от заполнения
			if percentage >= 80 {
				result += "🔴"
			} else if percentage >= 50 {
				result += "🟡"
			} else {
				result += "🟢"
			}
		} else {
			result += "▫️"
		}
	}
	return result
}

// formatFundingWithEmoji форматирует фандинг с эмодзи
func (f *MarketMessageFormatter) formatFundingWithEmoji(rate float64) string {
	ratePercent := rate * 100

	// Выбираем эмодзи в зависимости от величины
	var icon string
	if ratePercent > 0.015 {
		icon = "🟢"
	} else if ratePercent > 0.005 {
		icon = "🟡"
	} else if ratePercent > -0.005 {
		icon = "⚪"
	} else if ratePercent > -0.015 {
		icon = "🟠"
	} else {
		icon = "🔴"
	}

	return fmt.Sprintf("%s %.4f%%", icon, ratePercent)
}

// formatCompactTime форматирует время в компактном виде
func (f *MarketMessageFormatter) formatCompactTime(nextFundingTime time.Time) string {
	if nextFundingTime.IsZero() {
		return "─"
	}

	now := time.Now()
	if nextFundingTime.Before(now) {
		return "сейчас"
	}

	duration := nextFundingTime.Sub(now)

	// Компактный формат
	if duration.Hours() >= 1 {
		hours := int(duration.Hours())
		minutes := int(duration.Minutes()) % 60
		if minutes > 0 {
			return fmt.Sprintf("%dч %dм", hours, minutes)
		}
		return fmt.Sprintf("%dч", hours)
	} else {
		minutes := int(duration.Minutes())
		if minutes <= 0 {
			return "скоро!"
		}
		return fmt.Sprintf("%dм", minutes)
	}
}

// formatPrice форматирует цену с учетом ее величины
func (f *MarketMessageFormatter) formatPrice(price float64) string {
	if price <= 0 {
		return "0.00"
	}

	if price >= 1000 {
		return fmt.Sprintf("%.0f", math.Round(price))
	} else if price >= 100 {
		return fmt.Sprintf("%.1f", price)
	} else if price >= 10 {
		return fmt.Sprintf("%.2f", price)
	} else if price >= 1 {
		return fmt.Sprintf("%.3f", price)
	} else if price >= 0.1 {
		return fmt.Sprintf("%.4f", price)
	} else if price >= 0.01 {
		return fmt.Sprintf("%.5f", price)
	} else if price >= 0.001 {
		return fmt.Sprintf("%.6f", price)
	} else if price >= 0.0001 {
		return fmt.Sprintf("%.7f", price)
	} else {
		return fmt.Sprintf("%.8f", price)
	}
}

// formatDollarValue форматирует долларовые значения в читаемый вид
func (f *MarketMessageFormatter) formatDollarValue(num float64) string {
	if num <= 0 {
		return "0"
	}

	// Форматируем в M (миллионы) или K (тысячи)
	if num >= 1_000_000_000 {
		value := num / 1_000_000_000
		if value < 10 {
			return fmt.Sprintf("%.2fB", value)
		} else if value < 100 {
			return fmt.Sprintf("%.1fB", value)
		} else {
			return fmt.Sprintf("%.0fB", math.Round(value))
		}
	} else if num >= 1_000_000 {
		value := num / 1_000_000
		if value < 10 {
			return fmt.Sprintf("%.2fM", value)
		} else if value < 100 {
			return fmt.Sprintf("%.1fM", value)
		} else {
			return fmt.Sprintf("%.0fM", math.Round(value))
		}
	} else if num >= 1_000 {
		value := num / 1_000
		if value < 10 {
			return fmt.Sprintf("%.1fK", value)
		} else {
			return fmt.Sprintf("%.0fK", math.Round(value))
		}
	} else if num >= 1 {
		return fmt.Sprintf("%.0f", math.Round(num))
	} else {
		return fmt.Sprintf("%.2f", num)
	}
}
