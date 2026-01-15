// internal/delivery/telegram/app/bot/formatters/recommendation/scorer.go
package recommendation

import (
	"fmt"
	"math"
	"strings"
)

// Scorer подсчитывает баллы для рекомендаций
type Scorer struct{}

// NewScorer создает новый счетчик
func NewScorer() *Scorer {
	return &Scorer{}
}

// SignalScores баллы сигналов
type SignalScores struct {
	BullishScore int
	BearishScore int
	NeutralScore int
	WarningScore int // НОВОЕ: баллы предупреждений
}

// CalculateSignalScores подсчитывает баллы для сигналов
func (s *Scorer) CalculateSignalScores(recommendations []string) SignalScores {
	scores := SignalScores{}

	for _, rec := range recommendations {
		lowerRec := strings.ToLower(rec)

		// 1. СИЛЬНЫЕ МЕДВЕЖЬИ СИГНАЛЫ (RSI перекупленность)
		if strings.Contains(lowerRec, "осторожность с long") ||
			strings.Contains(lowerRec, "перекупленности") ||
			strings.Contains(lowerRec, "перекупленность") ||
			(strings.Contains(lowerRec, "rsi") && (strings.Contains(lowerRec, "70") || strings.Contains(lowerRec, "69"))) {
			scores.BearishScore += 3 // Сильный медвежий сигнал
			scores.WarningScore += 2 // Высокий уровень предупреждения

			// 2. СИЛЬНЫЕ БЫЧЬИ СИГНАЛЫ (RSI перепроданность)
		} else if strings.Contains(lowerRec, "осторожность с short") ||
			strings.Contains(lowerRec, "перепроданности") ||
			strings.Contains(lowerRec, "перепроданность") ||
			(strings.Contains(lowerRec, "rsi") && (strings.Contains(lowerRec, "30") || strings.Contains(lowerRec, "31"))) {
			scores.BullishScore += 3

			// 3. MACD СИГНАЛЫ (умеренный вес)
		} else if strings.Contains(lowerRec, "macd: сильный бычий") {
			scores.BullishScore += 2
		} else if strings.Contains(lowerRec, "macd: сильный медвежий") {
			scores.BearishScore += 2
		} else if strings.Contains(lowerRec, "macd: бычий") {
			scores.BullishScore += 1
		} else if strings.Contains(lowerRec, "macd: медвежий") {
			scores.BearishScore += 1
		} else if strings.Contains(lowerRec, "macd: слабый бычий") {
			scores.BullishScore += 1
		} else if strings.Contains(lowerRec, "macd: слабый медвежий") {
			scores.BearishScore += 1

			// 4. ДЕЛЬТА ОБЪЕМОВ
		} else if strings.Contains(lowerRec, "дельта покупок") && !strings.Contains(lowerRec, "при падении") {
			if strings.Contains(lowerRec, "сильная") {
				scores.BullishScore += 2
			} else if strings.Contains(lowerRec, "умеренная") {
				scores.BullishScore += 1
			}
		} else if strings.Contains(lowerRec, "дельта продаж") && !strings.Contains(lowerRec, "при росте") {
			if strings.Contains(lowerRec, "сильная") {
				scores.BearishScore += 2
			} else if strings.Contains(lowerRec, "умеренная") {
				scores.BearishScore += 1
			}

			// 5. КОНТРАДИКЦИИ (дельта противоречит направлению)
		} else if strings.Contains(lowerRec, "дельта продаж при росте") ||
			strings.Contains(lowerRec, "дельта покупок при падении") {
			scores.WarningScore += 2 // Высокий риск разворота
			scores.NeutralScore += 1

			// 6. ПОДТВЕРЖДЕНИЯ ОБЪЕМОВ
		} else if strings.Contains(lowerRec, "объемы подтверждают") {
			// Усиливаем существующий сигнал
			scores.BullishScore += 1
			scores.BearishScore += 1
		} else if strings.Contains(lowerRec, "объемы слабо подтверждают") {
			scores.NeutralScore += 1
		} else if strings.Contains(lowerRec, "объемы противоречат") {
			scores.WarningScore += 1
			scores.NeutralScore += 1

			// 7. ПРОТИВОРЕЧИЯ
		} else if strings.Contains(lowerRec, "противоречие") {
			scores.WarningScore += 2
			scores.NeutralScore += 2

			// 8. ЛИКВИДАЦИИ
		} else if strings.Contains(lowerRec, "long ликвидации") {
			scores.BullishScore += 1 // SHORT ликвидации = бычий сигнал
		} else if strings.Contains(lowerRec, "short ликвидации") {
			scores.BearishScore += 1 // LONG ликвидации = медвежий сигнал

			// 9. НЕЙТРАЛЬНЫЕ СИГНАЛЫ
		} else if strings.Contains(lowerRec, "нейтральный") ||
			strings.Contains(lowerRec, "слабый") ||
			strings.Contains(lowerRec, "незначительный") {
			scores.NeutralScore += 1
		}
	}

	return scores
}

// DeterminePrimarySignal определяет основной сигнал с учетом предупреждений
func (s *Scorer) DeterminePrimarySignal(
	scores SignalScores,
	recommendations []string,
) string {
	// Применяем штрафы за предупреждения
	if scores.WarningScore >= 3 {
		// Сильные предупреждения доминируют
		return "⚠️ ВЫСОКИЙ РИСК: ПРОТИВОРЕЧИВЫЕ СИГНАЛЫ"
	}

	totalScore := scores.BullishScore + scores.BearishScore + scores.NeutralScore
	if totalScore == 0 {
		return ""
	}

	// Проверяем наличие сильных RSI сигналов
	hasStrongBearishRSI := false
	hasStrongBullishRSI := false

	for _, rec := range recommendations {
		lowerRec := strings.ToLower(rec)
		if strings.Contains(lowerRec, "осторожность с long") {
			hasStrongBearishRSI = true
		}
		if strings.Contains(lowerRec, "осторожность с short") {
			hasStrongBullishRSI = true
		}
	}

	// RSI ПЕРЕКУПЛЕННОСТЬ = сильный медвежий сигнал (даже если MACD бычий)
	if hasStrongBearishRSI && scores.BearishScore > scores.BullishScore {
		return "🔴 ВЫСОКИЙ РИСК: ПЕРЕКУПЛЕННОСТЬ RSI"
	}

	// RSI ПЕРЕПРОДАННОСТЬ = сильный бычий сигнал
	if hasStrongBullishRSI && scores.BullishScore > scores.BearishScore {
		return "🟢 ВОЗМОЖЕН ОТСКОК: ПЕРЕПРОДАННОСТЬ RSI"
	}

	bullishRatio := float64(scores.BullishScore) / float64(totalScore)
	bearishRatio := float64(scores.BearishScore) / float64(totalScore)
	neutralRatio := float64(scores.NeutralScore) / float64(totalScore)

	// Логика определения с учетом предупреждений
	switch {
	case scores.WarningScore >= 2:
		return "⚠️ ПРОТИВОРЕЧИВЫЕ СИГНАЛЫ - ОСТОРОЖНОСТЬ"

	case hasStrongBearishRSI:
		// Перекупленность RSI имеет приоритет
		if scores.BullishScore > 0 {
			return "🟡 СМЕШАННЫЕ СИГНАЛЫ: RSI перекупленность"
		}
		return "🔴 ПРЕДУПРЕЖДЕНИЕ: ПЕРЕКУПЛЕННОСТЬ"

	case hasStrongBullishRSI:
		// Перепроданность RSI имеет приоритет
		if scores.BearishScore > 0 {
			return "🟡 СМЕШАННЫЕ СИГНАЛЫ: RSI перепроданность"
		}
		return "🟢 ВОЗМОЖНОСТЬ: ПЕРЕПРОДАННОСТЬ"

	case bullishRatio > 0.6 && scores.BullishScore >= 4:
		return "🟢 БЫЧЬИ СИГНАЛЫ"

	case bearishRatio > 0.6 && scores.BearishScore >= 4:
		return "🔴 МЕДВЕЖЬИ СИГНАЛЫ"

	case bullishRatio > bearishRatio && bullishRatio > 0.4:
		if scores.BullishScore-scores.BearishScore >= 2 {
			return "🟡 ПРЕОБЛАДАЮТ БЫЧЬИ СИГНАЛЫ"
		}
		return "🟡 СЛАБЫЙ БЫЧИЙ ПЕРЕВЕС"

	case bearishRatio > bullishRatio && bearishRatio > 0.4:
		if scores.BearishScore-scores.BullishScore >= 2 {
			return "🟠 ПРЕОБЛАДАЮТ МЕДВЕЖЬИ СИГНАЛЫ"
		}
		return "🟠 СЛАБЫЙ МЕДВЕЖИЙ ПЕРЕВЕС"

	case neutralRatio > 0.5 || math.Abs(float64(scores.BullishScore-scores.BearishScore)) <= 1:
		return "⚪ СБАЛАНСИРОВАННЫЕ СИГНАЛЫ"

	default:
		return "🟡 СМЕШАННЫЕ СИГНАЛЫ"
	}
}

// GetTradingAction возвращает конкретное торговое действие
func (s *Scorer) GetTradingAction(
	scores SignalScores,
	recommendations []string,
	rsi float64,
	changePercent float64,
	volumeDelta float64,
) string {
	// 1. ПРОВЕРКА СИЛЬНЫХ ПРОТИВОПОКАЗАНИЙ

	// RSI перекупленность > 70 = НЕ открывать LONG
	if rsi >= 70 {
		return "❌ НЕ ОТКРЫВАТЬ LONG: RSI в перекупленности"
	}

	// RSI перепроданность < 30 = НЕ открывать SHORT
	if rsi <= 30 {
		return "❌ НЕ ОТКРЫВАТЬ SHORT: RSI в перепроданности"
	}

	// Сильные противоречия в объемах
	hasVolumeContradiction := false
	for _, rec := range recommendations {
		if strings.Contains(strings.ToLower(rec), "объемы противоречат") ||
			strings.Contains(strings.ToLower(rec), "дельта покупок при падении") ||
			strings.Contains(strings.ToLower(rec), "дельта продаж при росте") {
			hasVolumeContradiction = true
			break
		}
	}

	if hasVolumeContradiction && scores.WarningScore >= 3 {
		return "⏸️ ЖДАТЬ: противоречие объемов"
	}

	// 2. АНАЛИЗ СИГНАЛОВ ДЛЯ ЛОНГ

	longConditions := 0
	shortConditions := 0

	// ЛОНГ условия:
	// 1. Направление роста
	// 2. RSI < 65 (не перекуплен)
	// 3. MACD бычий
	// 4. Дельта покупок > 0
	// 5. Изменение > 0.5%

	if changePercent > 0.5 {
		longConditions++
	}

	if rsi < 65 && rsi > 40 { // Идеальная зона для лонга
		longConditions++
	}

	// Проверяем MACD и дельты
	for _, rec := range recommendations {
		lowerRec := strings.ToLower(rec)
		if strings.Contains(lowerRec, "macd: бычий") ||
			strings.Contains(lowerRec, "macd: сильный бычий") {
			longConditions++
		}
		if strings.Contains(lowerRec, "дельта покупок") && !strings.Contains(lowerRec, "при падении") {
			longConditions++
		}
	}

	// 3. АНАЛИЗ СИГНАЛОВ ДЛЯ ШОРТ

	// ШОРТ условия:
	// 1. Направление падения
	// 2. RSI > 35 (не перепродан)
	// 3. MACD медвежий
	// 4. Дельта продаж > 0
	// 5. Изменение < -0.5%

	if changePercent < -0.5 {
		shortConditions++
	}

	if rsi > 35 && rsi < 60 { // Идеальная зона для шорта
		shortConditions++
	}

	for _, rec := range recommendations {
		lowerRec := strings.ToLower(rec)
		if strings.Contains(lowerRec, "macd: медвежий") ||
			strings.Contains(lowerRec, "macd: сильный медвежий") {
			shortConditions++
		}
		if strings.Contains(lowerRec, "дельта продаж") && !strings.Contains(lowerRec, "при росте") {
			shortConditions++
		}
	}

	// 4. ПРИНЯТИЕ РЕШЕНИЯ

	// СИЛЬНЫЙ ЛОНГ
	if longConditions >= 4 && shortConditions <= 1 {
		return "✅ ОТКРЫТЬ ЛОНГ: сильные бычьи сигналы"
	}

	// УМЕРЕННЫЙ ЛОНГ
	if longConditions >= 3 && shortConditions <= 1 {
		// Проверяем RSI
		if rsi < 62 {
			return "🟢 ОТКРЫТЬ ЛОНГ: умеренные бычьи сигналы"
		}
		return "🟡 ОТКРЫТЬ ЛОНГ (малый размер): RSI близко к перекупленности"
	}

	// СИЛЬНЫЙ ШОРТ
	if shortConditions >= 4 && longConditions <= 1 {
		return "✅ ОТКРЫТЬ ШОРТ: сильные медвежьи сигналы"
	}

	// УМЕРЕННЫЙ ШОРТ
	if shortConditions >= 3 && longConditions <= 1 {
		// Проверяем RSI
		if rsi > 38 {
			return "🔴 ОТКРЫТЬ ШОРТ: умеренные медвежьи сигналы"
		}
		return "🟠 ОТКРЫТЬ ШОРТ (малый размер): RSI близко к перепроданности"
	}

	// ПРОТИВОРЕЧИВЫЕ СИГНАЛЫ
	if longConditions >= 2 && shortConditions >= 2 {
		// Сравниваем силу
		longStrength := scores.BullishScore
		shortStrength := scores.BearishScore

		if longStrength > shortStrength+2 {
			return "🟡 ОТКРЫТЬ ЛОНГ (очень малый размер): смешанные сигналы"
		} else if shortStrength > longStrength+2 {
			return "🟠 ОТКРЫТЬ ШОРТ (очень малый размер): смешанные сигналы"
		}
		return "⏸️ ЖДАТЬ: противоречивые сигналы"
	}

	// СЛАБЫЕ СИГНАЛЫ
	if longConditions == 2 && shortConditions <= 1 {
		return "🟡 РАССМОТРЕТЬ ЛОНГ: слабые сигналы"
	}

	if shortConditions == 2 && longConditions <= 1 {
		return "🟠 РАССМОТРЕТЬ ШОРТ: слабые сигналы"
	}

	// НЕТ ЯСНЫХ СИГНАЛОВ
	return "⏸️ ЖДАТЬ: недостаточно четких сигналов"
}

// GetEntryRecommendation возвращает полную рекомендацию по входу
func (s *Scorer) GetEntryRecommendation(
	recommendations []string,
	rsi float64,
	changePercent float64,
	volumeDelta float64,
	currentPrice float64,
) string {
	scores := s.CalculateSignalScores(recommendations)
	action := s.GetTradingAction(scores, recommendations, rsi, changePercent, volumeDelta)

	var result strings.Builder
	result.WriteString("🎯 ТОРГОВАЯ РЕКОМЕНДАЦИЯ:\n")
	result.WriteString(action + "\n\n")

	// Добавляем уровни стоп-лосса и тейк-профита
	if strings.Contains(action, "ЛОНГ") || strings.Contains(action, "ШОРТ") {
		result.WriteString("📊 УРОВНИ:\n")

		// Рассчитываем стоп-лосс на основе волатильности (примерно 2%)
		stopLossPercent := 2.0
		takeProfitPercent := 4.0 // Риск:прибыль = 1:2

		if strings.Contains(action, "ЛОНГ") {
			stopPrice := currentPrice * (1 - stopLossPercent/100)
			takeProfitPrice := currentPrice * (1 + takeProfitPercent/100)

			result.WriteString(fmt.Sprintf("Стоп-лосс: $%.4f (%.1f%%)\n", stopPrice, stopLossPercent))
			result.WriteString(fmt.Sprintf("Тейк-профит: $%.4f (%.1f%%)\n", takeProfitPrice, takeProfitPercent))
			result.WriteString(fmt.Sprintf("Риск/Прибыль: 1:%.1f\n", takeProfitPercent/stopLossPercent))

		} else if strings.Contains(action, "ШОРТ") {
			stopPrice := currentPrice * (1 + stopLossPercent/100)
			takeProfitPrice := currentPrice * (1 - takeProfitPercent/100)

			result.WriteString(fmt.Sprintf("Стоп-лосс: $%.4f (%.1f%%)\n", stopPrice, stopLossPercent))
			result.WriteString(fmt.Sprintf("Тейк-профит: $%.4f (%.1f%%)\n", takeProfitPrice, takeProfitPercent))
			result.WriteString(fmt.Sprintf("Риск/Прибыль: 1:%.1f\n", takeProfitPercent/stopLossPercent))
		}

		// Добавляем рекомендацию по размеру позиции
		result.WriteString("\n📈 РАЗМЕР ПОЗИЦИИ:\n")

		// Определяем агрессивность на основе баллов
		totalConfidence := scores.BullishScore + scores.BearishScore
		if strings.Contains(action, "сильные") || totalConfidence >= 6 {
			result.WriteString("Рекомендуемый размер: 2-3% капитала\n")
		} else if strings.Contains(action, "умеренные") || totalConfidence >= 4 {
			result.WriteString("Рекомендуемый размер: 1-2% капитала\n")
		} else {
			result.WriteString("Рекомендуемый размер: 0.5-1% капитала\n")
		}
	}

	return result.String()
}
