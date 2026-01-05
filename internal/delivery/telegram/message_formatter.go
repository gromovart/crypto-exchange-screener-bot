// internal/delivery/telegram/message_formatter.go
package telegram

import (
	"crypto-exchange-screener-bot/internal/infrastructure/api/exchanges/bybit"
	"crypto-exchange-screener-bot/internal/types"
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
		"",
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
	deltaSource string, // 🔴 НОВЫЙ ПАРАМЕТР: источник данных дельты (пустая строка если нет)
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

		// 🔴 ДОБАВЛЯЕМ ИСТОЧНИК ДАННЫХ
		if deltaSource != "" {
			sourceIndicator := getSourceIndicator(deltaSource)
			deltaStr += sourceIndicator
		}

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
	recommendation := f.getEnhancedTradingRecommendation(direction, rsi, macdSignal, volumeDelta, volumeDeltaPercent, longLiqVolume, shortLiqVolume)
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

// getSourceIndicator возвращает строку с индикатором источника
func getSourceIndicator(source string) string {
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

	// 🔴 УЛУЧШЕНИЕ: Более точное определение направления
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
	deltaStr := f.formatDollarValue(deltaFormatted)

	// Если есть процент изменения, добавляем его с проверкой согласованности
	if deltaPercent != 0 {
		percentIcon := "🟢"
		percentPrefix := "+"

		if deltaPercent < 0 {
			percentIcon = "🔴"
			percentPrefix = "-"
		}

		// 🔴 УЛУЧШЕНИЕ: Более сложная проверка согласованности
		deltaSignPositive := delta > 0
		deltaPercentSignPositive := deltaPercent > 0

		if deltaSignPositive == deltaPercentSignPositive {
			// Согласованные знаки - покупатели/продавцы усиливают давление

			// 🔴 УЛУЧШЕНИЕ: Определяем силу согласованности
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
			// Противоречивые знаки - возможен разворот

			// 🔴 УЛУЧШЕНИЕ: Определяем силу противоречия
			contradictionStrength := math.Min(math.Abs(deltaPercent)/10, 1.0)

			switch {
			case contradictionStrength > 0.7:
				// Сильное противоречие - высокая вероятность разворота
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

// formatVolumeDeltaWithDetails форматирует дельту с дополнительной информацией
func (f *MarketMessageFormatter) formatVolumeDeltaWithDetails(
	delta float64,
	deltaPercent float64,
	direction string,
	buyVolume float64,
	sellVolume float64,
	totalTrades int,
) string {
	if delta == 0 && deltaPercent == 0 && buyVolume == 0 && sellVolume == 0 {
		return "─"
	}

	// Базовое форматирование
	baseString := f.formatVolumeDelta(delta, deltaPercent, direction)

	// Добавляем дополнительную информацию если доступна
	var details strings.Builder

	if buyVolume > 0 && sellVolume > 0 {
		// Рассчитываем соотношение покупок/продаж
		totalVolume := buyVolume + sellVolume
		buyRatio := (buyVolume / totalVolume) * 100
		sellRatio := 100 - buyRatio

		// Форматируем объемы
		buyStr := f.formatDollarValue(buyVolume)
		sellStr := f.formatDollarValue(sellVolume)

		// Определяем доминирующую сторону
		var dominanceIcon string
		if buyRatio > 55 {
			dominanceIcon = "🟢"
		} else if sellRatio > 55 {
			dominanceIcon = "🔴"
		} else {
			dominanceIcon = "⚪"
		}

		details.WriteString(fmt.Sprintf("\n   %s Покупки: $%s (%.0f%%)",
			dominanceIcon, buyStr, buyRatio))
		details.WriteString(fmt.Sprintf("\n   %s Продажи: $%s (%.0f%%)",
			dominanceIcon, sellStr, sellRatio))
	}

	if totalTrades > 0 {
		tradesPerMinute := float64(totalTrades) / 5.0 // Для 5-минутного периода
		var activityIcon string

		switch {
		case tradesPerMinute > 50:
			activityIcon = "⚡" // Очень высокая активность
		case tradesPerMinute > 20:
			activityIcon = "🔥" // Высокая активность
		case tradesPerMinute > 5:
			activityIcon = "📊" // Средняя активность
		default:
			activityIcon = "📉" // Низкая активность
		}

		details.WriteString(fmt.Sprintf("\n   %s Сделок: %d (%.1f/мин)",
			activityIcon, totalTrades, tradesPerMinute))
	}

	if details.Len() > 0 {
		return baseString + details.String()
	}

	return baseString
}

// getEnhancedTradingRecommendation возвращает улучшенные рекомендации по торговле
func (f *MarketMessageFormatter) getEnhancedTradingRecommendation(
	direction string,
	rsi float64,
	macdSignal float64,
	volumeDelta float64,
	volumeDeltaPercent float64,
	longLiqVolume float64,
	shortLiqVolume float64,
) string {
	var recommendations []string

	// 🔴 УЛУЧШЕНИЕ: Анализ силы движения
	priceDirectionStrength := "слабое"
	if math.Abs(volumeDelta) > 50000 {
		priceDirectionStrength = "сильное"
	} else if math.Abs(volumeDelta) > 10000 {
		priceDirectionStrength = "умеренное"
	}

	// Анализ RSI - определяем зоны перекупленности/перепроданности
	if rsi >= 70 {
		recommendations = append(recommendations,
			fmt.Sprintf("RSI в зоне перекупленности (%.1f) - осторожность с LONG", rsi))
	} else if rsi >= 62 {
		recommendations = append(recommendations,
			fmt.Sprintf("RSI показывает перекупленность (%.1f)", rsi))
	} else if rsi <= 30 {
		recommendations = append(recommendations,
			fmt.Sprintf("RSI в зоне перепроданности (%.1f) - осторожность с SHORT", rsi))
	} else if rsi <= 38 {
		recommendations = append(recommendations,
			fmt.Sprintf("RSI показывает перепроданность (%.1f)", rsi))
	} else if rsi >= 55 {
		recommendations = append(recommendations,
			fmt.Sprintf("RSI бычий настрой (%.1f)", rsi))
	} else if rsi < 45 {
		recommendations = append(recommendations,
			fmt.Sprintf("RSI медвежий настрой (%.1f)", rsi))
	}

	// Анализ MACD - определяем тренд
	if macdSignal > 0.1 {
		recommendations = append(recommendations, "MACD: сильный бычий тренд")
	} else if macdSignal > 0.05 {
		recommendations = append(recommendations, "MACD: бычий тренд")
	} else if macdSignal > 0.01 {
		recommendations = append(recommendations, "MACD: слабый бычий сигнал")
	} else if macdSignal < -0.1 {
		recommendations = append(recommendations, "MACD: сильный медвежий тренд")
	} else if macdSignal < -0.05 {
		recommendations = append(recommendations, "MACD: медвежий тренд")
	} else if macdSignal < -0.01 {
		recommendations = append(recommendations, "MACD: слабый медвежий сигнал")
	} else {
		recommendations = append(recommendations, "MACD: нейтральный")
	}

	// 🔴 УЛУЧШЕНИЕ: Более детальный анализ дельты объемов
	if math.Abs(volumeDelta) > 0 {
		// Определяем силу дельты
		deltaStrength := math.Abs(volumeDelta)
		var strengthLevel, deltaDescription string

		switch {
		case deltaStrength > 100000:
			strengthLevel = "сильная"
			deltaDescription = "значительное давление"
		case deltaStrength > 10000:
			strengthLevel = "умеренная"
			deltaDescription = "заметное давление"
		case deltaStrength > 1000:
			strengthLevel = "слабая"
			deltaDescription = "небольшое давление"
		default:
			strengthLevel = "незначительная"
			deltaDescription = "минимальное давление"
		}

		// 🔴 ИСПРАВЛЕНИЕ: Убираем дублирующуюся иконку, оставляем только одну
		if volumeDelta > 0 {
			if direction == "growth" {
				recommendations = append(recommendations,
					fmt.Sprintf("%s дельта покупок ($%.0f) - %s покупателей",
						strengthLevel, volumeDelta, deltaDescription))
			} else {
				recommendations = append(recommendations,
					fmt.Sprintf("⚠️ %s дельта покупок при падении ($%.0f) - возможен разворот",
						strengthLevel, volumeDelta))
			}
		} else {
			if direction == "fall" {
				recommendations = append(recommendations,
					fmt.Sprintf("%s дельта продаж ($%.0f) - %s продавцов",
						strengthLevel, math.Abs(volumeDelta), deltaDescription))
			} else {
				recommendations = append(recommendations,
					fmt.Sprintf("⚠️ %s дельта продаж при росте ($%.0f) - возможна коррекция",
						strengthLevel, math.Abs(volumeDelta)))
			}
		}

		// Анализ согласованности с ценовым движением
		if volumeDeltaPercent != 0 {
			if (volumeDelta > 0 && volumeDeltaPercent > 0) || (volumeDelta < 0 && volumeDeltaPercent < 0) {
				// Согласованность
				consistencyStrength := math.Min(math.Abs(volumeDeltaPercent)/10, 1.0)
				if consistencyStrength > 0.5 {
					recommendations = append(recommendations,
						"✅ Объемы подтверждают ценовое движение")
				} else {
					recommendations = append(recommendations,
						"🟡 Объемы слабо подтверждают движение")
				}
			} else {
				// Противоречие
				contradictionStrength := math.Min(math.Abs(volumeDeltaPercent)/10, 1.0)
				if contradictionStrength > 0.5 {
					recommendations = append(recommendations,
						"🔄 Сильное противоречие объемов - возможен разворот")
				} else {
					recommendations = append(recommendations,
						"⚠️ Объемы противоречат ценовому движению")
				}
			}
		}
	}

	// Анализ ликвидаций - определяем давление на рынок
	liquidationRatio := 0.0
	if shortLiqVolume > 0 {
		liquidationRatio = longLiqVolume / shortLiqVolume
	}

	totalLiq := longLiqVolume + shortLiqVolume
	if totalLiq > 0 {
		var liqDescription string

		if totalLiq > 100000 {
			liqDescription = "значительные"
		} else if totalLiq > 10000 {
			liqDescription = "заметные"
		} else {
			liqDescription = "небольшие"
		}

		if liquidationRatio > 2.0 {
			recommendations = append(recommendations,
				fmt.Sprintf("💥 %s LONG ликвидации ($%.0f) - возможен отскок вверх",
					liqDescription, longLiqVolume))
		} else if liquidationRatio < 0.5 {
			recommendations = append(recommendations,
				fmt.Sprintf("💥 %s SHORT ликвидации ($%.0f) - возможен отскок вниз",
					liqDescription, shortLiqVolume))
		} else if totalLiq > 50000 {
			recommendations = append(recommendations,
				fmt.Sprintf("💥 %s ликвидации ($%.0f) - повышенная волатильность",
					liqDescription, totalLiq))
		}
	}

	// Если рекомендаций нет - возвращаем пустую строку
	if len(recommendations) == 0 {
		return ""
	}

	// Определяем общий сигнал на основе всех рекомендаций
	var primarySignal string

	// Подсчитываем баллы для каждого типа сигналов
	bullishScore := 0
	bearishScore := 0
	neutralScore := 0

	for _, rec := range recommendations {
		lowerRec := strings.ToLower(rec)

		// 🔴 УЛУЧШЕНИЕ: Более точная система баллов
		if strings.Contains(lowerRec, "long") || strings.Contains(lowerRec, "рост") ||
			strings.Contains(lowerRec, "бычий") || strings.Contains(lowerRec, "покуп") ||
			strings.Contains(lowerRec, "дельта покупок") ||
			strings.Contains(lowerRec, "сильный бычий") {

			// Определяем силу бычьего сигнала
			if strings.Contains(lowerRec, "сильный") || strings.Contains(lowerRec, "значительное") {
				bullishScore += 3
			} else if strings.Contains(lowerRec, "умерен") || strings.Contains(lowerRec, "заметное") {
				bullishScore += 2
			} else {
				bullishScore += 1
			}

		} else if strings.Contains(lowerRec, "short") || strings.Contains(lowerRec, "падение") ||
			strings.Contains(lowerRec, "медвежий") || strings.Contains(lowerRec, "продаж") ||
			strings.Contains(lowerRec, "дельта продаж") ||
			strings.Contains(lowerRec, "сильный медвежий") {

			// Определяем силу медвежьего сигнала
			if strings.Contains(lowerRec, "сильный") || strings.Contains(lowerRec, "значительное") {
				bearishScore += 3
			} else if strings.Contains(lowerRec, "умерен") || strings.Contains(lowerRec, "заметное") {
				bearishScore += 2
			} else {
				bearishScore += 1
			}

		} else if strings.Contains(lowerRec, "нейтраль") || strings.Contains(lowerRec, "слабый") ||
			strings.Contains(lowerRec, "незначитель") {
			neutralScore += 1
		} else if strings.Contains(lowerRec, "⚠️") || strings.Contains(lowerRec, "🔄") {
			// Противоречивые сигналы уменьшают уверенность
			bullishScore -= 1
			bearishScore -= 1
			neutralScore += 2
		}
	}

	// Определяем итоговый сигнал на основе баллов
	totalWeightedScore := bullishScore + bearishScore + neutralScore

	if totalWeightedScore == 0 {
		return ""
	}

	// 🔴 УЛУЧШЕНИЕ: Градация силы сигнала
	bullishRatio := float64(bullishScore) / float64(totalWeightedScore)
	bearishRatio := float64(bearishScore) / float64(totalWeightedScore)

	switch {
	case bullishRatio > 0.7:
		if bullishScore >= 6 {
			primarySignal = "🟢🔼 СИЛЬНЫЕ БЫЧЬИ СИГНАЛЫ"
		} else if bullishScore >= 3 {
			primarySignal = "🟢 Бычьи сигналы"
		} else {
			primarySignal = "🟡 Слабые бычьи сигналы"
		}

	case bearishRatio > 0.7:
		if bearishScore >= 6 {
			primarySignal = "🔴🔽 СИЛЬНЫЕ МЕДВЕЖЬИ СИГНАЛЫ"
		} else if bearishScore >= 3 {
			primarySignal = "🔴 Медвежьи сигналы"
		} else {
			primarySignal = "🟠 Слабые медвежьи сигналы"
		}

	case bullishRatio > bearishRatio && bullishRatio > 0.4:
		if bullishScore-bearishScore >= 3 {
			primarySignal = "🟢 Преобладают бычьи сигналы"
		} else {
			primarySignal = "🟡 Слабый бычий перевес"
		}

	case bearishRatio > bullishRatio && bearishRatio > 0.4:
		if bearishScore-bullishScore >= 3 {
			primarySignal = "🔴 Преобладают медвежьи сигналы"
		} else {
			primarySignal = "🟠 Слабый медвежий перевес"
		}

	default:
		if neutralScore > 0 {
			scoreDiff := math.Abs(float64(bullishScore - bearishScore))
			if scoreDiff <= 1 {
				primarySignal = "⚪ СБАЛАНСИРОВАННЫЕ СИГНАЛЫ"
			} else {
				primarySignal = "⚪ Смешанные сигналы"
			}
		} else {
			primarySignal = "🟡 ПРОТИВОРЕЧИВЫЕ СИГНАЛЫ"
		}
	}

	// 🔴 ИСПРАВЛЕНИЕ: Формируем итоговое сообщение без дублирующихся иконок
	result := primarySignal + "\n"
	for i, rec := range recommendations {
		// Определяем соответствующую иконку для каждой рекомендации
		lowerRec := strings.ToLower(rec)
		var icon string

		switch {
		case strings.Contains(lowerRec, "дельта покупок"):
			icon = "📈"
		case strings.Contains(lowerRec, "дельта продаж"):
			icon = "📉"
		case strings.Contains(lowerRec, "long"):
			icon = "📈"
		case strings.Contains(lowerRec, "short"):
			icon = "📉"
		case strings.Contains(lowerRec, "рост"):
			icon = "📈"
		case strings.Contains(lowerRec, "падение"):
			icon = "📉"
		case strings.Contains(lowerRec, "бычий"):
			icon = "📈"
		case strings.Contains(lowerRec, "медвежий"):
			icon = "📉"
		case strings.Contains(lowerRec, "покуп"):
			icon = "📈"
		case strings.Contains(lowerRec, "продаж"):
			icon = "📉"
		case strings.Contains(lowerRec, "⚠️"):
			icon = "⚠️"
		case strings.Contains(lowerRec, "🔄"):
			icon = "🔄"
		case strings.Contains(lowerRec, "💥"):
			icon = "💥"
		case strings.Contains(lowerRec, "✅"):
			icon = "✅"
		case strings.Contains(lowerRec, "🟡"):
			icon = "🟡"
		case strings.Contains(lowerRec, "rsi"):
			icon = "📊"
		case strings.Contains(lowerRec, "macd"):
			icon = "📈"
		default:
			// 🔴 ИСПРАВЛЕНИЕ: Если строка уже содержит эмодзи в начале, не добавляем повторно
			if len(rec) > 0 {
				// Проверяем первый символ как руну
				firstRune := []rune(rec)[0]
				// Проверяем, является ли руна эмодзи
				if (firstRune >= 0x1F600 && firstRune <= 0x1F64F) || // Эмодзи диапазон лиц
					(firstRune >= 0x1F300 && firstRune <= 0x1F5FF) || // Символы и пиктограммы
					(firstRune >= 0x1F680 && firstRune <= 0x1F6FF) { // Транспорт и карты
					// Уже есть эмодзи в начале
					icon = ""
				} else {
					icon = "•"
				}
			} else {
				icon = "•"
			}
		}

		// 🔴 ИСПРАВЛЕНИЕ: Убираем дублирующиеся иконки в строке
		cleanRec := rec
		if icon != "" && strings.HasPrefix(cleanRec, icon+" ") {
			// Если строка уже начинается с этой иконки, не добавляем еще раз
			cleanRec = strings.TrimPrefix(cleanRec, icon+" ")
		}

		result += fmt.Sprintf("%d. %s%s\n", i+1,
			func() string {
				if icon != "" {
					return icon + " "
				}
				return ""
			}(),
			cleanRec)
	}

	// 🔴 УЛУЧШЕНИЕ: Добавляем итоговую оценку
	result += fmt.Sprintf("\n🎯 ИТОГ: %s движение с %s дельтой объемов",
		priceDirectionStrength,
		func() string {
			if math.Abs(volumeDelta) > 50000 {
				return "сильной"
			} else if math.Abs(volumeDelta) > 10000 {
				return "умеренной"
			} else {
				return "слабой"
			}
		}())

	return strings.TrimSpace(result)
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

// FormatMessageWithFullDelta создает сообщение с полными данными дельты
func (f *MarketMessageFormatter) FormatMessageWithFullDelta(
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
	volumeDelta *bybit.VolumeDelta, // 🔴 Полные данные дельты
	rsi float64,
	macdSignal float64,
) string {
	// Извлекаем данные из volumeDelta
	var delta, deltaPercent, buyVolume, sellVolume float64
	var totalTrades int
	var isRealData bool

	if volumeDelta != nil {
		delta = volumeDelta.Delta
		deltaPercent = volumeDelta.DeltaPercent
		buyVolume = volumeDelta.BuyVolume
		sellVolume = volumeDelta.SellVolume
		totalTrades = volumeDelta.TotalTrades
		isRealData = true

		// Логируем источник данных
		log.Printf("📊 Используем реальные данные дельты для %s:", symbol)
		log.Printf("   Период: %s", volumeDelta.Period)
		log.Printf("   Время: %s - %s",
			volumeDelta.StartTime.Format("15:04:05"),
			volumeDelta.EndTime.Format("15:04:05"))
		log.Printf("   Покупки: $%.0f, Продажи: $%.0f", buyVolume, sellVolume)
		log.Printf("   Дельта: $%.0f (%.1f%%)", delta, deltaPercent)
		log.Printf("   Сделок: %d", totalTrades)
	} else {
		isRealData = false
		log.Printf("⚠️ Данные дельты для %s не получены", symbol)
	}

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

	// Дельта объемов с полной информацией
	if volumeDelta != nil {
		deltaStr := f.formatVolumeDeltaWithFullInfo(
			delta, deltaPercent, direction,
			buyVolume, sellVolume, totalTrades,
			isRealData,
		)
		builder.WriteString(fmt.Sprintf("📈 Дельта: %s\n\n", deltaStr))
	} else {
		builder.WriteString("\n")
	}

	// ==================== БЛОК 4: ТЕХНИЧЕСКИЙ АНАЛИЗ ====================
	if rsi > 0 || macdSignal != 0 {
		builder.WriteString(fmt.Sprintf("📊 Тех. анализ:\n"))

		if rsi > 0 {
			rsiStr := f.formatRSI(rsi)
			builder.WriteString(fmt.Sprintf("%s\n", rsiStr))
		}

		if macdSignal != 0 {
			macdStr := f.formatMACD(macdSignal)
			builder.WriteString(fmt.Sprintf("%s\n", macdStr))
		}

		builder.WriteString("\n")
	}

	// ==================== БЛОК 5: ЛИКВИДАЦИИ ====================
	if liquidationVolume > 0 && volume24h > 0 {
		longPercent := safeDivide(longLiqVolume, liquidationVolume) * 100
		shortPercent := safeDivide(shortLiqVolume, liquidationVolume) * 100
		volumePercent := safeDivide(liquidationVolume, volume24h) * 100

		liqPeriod := "5мин"
		if strings.Contains(period, "15") {
			liqPeriod = "15мин"
		} else if strings.Contains(period, "30") {
			liqPeriod = "30мин"
		} else if strings.Contains(period, "1 час") {
			liqPeriod = "1ч"
		}

		builder.WriteString(fmt.Sprintf("💥 ЛИКВИДАЦИИ (%s)\n", liqPeriod))

		liqStr := f.formatDollarValue(liquidationVolume)
		if volumePercent > 0 {
			builder.WriteString(fmt.Sprintf("$%s • %.2f%% от объема\n", liqStr, volumePercent))
		} else {
			builder.WriteString(fmt.Sprintf("$%s\n", liqStr))
		}

		longBar := f.formatCompactBar(longPercent, "🟢")
		shortBar := f.formatCompactBar(shortPercent, "🔴")

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
	percentage := float64(signalCount) / float64(maxSignals) * 100
	progressBar := f.formatCompactProgressBar(percentage)

	builder.WriteString(fmt.Sprintf("📡 %d/%d %s (%.0f%%)\n",
		signalCount, maxSignals, progressBar, percentage))

	// Период анализа
	builder.WriteString(fmt.Sprintf("🕐 Период: %s\n\n", period))

	// ==================== БЛОК 7: РЕКОМЕНДАЦИИ ПО ТОРГОВЛЕ ====================
	recommendation := f.getEnhancedTradingRecommendationWithFullDelta(
		direction, rsi, macdSignal,
		volumeDelta, isRealData,
		longLiqVolume, shortLiqVolume,
	)
	if recommendation != "" {
		builder.WriteString(fmt.Sprintf("🎯 Рекомендация:\n%s\n\n", recommendation))
	}

	// ==================== БЛОК 8: ФАНДИНГ ====================
	fundingStr := f.formatFundingWithEmoji(fundingRate)
	timeUntil := f.formatCompactTime(nextFundingTime)

	builder.WriteString(fmt.Sprintf("🎯 Фандинг: %s\n", fundingStr))
	builder.WriteString(fmt.Sprintf("⏰ Через: %s", timeUntil))

	return builder.String()
}

// safeDivide безопасное деление
func safeDivide(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}

// formatVolumeDeltaWithFullInfo форматирует дельту с полной информацией
func (f *MarketMessageFormatter) formatVolumeDeltaWithFullInfo(
	delta, deltaPercent float64,
	direction string,
	buyVolume, sellVolume float64,
	totalTrades int,
	isRealData bool,
) string {
	// Базовое форматирование
	baseString := f.formatVolumeDelta(delta, deltaPercent, direction)

	// Добавляем источник данных
	sourceIndicator := ""
	if !isRealData {
		sourceIndicator = " [эмуляция]"
	}

	var details strings.Builder

	// Если есть данные о покупках/продажах
	if buyVolume > 0 && sellVolume > 0 {
		// Рассчитываем соотношение
		totalVolume := buyVolume + sellVolume
		buyRatio := (buyVolume / totalVolume) * 100
		sellRatio := 100 - buyRatio

		// Форматируем объемы
		buyStr := f.formatDollarValue(buyVolume)
		sellStr := f.formatDollarValue(sellVolume)

		// Определяем доминирующую сторону
		var dominanceIcon string
		if buyRatio > 55 {
			dominanceIcon = "🟢"
		} else if sellRatio > 55 {
			dominanceIcon = "🔴"
		} else {
			dominanceIcon = "⚪"
		}

		details.WriteString(fmt.Sprintf("\n   %s Покупки: $%s (%.0f%%)",
			dominanceIcon, buyStr, buyRatio))
		details.WriteString(fmt.Sprintf("\n   %s Продажи: $%s (%.0f%%)",
			dominanceIcon, sellStr, sellRatio))
	}

	// Если есть данные о количестве сделок
	if totalTrades > 0 {
		// Рассчитываем активность (сделок в минуту)
		// Предполагаем период 5 минут для real-time дельты
		tradesPerMinute := float64(totalTrades) / 5.0
		var activityIcon string

		switch {
		case tradesPerMinute > 50:
			activityIcon = "⚡" // Очень высокая активность
		case tradesPerMinute > 20:
			activityIcon = "🔥" // Высокая активность
		case tradesPerMinute > 5:
			activityIcon = "📊" // Средняя активность
		default:
			activityIcon = "📉" // Низкая активность
		}

		details.WriteString(fmt.Sprintf("\n   %s Сделок: %d (%.1f/мин)",
			activityIcon, totalTrades, tradesPerMinute))
	}

	// Добавляем индикатор источника данных
	if details.Len() > 0 {
		details.WriteString(fmt.Sprintf("\n   📡 Источник: %s",
			func() string {
				if isRealData {
					return "API Bybit"
				} else {
					return "Эмуляция (2% от объема)"
				}
			}()))
	}

	if details.Len() > 0 {
		return baseString + sourceIndicator + details.String()
	}

	return baseString + sourceIndicator
}

// getEnhancedTradingRecommendationWithFullDelta улучшенные рекомендации с полными данными дельты
func (f *MarketMessageFormatter) getEnhancedTradingRecommendationWithFullDelta(
	direction string,
	rsi float64,
	macdSignal float64,
	volumeDelta *bybit.VolumeDelta,
	isRealData bool,
	longLiqVolume, shortLiqVolume float64,
) string {
	var recommendations []string

	// Добавляем информацию о качестве данных
	if !isRealData {
		recommendations = append(recommendations, "⚠️ Внимание: данные дельты эмулированы")
	}

	if volumeDelta == nil {
		// Без данных дельты - используем базовые рекомендации
		return f.getEnhancedTradingRecommendation(
			direction, rsi, macdSignal, 0, 0, longLiqVolume, shortLiqVolume,
		)
	}

	// Анализ объема сделок
	if volumeDelta.TotalTrades > 0 {
		tradesPerMinute := float64(volumeDelta.TotalTrades) / 5.0

		switch {
		case tradesPerMinute > 50:
			recommendations = append(recommendations,
				fmt.Sprintf("📊 Высокая торговая активность: %.1f сделок/мин", tradesPerMinute))
		case tradesPerMinute > 20:
			recommendations = append(recommendations,
				fmt.Sprintf("📊 Средняя активность: %.1f сделок/мин", tradesPerMinute))
		case volumeDelta.TotalTrades < 10:
			recommendations = append(recommendations,
				"📊 Низкая торговая активность")
		}
	}

	// Анализ соотношения покупок/продаж
	if volumeDelta.BuyVolume > 0 && volumeDelta.SellVolume > 0 {
		totalVolume := volumeDelta.BuyVolume + volumeDelta.SellVolume
		buyRatio := (volumeDelta.BuyVolume / totalVolume) * 100
		sellRatio := 100 - buyRatio

		// Определяем дисбаланс
		var imbalance string
		if buyRatio > 60 {
			imbalance = fmt.Sprintf("сильный перевес покупок (%.0f%%)", buyRatio)
		} else if sellRatio > 60 {
			imbalance = fmt.Sprintf("сильный перевес продаж (%.0f%%)", sellRatio)
		} else if math.Abs(buyRatio-50) > 10 {
			imbalance = fmt.Sprintf("умеренный перевес %s (%.0f%%)",
				func() string {
					if buyRatio > 50 {
						return "покупок"
					} else {
						return "продаж"
					}
				}(),
				math.Max(buyRatio, sellRatio))
		}

		if imbalance != "" {
			recommendations = append(recommendations,
				fmt.Sprintf("📈 Дисбаланс объемов: %s", imbalance))
		}

		// Анализ качества сделок
		averageTradeSize := totalVolume / float64(volumeDelta.TotalTrades)
		if averageTradeSize > 10000 {
			recommendations = append(recommendations,
				fmt.Sprintf("💰 Крупные сделки: $%.0f в среднем", averageTradeSize))
		} else if averageTradeSize < 100 {
			recommendations = append(recommendations,
				"💰 Мелкие сделки преобладают")
		}
	}

	// Анализ дельты
	if math.Abs(volumeDelta.Delta) > 0 {
		deltaDirection := "покупок"
		if volumeDelta.Delta < 0 {
			deltaDirection = "продаж"
		}

		strength := "слабая"
		if math.Abs(volumeDelta.Delta) > 50000 {
			strength = "сильная"
		} else if math.Abs(volumeDelta.Delta) > 10000 {
			strength = "умеренная"
		}

		recommendations = append(recommendations,
			fmt.Sprintf("📈 %s дельта %s ($%.0f)", strength, deltaDirection, math.Abs(volumeDelta.Delta)))
	}

	// Добавляем базовые рекомендации
	baseRecommendations := f.getEnhancedTradingRecommendation(
		direction, rsi, macdSignal,
		volumeDelta.Delta, volumeDelta.DeltaPercent,
		longLiqVolume, shortLiqVolume,
	)

	if baseRecommendations != "" {
		// Парсим базовые рекомендации и добавляем к нашим
		lines := strings.Split(baseRecommendations, "\n")
		for _, line := range lines[1:] { // Пропускаем заголовок
			if strings.TrimSpace(line) != "" {
				recommendations = append(recommendations, line)
			}
		}
	}

	// Если рекомендаций нет
	if len(recommendations) == 0 {
		return ""
	}

	// Формируем итоговое сообщение
	var result strings.Builder

	// Определяем общий сигнал
	if isRealData {
		result.WriteString("📊 Анализ на основе реальных данных:\n")
	} else {
		result.WriteString("📊 Анализ на основе эмулированных данных:\n")
	}

	for _, rec := range recommendations {
		// Добавляем маркер
		var marker string
		if strings.Contains(rec, "⚠️") || strings.Contains(rec, "Внимание") {
			marker = "⚠️"
		} else if strings.Contains(rec, "💰") {
			marker = "💰"
		} else if strings.Contains(rec, "📊") {
			marker = "📊"
		} else if strings.Contains(rec, "📈") {
			marker = "📈"
		} else {
			marker = "•"
		}

		result.WriteString(fmt.Sprintf("%s %s\n", marker, rec))
	}

	return strings.TrimSpace(result.String())
}

// formatVolumeDeltaWithSource форматирует дельту с указанием источника
func (f *MarketMessageFormatter) formatVolumeDeltaWithSource(
	deltaData *types.VolumeDeltaData,
	direction string,
) string {
	// Если данных нет
	if deltaData.Delta == 0 && deltaData.DeltaPercent == 0 {
		return "─"
	}

	// Форматируем базовую дельту
	baseString := f.formatVolumeDelta(deltaData.Delta, deltaData.DeltaPercent, direction)

	// Добавляем индикатор источника
	var sourceIndicator string
	switch deltaData.Source {
	case types.VolumeDeltaSourceAPI:
		sourceIndicator = " [API]"
	case types.VolumeDeltaSourceStorage:
		sourceIndicator = " [Хранилище]"
	case types.VolumeDeltaSourceEmulated:
		sourceIndicator = " [Эмуляция]"
	case types.VolumeDeltaSourceCache:
		sourceIndicator = " [Кэш]"
	default:
		sourceIndicator = ""
	}

	return baseString + sourceIndicator
}
